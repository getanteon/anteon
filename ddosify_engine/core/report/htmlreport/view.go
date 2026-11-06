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

package htmlreport

import (
	_ "embed"
	"fmt"
	"go.ddosify.com/ddosify/v2/core/llm"
	"html/template"
	"math"
	"strings"

	"go.ddosify.com/ddosify/v2/core/report"
)

//go:embed page.html.tmpl
var pageSource string

//go:embed charts.js
var chartsJS string

var page = template.Must(template.New("page").Funcs(template.FuncMap{"inc": func(i int) int { return i + 1 }}).Parse(pageSource))

type card struct {
	Label, Value, Note string
	Alarm              bool
}

type outcomeRow struct {
	Label    string
	Count    string
	Perc     string
	Width    string // CSS width of the bar
	Alarm    bool
	Nonzero  bool
	rawCount int64
}

type stageRow struct {
	Index                          int
	Name                           string
	Warmup                         bool
	WarmupNote                     bool // warm-up, and the name does not say so
	Iterations, OK, Goodput        string
	R429, R503, R5xx, Timeout, Cut string
	Network                        string
	FTp50, FTp95, Totp50, Totp95   string
	Has429, Has503                 bool
}

type metricRow struct {
	Name                                 string
	Count, P50, P90, P95, P99, Max, Mean string
}

type stepRow struct {
	ID       uint16
	Name     string
	Success  string
	Failed   string
	Metrics  []metricRow
	Outcomes []outcomeRow
}

type pageView struct {
	R           Report
	Generated   string
	Duration    string
	StatusText  string
	StatusAlarm bool
	Cards       []card
	Outcomes    []outcomeRow
	Data        chartData
	Steady      *ReqData
	SteadyRows  []steadyRow
	StepRows    []stepStageRow
	Script      template.JS
	PerStep     string
	Stages      []stageRow
	Steps       []stepRow
	ITLName     string
	GoodputText string
	SLOText     []string

	// The answers' output: Unit is tokens, or chunks when no answer could be counted in tokens
	TokenUnit    string
	TokenMetrics []metricRow
	TokenFacts   []string
}

func fromLLM(d *llm.Dist) *report.Dist {
	if d == nil {
		return nil
	}
	return &report.Dist{Count: d.Count, Min: d.Min, Mean: d.Mean, P50: d.P50, P90: d.P90, P95: d.P95, P99: d.P99, Max: d.Max}
}

// tokenView never calls a chunk a token: without a token count it reports content events.
func tokenView(r Report, v *pageView) {
	s := r.LLM
	if s == nil && r.Schedule == nil {
		return
	}
	if s != nil {
		out := s.Tokens
		v.TokenUnit = "tokens"
		if out == nil {
			out, v.TokenUnit = s.Chunks, "chunks"
		}
		if out != nil {
			v.TokenMetrics = append(v.TokenMetrics, metric("Output per answer ("+v.TokenUnit+")", fromLLM(out.Output)))
			if v.TokenUnit == "tokens" {
				v.TokenMetrics = append(v.TokenMetrics, metric("TPOT, ms per output token", fromLLM(out.TimePer)))
			}
			v.TokenMetrics = append(v.TokenMetrics, metric("Output "+v.TokenUnit+"/s per answer", fromLLM(out.OutputPerSec)))
			v.TokenFacts = append(v.TokenFacts, fmt.Sprintf("%s %s over %d answers, %.1f %s/s from the first send to the last answer's end",
				fmtInt(out.Total), v.TokenUnit, out.Requests, out.SystemPerSec, v.TokenUnit))
			if v.TokenUnit == "chunks" {
				v.TokenFacts = append(v.TokenFacts, "No answer carried a token count: these are content events, not tokens. Give a tokenizer.json to count tokens.")
			}
		}
		v.TokenMetrics = append(v.TokenMetrics, metric("ITL, ms between content events", fromLLM(s.ITL)))
		if s.InputTokens != nil {
			v.TokenMetrics = append(v.TokenMetrics, metric("Input tokens", fromLLM(s.InputTokens)))
		}
	}
	switch {
	case r.PrefixCache != "":
		v.TokenFacts = append(v.TokenFacts, "Prefix cache: "+r.PrefixCache+".")
	default:
		v.TokenFacts = append(v.TokenFacts, "Prefix cache: prompts sent as they are.")
	}
	if sc := r.Schedule; sc != nil && sc.Lag != nil {
		v.TokenMetrics = append(v.TokenMetrics, metric("Scheduling lag, ms", fromLLM(sc.Lag)))
		if sc.Saturated {
			v.TokenFacts = append(v.TokenFacts, fmt.Sprintf("The client fell behind its schedule: p99 lag %.0f ms, %d iterations over 100 ms late.", sc.Lag.P99, sc.Late))
		}
	}
}

var outcomeOrder = []struct {
	key, label string
	alarm      bool
}{
	{report.OutcomeOK, "ok", false},
	{report.OutcomeRateLimited, "429 rate limited", true},
	{report.OutcomeUnavailable, "503 unavailable", true},
	{report.OutcomeServerError, "other 5xx", true},
	{report.OutcomeClientError, "other 4xx", true},
	{report.OutcomeTimeout, "timeout", true},
	{report.OutcomeNetworkError, "network error", true},
	{report.OutcomeStreamCut, "stream cut", true},
	{report.OutcomeStreamError, "error event in stream", true},
	{report.OutcomeInterrupted, "interrupted (in flight when cut)", false},
}

func outcomeRows(m map[string]int64) []outcomeRow {
	var total int64
	for _, v := range m {
		total += v
	}
	rows := make([]outcomeRow, 0, len(outcomeOrder))
	for _, o := range outcomeOrder {
		v := m[o.key]
		p := 0.0
		if total > 0 {
			p = float64(v) * 100 / float64(total)
		}
		rows = append(rows, outcomeRow{
			Label:    o.label,
			Count:    fmtInt(v),
			Perc:     fmt.Sprintf("%.1f%%", p),
			Width:    fmt.Sprintf("%.2f%%", p),
			Alarm:    o.alarm && v > 0,
			Nonzero:  v > 0,
			rawCount: v,
		})
	}
	return rows
}

func distCell(d *report.Dist, f func(*report.Dist) float64) string {
	if d == nil || d.Count == 0 {
		return "-"
	}
	return fmtMs(f(d))
}

func metric(name string, d *report.Dist) metricRow {
	if d == nil {
		return metricRow{Name: name, Count: "0", P50: "-", P90: "-", P95: "-", P99: "-", Max: "-", Mean: "-"}
	}
	return metricRow{
		Name:  name,
		Count: fmtInt(d.Count),
		P50:   distCell(d, func(d *report.Dist) float64 { return d.P50 }),
		P90:   distCell(d, func(d *report.Dist) float64 { return d.P90 }),
		P95:   distCell(d, func(d *report.Dist) float64 { return d.P95 }),
		P99:   distCell(d, func(d *report.Dist) float64 { return d.P99 }),
		Max:   distCell(d, func(d *report.Dist) float64 { return d.Max }),
		Mean:  distCell(d, func(d *report.Dist) float64 { return d.Mean }),
	}
}

func buildView(r Report) pageView {
	v := pageView{
		R:         r,
		Generated: r.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
		Duration:  clock(r.Duration.Seconds()),
		ITLName:   "ITL",
	}
	if r.ITLIsEventGap {
		v.ITLName = "Event gap"
	}
	switch {
	case r.Stopped:
		v.StatusText, v.StatusAlarm = "stopped early", true
	case r.Status == "failed":
		v.StatusText, v.StatusAlarm = "failed", true
	case r.Status == "success":
		v.StatusText = "success"
	default:
		v.StatusText = "unknown"
	}
	if r.FailedCriteria > 0 {
		v.StatusText += fmt.Sprintf(", %d success criteria failed", r.FailedCriteria)
	}

	for _, t := range r.Config.SLO {
		v.SLOText = append(v.SLOText, fmt.Sprintf("%s <= %s ms", t.Name, fmtInt(t.Ms)))
	}
	if g := r.Goodput; g != nil {
		v.GoodputText = fmt.Sprintf("%.1f%%", g.Perc)
		v.Cards = append(v.Cards, card{Label: "Goodput", Value: v.GoodputText,
			Note: fmt.Sprintf("%s of %s iterations met the SLO", fmtInt(g.Good), fmtInt(g.Iterations))})
	}

	v.Outcomes = outcomeRows(r.Outcomes)
	var requests int64
	for _, o := range v.Outcomes {
		requests += o.rawCount
	}
	reqNote := "warm-up excluded"
	if n := len(r.Steps); n > 1 {
		reqNote = fmt.Sprintf("%d per iteration, warm-up excluded", n)
	}
	v.Cards = append(v.Cards, card{Label: "Requests", Value: fmtInt(requests), Note: reqNote})
	for _, o := range v.Outcomes[1:3] {
		v.Cards = append(v.Cards, card{Label: o.Label, Value: o.Count, Note: o.Perc + " of results", Alarm: o.Alarm})
	}

	for _, s := range r.Steps {
		row := stepRow{ID: s.ID, Name: s.Name, Success: fmtInt(s.Success), Failed: fmtInt(s.Failed)}
		if s.Outcomes != nil {
			row.Outcomes = outcomeRows(s.Outcomes)
		}
		if st := s.Stream; st != nil {
			row.Metrics = []metricRow{
				metric("TTFB", st.TTFB),
				metric("First event", st.FirstEvent),
				metric("First token (TTFT)", st.FirstToken),
				metric("Total (E2E)", st.Total),
				metric("Gap p50 per stream", st.GapP50),
				metric("Gap p95 per stream", st.GapP95),
				metric("Gap max per stream", st.GapMax),
			}
			if len(v.Cards) < 7 && st.FirstToken != nil {
				v.Cards = append(v.Cards,
					card{Label: "TTFT p95", Value: fmtDur(st.FirstToken.P95), Note: s.Name},
				)
			}
			if len(v.Cards) < 7 && st.Total != nil {
				v.Cards = append(v.Cards,
					card{Label: "E2E p95", Value: fmtDur(st.Total.P95), Note: s.Name},
				)
			}
		}
		v.Steps = append(v.Steps, row)
	}

	for _, s := range r.Stages {
		o := s.Outcomes
		row := stageRow{
			Index: s.Index + 1, Name: s.Name, Warmup: s.Warmup,
			WarmupNote: s.Warmup && !strings.Contains(strings.ToLower(s.Name), "warm"),
			Iterations: fmtInt(s.Iterations), OK: fmtInt(s.OK), Goodput: "-",
			R429: fmtInt(o[report.OutcomeRateLimited]), R503: fmtInt(o[report.OutcomeUnavailable]),
			R5xx: fmtInt(o[report.OutcomeServerError]), Timeout: fmtInt(o[report.OutcomeTimeout]),
			Network: fmtInt(o[report.OutcomeNetworkError]), Cut: fmtInt(o[report.OutcomeStreamCut]),
			FTp50: "-", FTp95: "-", Totp50: "-", Totp95: "-",
			Has429: o[report.OutcomeRateLimited] > 0, Has503: o[report.OutcomeUnavailable] > 0,
		}
		if s.GoodputPerc != nil {
			row.Goodput = fmt.Sprintf("%.1f%%", *s.GoodputPerc)
		}
		if st := s.Stream; st != nil {
			row.FTp50 = distCell(st.FirstToken, func(d *report.Dist) float64 { return d.P50 })
			row.FTp95 = distCell(st.FirstToken, func(d *report.Dist) float64 { return d.P95 })
			row.Totp50 = distCell(st.Total, func(d *report.Dist) float64 { return d.P50 })
			row.Totp95 = distCell(st.Total, func(d *report.Dist) float64 { return d.P95 })
		}
		v.Stages = append(v.Stages, row)
	}

	v.Data, v.Steady = buildData(r, v.ITLName)
	v.SteadyRows, v.StepRows = steadyRows(v.Steady)
	v.Script = template.JS(chartsJS)
	v.PerStep = fmtInt(PerStepSamples)
	tokenView(r, &v)
	return v
}

type steadyRow struct {
	Name, Settle, Steady, Answered, NoText, Rejected, Failed, Good string
	FTp50, FTp95, Totp50, Totp95                                   string
	Warmup                                                         bool
}

type stepStageRow struct {
	Stage, Step, N               string
	FTp50, FTp95, Totp50, Totp95 string
	Warmup                       bool
}

func msCell(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmtMs(*v)
}

// p95Cell is p95 with its 95% interval.
func p95Cell(d Dist) string {
	if d.P95 == nil {
		return "-"
	}
	if len(d.P95CI) == 2 {
		return fmt.Sprintf("%s [%s-%s]", fmtMs(*d.P95), fmtMs(d.P95CI[0]), fmtMs(d.P95CI[1]))
	}
	return fmtMs(*d.P95)
}

func steadyRows(q *ReqData) ([]steadyRow, []stepStageRow) {
	if q == nil {
		return nil, nil
	}
	var rows []steadyRow
	var steps []stepStageRow
	for _, s := range q.Stages {
		good := "-"
		if s.Steady > 0 {
			good = fmt.Sprintf("%.1f%%", float64(s.Good)/float64(s.Steady)*100)
		}
		rows = append(rows, steadyRow{Name: s.Name, Warmup: s.Warmup, Settle: fmtInt(int64(math.Round(s.SettleMs / 1000))),
			Steady: fmtInt(int64(s.Steady)), Answered: fmtInt(int64(s.Answered)), NoText: fmtInt(int64(s.NoText)),
			Rejected: fmtInt(int64(s.Rejected)), Failed: fmtInt(int64(s.Failed)), Good: good,
			FTp50: msCell(s.FT.P50), FTp95: p95Cell(s.FT), Totp50: msCell(s.Total.P50), Totp95: p95Cell(s.Total)})
		if len(q.Steps) < 2 {
			continue
		}
		for _, st := range s.Steps {
			steps = append(steps, stepStageRow{Stage: s.Name, Step: st.Name, Warmup: s.Warmup, N: fmtInt(int64(st.Total.N)),
				FTp50: msCell(st.FT.P50), FTp95: msCell(st.FT.P95), Totp50: msCell(st.Total.P50), Totp95: msCell(st.Total.P95)})
		}
	}
	return rows, steps
}
