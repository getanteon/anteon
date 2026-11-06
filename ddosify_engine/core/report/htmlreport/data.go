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
	"math"

	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/report/live"
)

// chartData is what the page's script draws. Times are seconds from the start, latencies milliseconds.
type chartData struct {
	SLO      sloData     `json:"slo"`
	RateUnit string      `json:"rateUnit"` // "s", or "min" for a load under one iteration a second
	Window   float64     `json:"window"`   // the sliding window, seconds
	ITLName  string      `json:"itlName"`
	Stages   []stageData `json:"stages"`
	TS       *tsData     `json:"ts"`
	Req      *reqChart   `json:"req"`
}

type sloData struct {
	FT    float64 `json:"ft"`
	Total float64 `json:"total"`
}

type pctData struct {
	P50 *float64 `json:"p50"`
	P95 *float64 `json:"p95"`
	P99 *float64 `json:"p99"`
	Min *float64 `json:"min"`
	Max *float64 `json:"max"`
}

type stageData struct {
	Name        string   `json:"name"`
	Warmup      bool     `json:"warmup"`
	Start       *float64 `json:"start"`
	End         *float64 `json:"end"`
	Iterations  int64    `json:"iterations"`
	OK          int64    `json:"ok"`
	Good        *int64   `json:"good"`
	Rejected    int64    `json:"rejected"`
	RateLimited int64    `json:"rateLimited"`
	OtherFail   int64    `json:"otherFail"`
	FT          *pctData `json:"ft"`
	Total       *pctData `json:"total"`
}

type bandData struct {
	P50 []*float64 `json:"p50"`
	P95 []*float64 `json:"p95"`
}

type tsData struct {
	T        []float64  `json:"t"`
	Target   []float64  `json:"target"`
	Achieved []*float64 `json:"achieved"`
	Failed   []*float64 `json:"failed"`
	InFlight []int64    `json:"inflight"`
	TTFT     *bandData  `json:"ttft"`
	ITL      *bandData  `json:"itl"`
	E2E      bandData   `json:"e2e"`
	Goodput  []*float64 `json:"goodput"`
}

type reqChart struct {
	T       []float64  `json:"t"`
	FTP50   []*float64 `json:"ftP50"`
	FTP95   []*float64 `json:"ftP95"`
	TotP50  []*float64 `json:"totP50"`
	TotP95  []*float64 `json:"totP95"`
	N       []int      `json:"n"`
	Win     []float64  `json:"win"`
	Stream  bool       `json:"stream"`
	Sampled bool       `json:"sampled"`
}

func pctOf(d *report.Dist) *pctData {
	if d == nil || d.Count == 0 {
		return nil
	}
	return &pctData{P50: pf(d.P50), P95: pf(d.P95), P99: pf(d.P99), Min: pf(d.Min), Max: pf(d.Max)}
}

func latOrNil(l live.Latency, v float64) *float64 {
	if l.Count == 0 {
		return nil
	}
	return pf(v)
}

// stageWindows is when each listed stage ran, by its own index: a stage with no result is not
// listed, so the position in the list is not the index. Seconds from the start, and the absolute
// times in milliseconds when the snapshots carried them.
func stageWindows(spans map[int]Span, stages []*report.StageSummary) (rel [][2]float64, abs []StageWindow) {
	rel = make([][2]float64, len(stages))
	abs = make([]StageWindow, len(stages))
	for i, st := range stages {
		abs[i] = StageWindow{Index: st.Index, Name: st.Name, Warmup: st.Warmup}
		sp, ok := spans[st.Index]
		if !ok {
			rel[i] = [2]float64{math.NaN(), math.NaN()}
			continue
		}
		rel[i] = [2]float64{sp.Start, sp.End}
		// A stage ends where the next one, listed or not, begins.
		if next, ok := spans[st.Index+1]; ok {
			rel[i][1] = next.Start
		}
		abs[i].Start, abs[i].End, abs[i].Known = float64(sp.StartAt), float64(sp.EndAt), sp.StartAt > 0
	}
	return rel, abs
}

func buildData(r Report, itlName string) (chartData, *ReqData) {
	var steady *ReqData
	d := chartData{RateUnit: "s", ITLName: itlName}
	if g := r.Goodput; g != nil {
		d.SLO = sloData{FT: float64(g.SLO.FirstTokenMs), Total: float64(g.SLO.TotalMs)}
	}
	pts := r.Series
	rel, abs := stageWindows(r.Spans, r.Stages)
	for i, s := range r.Stages {
		sd := stageData{Name: s.Name, Warmup: s.Warmup, Iterations: s.Iterations, OK: s.OK, Good: s.Good,
			Rejected: s.Outcomes[report.OutcomeUnavailable], RateLimited: s.Outcomes[report.OutcomeRateLimited]}
		for k, v := range s.Outcomes {
			if k != report.OutcomeOK && k != report.OutcomeUnavailable && k != report.OutcomeRateLimited && k != report.OutcomeInterrupted {
				sd.OtherFail += v
			}
		}
		if !math.IsNaN(rel[i][0]) {
			sd.Start, sd.End = pf(rel[i][0]), pf(rel[i][1])
		}
		if st := s.Stream; st != nil {
			sd.FT, sd.Total = pctOf(st.FirstToken), pctOf(st.Total)
		}
		d.Stages = append(d.Stages, sd)
	}

	win := smoothing(r)
	d.Window = win
	started := windowRate(pts, win, func(i int) float64 { return float64(pts[i].Iterations) })
	target := windowMean(pts, win, func(i int) float64 { return pts[i].TargetRPS })
	// A stage's planned rate is its iterations over its length, flat as the plan was written: the
	// plan's own count when the report was written by the run, else the iterations the stage started.
	planned := map[int]float64{}
	for i, s := range r.Stages {
		count, dur := float64(s.Iterations), 0.0
		if s.Index < len(r.Config.Stages) && r.Config.Stages[s.Index].DurationS > 0 {
			c := r.Config.Stages[s.Index]
			count, dur = float64(c.Count), float64(c.DurationS)
		} else if !math.IsNaN(rel[i][0]) {
			dur = rel[i][1] - rel[i][0]
		}
		if dur >= 10 {
			planned[s.Index] = count / dur
		}
	}
	for i, p := range pts {
		if v, ok := planned[p.Stage]; ok && v > 0 {
			target[i] = v
		}
	}
	maxRate := 0.0
	for i := range pts {
		maxRate = max(maxRate, target[i])
		if started[i] != nil {
			maxRate = max(maxRate, *started[i])
		}
	}
	scale := 1.0
	if maxRate > 0 && maxRate < 1 {
		d.RateUnit, scale = "min", 60
	}
	if len(pts) >= 2 {
		ts := &tsData{}
		var ttft, itl bandData
		hasTTFT, hasITL := false, false
		// The counts are the stage's own: carry each finished stage's final count, kept by the
		// recorder from every snapshot, across a stage change.
		var carried int64
		prevStage := pts[0].Stage
		failed := make([]float64, len(pts))
		for i, p := range pts {
			if p.Stage != prevStage {
				carried += r.Spans[prevStage].Failed
				prevStage = p.Stage
			}
			failed[i] = float64(carried + p.Failed)
			ts.T = append(ts.T, p.T)
			ts.Target = append(ts.Target, target[i]*scale)
			if started[i] != nil {
				ts.Achieved = append(ts.Achieved, pf(*started[i]*scale))
			} else {
				ts.Achieved = append(ts.Achieved, nil)
			}
			ts.InFlight = append(ts.InFlight, p.InFlight)
			ttft.P50, ttft.P95 = append(ttft.P50, latOrNil(p.TTFT, p.TTFT.P50)), append(ttft.P95, latOrNil(p.TTFT, p.TTFT.P95))
			itl.P50, itl.P95 = append(itl.P50, latOrNil(p.ITL, p.ITL.P50)), append(itl.P95, latOrNil(p.ITL, p.ITL.P95))
			ts.E2E.P50, ts.E2E.P95 = append(ts.E2E.P50, latOrNil(p.E2E, p.E2E.P50)), append(ts.E2E.P95, latOrNil(p.E2E, p.E2E.P95))
			hasTTFT, hasITL = hasTTFT || p.TTFT.Count > 0, hasITL || p.ITL.Count > 0
			var g *float64
			if p.Goodput.Total > 0 {
				g = pf(p.Goodput.Perc())
			}
			ts.Goodput = append(ts.Goodput, g)
		}
		for _, v := range windowRate(pts, win, func(i int) float64 { return failed[i] }) {
			if v != nil {
				v = pf(*v * scale)
			}
			ts.Failed = append(ts.Failed, v)
		}
		if hasTTFT {
			ts.TTFT = &ttft
		}
		if hasITL {
			ts.ITL = &itl
		}
		d.TS = ts
	}

	if r.Requests != nil {
		start := 0.0
		for _, p := range pts {
			if p.At > 0 {
				start = float64(p.At) - p.T*1000
				break
			}
		}
		rq := r.Requests.Analyze(abs, d.SLO.FT, d.SLO.Total, win*1000)
		if rq != nil {
			if start == 0 && len(rq.Series.T) > 0 {
				start = rq.Series.T[0]
			}
			c := &reqChart{FTP50: rq.Series.FTP50, FTP95: rq.Series.FTP95, TotP50: rq.Series.TotP50, TotP95: rq.Series.TotP95,
				N: rq.Series.N, Win: rq.Series.Win, Sampled: rq.Sampled}
			for _, t := range rq.Series.T {
				c.T = append(c.T, (t-start)/1000)
			}
			for _, v := range rq.Series.FTP50 {
				c.Stream = c.Stream || v != nil
			}
			d.Req = c
			steady = rq
		}
	}
	return d, steady
}

// smoothing is the sliding window of the charts, in seconds: half the shortest stage past warm-up, from 10 to
// 60 seconds, so a short stage is not smeared into the next. A run with no stages takes a tenth of it.
func smoothing(r Report) float64 {
	warm := map[int]bool{}
	for i, s := range r.Config.Stages {
		warm[i] = s.Warmup
	}
	for _, s := range r.Stages {
		warm[s.Index] = warm[s.Index] || s.Warmup
	}
	shortest := math.Inf(1)
	for i, s := range r.Spans {
		if warm[i] {
			continue
		}
		if d := s.End - s.Start; d > 0 {
			shortest = min(shortest, d)
		}
	}
	if math.IsInf(shortest, 1) {
		if n := len(r.Series); n > 0 {
			return min(60, max(10, r.Series[n-1].T/10))
		}
		return 60
	}
	return min(60, max(10, math.Round(shortest/2)))
}

// windowRate is a cumulative count's rate a second over the last win seconds, or over the
// last two points when they are further apart; nil until 15 seconds have passed.
func windowRate(pts []Point, win float64, cum func(int) float64) []*float64 {
	out := make([]*float64, len(pts))
	j := 0
	for i := range pts {
		for j < i-1 && pts[i].T-pts[j+1].T >= win {
			j++
		}
		if dt := pts[i].T - pts[j].T; dt >= min(15, win) {
			out[i] = pf(max(0, cum(i)-cum(j)) / dt)
		}
	}
	return out
}

// windowMean is the mean of a per-second value over the last win seconds.
func windowMean(pts []Point, win float64, v func(int) float64) []float64 {
	out := make([]float64, len(pts))
	j, sum := 0, 0.0
	for i := range pts {
		sum += v(i)
		for j < i && pts[i].T-pts[j].T >= win {
			sum -= v(j)
			j++
		}
		out[i] = sum / float64(i-j+1)
	}
	return out
}
