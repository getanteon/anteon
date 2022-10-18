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
	"math"
	"os"

	"go.ddosify.com/ddosify/core/types"
)

const OutputTypeStdoutJson = "stdout-json"

func init() {
	AvailableOutputServices[OutputTypeStdoutJson] = &stdoutJson{}
}

type stdoutJson struct {
	doneChan          chan struct{}
	result            *Result
	reportPercentiles bool
}

func (s *stdoutJson) Init(reportPercentiles bool) (err error) {
	s.doneChan = make(chan struct{})
	s.result = &Result{
		ItemReports: make(map[int16]*ScenarioItemReport),
	}
	s.reportPercentiles = reportPercentiles

	return
}

func (s *stdoutJson) Start(input chan *types.Response) {
	for r := range input {
		aggregate(s.result, r)
	}
	s.doneChan <- struct{}{}
}

type jsonResult struct {
	SuccessCount int64                             `json:"success_count"`
	FailedCount  int64                             `json:"fail_count"`
	AvgDuration  float32                           `json:"avg_duration"`
	ItemReports  map[int16]*jsonScenarioItemReport `json:"steps"`
}

type jsonScenarioItemReport struct {
	Name           string             `json:"name"`
	StatusCodeDist map[int]int        `json:"status_code_dist"`
	ErrorDist      map[string]int     `json:"error_dist"`
	Durations      map[string]float32 `json:"durations"`
	Percentiles    map[string]float32 `json:"percentiles"`
	SuccessCount   int64              `json:"success_count"`
	FailedCount    int64              `json:"fail_count"`
}

func (s *stdoutJson) Report() {
	jsonResult := jsonResult{
		SuccessCount: s.result.SuccessCount,
		FailedCount:  s.result.FailedCount,
		AvgDuration:  s.result.AvgDuration,
		ItemReports:  map[int16]*jsonScenarioItemReport{},
	}

	for key, item := range s.result.ItemReports {
		jsonResult.ItemReports[key] = &jsonScenarioItemReport{
			Name:           item.Name,
			StatusCodeDist: item.StatusCodeDist,
			ErrorDist:      item.ErrorDist,
			Durations:      item.Durations,
			SuccessCount:   item.SuccessCount,
			FailedCount:    item.FailedCount,
		}

		if s.reportPercentiles {
			jsonResult.ItemReports[key].Percentiles = map[string]float32{
				"p99": item.DurationPercentile(99),
				"p95": item.DurationPercentile(95),
				"p90": item.DurationPercentile(90),
				"p80": item.DurationPercentile(80),
			}
		}

	}

	p := 1e3
	s.result.AvgDuration = float32(math.Round(float64(s.result.AvgDuration)*p) / p)

	for _, item := range jsonResult.ItemReports {
		itemReport := item
		durations := make(map[string]float32)
		for d, s := range itemReport.Durations {
			// Less precision for durations.
			t := math.Round(float64(s)*p) / p
			durations[strKeyToJsonKey[d]] = float32(t)
		}
		itemReport.Durations = durations
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.Encode(&jsonResult)
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

var strKeyToJsonKey = map[string]string{
	"dnsDuration":           "dns",
	"connDuration":          "connection",
	"tlsDuration":           "tls",
	"reqDuration":           "request_write",
	"serverProcessDuration": "server_processing",
	"resDuration":           "response_read",
	"duration":              "total",
}
