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
	"strings"
	"time"

	"go.ddosify.com/ddosify/core/assertion"
	"go.ddosify.com/ddosify/core/types"
)

func aggregate(result *Result, scr *types.ScenarioResult, samplingCount map[uint16]map[string]int, samplingRate int) {
	var scenarioDuration float32
	errOccured := false
	assertionFail := false
	for _, sr := range scr.StepResults {
		scenarioDuration += float32(sr.Duration.Seconds())

		fv := FailVerbose{}
		fv.AssertionErrorDist.Conditions = make(map[string]*AssertInfo)
		fv.ServerErrorDist.Reasons = make(map[string]int)

		if _, ok := result.StepResults[sr.StepID]; !ok {
			result.StepResults[sr.StepID] = &ScenarioStepResultSummary{
				Name:           sr.StepName,
				StatusCodeDist: make(map[int]int, 0),
				Fail:           fv,
				Durations:      map[string]float32{},
				SuccessCount:   0,
			}
		}
		stepResult := result.StepResults[sr.StepID]

		if len(sr.FailedAssertions) > 0 { // assertion error
			errOccured = true
			assertionFail = true
			stepResult.Fail.Count++
			stepResult.Fail.AssertionErrorDist.Count++
			stepResult.StatusCodeDist[sr.StatusCode]++
			for _, fa := range sr.FailedAssertions {
				if aed, ok := stepResult.Fail.AssertionErrorDist.Conditions[fa.Rule]; !ok {
					samplingCount[sr.StepID] = make(map[string]int)
					samplingCount[sr.StepID][fa.Rule] = 1
					ae := &AssertInfo{
						Count:    1,
						Received: make(map[string][]interface{}),
						Reason:   fa.Reason,
					}

					for ident, value := range fa.Received {
						ae.Received[ident] = []interface{}{value}
					}

					stepResult.Fail.AssertionErrorDist.Conditions[fa.Rule] = ae
				} else {
					aed.Count++
					samplingCount[sr.StepID][fa.Rule]++
					if samplingCount[sr.StepID][fa.Rule] <= samplingRate {

						for ident, value := range fa.Received {
							// do not append if the value is already in the list
							exists := false
							for _, v := range aed.Received[ident] {
								if v == value {
									exists = true
								}
							}
							if !exists {
								aed.Received[ident] = append(aed.Received[ident], value)
							}
						}
					}
				}
			}
			totalDur := float32(stepResult.SuccessCount+stepResult.Fail.Count-1)*stepResult.Durations["duration"] + float32(sr.Duration.Seconds())
			stepResult.Durations["duration"] = totalDur / float32(stepResult.SuccessCount+stepResult.Fail.Count)
			for k, v := range sr.Custom {
				if strings.Contains(k, "Duration") {
					totalDur := float32(stepResult.SuccessCount+stepResult.Fail.Count-1)*stepResult.Durations[k] + float32(v.(time.Duration).Seconds())
					stepResult.Durations[k] = float32(totalDur / float32(stepResult.SuccessCount+stepResult.Fail.Count))
				}
			}
		} else if sr.Err.Type != "" { // server error
			errOccured = true
			stepResult.Fail.Count++
			stepResult.Fail.ServerErrorDist.Count++
			stepResult.Fail.ServerErrorDist.Reasons[sr.Err.Reason]++
		} else { // success
			stepResult.StatusCodeDist[sr.StatusCode]++
			stepResult.SuccessCount++

			totalDur := float32(stepResult.SuccessCount+stepResult.Fail.Count-1)*stepResult.Durations["duration"] + float32(sr.Duration.Seconds())
			stepResult.Durations["duration"] = totalDur / float32(stepResult.SuccessCount+stepResult.Fail.Count)
			for k, v := range sr.Custom {
				if strings.Contains(k, "Duration") {
					totalDur := float32(stepResult.SuccessCount-1)*stepResult.Durations[k] + float32(v.(time.Duration).Seconds())
					stepResult.Durations[k] = float32(totalDur / float32(stepResult.SuccessCount+stepResult.Fail.Count))
				}
			}
		}

	}

	// Don't change avg duration if there is a error
	if !errOccured {
		totalDuration := float32(result.SuccessCount)*result.AvgDuration + scenarioDuration
		result.SuccessCount++
		result.AvgDuration = totalDuration / float32(result.SuccessCount)
	} else if errOccured {
		if assertionFail { // if any step failed because of assertion, that iteration counts as assertion fail
			result.AssertionFailCount++
		} else { // server error
			result.ServerFailedCount++
		}
	}
}

// Total test result, all scenario iterations combined
type Result struct {
	TestStatus           string                                `json:"test_status"`
	TestFailedAssertions []assertion.FailedRule                `json:"failed_criterias"`
	SuccessCount         int64                                 `json:"success_count"`
	ServerFailedCount    int64                                 `json:"server_fail_count"`
	AssertionFailCount   int64                                 `json:"assertion_fail_count"`
	AvgDuration          float32                               `json:"avg_duration"`
	StepResults          map[uint16]*ScenarioStepResultSummary `json:"steps"`
}

func (r *Result) successPercentage() int {
	if r.SuccessCount+r.ServerFailedCount+r.AssertionFailCount == 0 {
		return 0
	}
	t := float32(r.SuccessCount) / float32(r.SuccessCount+r.ServerFailedCount+r.AssertionFailCount)
	return int(t * 100)
}

func (r *Result) failedPercentage() int {
	if r.SuccessCount+r.ServerFailedCount+r.AssertionFailCount == 0 {
		return 0
	}
	return 100 - r.successPercentage()
}

type AssertionErrVerbose struct {
	Count      int64                  `json:"count"`
	Conditions map[string]*AssertInfo `json:"conditions"`
}

type ServerErrVerbose struct {
	Count   int64          `json:"count"`
	Reasons map[string]int `json:"reasons"`
}

type FailVerbose struct {
	Count              int64               `json:"count"`
	AssertionErrorDist AssertionErrVerbose `json:"assertions"`
	ServerErrorDist    ServerErrVerbose    `json:"server"`
}

type ScenarioStepResultSummary struct {
	Name           string             `json:"name"`
	StatusCodeDist map[int]int        `json:"status_code_dist"`
	Fail           FailVerbose        `json:"fail"`
	Durations      map[string]float32 `json:"durations"`
	SuccessCount   int64              `json:"success_count"`
}

func (s *ScenarioStepResultSummary) successPercentage() int {
	if s.SuccessCount+s.Fail.Count == 0 {
		return 0
	}
	t := float32(s.SuccessCount) / float32(s.SuccessCount+s.Fail.Count)
	return int(t * 100)
}

func (s *ScenarioStepResultSummary) failedPercentage() int {
	if s.SuccessCount+s.Fail.Count == 0 {
		return 0
	}
	return 100 - s.successPercentage()
}

type AssertInfo struct {
	Count    int
	Received map[string][]interface{}
	Reason   string
}
