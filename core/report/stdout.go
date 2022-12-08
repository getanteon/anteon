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
	"net/http"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/enescakir/emoji"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"go.ddosify.com/ddosify/core/types"
	"go.ddosify.com/ddosify/core/util"
)

const OutputTypeStdout = "stdout"

var out = colorable.NewColorableStdout()

func init() {
	AvailableOutputServices[OutputTypeStdout] = &stdout{}
}

type stdout struct {
	doneChan    chan struct{}
	result      *Result
	printTicker *time.Ticker
	mu          sync.Mutex
	debug       bool
}

var white = color.New(color.FgHiWhite).SprintFunc()
var blue = color.New(color.FgHiBlue).SprintFunc()
var green = color.New(color.FgHiGreen).SprintFunc()
var red = color.New(color.FgHiRed).SprintFunc()
var realTimePrintInterval = time.Duration(1500) * time.Millisecond

func (s *stdout) Init(debug bool) (err error) {
	s.doneChan = make(chan struct{})
	s.result = &Result{
		StepResults: make(map[uint16]*ScenarioStepResult),
	}
	s.debug = debug

	color.Cyan("%s  Initializing... \n", emoji.Gear)
	if s.debug {
		color.Cyan("%s Running in debug mode, 1 iteration will be played... \n", emoji.Bug)
	}
	return
}

func (s *stdout) Start(input chan *types.ScenarioResult) {
	if s.debug {
		s.printInDebugMode(input)
		s.doneChan <- struct{}{}
		return
	}
	go s.realTimePrintStart()

	for r := range input {
		s.mu.Lock()
		aggregate(s.result, r)
		s.mu.Unlock()
	}

	s.realTimePrintStop()
	s.doneChan <- struct{}{}
}

type verboseHttpRequestInfo struct {
	Request struct {
		Url     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
	} `json:"request"`
	Response struct {
		StatusCode int               `json:"statusCode"`
		Headers    map[string]string `json:"headers"`
		Body       interface{}       `json:"body"`
	} `json:"response"`
}

func (s *stdout) printInDebugMode(input chan *types.ScenarioResult) {
	color.Cyan("%s Engine fired. \n\n", emoji.Fire)
	color.Cyan("%s CTRL+C to gracefully stop.\n", emoji.StopSign)

	for r := range input { // only 1 sc result expected
		for _, sr := range r.StepResults {
			requestHeaders := make(map[string]string, 0)
			for k, v := range sr.DebugInfo["requestHeaders"].(http.Header) {
				values := strings.Join(v, ",")
				requestHeaders[k] = values
			}

			responseHeaders := make(map[string]string, 0)
			for k, v := range sr.DebugInfo["responseHeaders"].(http.Header) {
				values := strings.Join(v, ",")
				responseHeaders[k] = values
			}

			// TODO: may be a connection error or timeout, handle that case
			// sr.Err.Reason

			contentType := sr.DebugInfo["responseHeaders"].(http.Header).Get("content-type")
			var respBody string
			if strings.Contains(contentType, "text/html") {
				// decode text/html
				respBody = string(sr.DebugInfo["responseBody"].([]byte))
			} else if strings.Contains(contentType, "application/json") {
				err := json.Unmarshal(sr.DebugInfo["responseBody"].([]byte), &respBody)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println("application/json")
			}

			verboseInfo := verboseHttpRequestInfo{
				Request: struct {
					Url     string            "json:\"url\""
					Method  string            "json:\"method\""
					Headers map[string]string "json:\"headers\""
				}{
					Url:     sr.DebugInfo["url"].(string),
					Method:  sr.DebugInfo["method"].(string),
					Headers: requestHeaders,
				},

				Response: struct {
					StatusCode int               "json:\"statusCode\""
					Headers    map[string]string "json:\"headers\""
					Body       interface{}       `json:"body"`
				}{
					StatusCode: sr.StatusCode,
					Headers:    responseHeaders,
					Body:       respBody,
				},
			}

			valPretty, _ := json.MarshalIndent(verboseInfo, "", "  ")
			fmt.Fprintf(out, "%s http request: %s \n",
				sr.RequestTime.UTC().String(), white(fmt.Sprintf(" %-6s",
					valPretty)))

		}
		aggregate(s.result, r)
	}
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

	s.printTicker = time.NewTicker(realTimePrintInterval)

	color.Cyan("%s Engine fired. \n\n", emoji.Fire)
	color.Cyan("%s CTRL+C to gracefully stop.\n", emoji.StopSign)

	for range s.printTicker.C {
		go func() {
			s.mu.Lock()
			s.liveResultPrint()
			s.mu.Unlock()
		}()
	}
}

func (s *stdout) liveResultPrint() {
	fmt.Fprintf(out, "%s %s %s\n",
		green(fmt.Sprintf("%s  Successful Run: %-6d %3d%% %5s",
			emoji.CheckMark, s.result.SuccessCount, s.result.successPercentage(), "")),
		red(fmt.Sprintf("%s Failed Run: %-6d %3d%% %5s",
			emoji.CrossMark, s.result.FailedCount, s.result.failedPercentage(), "")),
		blue(fmt.Sprintf("%s  Avg. Duration: %.5fs", emoji.Stopwatch, s.result.AvgDuration)))
}

func (s *stdout) realTimePrintStop() {
	if util.IsSystemInTestMode() {
		return
	}
	// Last print.
	s.liveResultPrint()
	s.printTicker.Stop()
}

// TODO:REFACTOR use template
func (s *stdout) printDetails() {
	color.Set(color.FgHiCyan)
	defer color.Unset()

	b := strings.Builder{}
	w := tabwriter.NewWriter(&b, 0, 0, 4, ' ', 0)

	fmt.Fprintln(w, "\n\nRESULT")
	fmt.Fprintln(w, "-------------------------------------")

	keys := make([]int, 0)
	for k := range s.result.StepResults {
		keys = append(keys, int(k))
	}

	// Since map is not a ordered data structure,
	// We should sort scenarioItemIDs to traverse itemReports
	sort.Ints(keys)

	for _, k := range keys {
		v := s.result.StepResults[uint16(k)]

		if len(keys) > 1 {
			stepHeader := v.Name
			if v.Name == "" {
				stepHeader = fmt.Sprintf("Step %d", k)
			}
			fmt.Fprintf(w, "\n%d. "+stepHeader+"\n", k)
			fmt.Fprintln(w, "---------------------------------")
		}

		fmt.Fprintf(w, "Success Count:\t%-5d (%d%%)\n", v.SuccessCount, v.successPercentage())
		fmt.Fprintf(w, "Failed Count:\t%-5d (%d%%)\n", v.FailedCount, v.failedPercentage())

		fmt.Fprintln(w, "\nDurations (Avg):")
		var durationList = make([]duration, 0)
		for d, s := range v.Durations {
			dur := keyToStr[d]
			dur.duration = s
			durationList = append(durationList, dur)
		}
		sort.Slice(durationList, func(i, j int) bool {
			return durationList[i].order < durationList[j].order
		})
		for _, v := range durationList {
			fmt.Fprintf(w, "  %s\t:%.4fs\n", v.name, v.duration)
		}

		if len(v.StatusCodeDist) > 0 {
			fmt.Fprintln(w, "\nStatus Code (Message) :Count")
			for s, c := range v.StatusCodeDist {
				desc := fmt.Sprintf("%3d (%s)", s, http.StatusText(s))
				fmt.Fprintf(w, "  %s\t:%d\n", desc, c)
			}
		}

		if len(v.ErrorDist) > 0 {
			fmt.Fprintln(w, "\nError Distribution (Count:Reason):")
			for e, c := range v.ErrorDist {
				fmt.Fprintf(w, "  %d\t :%s\n", c, e)
			}
		}
		fmt.Fprintln(w)
	}

	w.Flush()
	fmt.Fprint(out, b.String())
}

type duration struct {
	name     string
	duration float32
	order    int
}

var keyToStr = map[string]duration{
	"dnsDuration":           {name: "DNS", order: 1},
	"connDuration":          {name: "Connection", order: 2},
	"tlsDuration":           {name: "TLS", order: 3},
	"reqDuration":           {name: "Request Write", order: 4},
	"serverProcessDuration": {name: "Server Processing", order: 5},
	"resDuration":           {name: "Response Read", order: 6},
	"duration":              {name: "Total", order: 7},
}
