package scenario

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/scenario/scripting/injection"
	"go.ddosify.com/ddosify/v2/core/types"
)

type sessionStep struct {
	skipIfEmpty string
	result      *types.ScenarioStepResult
	sleeps      bool
}

// sessionService runs a session of steps against mock requesters; the data set "sessions" has the rows.
func sessionService(t *testing.T, stopOnFailure bool, rows []map[string]interface{}, steps []sessionStep) (*ScenarioService, *url.URL, []*MockHttpRequester, []*MockSleep) {
	t.Helper()
	ei := &injection.EnvironmentInjector{}
	ei.Init()
	p, _ := url.Parse("http://proxy:80")
	var reqs []scenarioItemRequester
	var mocks []*MockHttpRequester
	var sleeps []*MockSleep
	var scenarioSteps []types.ScenarioStep
	for i, st := range steps {
		id := uint16(i + 1)
		m := &MockHttpRequester{ReturnSend: st.result}
		if m.ReturnSend == nil {
			m.ReturnSend = &types.ScenarioStepResult{StepID: id, StatusCode: 200}
		}
		sl := &MockSleep{}
		r := scenarioItemRequester{scenarioItemID: id, requester: m, skipIfEmpty: st.skipIfEmpty}
		if st.sleeps {
			r.sleeper = sl
		}
		reqs = append(reqs, r)
		mocks = append(mocks, m)
		sleeps = append(sleeps, sl)
		scenarioSteps = append(scenarioSteps, types.ScenarioStep{ID: id, Method: http.MethodPost, URL: "http://test.com", SkipIfEmpty: st.skipIfEmpty})
	}
	s := &ScenarioService{
		clients:       map[*url.URL][]scenarioItemRequester{p: reqs},
		scenario:      types.Scenario{Steps: scenarioSteps, Data: map[string]types.CsvData{"sessions": {Rows: rows}}},
		ctx:           context.Background(),
		ei:            ei,
		stopOnFailure: stopOnFailure,
	}
	return s, p, mocks, sleeps
}

func sentIDs(res *types.ScenarioResult) string {
	ids := ""
	for _, sr := range res.StepResults {
		ids += fmt.Sprint(sr.StepID)
	}
	return ids
}

func TestSkipIfEmpty(t *testing.T) {
	rows := []map[string]interface{}{
		{"turn1": "q1", "turn2": "f1", "turn3": "g1"},
		{"turn1": "q2", "turn2": "  ", "turn3": "g2"}, // blank after trimming
		{"turn1": "q3", "turn2": "", "turn3": ""},     // one turn
	}
	steps := []sessionStep{
		{sleeps: true},
		{skipIfEmpty: "{{data.sessions.turn1}}", sleeps: true},
		{skipIfEmpty: "{{data.sessions.turn2}}", sleeps: true},
		{skipIfEmpty: "{{data.sessions.turn3}}", sleeps: true},
		{skipIfEmpty: "{{NEVER_CAPTURED}}"}, // a variable that is not there skips the step too
	}
	want := []struct {
		sent   string
		sleeps []int
	}{
		{"1234", []int{1, 1, 1, 0, 0}},
		{"124", []int{1, 1, 0, 0, 0}}, // turn 2 skipped: its think time is not applied, turn 1's is, before turn 3
		{"12", []int{1, 0, 0, 0, 0}},  // nothing after turn 1 is sent: turn 1 does not sleep
	}
	for i, w := range want {
		s, p, mocks, sleeps := sessionService(t, false, rows, steps)
		s.iterIndex = int64(i)
		res, err := s.Do(p, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if got := sentIDs(res); got != w.sent {
			t.Errorf("row %d: expected steps %s sent, got %s", i, w.sent, got)
		}
		for j, m := range mocks {
			if m.SendCalled != containsID(w.sent, j+1) {
				t.Errorf("row %d: step %d sent: %v", i, j+1, m.SendCalled)
			}
			if sleeps[j].SleepCallCount != w.sleeps[j] {
				t.Errorf("row %d: step %d slept %d times, expected %d", i, j+1, sleeps[j].SleepCallCount, w.sleeps[j])
			}
		}
		if res.Iteration != int64(i) {
			t.Errorf("expected iteration %d, got %d", i, res.Iteration)
		}
	}
}

func containsID(ids string, id int) bool {
	for _, c := range ids {
		if int(c-'0') == id {
			return true
		}
	}
	return false
}

func TestSkipIfEmptyEveryStep(t *testing.T) {
	s, p, _, sleeps := sessionService(t, false, []map[string]interface{}{{"turn1": ""}},
		[]sessionStep{{skipIfEmpty: "{{data.sessions.turn1}}", sleeps: true}})
	res, err := s.Do(p, time.Now())
	if err != nil || len(res.StepResults) != 0 || sleeps[0].SleepCalled {
		t.Errorf("a skipped only step: err %v, %d results, slept %v", err, len(res.StepResults), sleeps[0].SleepCalled)
	}
}

func TestStopOnFailure(t *testing.T) {
	failures := map[string]*types.ScenarioStepResult{
		"503":          {StepID: 2, StatusCode: 503},
		"429":          {StepID: 2, StatusCode: 429},
		"timeout":      {StepID: 2, Err: types.RequestError{Type: types.ErrorConn, Reason: types.ReasonReadTimeout}},
		"stream cut":   {StepID: 2, StatusCode: 200, Stream: &types.StreamResult{End: types.StreamEndCut}},
		"error event":  {StepID: 2, StatusCode: 200, Stream: &types.StreamResult{End: types.StreamEndDone, ErrorEvents: 1}},
		"network fail": {StepID: 2, Err: types.RequestError{Type: types.ErrorConn, Reason: types.ReasonConnRefused}},
	}
	rows := []map[string]interface{}{{"turn2": "f"}}
	for name, failed := range failures {
		steps := []sessionStep{{sleeps: true}, {result: failed, sleeps: true}, {skipIfEmpty: "{{data.sessions.turn2}}"}, {}}
		s, p, mocks, sleeps := sessionService(t, true, rows, steps)
		res, _ := s.Do(p, time.Now())
		if got := sentIDs(res); got != "12" {
			t.Errorf("%s: expected steps 12 sent, got %s", name, got)
		}
		if mocks[2].SendCalled || mocks[3].SendCalled {
			t.Errorf("%s: a step after the failure was sent", name)
		}
		if sleeps[1].SleepCalled {
			t.Errorf("%s: the failed step's think time was applied", name)
		}

		// off by default: the session goes on
		s, p, _, _ = sessionService(t, false, rows, steps)
		res, _ = s.Do(p, time.Now())
		if got := sentIDs(res); got != "1234" {
			t.Errorf("%s without stop_on_failure: expected steps 1234 sent, got %s", name, got)
		}
	}

	// a clean stream and a 2xx go on
	ok := &types.ScenarioStepResult{StepID: 2, StatusCode: 200, Stream: &types.StreamResult{End: types.StreamEndDone}}
	s, p, _, _ := sessionService(t, true, rows, []sessionStep{{}, {result: ok}, {}})
	if res, _ := s.Do(p, time.Now()); sentIDs(res) != "123" {
		t.Errorf("an ok step stopped the iteration: %s", sentIDs(res))
	}
}

func TestRandomRowsFollowTheSeed(t *testing.T) {
	rows := make([]map[string]interface{}, 50)
	for i := range rows {
		rows[i] = map[string]interface{}{"q": fmt.Sprint(i)}
	}
	picks := func(seed int64) []interface{} {
		s := &ScenarioService{seed: seed, scenario: types.Scenario{Data: map[string]types.CsvData{"d": {Rows: rows, Random: true}}}}
		var out []interface{}
		for i := int64(0); i < 40; i++ {
			envs := map[string]interface{}{}
			s.enrichEnvFromData(envs, i)
			out = append(out, envs["data.d.q"])
		}
		return out
	}
	a, b, c := picks(1), picks(1), picks(2)
	if fmt.Sprint(a) != fmt.Sprint(b) {
		t.Error("the same seed should pick the same rows")
	}
	if fmt.Sprint(a) == fmt.Sprint(c) {
		t.Error("another seed should pick other rows")
	}
	distinct := map[interface{}]bool{}
	for _, v := range a {
		distinct[v] = true
	}
	if len(distinct) < 15 {
		t.Errorf("rows do not look random: %d distinct of 40", len(distinct))
	}
}
