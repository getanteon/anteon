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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

//TODO:move aggregator.go related tests cases to aggregator_test.go

func TestScenarioStepReport(t *testing.T) {
	tests := []struct {
		name              string
		s                 ScenarioStepResultSummary
		successPercentage int
		failedPercentage  int
	}{
		{"S:0-F:0", ScenarioStepResultSummary{FailedCount: 0, SuccessCount: 0}, 0, 0},
		{"S:0-F:1", ScenarioStepResultSummary{FailedCount: 1, SuccessCount: 0}, 0, 100},
		{"S:1-F:0", ScenarioStepResultSummary{FailedCount: 0, SuccessCount: 1}, 100, 0},
		{"S:3-F:9", ScenarioStepResultSummary{FailedCount: 9, SuccessCount: 3}, 25, 75},
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
	debug := false
	s.Init(debug)

	if s.doneChan == nil {
		t.Errorf("DoneChan should be initialized")
	}

	if s.result == nil {
		t.Errorf("Result map should be initialized")
	}
}

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
			},
		},
	}

	itemReport1 := &ScenarioStepResultSummary{
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
	itemReport2 := &ScenarioStepResultSummary{
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
		StepResults: map[uint16]*ScenarioStepResultSummary{
			uint16(1): itemReport1,
			uint16(2): itemReport2,
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

	if !reflect.DeepEqual(*s.result, expectedResult) {
		t.Errorf("2Expected %#v, Found %#v", expectedResult, *s.result)
	}
}

func TestPrintJsonBody(t *testing.T) {
	var byteArr []byte
	buffer := bytes.NewBuffer(byteArr)

	contentTypeJson := "application/json"
	body := map[string]interface{}{"x": "y"}
	printBody(buffer, contentTypeJson, body)

	printedBody := buffer.Bytes()

	if !json.Valid(printedBody) {
		t.Errorf("Printed body is not valid json: %v", string(printedBody))
	}
}

func TestPrintBodyAsString(t *testing.T) {
	var byteArr []byte
	buffer := bytes.NewBuffer(byteArr)

	contentTypeAny := "any"
	body := "argentina"
	printBody(buffer, contentTypeAny, body)

	printedBody := buffer.Bytes()

	if !strings.Contains(string(printedBody), body) {
		t.Errorf("Printed body does not match expected: %s, found: %v", body, string(printedBody))
	}
}

func TestStdoutPrintsHeadlinesInDebugMode(t *testing.T) {
	s := &stdout{}
	s.Init(true)
	testDoneChan := make(chan struct{}, 1)

	// listen to output
	realOut := out
	r, w, _ := os.Pipe()
	out = w
	defer func() {
		out = realOut
	}()

	inputChan := make(chan *types.ScenarioResult, 1)
	inputChan <- &types.ScenarioResult{
		StepResults: []*types.ScenarioStepResult{
			{
				StepID:        0,
				StepName:      "",
				RequestID:     [16]byte{},
				StatusCode:    0,
				RequestTime:   time.Time{},
				Duration:      0,
				ContentLength: 0,
				Err:           types.RequestError{},
				DebugInfo: map[string]interface{}{
					"requestBody":     []byte{},
					"requestHeaders":  http.Header{},
					"url":             "",
					"method":          "",
					"responseBody":    []byte{},
					"responseHeaders": http.Header{},
				},
				Custom: map[string]interface{}{},
			},
		},
	}
	close(inputChan)

	go func() {
		s.Start(inputChan)
		w.Close()
	}()

	go func() {
		// wait for print and debug
		<-s.DoneChan()

		printedOutput, err := ioutil.ReadAll(r)
		t.Log(err)
		t.Log(printedOutput)

		outStr := string(printedOutput)
		if !strings.Contains(outStr, "Environment Variables") ||
			!strings.Contains(outStr, "- Request") ||
			!strings.Contains(outStr, "Headers:") ||
			!strings.Contains(outStr, "Body:") ||
			!strings.Contains(outStr, "- Response") ||
			!strings.Contains(outStr, "StatusCode:") {

			t.Errorf("One or multiple headlines are missing in stdout debug mode")
		}

		testDoneChan <- struct{}{}
	}()

	<-testDoneChan

}
