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

	"go.ddosify.com/ddosify/core/types"
)

const samplingMax = 3 // per second

func aggregate(result *Result, scr *types.ScenarioResult, samplingCount map[uint16]map[string]int) {
	var scenarioDuration float32
	errOccured := false
	for _, sr := range scr.StepResults {
		scenarioDuration += float32(sr.Duration.Seconds())

		if _, ok := result.StepResults[sr.StepID]; !ok {
			result.StepResults[sr.StepID] = &ScenarioStepResultSummary{
				Name:               sr.StepName,
				StatusCodeDist:     make(map[int]int, 0),
				AssertionErrorDist: map[string]*AssertInfo{},
				ServerErrorDist:    make(map[string]int),
				Durations:          map[string]float32{},
			}
		}
		stepResult := result.StepResults[sr.StepID]

		if len(sr.FailedAssertions) > 0 { // assertion error
			errOccured = true
			stepResult.AssertionFailCount++
			stepResult.StatusCodeDist[sr.StatusCode]++
			for _, fa := range sr.FailedAssertions {
				if aed, ok := stepResult.AssertionErrorDist[fa.Rule]; !ok {
					samplingCount[sr.StepID] = make(map[string]int)
					samplingCount[sr.StepID][fa.Rule] = 1
					ae := &AssertInfo{
						Count:    1,
						Received: make(map[string][]interface{}),
					}

					for ident, value := range fa.Received {
						ae.Received[ident] = []interface{}{value}
					}

					stepResult.AssertionErrorDist[fa.Rule] = ae
				} else {
					aed.Count++
					samplingCount[sr.StepID][fa.Rule]++
					if samplingCount[sr.StepID][fa.Rule] <= samplingMax {
						for ident, value := range fa.Received {
							aed.Received[ident] = append(aed.Received[ident], value)
						}
					}
				}
			}
			totalDur := float32(stepResult.SuccessCount+stepResult.AssertionFailCount-1)*stepResult.Durations["duration"] + float32(sr.Duration.Seconds())
			stepResult.Durations["duration"] = totalDur / float32(stepResult.SuccessCount+stepResult.AssertionFailCount)
			for k, v := range sr.Custom {
				if strings.Contains(k, "Duration") {
					totalDur := float32(stepResult.SuccessCount+stepResult.AssertionFailCount-1)*stepResult.Durations[k] + float32(v.(time.Duration).Seconds())
					stepResult.Durations[k] = float32(totalDur / float32(stepResult.SuccessCount+stepResult.AssertionFailCount))
				}
			}
		} else if sr.Err.Type != "" { // server error
			errOccured = true
			stepResult.FailedCount++
			stepResult.ServerErrorDist[sr.Err.Reason]++
		} else { // success
			stepResult.StatusCodeDist[sr.StatusCode]++
			stepResult.SuccessCount++

			totalDur := float32(stepResult.SuccessCount+stepResult.AssertionFailCount-1)*stepResult.Durations["duration"] + float32(sr.Duration.Seconds())
			stepResult.Durations["duration"] = totalDur / float32(stepResult.SuccessCount+stepResult.AssertionFailCount)
			for k, v := range sr.Custom {
				if strings.Contains(k, "Duration") {
					totalDur := float32(stepResult.SuccessCount-1)*stepResult.Durations[k] + float32(v.(time.Duration).Seconds())
					stepResult.Durations[k] = float32(totalDur / float32(stepResult.SuccessCount+stepResult.AssertionFailCount))
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
		result.FailedCount++
	}
}

// Total test result, all scenario iterations combined
type Result struct {
	SuccessCount int64                                 `json:"success_count"`
	FailedCount  int64                                 `json:"fail_count"`
	AvgDuration  float32                               `json:"avg_duration"`
	StepResults  map[uint16]*ScenarioStepResultSummary `json:"steps"`
}

func (r *Result) successPercentage() int {
	if r.SuccessCount+r.FailedCount == 0 {
		return 0
	}
	t := float32(r.SuccessCount) / float32(r.SuccessCount+r.FailedCount)
	return int(t * 100)
}

func (r *Result) failedPercentage() int {
	if r.SuccessCount+r.FailedCount == 0 {
		return 0
	}
	return 100 - r.successPercentage()
}

type ScenarioStepResultSummary struct {
	Name               string                 `json:"name"`
	StatusCodeDist     map[int]int            `json:"status_code_dist"`
	AssertionErrorDist map[string]*AssertInfo `json:"assertion_error_dist"`
	ServerErrorDist    map[string]int         `json:"server_error_dist"`
	Durations          map[string]float32     `json:"durations"`
	SuccessCount       int64                  `json:"success_count"`
	FailedCount        int64                  `json:"fail_count"`
	AssertionFailCount int64                  `json:"assertion_fail_count"`
}

func (s *ScenarioStepResultSummary) successPercentage() int {
	if s.SuccessCount+s.FailedCount+s.AssertionFailCount == 0 {
		return 0
	}
	t := float32(s.SuccessCount) / float32(s.SuccessCount+s.FailedCount+s.AssertionFailCount)
	return int(t * 100)
}

func (s *ScenarioStepResultSummary) failedPercentage() int {
	if s.SuccessCount+s.FailedCount+s.AssertionFailCount == 0 {
		return 0
	}
	return 100 - s.successPercentage()
}

type AssertInfo struct {
	Count    int
	Received map[string][]interface{}
}

func cleanSamplingCount(samplingCount map[uint16]map[string]int, stopSampling chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			for stepId, ruleMap := range samplingCount {
				for rule, count := range ruleMap {
					if count >= samplingMax {
						samplingCount[stepId][rule] = 0
					}
				}
			}
		case <-stopSampling:
			return
		}
	}
}
