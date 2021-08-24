package report

import (
	"fmt"
	"time"

	"ddosify.com/hammer/core/types"
	"github.com/gosuri/uilive"
)

type stdout struct {
	doneChan    chan struct{}
	result      *result
	writer      *uilive.Writer
	printTicker *time.Ticker
}

func (s *stdout) init() {
	s.doneChan = make(chan struct{})
	s.result = &result{}
	s.writer = uilive.New()
}

func (s *stdout) Start(input chan *types.Response) {
	go s.realTimePrintStart()

	for r := range input {

		var scenarioDuration float32
		for _, rr := range r.ResponseItems {
			scenarioDuration += float32(rr.Duration.Seconds())
			if rr.Err.Reason == types.ReasonConnTimeout {
				s.result.timeoutCount++
			}
		}
		totalDuration := float32(s.result.responseCount)*s.result.avgDuration + scenarioDuration
		s.result.responseCount++
		s.result.avgDuration = totalDuration / float32(s.result.responseCount)

	}

	s.realTimePrintStop()
	s.doneChan <- struct{}{}
}

func (s *stdout) Report() {
	fmt.Printf("Reported! %d items\n", s.result.responseCount)
}

func (s *stdout) DoneChan() <-chan struct{} {
	return s.doneChan
}

func (s *stdout) realTimePrintStart() {
	s.writer.Start()
	s.printTicker = time.NewTicker(time.Duration(1) * time.Second)
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
Run Count  -  Average Response Time (s)  -  Timeout Count
%d %20f %30d
`
}

type result struct {
	responseCount int64
	avgDuration   float32
	timeoutCount  int64
	itemReports   map[int]scenarioItemReport
}

type scenarioItemReport struct {
	statusCodeDist map[int]int
	errorDist      map[string]int
	durations      map[string]duration
}

type duration struct {
	avg     float32
	slowest float32
	fastest float32
}
