package report

import (
	"reflect"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

func TestStart(t *testing.T) {
	responses := []*types.ScenarioResult{
		{
			StartTime: time.Now(),
			StepResults: []*types.ScenarioStepResult{
				{
					StepID:      1,
					StatusCode:  200,
					RequestTime: time.Now().Add(1),
					Duration:    time.Duration(10) * time.Second,
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(5) * time.Second,
						"connDuration": time.Duration(5) * time.Second,
					},
				},
				{
					StepID:      2,
					RequestTime: time.Now().Add(2),
					Duration:    time.Duration(30) * time.Second,
					Err:         types.RequestError{Type: types.ErrorConn, Reason: types.ReasonConnTimeout},
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(10) * time.Second,
						"connDuration": time.Duration(20) * time.Second,
					},
				},
				{
					StepID:      3,
					StatusCode:  400,
					RequestTime: time.Now().Add(2),
					Duration:    time.Duration(30) * time.Second,
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(10) * time.Second,
						"connDuration": time.Duration(20) * time.Second,
					},
					FailedAssertions: []types.FailedAssertion{{
						Rule:     "equals(status_code,200)",
						Received: map[string]interface{}{"status_code": 400},
					}},
				},
			},
		},
		{
			StartTime: time.Now().Add(10),
			StepResults: []*types.ScenarioStepResult{
				{
					StepID:      1,
					StatusCode:  200,
					RequestTime: time.Now().Add(11),
					Duration:    time.Duration(30) * time.Second,
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(10) * time.Second,
						"connDuration": time.Duration(20) * time.Second,
					},
				},
				{
					StepID:      2,
					StatusCode:  401,
					RequestTime: time.Now().Add(12),
					Duration:    time.Duration(60) * time.Second,
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(20) * time.Second,
						"connDuration": time.Duration(40) * time.Second,
					},
				},
				{
					StepID:      3,
					StatusCode:  200,
					RequestTime: time.Now().Add(2),
					Duration:    time.Duration(30) * time.Second,
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(10) * time.Second,
						"connDuration": time.Duration(20) * time.Second,
					},
				},
			},
		},
	}

	fail1 := FailVerbose{}
	fail1.Count = 0
	fail1.ServerErrorDist.Count = 0
	fail1.ServerErrorDist.Reasons = make(map[string]int)
	fail1.AssertionErrorDist.Conditions = make(map[string]*AssertInfo)
	itemReport1 := &ScenarioStepResultSummary{
		StatusCodeDist: map[int]int{200: 2},
		SuccessCount:   2,
		Fail:           fail1,
		Durations: map[string]float32{
			"dnsDuration":  7.5,
			"connDuration": 12.5,
			"duration":     20,
		},
	}

	fail2 := FailVerbose{}
	fail2.Count = 1
	fail2.ServerErrorDist.Count = 1
	fail2.ServerErrorDist.Reasons = make(map[string]int)
	fail2.ServerErrorDist.Reasons[types.ReasonConnTimeout] = 1
	fail2.AssertionErrorDist.Conditions = make(map[string]*AssertInfo)
	itemReport2 := &ScenarioStepResultSummary{
		StatusCodeDist: map[int]int{401: 1},
		SuccessCount:   1,
		Fail:           fail2,
		Durations: map[string]float32{
			"dnsDuration":  20,
			"connDuration": 40,
			"duration":     60,
		},
	}

	fail3 := FailVerbose{}
	fail3.Count = 1
	fail3.AssertionErrorDist.Count = 1
	fail3.ServerErrorDist.Reasons = make(map[string]int)
	fail3.AssertionErrorDist.Conditions = make(map[string]*AssertInfo)
	fail3.AssertionErrorDist.Conditions["equals(status_code,200)"] = &AssertInfo{
		Count: 1,
		Received: map[string][]interface{}{
			"status_code": {400},
		},
	}

	itemReport3 := &ScenarioStepResultSummary{
		StatusCodeDist: map[int]int{400: 1, 200: 1},
		SuccessCount:   1,
		Fail:           fail3,
		Durations: map[string]float32{
			"dnsDuration":  20,
			"connDuration": 40,
			"duration":     60,
		},
	}

	expectedResult := Result{
		SuccessCount:       1,
		ServerFailedCount:  0,
		AssertionFailCount: 1,
		AvgDuration:        120,
		StepResults: map[uint16]*ScenarioStepResultSummary{
			uint16(1): itemReport1,
			uint16(2): itemReport2,
			uint16(3): itemReport3,
		},
	}

	s := &stdout{}
	debug := false
	s.Init(debug, 0)

	responseChan := make(chan *types.ScenarioResult, len(responses))
	go s.Start(responseChan, nil)

	go func() {
		for _, r := range responses {
			responseChan <- r
		}
		close(responseChan)
	}()

	doneChanSignaled := false
	select {
	case <-s.doneChan:
		doneChanSignaled = true
	case <-time.After(time.Duration(1) * time.Second):
	}

	if !doneChanSignaled {
		t.Errorf("DoneChan is not signaled")
	}

	if !compareResults(s.result, &expectedResult) {
		t.Errorf("Expected %#v, Found %#v", s.result, expectedResult)

	}
}

func compareResults(r1, r2 *Result) bool {

	if r1.successPercentage() != r2.successPercentage() ||
		r1.failedPercentage() != r2.failedPercentage() ||
		r1.SuccessCount != r2.SuccessCount ||
		r1.AvgDuration != r2.AvgDuration ||
		r1.ServerFailedCount != r2.ServerFailedCount ||
		r1.AssertionFailCount != r2.AssertionFailCount {
		return false
	}

	for stepId, sr := range r1.StepResults {
		if !compareStepResults(sr, r2.StepResults[stepId]) {
			return false
		}
	}

	return true
}

func compareStepResults(s1, s2 *ScenarioStepResultSummary) bool {
	if s1.successPercentage() != s2.successPercentage() ||
		s1.failedPercentage() != s2.failedPercentage() ||
		s1.SuccessCount != s2.SuccessCount ||
		s1.Name != s2.Name ||
		s1.Fail.Count != s2.Fail.Count ||
		s1.Fail.AssertionErrorDist.Count != s2.Fail.AssertionErrorDist.Count ||
		s1.Fail.ServerErrorDist.Count != s2.Fail.ServerErrorDist.Count ||
		!reflect.DeepEqual(s1.Fail.AssertionErrorDist.Conditions, s2.Fail.AssertionErrorDist.Conditions) ||
		!reflect.DeepEqual(s1.Fail.ServerErrorDist.Reasons, s2.Fail.ServerErrorDist.Reasons) {
		return false
	}
	return true
}
