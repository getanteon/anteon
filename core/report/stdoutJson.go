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
	"encoding/json"
	"fmt"
	"io"
	"math"

	"go.ddosify.com/ddosify/core/types"
)

const OutputTypeStdoutJson = "stdout-json"

func init() {
	AvailableOutputServices[OutputTypeStdoutJson] = &stdoutJson{}
}

type stdoutJson struct {
	doneChan chan struct{}
	result   *Result
	debug    bool
}

func (s *stdoutJson) Init(debug bool) (err error) {
	s.doneChan = make(chan struct{})
	s.result = &Result{
		StepResults: make(map[uint16]*ScenarioStepResultSummary),
	}
	s.debug = debug
	return
}

func (s *stdoutJson) Start(input chan *types.ScenarioResult) {
	if s.debug {
		s.printInDebugMode(input)
		s.doneChan <- struct{}{}
		return
	}
	s.listenAndAggregate(input)
	s.report()
	s.doneChan <- struct{}{}
}

func (s *stdoutJson) report() {
	p := 1e3

	s.result.AvgDuration = float32(math.Round(float64(s.result.AvgDuration)*p) / p)

	for _, itemReport := range s.result.StepResults {
		durations := make(map[string]float32)
		for d, s := range itemReport.Durations {
			// Less precision for durations.
			t := math.Round(float64(s)*p) / p
			durations[strKeyToJsonKey[d]] = float32(t)
		}
		itemReport.Durations = durations
	}

	j, _ := json.Marshal(s.result)
	printJson(j)
}

func (s *stdoutJson) DoneChan() <-chan struct{} {
	return s.doneChan
}

func (s *stdoutJson) listenAndAggregate(input chan *types.ScenarioResult) {
	for r := range input {
		aggregate(s.result, r)
	}
}

func (s *stdoutJson) printInDebugMode(input chan *types.ScenarioResult) {
	stepDebugResults := struct {
		DebugResults map[uint16]verboseHttpRequestInfo "json:\"steps\""
	}{
		DebugResults: map[uint16]verboseHttpRequestInfo{},
	}
	for r := range input { // only 1 sc ScenarioResult expected
		for _, sr := range r.StepResults {
			verboseInfo := ScenarioStepResultToVerboseHttpRequestInfo(sr)
			stepDebugResults.DebugResults[verboseInfo.StepId] = verboseInfo
		}
	}

	printPretty(out, stepDebugResults)
}

func printPretty(w io.Writer, info any) {
	valPretty, _ := json.MarshalIndent(info, "", "  ")
	fmt.Fprintf(out, "%s \n",
		white(fmt.Sprintf(" %-6s",
			valPretty)))
}

// Report wraps Result to add success/fails percentage values
type Report Result

func (r Result) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		SuccesPerc int `json:"success_perc"`
		FailPerc   int `json:"fail_perc"`
		Report
	}{
		SuccesPerc: r.successPercentage(),
		FailPerc:   r.failedPercentage(),
		Report:     Report(r),
	})
}

// ItemReport wraps ScenarioStepReport to add success/fails percentage values
type ItemReport ScenarioStepResultSummary

func (s ScenarioStepResultSummary) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ItemReport
		SuccesPerc int `json:"success_perc"`
		FailPerc   int `json:"fail_perc"`
	}{
		ItemReport: ItemReport(s),
		SuccesPerc: s.successPercentage(),
		FailPerc:   s.failedPercentage(),
	})
}

var printJson = func(j []byte) {
	fmt.Println(string(j))
}

var strKeyToJsonKey = map[string]string{
	"dnsDuration":           "dns",
	"connDuration":          "connection",
	"tlsDuration":           "tls",
	"reqDuration":           "request_write",
	"serverProcessDuration": "server_processing",
	"resDuration":           "response_read",
	"duration":              "total",
}

func (v verboseHttpRequestInfo) MarshalJSON() ([]byte, error) {
	// could not prepare req, correlation
	if v.Error != "" && isVerboseInfoRequestEmpty(v.Request) {
		type alias struct {
			StepId         uint16                 `json:"stepId"`
			StepName       string                 `json:"stepName"`
			Envs           map[string]interface{} `json:"envs"`
			TestData       map[string]interface{} `json:"testData"`
			FailedCaptures map[string]string      `json:"failedCaptures"`
			Error          string                 `json:"error"`
		}

		a := alias{
			Error:          v.Error,
			StepId:         v.StepId,
			StepName:       v.StepName,
			FailedCaptures: v.FailedCaptures,
			Envs:           v.Envs,
			TestData:       v.TestData,
		}
		return json.Marshal(a)
	}

	if v.Error != "" {
		type alias struct {
			StepId         uint16                 `json:"stepId"`
			StepName       string                 `json:"stepName"`
			Envs           map[string]interface{} `json:"envs"`
			TestData       map[string]interface{} `json:"testData"`
			FailedCaptures map[string]string      `json:"failedCaptures"`
			Request        struct {
				Url     string            `json:"url"`
				Method  string            `json:"method"`
				Headers map[string]string `json:"headers"`
				Body    interface{}       `json:"body"`
			} `json:"request"`
			Error string `json:"error"`
		}

		a := alias{
			Request:        v.Request,
			Error:          v.Error,
			StepId:         v.StepId,
			StepName:       v.StepName,
			FailedCaptures: v.FailedCaptures,
			Envs:           v.Envs,
			TestData:       v.TestData,
		}
		return json.Marshal(a)
	}

	type alias struct {
		StepId         uint16                 `json:"stepId"`
		StepName       string                 `json:"stepName"`
		Envs           map[string]interface{} `json:"envs"`
		TestData       map[string]interface{} `json:"testData"`
		FailedCaptures map[string]string      `json:"failedCaptures"`
		Request        struct {
			Url     string            `json:"url"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
			Body    interface{}       `json:"body"`
		} `json:"request"`
		Response struct {
			StatusCode int               `json:"statusCode"`
			Headers    map[string]string `json:"headers"`
			Body       interface{}       `json:"body"`
		} `json:"response"`
	}

	a := alias{
		StepId:         v.StepId,
		StepName:       v.StepName,
		Request:        v.Request,
		Response:       v.Response,
		FailedCaptures: v.FailedCaptures,
		Envs:           v.Envs,
		TestData:       v.TestData,
	}
	return json.Marshal(a)

}
