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
	"reflect"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

func TestInitStdoutJson(t *testing.T) {
	sj := &stdoutJson{}
	sj.Init()

	if sj.doneChan == nil {
		t.Errorf("DoneChan should be initialized")
	}

	if sj.result == nil {
		t.Errorf("Result map should be initialized")
	}
}

func TestStdoutJsonStart(t *testing.T) {
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

	s := &stdoutJson{}
	s.Init()

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
		t.Errorf("Expected %#v, Found %#v", expectedResult, *s.result)
	}
}

func TestStdoutJsonOutput(t *testing.T) {
	// Arrange
	itemReport1 := &ScenarioItemReport{
		StatusCodeDist: map[int]int{200: 11},
		SuccessCount:   11,
		FailedCount:    0,
		Durations: map[string]float32{
			"dnsDuration":  0.1897,
			"connDuration": 0.0003,
			"duration":     0.1900,
		},
		ErrorDist: map[string]int{},
	}
	itemReport2 := &ScenarioItemReport{
		StatusCodeDist: map[int]int{401: 1, 200: 9},
		SuccessCount:   9,
		FailedCount:    2,
		Durations: map[string]float32{
			"dnsDuration":  0.48000,
			"connDuration": 0.01356,
			"duration":     0.493566,
		},
		ErrorDist: map[string]int{types.ReasonConnTimeout: 2},
	}
	result := Result{
		SuccessCount: 9,
		FailedCount:  2,
		AvgDuration:  0.25637,
		ItemReports: map[int16]*ScenarioItemReport{
			int16(1): itemReport1,
			int16(2): itemReport2,
		},
	}

	var output string
	printJson = func(j []byte) {
		output = string(j)
	}

	expectedOutputByte := []byte(`{
		"success_perc": 81,
		"fail_perc": 19,
		"success_count": 9,
		"fail_count": 2,
		"avg_duration": 0.256,
		"steps": {
			"1": {
				"name": "",
				"status_code_dist": {
					"200": 11
				},
				"error_dist": {},
				"durations": {
					"connection": 0,
					"dns": 0.19,
					"total": 0.19
				},
				"success_count": 11,
				"fail_count": 0,
				"success_perc": 100,
				"fail_perc": 0
			},
			"2": {
				"name": "",
				"status_code_dist": {
					"200": 9,
					"401": 1
				},
				"error_dist": {
					"connection timeout": 2
				},
				"durations": {
					"connection": 0.014,
					"dns": 0.48,
					"total": 0.494
				},
				"success_count": 9,
				"fail_count": 2,
				"success_perc": 81,
				"fail_perc": 19
			}
		}
	}`)
	buffer := new(bytes.Buffer)
	json.Compact(buffer, expectedOutputByte)
	expectedOutput := buffer.String()

	// Act
	s := &stdoutJson{result: &result}
	s.Report()

	// Assert
	if output != expectedOutput {
		t.Errorf("Expected: %v, Found: %v", expectedOutput, output)
	}
}
