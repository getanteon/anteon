package report

import (
	"fmt"
	"strings"
	"time"

	"ddosify.com/hammer/core/types"
	"github.com/gosuri/uilive"
)

var keyToStr = map[string]string{
	"avgDuration": "Total", "dnsDuration": "DNS", "connDuration": "Connection",
	"reqDuration": "Request Write", "resDuration": "Response Read",
	"serverProcessDuration": "Server Processing", "tlsDuration": "TLS"}

type stdout struct {
	doneChan    chan struct{}
	result      *result
	writer      *uilive.Writer
	printTicker *time.Ticker
}

func (s *stdout) init() {
	s.doneChan = make(chan struct{})
	s.result = &result{
		itemReports: make(map[int16]*scenarioItemReport),
	}
	s.writer = uilive.New()
}

func (s *stdout) Start(input chan *types.Response) {
	go s.realTimePrintStart()

	for r := range input {
		s.result.responseCount++

		var scenarioDuration float32
		timeout := 0
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
				if rr.Err.Reason == types.ReasonConnTimeout {
					timeout++
					item.timeoutCount++
				}
				item.errorDist[rr.Err.Reason]++
			} else {
				item.statusCodeDist[rr.StatusCode]++
				item.responseCount++

				totalDur := float32(item.responseCount-1)*item.durations["avgDuration"] + float32(rr.Duration.Seconds())
				item.durations["avgDuration"] = totalDur / float32(item.responseCount)
				for k, v := range rr.Custom {
					if strings.Contains(k, "Duration") {
						totalDur := float32(item.responseCount-1)*item.durations[k] + float32(v.(time.Duration).Seconds())
						item.durations[k] = totalDur / float32(item.responseCount)
					}
				}
			}

		}

		// Don't change avg duration if there is a timeout
		if timeout == 0 {
			totalDuration := float32(s.result.responseCount-1)*s.result.avgDuration + scenarioDuration
			s.result.avgDuration = totalDuration / float32(s.result.responseCount)
		} else {
			s.result.timeoutCount += int64(timeout)
		}

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
	s.writer.Start()
	s.printTicker = time.NewTicker(time.Duration(1) * time.Second)

	// First print.
	_, _ = fmt.Fprintf(s.writer, summaryTemplate(), s.result.responseCount, s.result.avgDuration, s.result.timeoutCount)
	for range s.printTicker.C {
		_, _ = fmt.Fprintf(s.writer, summaryTemplate(), s.result.responseCount, s.result.avgDuration, s.result.timeoutCount)
	}
}

func (s *stdout) realTimePrintStop() {
	// Last print.
	_, _ = fmt.Fprintf(s.writer, summaryTemplate(), s.result.responseCount, s.result.avgDuration, s.result.timeoutCount)
	s.printTicker.Stop()
	s.writer.Stop()
}

func summaryTemplate() string {
	return `
SUMMARY
----------------------------------------------------
Run Count  -  Average Duration (s)  -  Timeout Count
%d %20f %20d
`
}

// TODO:REFACTOR use template
func (s *stdout) printDetails() {
	fmt.Println("\nDETAILS")
	fmt.Println("----------------------------------------------------")
	for k, v := range s.result.itemReports {
		fmt.Println("Step", k)
		fmt.Println("-------------------------------------")

		fmt.Println(" Response Count:", v.responseCount)
		fmt.Println(" Timeout Count:", v.timeoutCount)

		fmt.Println("\n Durations (Avg);")
		for d, s := range v.durations {
			fmt.Printf("\t%-20s:%.4fs\n", keyToStr[d], s)
		}

		fmt.Println("\n Status Codes;")
		for s, c := range v.statusCodeDist {
			fmt.Printf("\t%3d : %d\n", s, c)
		}

		fmt.Println("\n Error Distribution;")
		for e, c := range v.errorDist {
			fmt.Printf("\t%-15s:%d\n", e, c)
		}
		fmt.Println()
	}
}

type result struct {
	responseCount int64
	avgDuration   float32
	timeoutCount  int64
	itemReports   map[int16]*scenarioItemReport
}

type scenarioItemReport struct {
	statusCodeDist map[int]int
	errorDist      map[string]int
	durations      map[string]float32
	timeoutCount   int64
	responseCount  int64
}
