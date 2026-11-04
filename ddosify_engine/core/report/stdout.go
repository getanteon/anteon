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
	"go.ddosify.com/ddosify/v2/core/assertion"
	"go.ddosify.com/ddosify/v2/core/types"
	"go.ddosify.com/ddosify/v2/core/util"
)

const OutputTypeStdout = "stdout"

// out takes the report, the primary output; errOut takes progress and banners.
var out io.Writer = colorable.NewColorableStdout()
var errOut io.Writer = colorable.NewColorableStderr()
var progress = true

// SetOutputs sends the report to stdout, and banners and progress lines to stderr.
// showProgress false prints no live progress lines, for a pipe or while a TUI owns the terminal.
func SetOutputs(stdout, stderr io.Writer, showProgress bool) {
	out, errOut, progress = stdout, stderr, showProgress
}

var cyan = color.New(color.FgCyan).SprintFunc()
var hiCyan = color.New(color.FgHiCyan).SprintFunc()

func init() {
	AvailableOutputServices[OutputTypeStdout] = &stdout{}
}

type stdout struct {
	doneChan     chan bool
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
	s.doneChan = make(chan bool, 1)
	s.result = &Result{
		StepResults: make(map[uint16]*ScenarioStepResultSummary),
		ext:         newExtended(),
	}
	s.debug = debug
	s.samplingRate = samplingRate

	fmt.Fprint(errOut, cyan(fmt.Sprintf("%s  Initializing... \n", emoji.Gear)))
	if s.debug {
		fmt.Fprint(errOut, cyan(fmt.Sprintf("%s Running in debug mode, 1 iteration will be played... \n", emoji.Bug)))
	}
	return
}

func (s *stdout) Start(input chan *types.ScenarioResult, assertionResultChan <-chan assertion.TestAssertionResult) {
	if s.debug {
		s.result.TestStatus = "success"
		if assertionResultChan != nil {
			result := <-assertionResultChan
			if result.Fail {
				s.result.TestStatus = "failed"
				s.result.TestFailedAssertions = result.FailedRules
			}
		}
		s.printInDebugMode(input)
		if s.result.TestStatus == "success" {
			s.doneChan <- true
		} else {
			s.doneChan <- false
		}
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

	// listen for assertion result
	s.result.TestStatus = "success"
	if assertionResultChan != nil {
		result := <-assertionResultChan
		if result.Fail {
			s.result.TestStatus = "failed"
			s.result.TestFailedAssertions = result.FailedRules
		}
	}
	s.realTimePrintStop()
	s.report()
	stopSampling <- struct{}{}

	if s.result.TestStatus == "success" {
		s.doneChan <- true
	} else {
		s.doneChan <- false
	}
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
	s.result.ext.finalize(s.result)
	s.printDetails()
}

func (s *stdout) Result() *Result {
	return s.result
}

func (s *stdout) SetSLO(slo *types.SLO) {
	s.result.ext.slo = slo
}

func (s *stdout) DoneChan() <-chan bool {
	return s.doneChan
}

func (s *stdout) realTimePrintStart() {
	if util.IsSystemInTestMode() || !progress {
		return
	}

	s.printTicker = time.NewTicker(realTimePrintInterval)

	fmt.Fprint(errOut, cyan(fmt.Sprintf("%s Engine fired. \n\n", emoji.Fire)))
	fmt.Fprint(errOut, cyan(fmt.Sprintf("%s CTRL+C to gracefully stop.\n", emoji.StopSign)))

	for range s.printTicker.C {
		go func() {
			s.mu.Lock()
			s.liveResultPrint()
			s.mu.Unlock()
		}()
	}
}

func (s *stdout) liveResultPrint() {
	fmt.Fprintf(errOut, "%s %s %s\n",
		green(fmt.Sprintf("%s  Successful Run: %-6d %3d%% %5s",
			emoji.CheckMark, s.result.SuccessCount, s.result.successPercentage(), "")),
		red(fmt.Sprintf("%s Failed Run: %-6d %3d%% %5s",
			emoji.CrossMark, s.result.ServerFailedCount+s.result.AssertionFailCount, s.result.failedPercentage(), "")),
		blue(fmt.Sprintf("%s  Avg. Duration: %.5fs", emoji.Stopwatch, s.result.AvgDuration)))
}

func (s *stdout) realTimePrintStop() {
	if util.IsSystemInTestMode() || !progress || s.printTicker == nil {
		return
	}
	// Last print.
	s.liveResultPrint()
	s.printTicker.Stop()
}

func (s *stdout) printInDebugMode(input chan *types.ScenarioResult) {
	fmt.Fprint(errOut, cyan(fmt.Sprintf("%s Engine fired. \n\n", emoji.Fire)))
	fmt.Fprint(errOut, cyan(fmt.Sprintf("%s CTRL+C to gracefully stop.\n", emoji.StopSign)))

	for r := range input { // only 1 ScenarioResult expected
		for _, sr := range r.StepResults {
			verboseInfo := ScenarioStepResultToVerboseHttpRequestInfo(sr)

			b := strings.Builder{}
			w := tabwriter.NewWriter(&b, 0, 0, 4, ' ', 0)
			fmt.Fprint(out, cyan(fmt.Sprintf("\n\nSTEP (%d) %-5s\n", verboseInfo.StepId, verboseInfo.StepName)))
			fmt.Fprint(out, cyan("-------------------------------------"))
			if len(verboseInfo.Envs) > 0 {
				fmt.Fprintf(w, "%s\n", blue("- Environment Variables"))
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
			}
			if len(verboseInfo.TestData) > 0 {
				fmt.Fprintf(w, "%s\n", blue("- Test Data"))

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
				fmt.Fprintf(w, "\n")
			}

			if verboseInfo.Error != "" && isVerboseInfoRequestEmpty(verboseInfo.Request) {
				fmt.Fprintf(w, "%s Error: \t%-5s \n", emoji.SosButton, verboseInfo.Error)
				fmt.Fprintln(w)
				fmt.Fprint(out, b.String())
				break
			}
			fmt.Fprintf(w, "%s\n", blue("- Request"))
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
				fmt.Fprintf(w, "\n%s\n", blue("- Response"))
				fmt.Fprintf(w, "\tStatusCode:\t%-5d \n", verboseInfo.Response.StatusCode)
				fmt.Fprintf(w, "\tResponseTime:\t%-5d(ms) \n", verboseInfo.Response.ResponseTime)
				if st := verboseInfo.Response.Stream; st != nil {
					fmt.Fprintf(w, "\tStream:\tend %s, %d events, %d comments, %d error events, %d bytes\n", st.End, st.Events, st.Comments, st.ErrorEvents, st.Bytes)
					fmt.Fprintf(w, "\t\tTTFB %.1fms, first event %.1fms, first token %.1fms, total %.1fms\n", st.TTFB, st.FirstEvent, st.FirstToken, st.Total)
					fmt.Fprintf(w, "\t\tgap max %.1fms, p50 %.1fms, p95 %.1fms\n", st.GapMax, st.GapP50, st.GapP95)
				}
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
				fmt.Fprintf(w, "%s\n", yellow("- Failed Captures"))
				for wKey, wVal := range verboseInfo.FailedCaptures {
					fmt.Fprintf(w, "\t\t%s: \t%s \n", wKey, wVal)
				}
			}

			if len(verboseInfo.FailedAssertions) > 0 {
				fmt.Fprintf(w, "%s", yellow("- Failed Assertions"))
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

	b := strings.Builder{}
	w := tabwriter.NewWriter(&b, 0, 0, 4, ' ', 0)
	if s.result.TestStatus == "success" {
		fmt.Fprintf(w, "%s", green("Test Status : Success\n"))

	} else if s.result.TestStatus == "failed" {
		fmt.Fprintf(w, "\n%s", red("Test Status: Failed\n"))
		for _, failedRule := range s.result.TestFailedAssertions {
			fmt.Fprintf(w, red("\nRule: %s\n"), failedRule.Rule)
			fmt.Fprint(w, red("Received: \n"))

			for ident, values := range failedRule.ReceivedMap {
				fmt.Fprintf(w, red("\t%s: %v\n"), ident, values)
			}
		}
	}
	fmt.Fprintln(w)
	fmt.Fprint(out, b.String())

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

		printOutcomes(w, v.Outcomes)
		printStream(w, v.Stream)

		if v.Fail.AssertionErrorDist.Count > 0 {
			fmt.Fprintln(w, "\nAssertion Error Distribution:")
			for e, c := range v.Fail.AssertionErrorDist.Conditions {
				fmt.Fprintf(w, "\tCondition: %s\n", e)
				fmt.Fprintf(w, "\t\tFail Count: %d\n", c.Count)
				fmt.Fprintf(w, "\t\tReceived: \n")

				for ident, values := range c.Received {
					// deduplication
					values = deduplicate(values)
					fmt.Fprintf(w, "\t\t\t %s : %v\n", ident, values)
				}

				fmt.Fprintf(w, "\t\tReason: %s \n\n", c.Reason)
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

	printGoodput(w, s.result.Goodput)
	printStages(w, s.result.Stages, s.result.Goodput != nil)

	if s.result.TestStatus == "success" {
		fmt.Fprintf(w, "%s", green("Test Status : Success\n"))

	} else if s.result.TestStatus == "failed" {
		fmt.Fprintf(w, "\n%s", red("Test Status: Failed\n"))
		for _, failedRule := range s.result.TestFailedAssertions {
			fmt.Fprintf(w, red("\nRule: %s\n"), failedRule.Rule)
			fmt.Fprint(w, red("Received: \n"))

			for ident, values := range failedRule.ReceivedMap {
				fmt.Fprintf(w, red("\t%s: %v\n"), ident, values)
			}
		}
	}

	w.Flush()
	fmt.Fprint(out, hiCyan(b.String()))
}

var outcomeOrder = []string{OutcomeOK, OutcomeRateLimited, OutcomeUnavailable, OutcomeServerError,
	OutcomeClientError, OutcomeTimeout, OutcomeNetworkError, OutcomeStreamCut, OutcomeStreamError}

var outcomeLabel = map[string]string{
	OutcomeOK:           "OK",
	OutcomeRateLimited:  "429 Rate Limited",
	OutcomeUnavailable:  "503 Unavailable",
	OutcomeServerError:  "Other 5xx",
	OutcomeClientError:  "Other 4xx",
	OutcomeTimeout:      "Timeout",
	OutcomeNetworkError: "Network Error",
	OutcomeStreamCut:    "Stream Cut",
	OutcomeStreamError:  "Error Event In Stream",
}

func printOutcomes(w io.Writer, outcomes map[string]int64) {
	if len(outcomes) == 0 {
		return
	}
	fmt.Fprintln(w, "\nOutcomes (warm-up excluded):")
	for _, k := range outcomeOrder {
		if c, ok := outcomes[k]; ok {
			fmt.Fprintf(w, "  %s\t:%d\n", outcomeLabel[k], c)
		}
	}
}

func printDistRow(w io.Writer, name string, d *Dist) {
	if d == nil {
		return
	}
	fmt.Fprintf(w, "  %s\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%d\n", name, d.P50, d.P90, d.P95, d.P99, d.Max, d.Mean, d.Count)
}

func printStream(w io.Writer, st *StreamSummary) {
	if st == nil {
		return
	}
	fmt.Fprintf(w, "\nStream (warm-up excluded): %d streams, ends: done %d, closed %d, cut %d, timeout %d; error events in %d; no first token in %d\n",
		st.Count, st.Ends[types.StreamEndDone], st.Ends[types.StreamEndClosed], st.Ends[types.StreamEndCut],
		st.Ends[types.StreamEndTimeout], st.ErrorEvents, st.NoFirstToken)
	fmt.Fprintln(w, "  (ms)\tp50\tp90\tp95\tp99\tmax\tmean\tcount")
	printDistRow(w, "TTFB", st.TTFB)
	printDistRow(w, "First Event", st.FirstEvent)
	printDistRow(w, "First Token", st.FirstToken)
	printDistRow(w, "Total", st.Total)
	printDistRow(w, "Max Gap", st.GapMax)
	printDistRow(w, "Gap p50", st.GapP50)
	printDistRow(w, "Gap p95", st.GapP95)
	if st.Events != nil {
		fmt.Fprintf(w, "  %s\t%.0f\t%.0f\t%.0f\t%.0f\t%.0f\t%.1f\t%d\n", "Events", st.Events.P50, st.Events.P90,
			st.Events.P95, st.Events.P99, st.Events.Max, st.Events.Mean, st.Events.Count)
	}
	if st.TruncatedBodies > 0 {
		fmt.Fprintf(w, "  Bodies cut at max_body_bytes: %d\n", st.TruncatedBodies)
	}
}

func sloText(slo types.SLO) string {
	parts := make([]string, 0)
	add := func(name string, v int64) {
		if v > 0 {
			parts = append(parts, fmt.Sprintf("%s <= %dms", name, v))
		}
	}
	add("ttfb", slo.TTFBMs)
	add("first event", slo.FirstEventMs)
	add("first token", slo.FirstTokenMs)
	add("total", slo.TotalMs)
	add("max gap", slo.MaxGapMs)
	if len(parts) == 0 {
		return "every step succeeds"
	}
	return strings.Join(parts, ", ")
}

func printGoodput(w io.Writer, g *Goodput) {
	if g == nil {
		return
	}
	fmt.Fprintf(w, "\nGoodput (warm-up excluded):\t%d / %d iterations (%.1f%%), SLO: %s\n", g.Good, g.Iterations, g.Perc, sloText(g.SLO))
}

func printStages(w io.Writer, stages []*StageSummary, withGoodput bool) {
	if len(stages) == 0 {
		return
	}
	fmt.Fprintln(w, "\nStages:")
	fmt.Fprintln(w, "  #\tName\tIter\tOK\tGood%\t429\t503\t5xx\tTimeout\tNet/Cut\tFirstTok p50\tp95\tTotal p50\tp95")
	for _, st := range stages {
		name := st.Name
		if name == "" {
			name = fmt.Sprintf("stage %d", st.Index+1)
		}
		if st.Warmup {
			name += " (warm-up)"
		}
		good := "-"
		if withGoodput && st.GoodputPerc != nil {
			good = fmt.Sprintf("%.1f", *st.GoodputPerc)
		}
		ftP50, ftP95, totP50, totP95 := "-", "-", "-", "-"
		if st.Stream != nil && st.Stream.FirstToken != nil {
			ftP50, ftP95 = fmt.Sprintf("%.0f", st.Stream.FirstToken.P50), fmt.Sprintf("%.0f", st.Stream.FirstToken.P95)
		}
		if st.Stream != nil && st.Stream.Total != nil {
			totP50, totP95 = fmt.Sprintf("%.0f", st.Stream.Total.P50), fmt.Sprintf("%.0f", st.Stream.Total.P95)
		}
		o := st.Outcomes
		fmt.Fprintf(w, "  %d\t%s\t%d\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\n", st.Index+1, name, st.Iterations, st.OK, good,
			o[OutcomeRateLimited], o[OutcomeUnavailable], o[OutcomeServerError], o[OutcomeTimeout],
			o[OutcomeNetworkError]+o[OutcomeStreamCut], ftP50, ftP95, totP50, totP95)
	}
}

func deduplicate(values []interface{}) []interface{} {
	seen := make(map[interface{}]bool)
	result := make([]interface{}, 0)

	for _, v := range values {
		if _, ok := seen[v]; !ok {
			seen[v] = true
			result = append(result, v)
		}
	}

	return result
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
