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
	"math"
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
			result.ItemReports[rr.ScenarioItemID] = &ScenarioItemReport{
				Name:           rr.ScenarioItemName,
				StatusCodeDist: make(map[int]int, 0),
				ErrorDist:      make(map[string]int),
				Durations:      map[string]float32{},
				TotalDurations: map[string][]float32{},
			}
		}
		item := result.ItemReports[rr.ScenarioItemID]

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
		for key, duration := range report.Durations {
			report.TotalDurations[key] = append(report.TotalDurations[key], duration)
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
	SuccessCount int64                         `json:"success_count"`
	FailedCount  int64                         `json:"fail_count"`
	AvgDuration  float32                       `json:"avg_duration"`
	ItemReports  map[int16]*ScenarioItemReport `json:"steps"`
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

type ScenarioItemReport struct {
	Name           string               `json:"name"`
	StatusCodeDist map[int]int          `json:"status_code_dist"`
	ErrorDist      map[string]int       `json:"error_dist"`
	Durations      map[string]float32   `json:"durations"`
	TotalDurations map[string][]float32 `json:"total_durations"`
	SuccessCount   int64                `json:"success_count"`
	FailedCount    int64                `json:"fail_count"`
}

func (s *ScenarioItemReport) DurationPercentile(p int) float32 {
	if p < 0 || p > 100 {
		return 0
	}

	durations, ok := s.TotalDurations["duration"]
	if !ok {
		return 0
	}

	// todo: it could be optimized by always sorted array being used in TotalDurations so we would not make this call.
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	return Percentile(durations, p)
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

func Percentile(list []float32, p int) float32 {
	if p < 0 || p > 100 {
		return 0
	}

	n := int(math.Round((float64(p) / 100.0) * float64(len(list))))
	if n > 0 {
		// I am not sure about the case where n == 0
		n--
	}

	return list[n]
}
