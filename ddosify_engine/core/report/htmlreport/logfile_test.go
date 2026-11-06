package htmlreport

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func readFixture(t *testing.T) Report {
	t.Helper()
	f, err := os.Open("testdata/run.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rep, err := ReadLog(f)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

type goldenDist struct {
	N                            *int
	Min, Max, P50, P90, P95, P99 *float64
	P50CI                        []float64 `json:"p50ci"`
	P95CI                        []float64 `json:"p95ci"`
}

// The golden file holds what an independent implementation computed from testdata/run.ndjson.
type golden struct {
	Stages []struct {
		Name                                                  string
		SettleMs                                              float64 `json:"settleMs"`
		All, Steady, Answered, Good, Rejected, Failed, NoText int
		FT, Total                                             goldenDist
	}
	Series struct {
		T                            []float64
		FTP50, FTP95, TotP50, TotP95 []*float64
		N                            []int
		Win                          []float64
	}
}

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(a)) }

func sameP(t *testing.T, what string, want, got *float64) {
	t.Helper()
	if (want == nil) != (got == nil) || (want != nil && !near(*want, *got)) {
		t.Errorf("%s: got %v, want %v", what, deref(got), deref(want))
	}
}

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestReadLogMatchesGolden(t *testing.T) {
	rep := readFixture(t)
	if rep.Requests == nil || len(rep.Stages) != 3 || len(rep.Series) < 100 {
		t.Fatalf("requests %v, %d stages, %d points", rep.Requests != nil, len(rep.Stages), len(rep.Series))
	}
	_, steady := buildData(rep, "ITL")
	var g golden
	b, err := os.ReadFile("testdata/run.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}
	if len(steady.Stages) != len(g.Stages) {
		t.Fatalf("%d stages, want %d", len(steady.Stages), len(g.Stages))
	}
	for i, w := range g.Stages {
		s := steady.Stages[i]
		got := []int{s.All, s.Steady, s.Answered, s.Good, s.Rejected, s.Failed, s.NoText}
		want := []int{w.All, w.Steady, w.Answered, w.Good, w.Rejected, w.Failed, w.NoText}
		for k := range got {
			if got[k] != want[k] {
				t.Errorf("%s: counts %v, want %v", w.Name, got, want)
				break
			}
		}
		if !near(s.SettleMs, w.SettleMs) {
			t.Errorf("%s: settle %v, want %v", w.Name, s.SettleMs, w.SettleMs)
		}
		for _, c := range []struct {
			name string
			got  Dist
			want goldenDist
		}{{"ft", s.FT, w.FT}, {"total", s.Total, w.Total}} {
			if c.want.N != nil && *c.want.N != c.got.N {
				t.Errorf("%s %s: n %d, want %d", w.Name, c.name, c.got.N, *c.want.N)
			}
			sameP(t, w.Name+" "+c.name+" p50", c.want.P50, c.got.P50)
			sameP(t, w.Name+" "+c.name+" p95", c.want.P95, c.got.P95)
			sameP(t, w.Name+" "+c.name+" p99", c.want.P99, c.got.P99)
			sameP(t, w.Name+" "+c.name+" min", c.want.Min, c.got.Min)
			for k := range c.want.P95CI {
				if !near(c.want.P95CI[k], c.got.P95CI[k]) {
					t.Errorf("%s %s p95 CI %v, want %v", w.Name, c.name, c.got.P95CI, c.want.P95CI)
				}
			}
		}
	}
	s := steady.Series
	if len(s.T) != len(g.Series.T) {
		t.Fatalf("%d series points, want %d", len(s.T), len(g.Series.T))
	}
	for i := range s.T {
		if !near(s.T[i], g.Series.T[i]) || s.N[i] != g.Series.N[i] || !near(s.Win[i], g.Series.Win[i]) {
			t.Errorf("point %d: t %v n %d win %v, want %v %d %v", i, s.T[i], s.N[i], s.Win[i], g.Series.T[i], g.Series.N[i], g.Series.Win[i])
		}
		sameP(t, "ft p50", g.Series.FTP50[i], s.FTP50[i])
		sameP(t, "ft p95", g.Series.FTP95[i], s.FTP95[i])
		sameP(t, "total p50", g.Series.TotP50[i], s.TotP50[i])
		sameP(t, "total p95", g.Series.TotP95[i], s.TotP95[i])
	}
}

func TestReadLogReport(t *testing.T) {
	var b strings.Builder
	if err := Write(&b, readFixture(t)); err != nil {
		t.Fatal(err)
	}
	doc := b.String()
	checkBalanced(t, doc)
	for _, want := range []string{"Steady state, from the request records", "Per step", "Turn 1, Turn 2", "6/min", "12/min"} {
		if !strings.Contains(doc, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	var d chartData
	if err := json.Unmarshal([]byte(dataJSON(t, doc)), &d); err != nil {
		t.Fatal(err)
	}
	if d.Req == nil || !d.Req.Stream || d.Req.Sampled || d.RateUnit != "min" {
		t.Errorf("chart data: req %+v, rate unit %q", d.Req, d.RateUnit)
	}
	if d.Stages[1].Start == nil || *d.Stages[1].Start < 59 || *d.Stages[1].Start > 61 {
		t.Errorf("the second stage starts at %v s, want 60", deref(d.Stages[1].Start))
	}
}

func TestReadLogWithoutReport(t *testing.T) {
	_, err := ReadLog(strings.NewReader(`{"type":"progress","elapsed_s":1}` + "\nnot json\n"))
	if !errors.Is(err, ErrNoReport) {
		t.Fatalf("err %v, want ErrNoReport", err)
	}
}

func TestReadLogSkipsOtherLines(t *testing.T) {
	in := "2026-01-01T09:00:00Z pod log prefix\n\n{broken\n" + `{"type":"report","test_status":"success","steps":{"1":{"name":"get"}}}`
	rep, err := ReadLog(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != "success" || len(rep.Steps) != 1 || rep.Requests != nil {
		t.Errorf("status %q, %d steps, requests %v", rep.Status, len(rep.Steps), rep.Requests != nil)
	}
}

func TestRequestsAreBounded(t *testing.T) {
	q := NewRequests()
	const n = 3 * PerStepSamples
	for i := range n {
		q.Add(Record{Stage: -1, StepID: 1, StepName: "get", Sent: float64(i), Done: float64(i) + 100, Total: 100, Outcome: "ok"})
	}
	if q.Len() != PerStepSamples {
		t.Fatalf("kept %d, want %d", q.Len(), PerStepSamples)
	}
	d := q.Analyze(nil, 0, 0, 60_000)
	if !d.Sampled || d.Count != n || d.Kept != PerStepSamples {
		t.Errorf("sampled %v, count %d, kept %d", d.Sampled, d.Count, d.Kept)
	}
	// 1,000 requests a second: the window holds the counted-back rate, not the sample's.
	if w := d.Series.Win[len(d.Series.Win)/2]; w != 60_000 {
		t.Errorf("window %v ms, want 60,000", w)
	}
	for i := 0; i < 2*maxSteps; i++ {
		q.Add(Record{StepID: uint16(100 + i), Outcome: "ok"})
	}
	if len(q.steps) != maxSteps {
		t.Errorf("%d steps kept, want %d", len(q.steps), maxSteps)
	}
}

func TestSpansSurviveThinning(t *testing.T) {
	rec := NewRecorder(8)
	for i := range 100 {
		stage := 0
		if i >= 37 {
			stage = 1
		}
		rec.Add(liveSnapshot(i, stage))
	}
	sp := rec.Spans()
	if sp[1].Start != 37 || sp[0].End != 36 || sp[1].End != 99 {
		t.Errorf("spans %+v, want stage 1 from 37 s", sp)
	}
}

// A log's report line carries error reasons and failed rules; none of it may reach the page.
func TestReadLogHoldsNoSecrets(t *testing.T) {
	in := `{"type":"report","test_status":"failed","failed_criterias":[{"rule":"RULESECRET","received":{"x":"RECEIVEDSECRET"}}],` +
		`"steps":{"1":{"name":"get","fail":{"count":1,"server":{"count":1,"reasons":{"Get \"https://llm.bank.internal/x?k=QUERYSECRET\"":1}}}}}}`
	rep, err := ReadLog(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := Write(&b, rep); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"RULESECRET", "RECEIVEDSECRET", "QUERYSECRET", "llm.bank.internal"} {
		if strings.Contains(b.String(), s) {
			t.Errorf("report contains %q", s)
		}
	}
	if !strings.Contains(b.String(), "1 success criteria failed") {
		t.Error("the failed criteria are not counted")
	}
}

// A long run's log is read as a stream: memory follows the sample, not the log.
func TestReadLogIsBounded(t *testing.T) {
	const n = 300_000
	pr, pw := io.Pipe()
	go func() {
		w := bufio.NewWriter(pw)
		t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := range n {
			at := t0.Add(time.Duration(i) * time.Millisecond)
			fmt.Fprintf(w, `{"type":"request","time":%q,"stage":"","warmup":false,"iteration":%d,"step_id":1,"step_name":"get","sent":%q,"outcome":"ok","total_ms":%d}`+"\n",
				at.Add(40*time.Millisecond).Format(time.RFC3339Nano), i, at.Format(time.RFC3339Nano), 30+i%20)
		}
		fmt.Fprintln(w, `{"type":"report","test_status":"success","steps":{"1":{"name":"get"}}}`)
		w.Flush()
		pw.Close()
	}()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	rep, err := ReadLog(pr)
	if err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	if rep.Requests.Len() != PerStepSamples {
		t.Fatalf("kept %d records, want %d", rep.Requests.Len(), PerStepSamples)
	}
	if grown := int64(after.HeapInuse) - int64(before.HeapInuse); grown > 32<<20 {
		t.Errorf("heap grew %d MB reading %d lines", grown>>20, n)
	}
}
