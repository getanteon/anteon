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

	itemReport1 := &ScenarioStepResultSummary{
		StatusCodeDist:    map[int]int{200: 2},
		SuccessCount:      2,
		ServerFailedCount: 0,
		Durations: map[string]float32{
			"dnsDuration":  7.5,
			"connDuration": 12.5,
			"duration":     20,
		},
		ServerErrorDist:    make(map[string]int),
		AssertionErrorDist: map[string]*AssertInfo{},
	}
	itemReport2 := &ScenarioStepResultSummary{
		StatusCodeDist:    map[int]int{401: 1},
		SuccessCount:      1,
		ServerFailedCount: 1,
		Durations: map[string]float32{
			"dnsDuration":  20,
			"connDuration": 40,
			"duration":     60,
		},
		ServerErrorDist:    map[string]int{types.ReasonConnTimeout: 1},
		AssertionErrorDist: map[string]*AssertInfo{},
	}
	itemReport3 := &ScenarioStepResultSummary{
		StatusCodeDist:     map[int]int{400: 1, 200: 1},
		SuccessCount:       1,
		AssertionFailCount: 1,
		Durations: map[string]float32{
			"dnsDuration":  20,
			"connDuration": 40,
			"duration":     60,
		},
		ServerErrorDist: map[string]int{types.ReasonConnTimeout: 1},
		AssertionErrorDist: map[string]*AssertInfo{
			"equals(status_code,200)": {
				Count: 1,
				Received: map[string][]interface{}{
					"status_code": {400},
				},
			},
		},
	}

	expectedResult := Result{
		SuccessCount:       1,
		ServerFailedCount:  1,
		AssertionFailCount: 0,
		AvgDuration:        90,
		StepResults: map[uint16]*ScenarioStepResultSummary{
			uint16(1): itemReport1,
			uint16(2): itemReport2,
			uint16(3): itemReport3,
		},
	}

	s := &stdout{}
	debug := false
	s.Init(debug)

	responseChan := make(chan *types.ScenarioResult, len(responses))
	go s.Start(responseChan)

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

	if !reflect.DeepEqual(s.result.StepResults[0], expectedResult.StepResults[0]) {
		t.Errorf("Expected %#v, Found %#v", expectedResult, *s.result)
	}
	if !reflect.DeepEqual(s.result.StepResults[1], expectedResult.StepResults[1]) {
		t.Errorf("Expected %#v, Found %#v", expectedResult, *s.result)
	}
	if !reflect.DeepEqual(s.result.StepResults[2], expectedResult.StepResults[2]) {
		t.Errorf("Expected %#v, Found %#v", expectedResult, *s.result)
	}
}
