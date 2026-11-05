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

package live

import (
	"encoding/json"
	"io"
	"math"
	"time"
)

type jsonLatency struct {
	Count int64   `json:"count"`
	P50   float64 `json:"p50"`
	P90   float64 `json:"p90"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
}

type jsonStage struct {
	Index    int     `json:"index"`
	Count    int     `json:"count"`
	Name     string  `json:"name,omitempty"`
	Warmup   bool    `json:"warmup,omitempty"`
	ElapsedS float64 `json:"elapsed_s"`
}

type jsonGoodput struct {
	Good  int64   `json:"good"`
	Total int64   `json:"total"`
	Perc  float64 `json:"perc"`
}

// jsonLine is one NDJSON progress line. Keys follow the final report's snake_case.
type jsonLine struct {
	Type        string           `json:"type"`
	Time        string           `json:"time,omitempty"`
	ElapsedS    float64          `json:"elapsed_s"`
	RemainingS  *float64         `json:"remaining_s,omitempty"`
	Stage       *jsonStage       `json:"stage,omitempty"`
	TargetRPS   float64          `json:"target_rps"`
	AchievedRPS float64          `json:"achieved_rps"`
	InFlight    int64            `json:"in_flight"`
	Iterations  int64            `json:"iterations"`
	TTFT        jsonLatency      `json:"ttft_ms"`
	ITL         jsonLatency      `json:"itl_ms"`
	ITLKind     string           `json:"itl_kind"`
	E2E         jsonLatency      `json:"e2e_ms"`
	Goodput     jsonGoodput      `json:"goodput"`
	Outcomes    map[string]int64 `json:"outcomes"`
	Tokens      *jsonTokens      `json:"tokens,omitempty"`
	SchedLag    *jsonLatency     `json:"scheduling_lag_ms,omitempty"`
	PrefixCache string           `json:"prefix_cache,omitempty"`
}

type jsonTokens struct {
	Unit   string      `json:"unit"`
	Output jsonLatency `json:"output"`
	TPOT   jsonLatency `json:"tpot_ms"`
	ITL    jsonLatency `json:"itl_ms"`
	PerSec float64     `json:"per_sec"`
}

func r3(v float64) float64 { return math.Round(v*1000) / 1000 }

func jl(l Latency) jsonLatency {
	return jsonLatency{Count: l.Count, P50: r3(l.P50), P90: r3(l.P90), P95: r3(l.P95), P99: r3(l.P99)}
}

func toJSONLine(s Snapshot) jsonLine {
	l := jsonLine{
		Type:        "progress",
		ElapsedS:    r3(s.Elapsed.Seconds()),
		TargetRPS:   r3(s.TargetRPS),
		AchievedRPS: r3(s.AchievedRPS),
		InFlight:    s.InFlight,
		Iterations:  s.Iterations,
		TTFT:        jl(s.TTFT),
		ITL:         jl(s.ITL),
		ITLKind:     "itl",
		E2E:         jl(s.E2E),
		Goodput:     jsonGoodput{Good: s.Goodput.Good, Total: s.Goodput.Total, Perc: r3(s.Goodput.Perc())},
		Outcomes:    s.Outcomes.Map(),
	}
	if s.ITLIsEventGap {
		l.ITLKind = "event_gap"
	}
	if t := s.Tokens; t != nil {
		l.Tokens = &jsonTokens{Unit: t.Unit, Output: jl(t.Output), TPOT: jl(t.TPOT), ITL: jl(t.ITL), PerSec: r3(t.PerSec)}
	}
	if s.SchedulingLag.Count > 0 {
		lag := jl(s.SchedulingLag)
		l.SchedLag = &lag
	}
	l.PrefixCache = s.PrefixCache
	if !s.At.IsZero() {
		l.Time = s.At.UTC().Format(time.RFC3339Nano)
	}
	if s.Duration > 0 {
		r := r3(math.Max(0, (s.Duration - s.Elapsed).Seconds()))
		l.RemainingS = &r
	}
	if s.Stage.Count > 0 {
		l.Stage = &jsonStage{
			Index:    s.Stage.Index,
			Count:    s.Stage.Count,
			Name:     s.Stage.Name,
			Warmup:   s.Stage.Warmup,
			ElapsedS: r3(s.Stage.Elapsed.Seconds()),
		}
	}
	return l
}

// WriteNDJSON writes one snapshot as one line.
func WriteNDJSON(w io.Writer, s Snapshot) error {
	b, err := json.Marshal(toJSONLine(s))
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}
