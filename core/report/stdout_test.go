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
	"os"
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
		{"S:0-SF:0-AF:0", ScenarioStepResultSummary{SuccessCount: 0, Fail: FailVerbose{Count: 0, ServerErrorDist: ServerErrVerbose{Count: 0}, AssertionErrorDist: AssertionErrVerbose{Count: 0}}}, 0, 0},
		{"S:0-SF:1-AF:0", ScenarioStepResultSummary{SuccessCount: 0, Fail: FailVerbose{Count: 1, ServerErrorDist: ServerErrVerbose{Count: 1}, AssertionErrorDist: AssertionErrVerbose{Count: 0}}}, 0, 100},
		{"S:1-SF:0-AF:0", ScenarioStepResultSummary{SuccessCount: 1, Fail: FailVerbose{Count: 0, ServerErrorDist: ServerErrVerbose{Count: 0}, AssertionErrorDist: AssertionErrVerbose{Count: 0}}}, 100, 0},
		{"S:3-SF:9-AF:6", ScenarioStepResultSummary{SuccessCount: 3, Fail: FailVerbose{Count: 9, ServerErrorDist: ServerErrVerbose{Count: 3}, AssertionErrorDist: AssertionErrVerbose{Count: 6}}}, 25, 75},
		{"S:5-SF:2-AF:3", ScenarioStepResultSummary{SuccessCount: 5, Fail: FailVerbose{Count: 5, ServerErrorDist: ServerErrVerbose{Count: 2}, AssertionErrorDist: AssertionErrVerbose{Count: 3}}}, 50, 50},
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
		{"S:0-F:0", Result{ServerFailedCount: 0, SuccessCount: 0}, 0, 0},
		{"S:0-F:1", Result{ServerFailedCount: 1, SuccessCount: 0}, 0, 100},
		{"S:1-F:0", Result{ServerFailedCount: 0, SuccessCount: 1}, 100, 0},
		{"S:3-F:9", Result{ServerFailedCount: 9, SuccessCount: 3}, 25, 75},
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
	s.Init(debug, 0)

	if s.doneChan == nil {
		t.Errorf("DoneChan should be initialized")
	}

	if s.result == nil {
		t.Errorf("Result map should be initialized")
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
	s.Init(true, 0)
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
				Custom:        map[string]interface{}{},
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
