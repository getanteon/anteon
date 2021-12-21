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
}

func (s *stdoutJson) Init() (err error) {
	s.doneChan = make(chan struct{})
	s.result = &Result{
		ItemReports: make(map[int16]*ScenarioItemReport),
	}
	return
}

func (s *stdoutJson) Start(input chan *types.Response) {
	for r := range input {
		aggregate(s.result, r)
	}
	s.doneChan <- struct{}{}
}

func (s *stdoutJson) Report() {
	p := 1e3

	s.result.AvgDuration = float32(math.Round(float64(s.result.AvgDuration)*p) / p)

	for _, itemReport := range s.result.ItemReports {
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

// ItemReport wraps ScenarioItemReport to add success/fails percentage values
type ItemReport ScenarioItemReport

func (s ScenarioItemReport) MarshalJSON() ([]byte, error) {
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
