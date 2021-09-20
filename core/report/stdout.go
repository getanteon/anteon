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
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"ddosify.com/hammer/core/types"
	"ddosify.com/hammer/core/util"
	"github.com/gosuri/uilive"
)

type stdout struct {
	doneChan    chan struct{}
	result      *result
	writer      *uilive.Writer
	printTicker *time.Ticker
	mu          sync.Mutex
}

func (s *stdout) Init() (err error) {
	s.doneChan = make(chan struct{})
	s.result = &result{
		itemReports: make(map[int16]*scenarioItemReport),
	}

	if !util.IsSystemInTestMode() {
		s.writer = uilive.New()
	}
	return
}

func (s *stdout) Start(input chan *types.Response) {
	go s.realTimePrintStart()

	for r := range input {
		s.mu.Lock()
		var scenarioDuration float32
		errOccured := false
		for _, rr := range r.ResponseItems {
			scenarioDuration += float32(rr.Duration.Seconds())

			if _, ok := s.result.itemReports[rr.ScenarioItemID]; !ok {
				s.result.itemReports[rr.ScenarioItemID] = &scenarioItemReport{
					statusCodeDist: make(map[int]int, 0),
					errorDist:      make(map[string]int),
					durations:      map[string]float32{},
				}
			}
			item := s.result.itemReports[rr.ScenarioItemID]

			if rr.Err.Type != "" {
				errOccured = true
				item.failedCount++
				item.errorDist[rr.Err.Reason]++
			} else {
				item.statusCodeDist[rr.StatusCode]++
				item.successCount++

				totalDur := float32(item.successCount-1)*item.durations["duration"] + float32(rr.Duration.Seconds())
				item.durations["duration"] = totalDur / float32(item.successCount)
				for k, v := range rr.Custom {
					if strings.Contains(k, "Duration") {
						totalDur := float32(item.successCount-1)*item.durations[k] + float32(v.(time.Duration).Seconds())
						item.durations[k] = float32(totalDur / float32(item.successCount))
					}
				}
			}

		}

		// Don't change avg duration if there is a error
		if !errOccured {
			totalDuration := float32(s.result.successCount)*s.result.avgDuration + scenarioDuration
			s.result.successCount++
			s.result.avgDuration = totalDuration / float32(s.result.successCount)
		} else if errOccured {
			s.result.failedCount++
		}
		s.mu.Unlock()
	}

	s.realTimePrintStop()
	s.doneChan <- struct{}{}
}

func (s *stdout) Report() {
	s.printDetails()
}

func (s *stdout) DoneChan() <-chan struct{} {
	return s.doneChan
}

func (s *stdout) realTimePrintStart() {
	if util.IsSystemInTestMode() {
		return
	}

	s.writer.Start()
	s.printTicker = time.NewTicker(time.Duration(1) * time.Second)

	for range s.printTicker.C {
		go func() {
			s.mu.Lock()
			_, _ = fmt.Fprintf(s.writer, summaryTemplate(), s.result.successCount, s.result.successPercentage(),
				s.result.failedCount, s.result.failedPercentage(), s.result.avgDuration, "\nCTRL+C to gracefully stop.")
			s.mu.Unlock()
		}()

	}
}

func (s *stdout) realTimePrintStop() {
	if util.IsSystemInTestMode() {
		return
	}

	// Last print.
	_, _ = fmt.Fprintf(s.writer, summaryTemplate(), s.result.successCount, s.result.successPercentage(),
		s.result.failedCount, s.result.failedPercentage(), s.result.avgDuration, "")
	s.printTicker.Stop()
	s.writer.Stop()
}

// TODO:REFACTOR use template
func (s *stdout) printDetails() {
	if util.IsSystemInTestMode() {
		return
	}

	fmt.Println("\nDETAILS")
	fmt.Println("----------------------------------------------------")

	keys := make([]int, 0)
	for k, _ := range s.result.itemReports {
		keys = append(keys, int(k))
	}

	// Since map is not a ordered data structure,
	// We should sort scenarioItemIDs to traverse itemReports
	sort.Ints(keys)

	for _, k := range keys {
		v := s.result.itemReports[int16(k)]

		fmt.Println("Step", k)
		fmt.Println("-------------------------------------")

		fmt.Printf("Success Count: %-5d (%d%%)\n", v.successCount, v.successPercentage())
		fmt.Printf("Failed Count:  %-5d (%d%%)\n", v.failedCount, v.failedPercentage())

		fmt.Println("\nDurations (Avg);")
		var durationList = make([]duration, 0)
		for d, s := range v.durations {
			dur := keyToStr[d]
			dur.duration = s
			durationList = append(durationList, dur)
		}
		sort.Slice(durationList, func(i, j int) bool {
			return durationList[i].order < durationList[j].order
		})
		for _, v := range durationList {
			fmt.Printf("\t%-20s:%.4fs\n", v.name, v.duration)
		}

		if len(v.statusCodeDist) > 0 {
			fmt.Println("\nStatus Codes;")
			for s, c := range v.statusCodeDist {
				fmt.Printf("\t%3d : %d\n", s, c)
			}
		}

		if len(v.errorDist) > 0 {
			fmt.Println("\nError Distribution (Count:Reason);")
			for e, c := range v.errorDist {
				fmt.Printf("\t%-5d : %s\n", c, e)
			}
		}
		fmt.Println()
	}
}

func summaryTemplate() string {
	return `
SUMMARY
----------------------------------------------------
Successful Run Count: %-5d (%d%%)
Failed Run Count    : %-5d (%d%%)
Average Duration(s) : %.3f

*Average duration calculated only for successful runs.
%s
`
}

type result struct {
	successCount int64
	avgDuration  float32
	failedCount  int64
	itemReports  map[int16]*scenarioItemReport
}

func (r *result) successPercentage() int {
	if r.successCount+r.failedCount == 0 {
		return 0
	}
	t := float32(r.successCount) / float32(r.successCount+r.failedCount)
	return int(t * 100)
}

func (r *result) failedPercentage() int {
	if r.successCount+r.failedCount == 0 {
		return 0
	}
	return 100 - r.successPercentage()
}

type scenarioItemReport struct {
	statusCodeDist map[int]int
	errorDist      map[string]int
	durations      map[string]float32
	failedCount    int64
	successCount   int64
}

func (s *scenarioItemReport) successPercentage() int {
	if s.successCount+s.failedCount == 0 {
		return 0
	}
	t := float32(s.successCount) / float32(s.successCount+s.failedCount)
	return int(t * 100)
}

func (s *scenarioItemReport) failedPercentage() int {
	if s.successCount+s.failedCount == 0 {
		return 0
	}
	return 100 - s.successPercentage()
}

type duration struct {
	name     string
	duration float32
	order    int
}

var keyToStr = map[string]duration{
	"dnsDuration":           duration{name: "DNS", order: 1},
	"connDuration":          duration{name: "Connection", order: 2},
	"tlsDuration":           duration{name: "TLS", order: 3},
	"reqDuration":           duration{name: "Request Write", order: 4},
	"serverProcessDuration": duration{name: "Server Processing", order: 5},
	"resDuration":           duration{name: "Response Read", order: 6},
	"duration":              duration{name: "Total", order: 7},
}
