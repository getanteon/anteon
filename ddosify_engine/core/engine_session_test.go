package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/config"
	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/types"
)

// Sessions of one or two turns, one of which is refused: a skipped turn and the turns after a
// failure are neither sent nor counted anywhere in the report.
func TestVariableLengthSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat" {
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) == "fail" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"text-delta\",\"delta\":\"x\"}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()

	dir := t.TempDir()
	csv := filepath.Join(dir, "sessions.csv")
	if err := os.WriteFile(csv, []byte("a,b\na,\nfail,c\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	conf := fmt.Sprintf(`{
		"load_spread": "poisson", "seed": 5, "stop_on_failure": true, "request_log": true, "output": "stdout-json",
		"slo": {"total_ms": 10000},
		"manual_load": [{"duration": 1, "count": 6, "name": "load"}],
		"data": {"s": {"path": %q, "vars": {"0": {"tag": "t1"}, "1": {"tag": "t2"}}, "order": "sequential"}},
		"steps": [
			{"id": 1, "name": "create", "url": "%[2]s/chat"},
			{"id": 2, "name": "turn 1", "url": "%[2]s/ask", "method": "POST", "payload": "{{data.s.t1}}", "stream": {"type": "sse"}, "sleep": "10"},
			{"id": 3, "name": "turn 2", "url": "%[2]s/ask", "method": "POST", "payload": "{{data.s.t2}}", "stream": {"type": "sse"}, "skip_if_empty": "{{data.s.t2}}"}
		]}`, csv, server.URL)

	reader, err := config.NewConfigReader([]byte(conf), config.ConfigTypeJson)
	if err != nil {
		t.Fatal(err)
	}
	h, err := reader.CreateHammer()
	if err != nil {
		t.Fatal(err)
	}
	if err = h.Validate(); err != nil {
		t.Fatal(err)
	}
	if !h.StopOnFailure || !h.RequestLog || h.Seed != 5 || h.Scenario.Steps[2].SkipIfEmpty != "{{data.s.t2}}" {
		t.Fatalf("config not read: %+v", h)
	}
	es, err := InitEngineServices(h)
	if err != nil {
		t.Fatal(err)
	}
	e, err := NewEngine(context.Background(), h, es)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Init(); err != nil {
		t.Fatal(err)
	}
	e.Start()
	r := es.ReportServ.(report.ResultProvider).Result()

	// rows a,b twice: three steps; a, twice: two; fail,c twice: two, the second a 503.
	// The legacy counts take a 503 as an answer; the outcomes below do not.
	if r.SuccessCount != 6 || r.ServerFailedCount != 0 {
		t.Errorf("iterations: %d ok, %d failed", r.SuccessCount, r.ServerFailedCount)
	}
	if s := r.StepResults[2]; s.SuccessCount != 6 || s.StatusCodeDist[503] != 2 || s.Outcomes[report.OutcomeUnavailable] != 2 {
		t.Errorf("turn 1: %+v", s)
	}
	s3 := r.StepResults[3]
	if s3 == nil || s3.SuccessCount != 2 || s3.Fail.Count != 0 || len(s3.StatusCodeDist) != 1 || s3.Stream.Count != 2 || s3.Stream.Total.Count != 2 {
		t.Fatalf("turn 2 should count only the two sent: %+v", s3)
	}
	if s3.Outcomes[report.OutcomeOK] != 2 || len(s3.Outcomes) != 1 {
		t.Errorf("turn 2 outcomes %v", s3.Outcomes)
	}
	if r.Goodput == nil || r.Goodput.Iterations != 6 || r.Goodput.Good != 4 {
		t.Errorf("goodput %+v", r.Goodput)
	}
	if len(r.Stages) != 1 || r.Stages[0].Outcomes[report.OutcomeOK] != 12 || r.Stages[0].Outcomes[report.OutcomeUnavailable] != 2 ||
		len(r.Stages[0].Outcomes) != 2 || r.Stages[0].Stream.Count != 6 {
		t.Errorf("stage %+v", r.Stages[0])
	}
	if r.LLM != nil && r.LLM.Tokens != nil && r.LLM.Tokens.Requests != 6 {
		t.Errorf("llm requests %+v", r.LLM.Tokens)
	}
}

// An interrupt ends a poisson schedule: arrivals not yet due never start, and the run returns.
func TestPoissonInterrupt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	h := newDummyHammer()
	h.LoadSpread = types.LoadSpreadPoisson
	h.Seed = 1
	h.TimeRunCountMap = types.TimeRunCount{{Duration: 3, Count: 60, Name: "a"}}
	h.TestDuration, h.IterationCount = 3, 60
	h.Scenario.Steps[0].URL = server.URL

	es, err := InitEngineServices(h)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recordingReport{}
	rec.Init(false, 0)
	es.ReportServ = rec
	ctx, cancel := context.WithCancel(context.Background())
	e, err := NewEngine(ctx, h, es)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Init(); err != nil {
		t.Fatal(err)
	}
	time.AfterFunc(time.Second, cancel)
	begin := time.Now()
	if status := e.Start(); status != resultStopped {
		t.Errorf("expected stopped, got %s", status)
	}
	if n := len(rec.results); n == 0 || n >= 60 || time.Since(begin) > 2500*time.Millisecond {
		t.Errorf("%d of 60 iterations ran in %v", n, time.Since(begin))
	}
}
