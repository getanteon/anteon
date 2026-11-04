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

// Package live shows a running load test: a terminal UI on a TTY, NDJSON lines otherwise.
// It reads aggregated snapshots, never per-request events, and keeps only short history rings.
package live

import (
	"time"

	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/types"
)

// Snapshot is the engine's aggregate at one instant. The engine sends one per interval.
// It mirrors the engine's own snapshot type; an adapter converts that one into this.
type Snapshot struct {
	At       time.Time
	Elapsed  time.Duration
	Duration time.Duration // planned length of the test, 0 when unknown

	Stage  StageRef
	Stages []StagePlan

	TargetRPS   float64
	AchievedRPS float64
	InFlight    int64
	Iterations  int64 // iterations started so far

	// Milliseconds.
	TTFT Latency
	ITL  Latency
	E2E  Latency

	// ITLIsEventGap labels ITL as the gap between SSE events, when no token-level ITL is measured.
	ITLIsEventGap bool

	Goodput  Goodput
	Outcomes Outcomes

	// Set when a step has an llm config
	Tokens *Tokens

	// Milliseconds the first request went out after the schedule meant it to
	SchedulingLag Latency

	// The prompt's prefix cache setting: "busted", "shared N tokens", or "" when untouched
	PrefixCache string
}

// Tokens is the answers' output. Unit is "tokens", or "chunks" when no answer could be counted in
// tokens: then Output and PerSec count content events and TPOT is empty.
type Tokens struct {
	Unit   string
	Output Latency // per answer, in Unit
	TPOT   Latency // ms, tokens only
	ITL    Latency // ms between content events
	PerSec float64 // in Unit a second, all answers
}

// StageRef is the stage the test is in. Index is 0-based, Count is the number of stages.
type StageRef struct {
	Index    int
	Count    int
	Name     string
	Warmup   bool
	Elapsed  time.Duration
	Duration time.Duration
}

// StagePlan is one stage of the load plan.
type StagePlan struct {
	Name     string
	Warmup   bool
	Duration time.Duration
	Count    int64 // iterations planned
}

// Latency holds percentiles in milliseconds over the samples so far.
type Latency struct {
	Count int64
	P50   float64
	P90   float64
	P95   float64
	P99   float64
}

// Goodput counts iterations that met the SLO. Warm-up excluded.
type Goodput struct {
	Good  int64
	Total int64
}

// Perc is the goodput percentage, 0 when nothing finished.
func (g Goodput) Perc() float64 {
	if g.Total == 0 {
		return 0
	}
	return float64(g.Good) * 100 / float64(g.Total)
}

// Outcomes counts step results by outcome, with the keys of the final report.
type Outcomes struct {
	OK           int64
	RateLimited  int64 // 429
	Unavailable  int64 // 503
	ServerError  int64 // other 5xx
	ClientError  int64 // other 4xx
	Timeout      int64
	NetworkError int64
	StreamCut    int64
	StreamError  int64
}

// Total is the sum of all outcomes.
func (o Outcomes) Total() int64 {
	return o.OK + o.RateLimited + o.Unavailable + o.ServerError + o.ClientError +
		o.Timeout + o.NetworkError + o.StreamCut + o.StreamError
}

// Map returns the counts keyed as in the final report's outcomes.
func (o Outcomes) Map() map[string]int64 {
	return map[string]int64{
		report.OutcomeOK:           o.OK,
		report.OutcomeRateLimited:  o.RateLimited,
		report.OutcomeUnavailable:  o.Unavailable,
		report.OutcomeServerError:  o.ServerError,
		report.OutcomeClientError:  o.ClientError,
		report.OutcomeTimeout:      o.Timeout,
		report.OutcomeNetworkError: o.NetworkError,
		report.OutcomeStreamCut:    o.StreamCut,
		report.OutcomeStreamError:  o.StreamError,
	}
}

// OutcomesFromMap reads counts keyed as in the final report's outcomes.
func OutcomesFromMap(m map[string]int64) Outcomes {
	return Outcomes{
		OK:           m[report.OutcomeOK],
		RateLimited:  m[report.OutcomeRateLimited],
		Unavailable:  m[report.OutcomeUnavailable],
		ServerError:  m[report.OutcomeServerError],
		ClientError:  m[report.OutcomeClientError],
		Timeout:      m[report.OutcomeTimeout],
		NetworkError: m[report.OutcomeNetworkError],
		StreamCut:    m[report.OutcomeStreamCut],
		StreamError:  m[report.OutcomeStreamError],
	}
}

// Threshold is one SLO limit, shown beside the goodput.
type Threshold struct {
	Name string
	Ms   int64
}

// ThresholdsFromSLO lists the limits an SLO sets, in a fixed order.
func ThresholdsFromSLO(s *types.SLO) []Threshold {
	if s == nil {
		return nil
	}
	var t []Threshold
	add := func(name string, ms int64) {
		if ms > 0 {
			t = append(t, Threshold{Name: name, Ms: ms})
		}
	}
	add("ttfb", s.TTFBMs)
	add("first event", s.FirstEventMs)
	add("first token", s.FirstTokenMs)
	add("total", s.TotalMs)
	add("max gap", s.MaxGapMs)
	add("tpot", s.TPOTMs)
	return t
}

// StagePlansFromHammer lists the manual_load stages of a test.
func StagePlansFromHammer(h *types.Hammer) []StagePlan {
	if h == nil {
		return nil
	}
	plans := make([]StagePlan, 0, len(h.TimeRunCountMap))
	for _, s := range h.TimeRunCountMap {
		plans = append(plans, StagePlan{
			Name:     s.Name,
			Warmup:   s.Warmup,
			Duration: time.Duration(s.Duration) * time.Second,
			Count:    int64(s.Count),
		})
	}
	return plans
}
