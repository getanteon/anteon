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
	"io"
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
	doneChan     chan struct{}
	result       *Result
	printTicker  *time.Ticker
	mu           sync.Mutex
	debug        bool
	samplingRate int
}

var white = color.New(color.FgHiWhite).SprintFunc()
var blue = color.New(color.FgHiBlue).SprintFunc()
var green = color.New(color.FgHiGreen).SprintFunc()
var yellow = color.New(color.FgHiYellow).SprintFunc()
var red = color.New(color.FgHiRed).SprintFunc()
var realTimePrintInterval = time.Duration(1500) * time.Millisecond

func (s *stdout) Init(debug bool, samplingRate int) (err error) {
	s.doneChan = make(chan struct{})
	s.result = &Result{
		StepResults: make(map[uint16]*ScenarioStepResultSummary),
	}
	s.debug = debug
	s.samplingRate = samplingRate

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

	stopSampling := make(chan struct{})
	samplingCount := make(map[uint16]map[string]int)
	go s.cleanSamplingCount(samplingCount, stopSampling, s.samplingRate)

	for r := range input {
		s.mu.Lock() // avoid race around samplingCount
		aggregate(s.result, r, samplingCount, s.samplingRate)
		s.mu.Unlock()
	}

	s.realTimePrintStop()
	s.report()
	stopSampling <- struct{}{}
	s.doneChan <- struct{}{}
}

func (s *stdout) cleanSamplingCount(samplingCount map[uint16]map[string]int, stopSampling chan struct{}, samplingRate int) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			s.mu.Lock() // avoid race around samplingCount
			for stepId, ruleMap := range samplingCount {
				for rule, count := range ruleMap {
					if count >= samplingRate {
						samplingCount[stepId][rule] = 0
					}
				}
			}
			s.mu.Unlock()
		case <-stopSampling:
			return
		}
	}
}

func (s *stdout) report() {
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
			emoji.CrossMark, s.result.ServerFailedCount+s.result.AssertionFailCount, s.result.failedPercentage(), "")),
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

func (s *stdout) printInDebugMode(input chan *types.ScenarioResult) {
	color.Cyan("%s Engine fired. \n\n", emoji.Fire)
	color.Cyan("%s CTRL+C to gracefully stop.\n", emoji.StopSign)

	for r := range input { // only 1 ScenarioResult expected
		for _, sr := range r.StepResults {
			verboseInfo := ScenarioStepResultToVerboseHttpRequestInfo(sr)

			b := strings.Builder{}
			w := tabwriter.NewWriter(&b, 0, 0, 4, ' ', 0)
			color.Cyan("\n\nSTEP (%d) %-5s\n", verboseInfo.StepId, verboseInfo.StepName)
			color.Cyan("-------------------------------------")
			fmt.Fprintf(w, "%s\n", blue(fmt.Sprintf("- Environment Variables")))

			for eKey, eVal := range verboseInfo.Envs {
				switch eVal.(type) {
				case map[string]interface{}:
					valPretty, _ := json.Marshal(eVal)
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), valPretty)
				case []string:
					valPretty, _ := json.Marshal(eVal)
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), valPretty)
				case []float64:
					valPretty, _ := json.Marshal(eVal)
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), valPretty)
				case []bool:
					valPretty, _ := json.Marshal(eVal)
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), valPretty)
				default:
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), fmt.Sprint(eVal))
				}
			}
			fmt.Fprintf(w, "\n")
			fmt.Fprintf(w, "%s\n", blue(fmt.Sprintf("- Test Data")))

			for eKey, eVal := range verboseInfo.TestData {
				switch eVal.(type) {
				case map[string]interface{}:
					valPretty, _ := json.Marshal(eVal)
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), valPretty)
				case []int:
					valPretty, _ := json.Marshal(eVal)
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), valPretty)
				case []string:
					valPretty, _ := json.Marshal(eVal)
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), valPretty)
				case []float64:
					valPretty, _ := json.Marshal(eVal)
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), valPretty)
				case []bool:
					valPretty, _ := json.Marshal(eVal)
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), valPretty)
				default:
					fmt.Fprintf(w, "\t%s:\t%-5s \n", fmt.Sprint(eKey), fmt.Sprint(eVal))
				}
			}

			if verboseInfo.Error != "" && isVerboseInfoRequestEmpty(verboseInfo.Request) {
				fmt.Fprintf(w, "%s Error: \t%-5s \n", emoji.SosButton, verboseInfo.Error)
				fmt.Fprintln(w)
				fmt.Fprint(out, b.String())
				break
			}
			fmt.Fprintf(w, "\n")
			fmt.Fprintf(w, "%s\n", blue(fmt.Sprintf("- Request")))
			fmt.Fprintf(w, "\tTarget: \t%s \n", verboseInfo.Request.Url)
			fmt.Fprintf(w, "\tMethod: \t%s \n", verboseInfo.Request.Method)

			fmt.Fprintf(w, "\t%s\n", "Headers: ")
			for hKey, hVal := range verboseInfo.Request.Headers {
				fmt.Fprintf(w, "\t\t%s:\t%-5s \n", hKey, hVal)
			}

			contentType := sr.ReqHeaders.Get("content-type")
			fmt.Fprintf(w, "\t%s", "Body: ")
			printBody(w, contentType, verboseInfo.Request.Body)
			fmt.Fprintf(w, "\n")

			if verboseInfo.Error == "" {
				// response
				fmt.Fprintf(w, "%s\n", blue(fmt.Sprintf("- Response")))
				fmt.Fprintf(w, "\tStatusCode:\t%-5d \n", verboseInfo.Response.StatusCode)
				fmt.Fprintf(w, "\tResponseTime:\t%-5d(ms) \n", verboseInfo.Response.ResponseTime)
				fmt.Fprintf(w, "\t%s\n", "Headers: ")
				for hKey, hVal := range verboseInfo.Response.Headers {
					fmt.Fprintf(w, "\t\t%s:\t%-5s \n", hKey, hVal)
				}

				contentType := sr.RespHeaders.Get("content-type")
				fmt.Fprintf(w, "\t%s", "Body: ")
				printBody(w, contentType, verboseInfo.Response.Body)
				fmt.Fprintf(w, "\n")
			}

			if len(verboseInfo.FailedCaptures) > 0 {
				fmt.Fprintf(w, "%s\n", yellow(fmt.Sprintf("- Failed Captures")))
				for wKey, wVal := range verboseInfo.FailedCaptures {
					fmt.Fprintf(w, "\t\t%s: \t%s \n", wKey, wVal)
				}
			}

			if len(verboseInfo.FailedAssertions) > 0 {
				fmt.Fprintf(w, "%s", yellow(fmt.Sprintf("- Failed Assertions")))
				for _, failAssertion := range verboseInfo.FailedAssertions {
					fmt.Fprintf(w, "\n\t\tRule: %s\n", failAssertion.Rule)
					prettyReceived, _ := json.MarshalIndent(failAssertion.Received, "\t\t", "\t")
					fmt.Fprintf(w, "\t\tReceived: %s\n", prettyReceived)
					fmt.Fprintf(w, "\t\tReason: %s\n", failAssertion.Reason)
				}
			}

			if verboseInfo.Error != "" { // server error
				fmt.Fprintf(w, "\n%s Error: \t%-5s \n", emoji.SosButton, verboseInfo.Error)
			}

			fmt.Fprintln(w)
			fmt.Fprint(out, b.String())
		}
	}
}

func printBody(w io.Writer, contentType string, body interface{}) {
	if body == nil {
		return
	}
	if strings.Contains(contentType, "application/json") {
		valPretty, _ := json.MarshalIndent(body, "\t\t", "\t")
		fmt.Fprintf(w, "\n\t\t%s\n", valPretty)
	} else {
		// html unescaped text
		// if xml came as decoded, we could pretty print it like json
		fmt.Fprintf(w, "%+v\n", fmt.Sprintf("%s", body))
	}
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
		fmt.Fprintf(w, "Failed Count:\t%-5d (%d%%)\n", v.Fail.Count, v.failedPercentage())

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

		if v.Fail.AssertionErrorDist.Count > 0 {
			fmt.Fprintln(w, "\nAssertion Error Distribution:")
			for e, c := range v.Fail.AssertionErrorDist.Conditions {
				fmt.Fprintf(w, "\tCondition : %s\n", e)
				fmt.Fprintf(w, "\t\tFail Count : %d\n", c.Count)
				fmt.Fprintf(w, "\t\tReceived : \n")

				for ident, values := range c.Received {
					fmt.Fprintf(w, "\t\t\t %s : %v\n", ident, values)
				}

				fmt.Fprintf(w, "\t\tReason : %s \n", c.Reason)
			}
		}

		if v.Fail.ServerErrorDist.Count > 0 {
			fmt.Fprintln(w, "\nServer Error Distribution (Count:Reason):")
			for e, c := range v.Fail.ServerErrorDist.Reasons {
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
