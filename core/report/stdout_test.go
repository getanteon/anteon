/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */
package report

import (
	"reflect"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

//TODO:move aggregator.go related tests cases to aggregator_test.go

func TestScenarioItemReport(t *testing.T) {
	tests := []struct {
		name              string
		s                 ScenarioItemReport
		successPercentage int
		failedPercentage  int
	}{
		{"S:0-F:0", ScenarioItemReport{FailedCount: 0, SuccessCount: 0}, 0, 0},
		{"S:0-F:1", ScenarioItemReport{FailedCount: 1, SuccessCount: 0}, 0, 100},
		{"S:1-F:0", ScenarioItemReport{FailedCount: 0, SuccessCount: 1}, 100, 0},
		{"S:3-F:9", ScenarioItemReport{FailedCount: 9, SuccessCount: 3}, 25, 75},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			sp := test.s.successPercentage()
			fp := test.s.failedPercentage()

			if test.successPercentage != sp {
				t.Errorf("SuccessPercentage Expected %d Found %d", test.successPercentage, sp)
			}

			if test.failedPercentage != fp {
				t.Errorf("FailedPercentage Expected %d Found %d", test.failedPercentage, fp)
			}
		}
		t.Run(test.name, tf)
	}
}

func TestResult(t *testing.T) {
	tests := []struct {
		name              string
		r                 Result
		successPercentage int
		failedPercentage  int
	}{
		{"S:0-F:0", Result{FailedCount: 0, SuccessCount: 0}, 0, 0},
		{"S:0-F:1", Result{FailedCount: 1, SuccessCount: 0}, 0, 100},
		{"S:1-F:0", Result{FailedCount: 0, SuccessCount: 1}, 100, 0},
		{"S:3-F:9", Result{FailedCount: 9, SuccessCount: 3}, 25, 75},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			sp := test.r.successPercentage()
			fp := test.r.failedPercentage()

			if test.successPercentage != sp {
				t.Errorf("SuccessPercentage Expected %d Found %d", test.successPercentage, sp)
			}

			if test.failedPercentage != fp {
				t.Errorf("FailedPercentage Expected %d Found %d", test.failedPercentage, fp)
			}
		}
		t.Run(test.name, tf)
	}
}

func TestInit(t *testing.T) {
	s := &stdout{}
	s.Init(false)

	if s.doneChan == nil {
		t.Errorf("DoneChan should be initialized")
	}

	if s.result == nil {
		t.Errorf("Result map should be initialized")
	}
}

func TestStart(t *testing.T) {
	responses := []*types.Response{
		{
			StartTime: time.Now(),
			ResponseItems: []*types.ResponseItem{
				{
					ScenarioItemID: 1,
					StatusCode:     200,
					RequestTime:    time.Now().Add(1),
					Duration:       time.Duration(10) * time.Second,
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(5) * time.Second,
						"connDuration": time.Duration(5) * time.Second,
					},
				},
				{
					ScenarioItemID: 2,
					RequestTime:    time.Now().Add(2),
					Duration:       time.Duration(30) * time.Second,
					Err:            types.RequestError{Type: types.ErrorConn, Reason: types.ReasonConnTimeout},
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(10) * time.Second,
						"connDuration": time.Duration(20) * time.Second,
					},
				},
			},
		},
		{
			StartTime: time.Now().Add(10),
			ResponseItems: []*types.ResponseItem{
				{
					ScenarioItemID: 1,
					StatusCode:     200,
					RequestTime:    time.Now().Add(11),
					Duration:       time.Duration(30) * time.Second,
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(10) * time.Second,
						"connDuration": time.Duration(20) * time.Second,
					},
				},
				{
					ScenarioItemID: 2,
					StatusCode:     401,
					RequestTime:    time.Now().Add(12),
					Duration:       time.Duration(60) * time.Second,
					Custom: map[string]interface{}{
						"dnsDuration":  time.Duration(20) * time.Second,
						"connDuration": time.Duration(40) * time.Second,
					},
				},
			},
		},
	}

	itemReport1 := &ScenarioItemReport{
		StatusCodeDist: map[int]int{200: 2},
		SuccessCount:   2,
		FailedCount:    0,
		Durations: map[string]float32{
			"dnsDuration":  7.5,
			"connDuration": 12.5,
			"duration":     20,
		},
		ErrorDist: map[string]int{},
	}
	itemReport2 := &ScenarioItemReport{
		StatusCodeDist: map[int]int{401: 1},
		SuccessCount:   1,
		FailedCount:    1,
		Durations: map[string]float32{
			"dnsDuration":  20,
			"connDuration": 40,
			"duration":     60,
		},
		ErrorDist: map[string]int{types.ReasonConnTimeout: 1},
	}

	expectedResult := Result{
		SuccessCount: 1,
		FailedCount:  1,
		AvgDuration:  90,
		ItemReports: map[int16]*ScenarioItemReport{
			int16(1): itemReport1,
			int16(2): itemReport2,
		},
	}

	s := &stdout{}
	s.Init(false)

	responseChan := make(chan *types.Response, len(responses))
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

	if !reflect.DeepEqual(*s.result, expectedResult) {
		t.Errorf("2Expected %#v, Found %#v", expectedResult, *s.result)
	}
}
