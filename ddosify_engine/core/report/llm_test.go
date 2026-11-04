package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/llm"
	"go.ddosify.com/ddosify/core/types"
)

func llmIteration(stage types.LoadStage, intended time.Time, lag time.Duration, tokens int) *types.ScenarioResult {
	st := streamRes(300*time.Millisecond, 2*time.Second)
	st.LLM = &types.LLMResult{SentAt: intended.Add(lag), Output: tokens, OutputSource: types.TokenSourceUsage,
		HasContent: true, FirstContent: 300 * time.Millisecond, LastContent: 2 * time.Second,
		TPOT: 1700 * time.Millisecond / time.Duration(tokens-1), HasTPOT: true, OutputPerSec: float64(tokens) / 2}
	res := iteration(stage, &types.ScenarioStepResult{StepID: 1, StatusCode: 200, RequestTime: intended.Add(lag), Stream: st})
	res.Others[types.OthersIntendedStart] = intended
	return res
}

func TestLLMReportJSON(t *testing.T) {
	sj := &stdoutJson{}
	sj.Init(false, 1)
	sj.SetSLO(&types.SLO{TPOTMs: 10})

	t0 := time.Now()
	warm := types.LoadStage{Index: 0, Name: "warm-up", Warmup: true}
	load := types.LoadStage{Index: 1, Name: "load"}
	input := make(chan *types.ScenarioResult, 4)
	input <- llmIteration(warm, t0, 0, 11)                                  // warm-up: left out
	input <- llmIteration(load, t0, time.Millisecond, 201)                  // TPOT 8.5ms: good
	input <- llmIteration(load, t0.Add(time.Second), time.Millisecond, 101) // TPOT 17ms: misses tpot_ms
	close(input)

	var output string
	printJson = func(j []byte) { output = string(j) }
	go sj.Start(input, nil)
	<-sj.DoneChan()

	var got struct {
		Goodput  Goodput
		LLM      *llm.Summary         `json:"llm"`
		Schedule *llm.ScheduleSummary `json:"schedule"`
		Steps    map[string]struct{ Stream *StreamSummary }
		Stages   []StageSummary
	}
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	if got.LLM == nil || got.LLM.Tokens == nil || got.LLM.Tokens.Total != 302 || got.LLM.Tokens.Requests != 2 {
		t.Fatalf("llm summary %s", output)
	}
	// 302 tokens from the first send to the last end: 1ms to 3.001s
	if got.LLM.Tokens.WindowSeconds != 3 || got.LLM.Tokens.SystemPerSec != 100.7 {
		t.Errorf("window %v rate %v", got.LLM.Tokens.WindowSeconds, got.LLM.Tokens.SystemPerSec)
	}
	if got.Goodput.Good != 1 || got.Goodput.Iterations != 2 {
		t.Errorf("tpot_ms goodput %+v", got.Goodput)
	}
	if got.Schedule == nil || got.Schedule.Iterations != 2 || got.Schedule.Lag.Max != 1 {
		t.Errorf("schedule %+v", got.Schedule)
	}
	if s := got.Steps["1"].Stream; s == nil || s.LLM == nil || s.LLM.Tokens.Requests != 2 {
		t.Errorf("step llm summary %+v", s)
	}
	if len(got.Stages) != 2 || got.Stages[0].Stream.LLM.Tokens.Total != 11 {
		t.Errorf("stage llm summary %+v", got.Stages)
	}
}

func TestLLMReportStdout(t *testing.T) {
	s := &stdout{}
	s.Init(false, 1)
	s.SetSLO(&types.SLO{TPOTMs: 50})
	t0 := time.Now()
	aggregate(s.result, llmIteration(types.LoadStage{Index: 1, Name: "x"}, t0, 300*time.Millisecond, 51), map[uint16]map[string]int{}, 1)
	s.result.ext.finalize(s.result)

	var b strings.Builder
	printStream(&b, s.result.StepResults[1].Stream)
	printGoodput(&b, s.result.Goodput)
	llm.WriteSystem(&b, s.result.LLM, s.result.Schedule)
	out := b.String()
	for _, want := range []string{"TPOT (ms)", "Tokens/s/Request", "tpot <= 50ms", "Output throughput", "Scheduling lag", "WARNING: the client could not keep the schedule"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout misses %q:\n%s", want, out)
		}
	}
}
