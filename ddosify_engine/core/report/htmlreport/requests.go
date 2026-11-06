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
	"math/rand/v2"
	"slices"
	"sort"

	"go.ddosify.com/ddosify/v2/core/report"
)

// PerStepSamples bounds the request records kept for one step. Below it every record is kept and
// the figures are exact; above it a seeded reservoir keeps a uniform sample, so memory is fixed.
const (
	PerStepSamples = 25_000
	maxSteps       = 16

	maxSeriesPoints = 2_000
)

// Record is one sent step, as the request log writes it. Times are milliseconds from the run's start.
type Record struct {
	Stage    int // index into the stage list, -1 when unknown
	Warmup   bool
	StepID   uint16
	StepName string
	Intended float64
	Sent     float64
	Done     float64
	Outcome  string
	FT       float64 // first token, NaN when the answer had none
	Total    float64
	HasFT    bool
}

type stepSample struct {
	id   uint16
	name string
	seen int64
	keep []Record
	ft   bool // some answer of this step carried a first token
}

// Requests keeps a bounded sample of request records per step.
type Requests struct {
	steps map[uint16]*stepSample
	rng   *rand.Rand
}

func NewRequests() *Requests {
	return &Requests{steps: map[uint16]*stepSample{}, rng: rand.New(rand.NewPCG(1, 2))}
}

// Add keeps r, or replaces a kept record with it at the reservoir's rate.
func (q *Requests) Add(r Record) {
	s := q.steps[r.StepID]
	if s == nil {
		if len(q.steps) >= maxSteps {
			return
		}
		s = &stepSample{id: r.StepID, name: r.StepName}
		q.steps[r.StepID] = s
	}
	s.seen++
	s.ft = s.ft || r.HasFT
	if len(s.keep) < PerStepSamples {
		s.keep = append(s.keep, r)
		return
	}
	if j := q.rng.Int64N(s.seen); j < PerStepSamples {
		s.keep[j] = r
	}
}

// Len is the number of records kept.
func (q *Requests) Len() int {
	n := 0
	for _, s := range q.steps {
		n += len(s.keep)
	}
	return n
}

// Dist is a sample's percentiles in milliseconds, with distribution-free 95% intervals.
type Dist struct {
	N     int       `json:"n"`
	Min   *float64  `json:"min,omitempty"`
	Max   *float64  `json:"max,omitempty"`
	P50   *float64  `json:"p50,omitempty"`
	P90   *float64  `json:"p90,omitempty"`
	P95   *float64  `json:"p95,omitempty"`
	P99   *float64  `json:"p99,omitempty"` // from 100 values
	P50CI []float64 `json:"p50ci,omitempty"`
	P95CI []float64 `json:"p95ci,omitempty"`
}

func quantile(sorted []float64, q float64) float64 {
	k := float64(len(sorted)-1) * q
	f := int(k)
	c := min(f+1, len(sorted)-1)
	return sorted[f] + (sorted[c]-sorted[f])*(k-float64(f))
}

// quantileCI is the order-statistic interval of a quantile, by the normal approximation to the binomial.
func quantileCI(sorted []float64, q float64) []float64 {
	n := float64(len(sorted))
	if n < 5 {
		return nil
	}
	sd := math.Sqrt(n * q * (1 - q))
	lo := max(0, int(n*q-1.96*sd))
	hi := min(len(sorted)-1, int(n*q+1.96*sd+1))
	return []float64{sorted[lo], sorted[hi]}
}

func pf(v float64) *float64 { return &v }

func distOf(vals []float64) Dist {
	if len(vals) == 0 {
		return Dist{}
	}
	v := slices.Clone(vals)
	sort.Float64s(v)
	d := Dist{N: len(v), Min: pf(v[0]), Max: pf(v[len(v)-1]),
		P50: pf(quantile(v, .5)), P90: pf(quantile(v, .9)), P95: pf(quantile(v, .95)),
		P50CI: quantileCI(v, .5), P95CI: quantileCI(v, .95)}
	if len(v) >= 100 {
		d.P99 = pf(quantile(v, .99))
	}
	return d
}

// StageWindow is when a stage ran, in milliseconds from the start.
type StageWindow struct {
	Name   string
	Warmup bool
	Start  float64
	End    float64
	Known  bool
}

// SteadyStage is a stage's figures once the system settled at its load.
type SteadyStage struct {
	Name     string     `json:"name"`
	Warmup   bool       `json:"warmup"`
	SettleMs float64    `json:"settleMs"`
	All      int        `json:"all"`
	Steady   int        `json:"steady"`
	Answered int        `json:"answered"`
	Good     int        `json:"good"`
	Rejected int        `json:"rejected"`
	Failed   int        `json:"failed"`
	NoText   int        `json:"noText"`
	FT       Dist       `json:"ft"`
	Total    Dist       `json:"total"`
	Steps    []StepDist `json:"steps"`
}

// StepDist is one step's latency within a stage.
type StepDist struct {
	ID    uint16 `json:"id"`
	Name  string `json:"name"`
	FT    Dist   `json:"ft"`
	Total Dist   `json:"total"`
}

// ReqSeries is latency by start time: every quarter window, the requests started over the last
// max(window, the time 30 requests took to start), at most 10 minutes. A point needs 20 answers.
type ReqSeries struct {
	T      []float64  `json:"t"` // ms from the start
	FTP50  []*float64 `json:"ftP50"`
	FTP95  []*float64 `json:"ftP95"`
	TotP50 []*float64 `json:"totP50"`
	TotP95 []*float64 `json:"totP95"`
	N      []int      `json:"n"`
	Win    []float64  `json:"win"`
}

// ReqData is what the request records add to the report.
type ReqData struct {
	Count   int64         `json:"count"`
	Kept    int           `json:"kept"`
	Sampled bool          `json:"sampled"`
	Steps   []string      `json:"steps"` // the measured steps
	Stages  []SteadyStage `json:"stages"`
	Series  ReqSeries     `json:"series"`
}

// Analyze measures the steps that stream an answer when any does, else every step.
// sloFT and sloTotal are milliseconds, 0 when not set; window is the smoothing window in
// milliseconds, and also the least a stage's start is left out for.
func (q *Requests) Analyze(stages []StageWindow, sloFT, sloTotal, window float64) *ReqData {
	var measured []*stepSample
	anyFT := false
	for _, s := range q.steps {
		anyFT = anyFT || s.ft
	}
	for _, s := range q.steps {
		if !anyFT || s.ft {
			measured = append(measured, s)
		}
	}
	if len(measured) == 0 {
		return nil
	}
	slices.SortFunc(measured, func(a, b *stepSample) int { return int(a.id) - int(b.id) })
	out := &ReqData{}
	var reqs []Record
	weight := map[uint16]float64{} // records each kept one stands for
	for _, s := range measured {
		out.Count += s.seen
		out.Steps = append(out.Steps, s.name)
		reqs = append(reqs, s.keep...)
		weight[s.id] = float64(s.seen) / float64(len(s.keep))
		if s.seen > int64(len(s.keep)) {
			out.Sampled = true
		}
	}
	out.Kept = len(reqs)
	if sloFT <= 0 {
		sloFT = math.Inf(1)
	}
	if sloTotal <= 0 {
		sloTotal = math.Inf(1)
	}
	ok := func(r Record) bool { return r.Outcome == report.OutcomeOK }
	good := func(r Record) bool {
		if !ok(r) || r.Total > sloTotal {
			return false
		}
		if s := q.steps[r.StepID]; s.ft {
			return r.HasFT && r.FT <= sloFT
		}
		return true
	}

	prevP95 := math.NaN()
	for i, st := range stages {
		var rs []Record
		for _, r := range reqs {
			if r.Stage == i {
				rs = append(rs, r)
			}
		}
		settle := window
		if !math.IsNaN(prevP95) {
			settle = max(settle, prevP95)
		}
		b := SteadyStage{Name: st.Name, Warmup: st.Warmup, SettleMs: settle, All: len(rs)}
		var ft, tot []float64
		perStep := map[uint16]*[2][]float64{}
		for _, r := range rs {
			if st.Known && r.Intended < st.Start+settle {
				continue
			}
			b.Steady++
			if good(r) {
				b.Good++
			}
			switch {
			case r.Outcome == report.OutcomeUnavailable:
				b.Rejected++
			case !ok(r):
				b.Failed++
			}
			if !ok(r) {
				continue
			}
			b.Answered++
			p := perStep[r.StepID]
			if p == nil {
				p = &[2][]float64{}
				perStep[r.StepID] = p
			}
			if r.HasFT {
				ft = append(ft, r.FT)
				p[0] = append(p[0], r.FT)
			} else if q.steps[r.StepID].ft {
				b.NoText++
			}
			tot = append(tot, r.Total)
			p[1] = append(p[1], r.Total)
		}
		b.FT, b.Total = distOf(ft), distOf(tot)
		for _, s := range measured {
			if p := perStep[s.id]; p != nil {
				b.Steps = append(b.Steps, StepDist{ID: s.id, Name: s.name, FT: distOf(p[0]), Total: distOf(p[1])})
			}
		}
		out.Stages = append(out.Stages, b)
		var all []float64
		for _, r := range rs {
			if ok(r) {
				all = append(all, r.Total)
			}
		}
		if len(all) > 0 {
			sort.Float64s(all)
			prevP95 = quantile(all, .95)
		}
	}

	slices.SortFunc(reqs, func(a, b Record) int {
		switch {
		case a.Sent < b.Sent:
			return -1
		case a.Sent > b.Sent:
			return 1
		}
		return 0
	})
	t0, t1 := reqs[0].Sent, reqs[0].Done
	for _, r := range reqs {
		t1 = max(t1, r.Done)
	}
	sent := make([]float64, len(reqs))
	for i, r := range reqs {
		sent[i] = r.Sent
	}
	// Index of the first request sent after t, so (a, b] windows are two binary searches.
	after := func(t float64) int { return sort.Search(len(sent), func(i int) bool { return sent[i] > t }) }
	s := &out.Series
	var ftv, tov []float64
	// A quarter window apart, and at most maxSeriesPoints over a long run.
	step := max(window/4, (t1-t0)/maxSeriesPoints)
	for t := t0; t <= t1; t += step {
		lo, hi := after(t-window), after(t)
		rate := 0.0 // per window, the records each kept one stands for counted back in
		for _, r := range reqs[lo:hi] {
			rate += weight[r.StepID]
		}
		win := min(600_000, max(window, math.Floor(30/max(rate, 0.1)*window)))
		ftv, tov = ftv[:0], tov[:0]
		for _, r := range reqs[after(t-win):hi] {
			if !ok(r) {
				continue
			}
			if r.HasFT {
				ftv = append(ftv, r.FT)
			}
			tov = append(tov, r.Total)
		}
		s.T = append(s.T, t)
		if anyFT {
			s.N = append(s.N, len(ftv))
		} else {
			s.N = append(s.N, len(tov))
		}
		s.Win = append(s.Win, win)
		p50, p95 := pctPair(ftv)
		s.FTP50, s.FTP95 = append(s.FTP50, p50), append(s.FTP95, p95)
		p50, p95 = pctPair(tov)
		s.TotP50, s.TotP95 = append(s.TotP50, p50), append(s.TotP95, p95)
	}
	return out
}

// pctPair is p50 and p95 of v, or nils under 20 values.
func pctPair(v []float64) (*float64, *float64) {
	if len(v) < 20 {
		return nil, nil
	}
	sort.Float64s(v)
	return pf(quantile(v, .5)), pf(quantile(v, .95))
}
