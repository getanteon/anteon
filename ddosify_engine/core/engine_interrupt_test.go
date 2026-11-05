package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

// slowStream answers with a stream that takes 2 seconds.
func slowStream(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
			return
		}
		fmt.Fprint(w, "data: {\"type\":\"text-delta\"}\n\ndata: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runInterrupted(t *testing.T, drain, interruptAt time.Duration) (string, *recordingReport, time.Duration) {
	t.Helper()
	srv := slowStream(t)
	h := newDummyHammer()
	h.TestDuration = 1
	h.IterationCount = 1
	h.Drain = drain
	h.Scenario.Steps[0].URL = srv.URL
	h.Scenario.Steps[0].Timeout = 10
	h.Scenario.Steps[0].Stream = types.DefaultStreamConf()

	es, err := InitEngineServices(h)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recordingReport{}
	rec.Init(false, 0)
	es.ReportServ = rec
	ctx, interrupt := context.WithCancel(context.Background())
	defer interrupt()
	e, err := NewEngine(ctx, h, es)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Init(); err != nil {
		t.Fatal(err)
	}
	time.AfterFunc(interruptAt, interrupt)
	begin := time.Now()
	status := e.Start()
	return status, rec, time.Since(begin)
}

func TestInterruptAfterTheScheduleIsStoppedAndCountsInFlight(t *testing.T) {
	// the schedule ends at 1 s, the stream at about 2.1 s; the interrupt comes in between
	status, rec, took := runInterrupted(t, 100*time.Millisecond, 1300*time.Millisecond)
	if status != resultStopped {
		t.Errorf("an interrupt after the schedule is still a stop, got %q", status)
	}
	if len(rec.results) != 1 || !rec.results[0].Interrupted() {
		t.Fatalf("the iteration cut in flight is reported as interrupted: %+v", rec.results)
	}
	if took > 1800*time.Millisecond {
		t.Errorf("the drain cut the stream early, the run took %v", took)
	}
}

func TestDrainLetsInFlightFinish(t *testing.T) {
	status, rec, _ := runInterrupted(t, 5*time.Second, 1300*time.Millisecond)
	if status != resultStopped {
		t.Errorf("status %q", status)
	}
	if len(rec.results) != 1 || rec.results[0].Interrupted() || rec.results[0].StepResults[0].Stream == nil ||
		rec.results[0].StepResults[0].Stream.End != types.StreamEndDone {
		t.Fatalf("within the drain the stream finishes and is reported: %+v", rec.results)
	}
}

func TestCutEndsTheDrain(t *testing.T) {
	srv := slowStream(t)
	h := newDummyHammer()
	h.TestDuration = 1
	h.IterationCount = 1
	h.Drain = time.Minute
	h.Scenario.Steps[0].URL = srv.URL
	h.Scenario.Steps[0].Timeout = 10
	es, _ := InitEngineServices(h)
	rec := &recordingReport{}
	rec.Init(false, 0)
	es.ReportServ = rec
	ctx, interrupt := context.WithCancel(context.Background())
	e, _ := NewEngine(ctx, h, es)
	if err := e.Init(); err != nil {
		t.Fatal(err)
	}
	time.AfterFunc(1200*time.Millisecond, interrupt)
	time.AfterFunc(1400*time.Millisecond, e.Cut)
	begin := time.Now()
	if status := e.Start(); status != resultStopped || time.Since(begin) > 1900*time.Millisecond {
		t.Errorf("a second interrupt cuts at once: %q after %v", status, time.Since(begin))
	}
	if len(rec.results) != 1 || !rec.results[0].Interrupted() {
		t.Errorf("cut iteration reported as interrupted: %+v", rec.results)
	}
}
