package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

func TestSamplerPercentiles(t *testing.T) {
	var s sampler
	for i := 1; i <= 1000; i++ {
		s.add(float64(i))
	}
	d := s.dist()
	if d.Count != 1000 || d.Min != 1 || d.Max != 1000 || d.P50 != 500 || d.P90 != 900 || d.P95 != 950 || d.P99 != 990 || d.Mean != 500.5 {
		t.Errorf("unexpected dist %#v", d)
	}

	var big sampler
	for i := 0; i < maxSamples*2; i++ {
		big.add(float64(i % 100))
	}
	if len(big.samples) != maxSamples {
		t.Errorf("samples should stay bounded, got %d", len(big.samples))
	}
	if bd := big.dist(); bd.Count != int64(maxSamples*2) || bd.P50 < 40 || bd.P50 > 60 {
		t.Errorf("reservoir percentiles drifted: %#v", bd)
	}

	var empty sampler
	if empty.dist() != nil {
		t.Error("an empty sampler has no distribution")
	}
}

func TestOutcomeOf(t *testing.T) {
	cases := []struct {
		sr   types.ScenarioStepResult
		want string
	}{
		{types.ScenarioStepResult{StatusCode: 200}, OutcomeOK},
		{types.ScenarioStepResult{StatusCode: 201}, OutcomeOK},
		{types.ScenarioStepResult{StatusCode: 429}, OutcomeRateLimited},
		{types.ScenarioStepResult{StatusCode: 503}, OutcomeUnavailable},
		{types.ScenarioStepResult{StatusCode: 502}, OutcomeServerError},
		{types.ScenarioStepResult{StatusCode: 404}, OutcomeClientError},
		{types.ScenarioStepResult{Err: types.RequestError{Type: types.ErrorConn, Reason: types.ReasonConnTimeout}}, OutcomeTimeout},
		{types.ScenarioStepResult{Err: types.RequestError{Type: types.ErrorConn, Reason: types.ReasonReadTimeout}}, OutcomeTimeout},
		{types.ScenarioStepResult{Err: types.RequestError{Type: types.ErrorConn, Reason: types.ReasonConnRefused}}, OutcomeNetworkError},
		{types.ScenarioStepResult{StatusCode: 200, Err: types.RequestError{Type: types.ErrorUnkown, Reason: "unexpected EOF"},
			Stream: &types.StreamResult{End: types.StreamEndCut}}, OutcomeStreamCut},
		{types.ScenarioStepResult{StatusCode: 200, Stream: &types.StreamResult{End: types.StreamEndCut}}, OutcomeStreamCut},
		{types.ScenarioStepResult{StatusCode: 200, Stream: &types.StreamResult{End: types.StreamEndDone, ErrorEvents: 1}}, OutcomeStreamError},
		{types.ScenarioStepResult{StatusCode: 200, Stream: &types.StreamResult{End: types.StreamEndClosed}}, OutcomeOK},
	}
	for i, c := range cases {
		sr := c.sr
		if got := OutcomeOf(&sr); got != c.want {
			t.Errorf("case %d: expected %s, got %s", i, c.want, got)
		}
	}
}

func TestSLOMet(t *testing.T) {
	slo := &types.SLO{FirstTokenMs: 1000, TotalMs: 5000}
	ok := &types.StreamResult{End: types.StreamEndDone, HasFirstToken: true, FirstToken: 900 * time.Millisecond, Total: 4 * time.Second}
	if !slo.Met(ok) {
		t.Error("within both thresholds")
	}
	slow := *ok
	slow.FirstToken = 1100 * time.Millisecond
	noToken := *ok
	noToken.HasFirstToken = false
	cut := *ok
	cut.End = types.StreamEndCut
	errEv := *ok
	errEv.ErrorEvents = 1
	for name, r := range map[string]*types.StreamResult{"slow": &slow, "no token": &noToken, "cut": &cut, "error event": &errEv} {
		if slo.Met(r) {
			t.Errorf("%s should miss the SLO", name)
		}
	}
}

func streamRes(firstToken, total time.Duration) *types.StreamResult {
	return &types.StreamResult{End: types.StreamEndDone, HasFirstEvent: true, FirstEvent: firstToken / 2,
		HasFirstToken: true, FirstToken: firstToken, Total: total, Events: 10, GapMax: 50 * time.Millisecond,
		GapP50: 10 * time.Millisecond, GapP95: 40 * time.Millisecond, TTFB: 5 * time.Millisecond}
}

func iteration(stage types.LoadStage, steps ...*types.ScenarioStepResult) *types.ScenarioResult {
	return &types.ScenarioResult{StepResults: steps, Others: map[string]interface{}{"loadStage": stage}}
}

func TestExtendedReportJSON(t *testing.T) {
	warm := types.LoadStage{Index: 0, Name: "warm-up", Warmup: true}
	s6 := types.LoadStage{Index: 1, Name: "6/min"}

	create := func(code int) *types.ScenarioStepResult {
		return &types.ScenarioStepResult{StepID: 1, StepName: "create chat", StatusCode: code, Duration: 10 * time.Millisecond}
	}
	ask := func(code int, st *types.StreamResult) *types.ScenarioStepResult {
		return &types.ScenarioStepResult{StepID: 2, StepName: "ask", StatusCode: code, Duration: 20 * time.Millisecond, Stream: st}
	}

	sj := &stdoutJson{}
	sj.Init(false, 1)
	sj.SetSLO(&types.SLO{FirstTokenMs: 1000, TotalMs: 5000})

	input := make(chan *types.ScenarioResult, 10)
	input <- iteration(warm, create(201), ask(200, streamRes(9*time.Second, 20*time.Second)))     // slow, but warm-up
	input <- iteration(s6, create(201), ask(200, streamRes(500*time.Millisecond, 3*time.Second))) // good
	input <- iteration(s6, create(201), ask(200, streamRes(800*time.Millisecond, 6*time.Second))) // too long
	input <- iteration(s6, create(201), ask(429, nil))
	input <- iteration(s6, create(201), ask(503, nil))
	close(input)

	var output string
	printJson = func(j []byte) { output = string(j) }
	go sj.Start(input, nil)
	<-sj.DoneChan()

	var got struct {
		SuccessCount int64 `json:"success_count"`
		Goodput      Goodput
		Stages       []StageSummary
		Steps        map[string]struct {
			Outcomes map[string]int64
			Stream   *StreamSummary
		}
	}
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	if got.SuccessCount != 5 {
		t.Errorf("existing counters still count every iteration, got %d", got.SuccessCount)
	}
	if got.Goodput.Iterations != 4 || got.Goodput.Good != 1 || got.Goodput.Perc != 25 {
		t.Errorf("goodput excludes the warm-up: %#v", got.Goodput)
	}
	if len(got.Stages) != 2 || !got.Stages[0].Warmup || got.Stages[0].Iterations != 1 || got.Stages[1].Iterations != 4 ||
		*got.Stages[1].Good != 1 || got.Stages[1].OK != 2 {
		t.Errorf("unexpected stages %#v", got.Stages)
	}
	ask2 := got.Steps["2"]
	if ask2.Outcomes[OutcomeOK] != 2 || ask2.Outcomes[OutcomeRateLimited] != 1 || ask2.Outcomes[OutcomeUnavailable] != 1 {
		t.Errorf("unexpected outcomes %#v", ask2.Outcomes)
	}
	if ask2.Stream == nil || ask2.Stream.Count != 2 || ask2.Stream.FirstToken.Max != 800 || ask2.Stream.Total.P50 != 3000 {
		t.Errorf("stream summary should hold the two streams after the warm-up: %#v", ask2.Stream)
	}
	if got.Steps["1"].Stream != nil || got.Steps["1"].Outcomes[OutcomeOK] != 4 {
		t.Errorf("step 1 is not streamed: %#v", got.Steps["1"])
	}
}

func TestExtendedReportSilentWithoutStreams(t *testing.T) {
	sj := &stdoutJson{}
	sj.Init(false, 1)
	input := make(chan *types.ScenarioResult, 1)
	input <- &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{{StepID: 1, StatusCode: 200}}}
	close(input)

	var output string
	printJson = func(j []byte) { output = string(j) }
	go sj.Start(input, nil)
	<-sj.DoneChan()
	for _, key := range []string{"goodput", "stages", "outcomes", "stream"} {
		if strings.Contains(output, `"`+key+`"`) {
			t.Errorf("a test without streams, SLO or stages should print no %q: %s", key, output)
		}
	}
}

func TestExtendedReportStdout(t *testing.T) {
	s := &stdout{}
	s.Init(false, 1)
	s.SetSLO(&types.SLO{FirstTokenMs: 1000})
	stage := types.LoadStage{Index: 1, Name: "12/min"}
	aggregate(s.result, iteration(stage, &types.ScenarioStepResult{StepID: 1, StatusCode: 200,
		Stream: streamRes(500*time.Millisecond, 2*time.Second)}), map[uint16]map[string]int{}, 1)
	s.result.ext.finalize(s.result)

	var b strings.Builder
	printStream(&b, s.result.StepResults[1].Stream)
	printOutcomes(&b, s.result.StepResults[1].Outcomes)
	printGoodput(&b, s.result.Goodput)
	printStages(&b, s.result.Stages, true)
	out := b.String()
	for _, want := range []string{"First Token", "Max Gap", "Outcomes", "Goodput", "1 / 1 iterations (100.0%)", "first token <= 1000ms", "12/min"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout report misses %q:\n%s", want, out)
		}
	}
}
