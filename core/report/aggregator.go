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
	"sort"
	"strings"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

func aggregate(result *Result, response *types.Response) {
	var scenarioDuration float32
	errOccured := false
	for _, rr := range response.ResponseItems {
		scenarioDuration += float32(rr.Duration.Seconds())

		if _, ok := result.ItemReports[rr.ScenarioItemID]; !ok {
			result.ItemReports[rr.ScenarioItemID] = &ScenarioResult{
				Report: &ScenarioItemReport{
					Name:           rr.ScenarioItemName,
					StatusCodeDist: make(map[int]int, 0),
					ErrorDist:      make(map[string]int),
					Durations:      map[string]float32{},
				},
				Durations: map[string]*ScenarioStats{},
			}
		}
		item := result.ItemReports[rr.ScenarioItemID].Report

		if rr.Err.Type != "" {
			errOccured = true
			item.FailedCount++
			item.ErrorDist[rr.Err.Reason]++
		} else {
			item.StatusCodeDist[rr.StatusCode]++
			item.SuccessCount++

			totalDur := float32(item.SuccessCount-1)*item.Durations["duration"] + float32(rr.Duration.Seconds())
			item.Durations["duration"] = totalDur / float32(item.SuccessCount)
			for k, v := range rr.Custom {
				if strings.Contains(k, "Duration") {
					totalDur := float32(item.SuccessCount-1)*item.Durations[k] + float32(v.(time.Duration).Seconds())
					item.Durations[k] = float32(totalDur / float32(item.SuccessCount))
				}
			}
		}
	}

	for _, report := range result.ItemReports {
		for key, duration := range report.Report.Durations {
			if report.Durations[key] == nil {
				report.Durations[key] = &ScenarioStats{}
			}

			report.Durations[key].Durations = append(report.Durations[key].Durations, duration)
			sort.Slice(report.Durations[key].Durations, func(i, j int) bool { return report.Durations[key].Durations[i] < report.Durations[key].Durations[j] })
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

type Result struct {
	SuccessCount int64                     `json:"success_count"`
	FailedCount  int64                     `json:"fail_count"`
	AvgDuration  float32                   `json:"avg_duration"`
	ItemReports  map[int16]*ScenarioResult `json:"steps"`
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

type ScenarioResult struct {
	Report    *ScenarioItemReport       `json:"report"`
	Durations map[string]*ScenarioStats `json:"durations"`
}

type ScenarioStats struct {
	Durations []float32 `json:"durations"`
}

func (r *ScenarioStats) Percentile(p int) float32 {
	if p < 0 || p > 100 {
		return 0
	}

	// n = (P/100) x N
	n := (p / 100) * len(r.Durations)
	if n > 0 {
		// I am not sure about the case where n == 0
		n--
	}

	return r.Durations[n]
}

type ScenarioItemReport struct {
	Name           string             `json:"name"`
	StatusCodeDist map[int]int        `json:"status_code_dist"`
	ErrorDist      map[string]int     `json:"error_dist"`
	Durations      map[string]float32 `json:"durations"`
	SuccessCount   int64              `json:"success_count"`
	FailedCount    int64              `json:"fail_count"`
}

func (s *ScenarioItemReport) successPercentage() int {
	if s.SuccessCount+s.FailedCount == 0 {
		return 0
	}
	t := float32(s.SuccessCount) / float32(s.SuccessCount+s.FailedCount)
	return int(t * 100)
}

func (s *ScenarioItemReport) failedPercentage() int {
	if s.SuccessCount+s.FailedCount == 0 {
		return 0
	}
	return 100 - s.successPercentage()
}
