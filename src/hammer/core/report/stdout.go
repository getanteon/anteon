package report

import (
	"fmt"

	"ddosify.com/hammer/core/types"
	"github.com/gosuri/uilive"
)

type stdout struct {
	doneChan chan struct{}
	result   *result
	writer   *uilive.Writer
}

func (s *stdout) init() {
	s.doneChan = make(chan struct{})
	s.result = &result{}
	// s.result.ResponseItems = []*types.ReskponseItem{}
	s.writer = uilive.New()
}

func (s *stdout) Start(input chan *types.Response) {
	s.writer.Start()
	i := 1
	delta := 2
	chunk := 1
	limit := 1000
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

		if i%chunk == 0 {
			_, _ = fmt.Fprintf(s.writer, summaryTemplate(), s.result.responseCount, s.result.avgDuration, s.result.timeoutCount)
			if chunk < limit {
				chunk *= delta
			}
		}
		i++
	}

	_, _ = fmt.Fprintf(s.writer, summaryTemplate(), s.result.responseCount, s.result.avgDuration, s.result.timeoutCount)
	s.writer.Stop() // flush and stop rendering
	s.doneChan <- struct{}{}
}

func (s *stdout) Report() {

	// for _, f := range [][]string{{"Foo.zip", "Bar.iso"}, {"Baz.tar.gz", "Qux.img"}} {
	// 	for i := 0; i <= 50; i++ {
	// 		_, _ = fmt.Fprintf(writer, "Downloading %s.. (%d/%d) GB\n", f[0], i, 50)
	// 		_, _ = fmt.Fprintf(writer.Newline(), "Downloading %s.. (%d/%d) GB\n", f[1], i, 50)
	// 		time.Sleep(time.Millisecond * 25)
	// 	}
	// 	_, _ = fmt.Fprintf(writer.Bypass(), "Downloaded %s\n", f[0])
	// 	_, _ = fmt.Fprintf(writer.Bypass(), "Downloaded %s\n", f[1])
	// }
	// _, _ = fmt.Fprintln(writer, "Finished: Downloaded 150GB")
	// fmt.Printf("Reported! %d items\n", len(s.result))
}

func (s *stdout) DoneChan() <-chan struct{} {
	return s.doneChan
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
