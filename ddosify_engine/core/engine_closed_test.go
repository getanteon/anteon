package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/types"
)

// planObserver keeps the plan and the stages snapshots named.
type planObserver struct {
	mu     sync.Mutex
	plan   live.Plan
	stages map[int]bool
}

func (p *planObserver) Start(pl live.Plan) { p.mu.Lock(); p.plan = pl; p.mu.Unlock() }
func (p *planObserver) Update(s live.Snapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stages == nil {
		p.stages = map[int]bool{}
	}
	p.stages[s.Stage.Index] = true
}
func (p *planObserver) Finish(live.Final) {}

// A closed-loop stage keeps at most its concurrency in flight, runs exactly its count, and starts
// a new iteration as soon as one ends; the next stage starts when the last of the previous ended.
func TestClosedLoopStages(t *testing.T) {
	var inFlight, peak atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
	}))
	defer server.Close()

	h := newDummyHammer()
	h.TimeRunCountMap = types.TimeRunCount{
		{Count: 6, Concurrency: 2, Name: "warm-up", Warmup: true},
		{Count: 30, Concurrency: 3, Name: "c=3"},
		{Count: 40, Concurrency: 8, Name: "c=8"},
	}
	h.IterationCount, h.TestDuration = h.TimeRunCountMap.Totals()
	h.Scenario.Steps[0].URL = server.URL
	if err := h.Validate(); err != nil {
		t.Fatal(err)
	}

	es, err := InitEngineServices(h)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recordingReport{}
	rec.Init(false, 0)
	es.ReportServ = rec
	obs := &planObserver{}
	es.Observers = []live.Observer{obs}
	e, err := NewEngine(context.Background(), h, es)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Init(); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if status := e.Start(); status != resultDone {
		t.Fatalf("status %s", status)
	}
	took := time.Since(started)

	if len(rec.results) != 76 {
		t.Fatalf("expected 76 iterations, got %d", len(rec.results))
	}
	count := map[string]int{}
	var order []string
	for _, res := range rec.results {
		st := res.Others["loadStage"].(types.LoadStage)
		count[st.Name]++
		if st.Concurrency == 0 {
			t.Fatal("a closed-loop stage must carry its concurrency")
		}
		if len(order) == 0 || order[len(order)-1] != st.Name {
			order = append(order, st.Name)
		}
	}
	if count["warm-up"] != 6 || count["c=3"] != 30 || count["c=8"] != 40 {
		t.Errorf("iterations per stage %v", count)
	}
	// results arrive as iterations end, so stages that do not overlap come in order
	if len(order) != 3 {
		t.Errorf("stages overlapped: %v", order)
	}
	if p := peak.Load(); p > 8 || p < 8 {
		t.Errorf("peak in flight %d, want exactly the largest concurrency, 8", p)
	}
	// 2 + 10 + 5 rounds of 40 ms at the least; a closed loop does not wait for a schedule
	if took > 3*time.Second {
		t.Errorf("took %v: a new iteration should start as soon as one ends", took)
	}
	obs.mu.Lock()
	defer obs.mu.Unlock()
	if !obs.plan.ClosedLoop || len(obs.plan.Stages) != 3 || obs.plan.Stages[2].Concurrency != 8 || obs.plan.Iterations != 76 {
		t.Errorf("plan %+v", obs.plan)
	}
}

// Within a stage the cap on in-flight iterations holds; a duration ends the stage early.
func TestClosedLoopConcurrencyCapAndDuration(t *testing.T) {
	var inFlight, peak atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	h := newDummyHammer()
	h.TimeRunCountMap = types.TimeRunCount{{Count: 1000, Concurrency: 4, Duration: 1, Name: "capped"}}
	h.IterationCount, h.TestDuration = h.TimeRunCountMap.Totals()
	h.Scenario.Steps[0].URL = server.URL
	es, _ := InitEngineServices(h)
	rec := &recordingReport{}
	rec.Init(false, 0)
	es.ReportServ = rec
	e, _ := NewEngine(context.Background(), h, es)
	if err := e.Init(); err != nil {
		t.Fatal(err)
	}
	e.Start()
	if p := peak.Load(); p != 4 {
		t.Errorf("peak in flight %d, want 4", p)
	}
	// about 1 s / 100 ms x 4 iterations, nowhere near 1000
	if n := len(rec.results); n < 20 || n > 60 {
		t.Errorf("%d iterations in a 1 s cap", n)
	}
}

func TestClosedLoopValidation(t *testing.T) {
	for _, st := range []types.TimeRunCount{
		{{Count: 5, Concurrency: 2}, {Duration: 10, Count: 5}},
		{{Count: 0, Concurrency: 2}},
		{{Count: 5, Concurrency: 2, Duration: -1}},
	} {
		h := newDummyHammer()
		h.TimeRunCountMap = st
		if err := h.Validate(); err == nil {
			t.Errorf("%+v must be refused", st)
		}
	}
	h := newDummyHammer()
	h.TimeRunCountMap = types.TimeRunCount{{Count: 5, Concurrency: 2}}
	h.LoadSpread = types.LoadSpreadPoisson
	if err := h.Validate(); err == nil {
		t.Error("poisson with closed-loop stages must be refused")
	}
}
