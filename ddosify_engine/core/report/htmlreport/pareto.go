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
	"fmt"
	"sort"

	"go.ddosify.com/ddosify/v2/core/llm"
	"go.ddosify.com/ddosify/v2/core/types"
)

// ParetoPoint is one stage read as a point of the throughput and interactivity trade-off.
type ParetoPoint struct {
	Stage       string   `json:"stage"`
	Concurrency int      `json:"c"`
	Requests    int64    `json:"n"`
	PerUser     *float64 `json:"user"` // output tokens a second of one answer, median: tokens / (end - first token)
	PerGPU      *float64 `json:"gpu"`  // output tokens a second of the stage, all answers, over the GPUs
	TTFT50      *float64 `json:"ttft50"`
	TTFT99      *float64 `json:"ttft99"`
	TPOT50      *float64 `json:"tpot50"`
	TPOT99      *float64 `json:"tpot99"`
	Input50     *float64 `json:"in50"`
	Output50    *float64 `json:"out50"`

	// Iterations started a second: an open-loop stage's planned rate when the config says it, else
	// those the stage ran over its span. In flight is the concurrency of a closed-loop stage, and of
	// an open-loop one the mean by Little's law: the rate times the mean time of an answer.
	Rate     *float64 `json:"rate"`
	InFlight *float64 `json:"inflight"`
}

// ParetoSeries is one run's points, in stage order.
type ParetoSeries struct {
	Label  string        `json:"label"`
	GPUs   int           `json:"gpus"`
	Unit   string        `json:"unit"` // tokens, or chunks when tokens could not be counted
	Points []ParetoPoint `json:"points"`
}

// Labeled is one run of a comparison.
type Labeled struct {
	Label  string
	Report Report
}

func ptr(d *llm.Dist, f func(*llm.Dist) float64) *float64 {
	if d == nil || d.Count == 0 {
		return nil
	}
	return pf(f(d))
}

// ParetoOf reads the stages of a run that streamed answers with counted output. A warm-up stage
// is left out. gpus 0 takes the run's capacity config, else 1.
func ParetoOf(r Report, label string, gpus int) *ParetoSeries {
	if gpus < 1 {
		gpus = r.Capacity.GPUCount()
	}
	s := &ParetoSeries{Label: label, GPUs: gpus}
	for _, st := range r.Stages {
		if st.Warmup || st.Stream == nil || st.Stream.LLM == nil {
			continue
		}
		out := st.Stream.LLM.Tokens
		s.Unit = "tokens"
		if out == nil {
			out, s.Unit = st.Stream.LLM.Chunks, "chunks"
		}
		if out == nil {
			continue
		}
		p := ParetoPoint{Stage: st.Name, Concurrency: st.Concurrency, Requests: out.Requests,
			PerUser:  ptr(out.OutputPerSec, func(d *llm.Dist) float64 { return d.P50 }),
			TPOT50:   ptr(out.TimePer, func(d *llm.Dist) float64 { return d.P50 }),
			TPOT99:   ptr(out.TimePer, func(d *llm.Dist) float64 { return d.P99 }),
			Output50: ptr(out.Output, func(d *llm.Dist) float64 { return d.P50 }),
			Input50:  ptr(st.Stream.LLM.InputTokens, func(d *llm.Dist) float64 { return d.P50 }),
		}
		if p.Stage == "" {
			p.Stage = fmt.Sprintf("S%d", st.Index+1)
			if st.Concurrency > 0 {
				p.Stage = fmt.Sprintf("c=%d", st.Concurrency)
			}
		}
		if out.SystemPerSec > 0 {
			p.PerGPU = pf(out.SystemPerSec / float64(gpus))
		}
		if ft := st.Stream.FirstToken; ft != nil && ft.Count > 0 {
			p.TTFT50, p.TTFT99 = pf(ft.P50), pf(ft.P99)
		}
		if st.Index < len(r.Config.Stages) && st.Concurrency == 0 {
			if c := r.Config.Stages[st.Index]; c.Concurrency == 0 && c.DurationS > 0 {
				p.Rate = pf(float64(c.Count) / float64(c.DurationS))
			}
		}
		if p.Rate == nil && out.WindowSeconds > 0 {
			p.Rate = pf(float64(st.Iterations) / out.WindowSeconds)
		}
		switch {
		case st.Concurrency > 0:
			p.InFlight = pf(float64(st.Concurrency))
		case p.Rate != nil && st.Stream.Total != nil && st.Stream.Total.Count > 0:
			p.InFlight = pf(*p.Rate * st.Stream.Total.Mean / 1000)
		}
		s.Points = append(s.Points, p)
	}
	if len(s.Points) == 0 {
		return nil
	}
	return s
}

// meets tells whether a point is within a tier: both p99s, each against the limit the tier sets.
func meets(p ParetoPoint, t types.CapacityTier) bool {
	if t.FirstTokenMs > 0 && (p.TTFT99 == nil || *p.TTFT99 > t.FirstTokenMs) {
		return false
	}
	if t.TPOTMs > 0 && (p.TPOT99 == nil || *p.TPOT99 > t.TPOTMs) {
		return false
	}
	return p.PerGPU != nil
}

// capacity is the point a tier allows: the highest concurrency within it, or, for open-loop stages
// such as MLPerf's Server scenario's Poisson arrivals, the highest arrival rate within it. nil when
// no stage meets the tier.
func capacity(s *ParetoSeries, t types.CapacityTier) *ParetoPoint {
	var best *ParetoPoint
	for i := range s.Points {
		p := &s.Points[i]
		if !meets(*p, t) {
			continue
		}
		switch {
		case best == nil:
			best = p
		case p.Concurrency > 0 || best.Concurrency > 0:
			if p.Concurrency > best.Concurrency {
				best = p
			}
		case p.Rate != nil && (best.Rate == nil || *p.Rate > *best.Rate):
			best = p
		}
	}
	return best
}

type capacityCell struct {
	Value, Note string
	None        bool
}

type capacityRow struct {
	Tier   string
	Limits string
	Cells  []capacityCell
}

type paretoStageRow struct {
	Series, Stage, Concurrency, Requests, Input, PerUser, PerGPU, TTFT50, TTFT99, TPOT50, TPOT99 string
	Rate, InFlight                                                                               string
	Tiers                                                                                        []bool
}

type paretoView struct {
	Series    []*ParetoSeries
	Tiers     []types.CapacityTier
	TierNames []string
	Rows      []capacityRow
	Stages    []paretoStageRow
	MinN      int64  // the fewest requests behind a point's p99
	Rate      string // tok/s, or chunks/s when a run's output could not be counted in tokens
}

func limitsOf(t types.CapacityTier) string {
	var s string
	if t.FirstTokenMs > 0 {
		s = "TTFT p99 <= " + fmtMs(t.FirstTokenMs) + " ms"
	}
	if t.TPOTMs > 0 {
		if s != "" {
			s += ", "
		}
		s += "TPOT p99 <= " + fmtMs(t.TPOTMs) + " ms"
	}
	return s
}

func fnum(v *float64, digits int) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.*f", digits, *v)
}

func fms(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmtMs(*v)
}

func buildPareto(series []*ParetoSeries, tiers []types.CapacityTier) *paretoView {
	var keep []*ParetoSeries
	for _, s := range series {
		if s != nil {
			keep = append(keep, s)
		}
	}
	if len(keep) == 0 {
		return nil
	}
	v := &paretoView{Series: keep, Tiers: tiers, MinN: -1, Rate: "tok/s"}
	for _, s := range keep {
		if s.Unit == "chunks" {
			v.Rate = "chunks/s"
		}
	}
	for _, t := range tiers {
		v.TierNames = append(v.TierNames, t.Name)
		row := capacityRow{Tier: t.Name, Limits: limitsOf(t)}
		for _, s := range keep {
			p := capacity(s, t)
			if p == nil {
				row.Cells = append(row.Cells, capacityCell{Value: "none", None: true, Note: "no stage meets it"})
				continue
			}
			value := fmt.Sprintf("concurrency %d", p.Concurrency)
			note := fmt.Sprintf("%s %s per GPU, %s %s per user", fnum(p.PerGPU, 0), v.Rate, fnum(p.PerUser, 1), v.Rate)
			if p.Concurrency == 0 {
				value = fmt.Sprintf("%s/s", fnum(p.Rate, 2))
				note = fmt.Sprintf("%s (%s), about %s in flight; %s", p.Stage, "arrivals a second", fnum(p.InFlight, 1), note)
			}
			row.Cells = append(row.Cells, capacityCell{Value: value, Note: note})
		}
		v.Rows = append(v.Rows, row)
	}
	for _, s := range keep {
		pts := append([]ParetoPoint(nil), s.Points...)
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].Concurrency < pts[j].Concurrency })
		for _, p := range pts {
			if v.MinN < 0 || p.Requests < v.MinN {
				v.MinN = p.Requests
			}
			r := paretoStageRow{Series: s.Label, Stage: p.Stage, Requests: fmtInt(p.Requests),
				Input: fnum(p.Input50, 0), PerUser: fnum(p.PerUser, 1), PerGPU: fnum(p.PerGPU, 0),
				TTFT50: fms(p.TTFT50), TTFT99: fms(p.TTFT99), TPOT50: fms(p.TPOT50), TPOT99: fms(p.TPOT99), Concurrency: "-",
				Rate: fnum(p.Rate, 2), InFlight: fnum(p.InFlight, 1)}
			if p.Concurrency > 0 {
				r.Concurrency = fmt.Sprintf("%d", p.Concurrency)
			}
			for _, t := range tiers {
				r.Tiers = append(r.Tiers, meets(p, t))
			}
			v.Stages = append(v.Stages, r)
		}
	}
	return v
}

// worth tells whether a run's report shows the trade-off: a closed-loop stage, or two stages or more.
func (s *ParetoSeries) worth(r Report) bool {
	if s == nil {
		return false
	}
	for _, st := range r.Stages {
		if st.Concurrency > 0 {
			return true
		}
	}
	return len(s.Points) > 1
}
