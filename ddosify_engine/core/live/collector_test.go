package live

import (
	"sync"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/types"
)

type recorder struct {
	mu      sync.Mutex
	plan    *Plan
	updates []Snapshot
	final   *Final
}

func (r *recorder) Start(p Plan)      { r.mu.Lock(); r.plan = &p; r.mu.Unlock() }
func (r *recorder) Update(s Snapshot) { r.mu.Lock(); r.updates = append(r.updates, s); r.mu.Unlock() }
func (r *recorder) Finish(f Final)    { r.mu.Lock(); r.final = &f; r.mu.Unlock() }

func streamed(ttft time.Duration, code int) *types.ScenarioResult {
	return &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{{StepID: 1, StatusCode: code,
		Stream: &types.StreamResult{End: types.StreamEndDone, HasFirstToken: true, FirstToken: ttft, Total: 2 * ttft,
			Events: 3, GapP50: 10 * time.Millisecond, Extra: map[string]float64{"output_tokens": 42}}}}}
}

func TestCollector(t *testing.T) {
	warm := &types.LoadStage{Index: 0, Name: "warm-up", Warmup: true}
	main := &types.LoadStage{Index: 1, Name: "10/s"}
	plan := Plan{Duration: 2 * time.Second, Iterations: 4, Schedule: make([]int, 20),
		Stages: []Stage{{Index: 0, Name: "warm-up", Warmup: true, End: time.Second}, {Index: 1, Name: "10/s", Start: time.Second, End: 2 * time.Second}}}
	for i := range plan.Schedule[10:] {
		plan.Schedule[10+i] = 1
	}
	rec := &recorder{}
	c := NewCollector(plan, &types.SLO{FirstTokenMs: 100}, []Observer{rec})
	c.Start()

	c.IterationStarted()
	c.IterationDone(streamed(500*time.Millisecond, 200), warm, time.Second)
	for _, code := range []int{200, 200, 429} {
		c.IterationStarted()
		c.IterationDone(streamed(50*time.Millisecond, code), main, 100*time.Millisecond)
	}
	c.IterationStarted() // still running

	time.Sleep(3 * Interval)
	s := c.Snapshot()
	if s.Started != 5 || s.Completed != 4 || s.InFlight != 1 || s.AchievedRPS != 5 {
		t.Errorf("unexpected counters %+v", s)
	}
	if s.Total.Iterations != 3 || s.Total.Goodput.Good != 2 || s.Total.Outcomes[report.OutcomeRateLimited] != 1 {
		t.Errorf("total leaves the warm-up out: %+v", s.Total)
	}
	if s.Total.TTFT.Max != 50 || s.Total.Extra["output_tokens"].P50 != 42 || s.Total.ITL.P50 != 10 || s.Total.E2E.P50 != 100 {
		t.Errorf("unexpected percentiles %+v", s.Total)
	}
	if s.Stage.Name != "warm-up" || s.Current.Iterations != 1 {
		t.Errorf("the first second is the warm-up: %+v", s.Stage)
	}

	c.Finish("done", true, &report.Result{TestStatus: "success"})
	c.Finish("done", true, nil) // once
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.plan == nil || len(rec.updates) < 2 || rec.final == nil || rec.final.Result == nil || !rec.final.Passed {
		t.Fatalf("observer calls missing: plan %v updates %d final %v", rec.plan != nil, len(rec.updates), rec.final)
	}
	for i := 1; i < len(rec.updates); i++ {
		if rec.updates[i].Seq <= rec.updates[i-1].Seq {
			t.Error("snapshots in order")
		}
	}
}

func TestRegistry(t *testing.T) {
	Register("test-obs", func(o Options) (Observer, error) { return &recorder{}, nil })
	if _, err := Build("test-obs", Options{}); err != nil {
		t.Error(err)
	}
	if _, err := Build("missing", Options{}); err == nil {
		t.Error("a missing observer is an error")
	}
}

func llmResult(source string, out int, intended, sent time.Time) *types.ScenarioResult {
	r := streamed(50*time.Millisecond, 200)
	r.StepResults[0].RequestTime = sent
	r.StepResults[0].Stream.LLM = &types.LLMResult{SentAt: sent, Output: out, OutputSource: source, HasContent: true,
		LastContent: time.Second, TPOT: 20 * time.Millisecond, HasTPOT: true, ITL: []time.Duration{10 * time.Millisecond, 30 * time.Millisecond},
		CacheBusted: true}
	r.Others = map[string]interface{}{types.OthersIntendedStart: intended}
	return r
}

func TestCollectorTokens(t *testing.T) {
	c := NewCollector(Plan{Duration: time.Second}, nil, nil)
	c.Start()
	t0 := time.Now()
	c.IterationStarted()
	c.IterationDone(llmResult(types.TokenSourceChunks, 9, t0, t0.Add(2*time.Millisecond)), nil, time.Second)
	s := c.Snapshot()
	if tk := s.Total.Tokens; tk == nil || tk.Unit != "chunks" || tk.TPOT.Count != 0 || tk.Output.P50 != 9 {
		t.Errorf("only chunks: never tokens, no TPOT: %+v", tk)
	}
	if s.Total.SchedulingLag.Max != 2 {
		t.Errorf("scheduling lag %+v", s.Total.SchedulingLag)
	}
	c.IterationStarted()
	c.IterationDone(llmResult(types.TokenSourceTokenizer, 100, t0, t0), nil, time.Second)
	tk := c.Snapshot().Total.Tokens
	if tk.Unit != "tokens" || tk.Output.P50 != 100 || tk.TPOT.P50 != 20 || tk.ITL.Count != 4 || tk.CacheBusted != 2 || tk.PerSec != 100 {
		t.Errorf("tokens counted in tokens: %+v", tk)
	}
	c.Finish("done", true, nil)
}

func TestE2EIsOfSuccessfulIterations(t *testing.T) {
	c := NewCollector(Plan{Duration: time.Second, Schedule: make([]int, 10)}, nil, []Observer{&recorder{}})
	c.Start()
	c.IterationDone(streamed(50*time.Millisecond, 429), nil, 2*time.Millisecond)
	s := c.Snapshot()
	if s.Total.E2E.Count != 0 {
		t.Fatalf("a 429 counted as an E2E sample: %+v", s.Total.E2E)
	}
	c.IterationDone(streamed(50*time.Millisecond, 200), nil, time.Second)
	s = c.Snapshot()
	if s.Total.E2E.Count != 1 || s.Total.E2E.P50 != 1000 || s.Total.Outcomes[report.OutcomeRateLimited] != 1 {
		t.Fatalf("E2E %+v, outcomes %v", s.Total.E2E, s.Total.Outcomes)
	}
}
