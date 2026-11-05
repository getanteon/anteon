package report

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

func TestInterruptedIterationsCountApart(t *testing.T) {
	s := &stdout{}
	s.Init(false, 1)
	ok := &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{{StepID: 1, StatusCode: 200, Duration: time.Second,
		Stream: streamRes(100*time.Millisecond, time.Second)}}}
	cut := &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{{StepID: 1, StatusCode: 200,
		Err:    types.RequestError{Type: types.ErrorIntented, Reason: types.ReasonCtxCanceled},
		Stream: &types.StreamResult{End: types.StreamEndCut, Total: 9 * time.Second}}},
		Others: map[string]interface{}{types.OthersInterrupted: true}}
	aggregate(s.result, ok, map[uint16]map[string]int{}, 1)
	aggregate(s.result, cut, map[uint16]map[string]int{}, 1)
	s.result.ext.finalize(s.result)
	r := s.result
	step := r.StepResults[1]
	if r.InterruptedCount != 1 || r.SuccessCount != 1 || r.ServerFailedCount != 0 || step.Fail.Count != 0 {
		t.Errorf("an interrupted iteration is not a success or a failure: %+v", r)
	}
	if step.Outcomes[OutcomeInterrupted] != 1 || step.Stream.Count != 1 || step.Stream.Total.Max != 1000 {
		t.Errorf("interrupted shown, latencies without it: %+v %+v", step.Outcomes, step.Stream)
	}
}

func TestNoSuccessfulIterationFails(t *testing.T) {
	s := &stdout{}
	s.Init(false, 1)
	for i := 0; i < 3; i++ {
		aggregate(s.result, &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{{StepID: 1, StatusCode: 429}}},
			map[uint16]map[string]int{}, 1)
	}
	s.result.TestStatus = "success"
	s.result.ext.finalize(s.result)
	if s.result.TestStatus != "failed" || len(s.result.TestFailedAssertions) != 1 {
		t.Errorf("every request refused is a failed test: %+v", s.result)
	}
}

func TestPercentilesOfSuccessfulStreamsOnly(t *testing.T) {
	var a streamAgg
	a.add(streamRes(100*time.Millisecond, time.Second))
	a.add(&types.StreamResult{End: types.StreamEndTimeout, HasFirstToken: true, FirstToken: 30 * time.Second, Total: 30 * time.Second})
	errEv := streamRes(200*time.Millisecond, 2*time.Second)
	errEv.ErrorEvents = 1
	a.add(errEv)
	s := a.summary()
	if s.Count != 3 || s.Excluded != 2 || s.FirstToken.Count != 1 || s.Total.Max != 1000 || s.Ends[types.StreamEndTimeout] != 1 {
		t.Errorf("timed out and failed streams counted apart: %+v", s)
	}
}

func TestDebugRedactsSecrets(t *testing.T) {
	sr := &types.ScenarioStepResult{StepID: 1, Url: "https://user:pw@api.example.com/v1?key=abc123&q=hello&api_key=s3",
		ReqHeaders:  http.Header{"Authorization": {"Bearer gbk_secret"}, "Proxy-Authorization": {"Basic x"}, "Cookie": {"sid=1"}, "Accept": {"*/*"}},
		RespHeaders: http.Header{"Set-Cookie": {"sid=2"}},
		StatusCode:  200}
	v := ScenarioStepResultToVerboseHttpRequestInfo(sr)
	dump := v.Request.Url + " " + strings.Join(mapValues(v.Request.Headers), " ") + " " + strings.Join(mapValues(v.Response.Headers), " ")
	for _, secret := range []string{"gbk_secret", "Basic x", "sid=1", "sid=2", "abc123", "s3", "pw@"} {
		if strings.Contains(dump, secret) {
			t.Errorf("debug output shows %q: %s", secret, dump)
		}
	}
	if !strings.Contains(dump, "q=hello") || !strings.Contains(dump, "*/*") {
		t.Errorf("only secrets are hidden: %s", dump)
	}
	sr.Err = types.RequestError{Type: types.ErrorConn, Reason: `Get "https://h/?token=t0k3n": dial tcp`}
	if v := ScenarioStepResultToVerboseHttpRequestInfo(sr); strings.Contains(v.Error, "t0k3n") {
		t.Errorf("errors that quote a URL are redacted too: %s", v.Error)
	}
}

func mapValues(m map[string]string) []string {
	out := []string{}
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func TestOnlyInterruptedStillShowsTheStep(t *testing.T) {
	s := &stdout{}
	s.Init(false, 1)
	cut := &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{{StepID: 2, StepName: "ask",
		Err: types.RequestError{Type: types.ErrorIntented}}}, Others: map[string]interface{}{types.OthersInterrupted: true}}
	aggregate(s.result, cut, map[uint16]map[string]int{}, 1)
	s.result.ext.finalize(s.result)
	st, ok := s.result.StepResults[2]
	if !ok || st.Name != "ask" || st.Outcomes[OutcomeInterrupted] != 1 {
		t.Errorf("a step cut every time is still in the report: %+v", s.result.StepResults)
	}
}
