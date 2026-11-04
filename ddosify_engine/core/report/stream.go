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
	"fmt"

	"go.ddosify.com/ddosify/v2/core/assertion"
	"math"
	"math/rand"
	"sort"
	"time"

	"go.ddosify.com/ddosify/v2/core/llm"
	"go.ddosify.com/ddosify/v2/core/types"
)

// Samples kept per metric for its percentiles; a uniform reservoir sample beyond this.
const maxSamples = 100000

// Outcome keys, one per step result
const (
	OutcomeOK           = "ok"
	OutcomeRateLimited  = "rate_limited_429"
	OutcomeUnavailable  = "unavailable_503"
	OutcomeServerError  = "server_error_5xx"
	OutcomeClientError  = "client_error_4xx"
	OutcomeTimeout      = "timeout"
	OutcomeNetworkError = "network_error"
	OutcomeStreamCut    = "stream_cut"
	OutcomeStreamError  = "stream_error_event"
)

// SLOSetter is implemented by the report services that compute goodput.
type SLOSetter interface {
	SetSLO(slo *types.SLO)
}

// ResultProvider is implemented by the report services that keep the aggregated result.
// Result is complete once DoneChan has delivered.
type ResultProvider interface {
	Result() *Result
}

type sampler struct {
	n       int64
	sum     float64
	min     float64
	max     float64
	samples []float64
	rnd     *rand.Rand
}

func (s *sampler) add(v float64) {
	if s.n == 0 || v < s.min {
		s.min = v
	}
	if s.n == 0 || v > s.max {
		s.max = v
	}
	s.n++
	s.sum += v
	if len(s.samples) < maxSamples {
		s.samples = append(s.samples, v)
		return
	}
	if s.rnd == nil {
		s.rnd = rand.New(rand.NewSource(1))
	}
	if j := s.rnd.Int63n(s.n); j < maxSamples {
		s.samples[int(j)] = v
	}
}

// Dist is a distribution in milliseconds, or in counts for events.
type Dist struct {
	Count int64   `json:"count"`
	Min   float64 `json:"min"`
	Mean  float64 `json:"mean"`
	P50   float64 `json:"p50"`
	P90   float64 `json:"p90"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Max   float64 `json:"max"`
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

// nearest rank
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func (s *sampler) dist() *Dist {
	if s == nil || s.n == 0 {
		return nil
	}
	sorted := make([]float64, len(s.samples))
	copy(sorted, s.samples)
	sort.Float64s(sorted)
	return &Dist{
		Count: s.n,
		Min:   round1(s.min),
		Mean:  round1(s.sum / float64(s.n)),
		P50:   round1(percentile(sorted, 0.50)),
		P90:   round1(percentile(sorted, 0.90)),
		P95:   round1(percentile(sorted, 0.95)),
		P99:   round1(percentile(sorted, 0.99)),
		Max:   round1(s.max),
	}
}

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// StreamSummary aggregates the streamed responses of a step, or of a stage.
type StreamSummary struct {
	Count           int64            `json:"count"`
	Ends            map[string]int64 `json:"ends"`
	ErrorEvents     int64            `json:"error_events"`
	NoFirstToken    int64            `json:"no_first_token"`
	TruncatedBodies int64            `json:"truncated_bodies,omitempty"`
	TTFB            *Dist            `json:"ttfb_ms,omitempty"`
	FirstEvent      *Dist            `json:"first_event_ms,omitempty"`
	FirstToken      *Dist            `json:"first_token_ms,omitempty"`
	Total           *Dist            `json:"total_ms,omitempty"`
	GapMax          *Dist            `json:"gap_max_ms,omitempty"`
	GapP50          *Dist            `json:"gap_p50_ms,omitempty"`
	GapP95          *Dist            `json:"gap_p95_ms,omitempty"`
	Events          *Dist            `json:"events,omitempty"`

	// Metrics stream observers returned, such as "output_tokens"
	Extra map[string]*Dist `json:"extra,omitempty"`
	LLM   *llm.Summary     `json:"llm,omitempty"`
}

type streamAgg struct {
	count, errorEvents, noFirstToken, truncated int64
	ends                                        map[string]int64
	ttfb, firstEvent, firstToken, total         sampler
	gapMax, gapP50, gapP95, events              sampler
	extra                                       map[string]*sampler
	llm                                         llm.Agg
}

func (a *streamAgg) add(r *types.StreamResult) {
	if a.ends == nil {
		a.ends = make(map[string]int64)
	}
	a.count++
	a.ends[r.End]++
	if r.ErrorEvents > 0 {
		a.errorEvents++
	}
	if r.BodyTruncated {
		a.truncated++
	}
	a.ttfb.add(ms(r.TTFB))
	if r.HasFirstEvent {
		a.firstEvent.add(ms(r.FirstEvent))
	}
	if r.HasFirstToken {
		a.firstToken.add(ms(r.FirstToken))
	} else {
		a.noFirstToken++
	}
	a.total.add(ms(r.Total))
	if r.Events > 1 {
		a.gapMax.add(ms(r.GapMax))
		a.gapP50.add(ms(r.GapP50))
		a.gapP95.add(ms(r.GapP95))
	}
	a.events.add(float64(r.Events))
	for k, v := range r.Extra {
		if a.extra == nil {
			a.extra = make(map[string]*sampler)
		}
		s, ok := a.extra[k]
		if !ok {
			s = &sampler{}
			a.extra[k] = s
		}
		s.add(v)
	}
	a.llm.Add(r.LLM)
}

func (a *streamAgg) summary() *StreamSummary {
	if a == nil || a.count == 0 {
		return nil
	}
	var extra map[string]*Dist
	for k, s := range a.extra {
		if extra == nil {
			extra = make(map[string]*Dist)
		}
		extra[k] = s.dist()
	}
	return &StreamSummary{
		Extra:           extra,
		Count:           a.count,
		Ends:            a.ends,
		ErrorEvents:     a.errorEvents,
		NoFirstToken:    a.noFirstToken,
		TruncatedBodies: a.truncated,
		TTFB:            a.ttfb.dist(),
		FirstEvent:      a.firstEvent.dist(),
		FirstToken:      a.firstToken.dist(),
		Total:           a.total.dist(),
		GapMax:          a.gapMax.dist(),
		GapP50:          a.gapP50.dist(),
		GapP95:          a.gapP95.dist(),
		Events:          a.events.dist(),
		LLM:             a.llm.Summary(),
	}
}

// Goodput is the share of iterations whose every step succeeded and whose every stream met the SLO.
type Goodput struct {
	SLO        types.SLO `json:"slo"`
	Iterations int64     `json:"iterations"`
	Good       int64     `json:"good"`
	Perc       float64   `json:"perc"`
}

// StageSummary is one manual_load stage.
type StageSummary struct {
	Index       int              `json:"index"`
	Name        string           `json:"name"`
	Warmup      bool             `json:"warmup"`
	Iterations  int64            `json:"iterations"`
	OK          int64            `json:"ok"`
	Good        *int64           `json:"good,omitempty"`
	GoodputPerc *float64         `json:"goodput_perc,omitempty"`
	Outcomes    map[string]int64 `json:"outcomes"`
	Stream      *StreamSummary   `json:"stream,omitempty"`
}

type stageAgg struct {
	stage      types.LoadStage
	iterations int64
	ok         int64
	good       int64
	outcomes   map[string]int64
	stream     streamAgg
}

type stepExt struct {
	outcomes map[string]int64
	stream   streamAgg
}

// extended holds what the reports add for streams, goodput and stages.
// It reports only once a stream, an SLO or a stage tag is seen, so other tests' output does not change.
type extended struct {
	slo        *types.SLO
	active     bool
	steps      map[uint16]*stepExt
	stages     map[int]*stageAgg
	iterations int64
	good       int64
	llmAll     llm.Agg
	sched      llm.ScheduleAgg
}

func newExtended() *extended {
	return &extended{steps: make(map[uint16]*stepExt), stages: make(map[int]*stageAgg)}
}

// OutcomeOf classifies one step result as one of the Outcome* keys.
func OutcomeOf(sr *types.ScenarioStepResult) string {
	if sr.Err.Type != "" {
		switch {
		case sr.Err.Reason == types.ReasonConnTimeout || sr.Err.Reason == types.ReasonReadTimeout:
			return OutcomeTimeout
		case sr.Stream != nil && sr.Stream.End == types.StreamEndTimeout:
			return OutcomeTimeout
		case sr.Stream != nil && sr.Stream.End == types.StreamEndCut:
			return OutcomeStreamCut
		default:
			return OutcomeNetworkError
		}
	}
	switch {
	case sr.StatusCode == 429:
		return OutcomeRateLimited
	case sr.StatusCode == 503:
		return OutcomeUnavailable
	case sr.StatusCode >= 500:
		return OutcomeServerError
	case sr.StatusCode >= 400:
		return OutcomeClientError
	}
	if sr.Stream != nil {
		if !sr.Stream.Clean() {
			return OutcomeStreamCut
		}
		if sr.Stream.ErrorEvents > 0 {
			return OutcomeStreamError
		}
	}
	return OutcomeOK
}

func stageOf(scr *types.ScenarioResult) (types.LoadStage, bool) {
	if scr.Others == nil {
		return types.LoadStage{}, false
	}
	st, ok := scr.Others["loadStage"].(types.LoadStage)
	return st, ok
}

func (x *extended) add(scr *types.ScenarioResult) {
	stage, staged := stageOf(scr)

	ok := true
	good := true
	for _, sr := range scr.StepResults {
		if sr.Stream != nil {
			x.active = true
		}
		outcome := OutcomeOf(sr)
		if outcome != OutcomeOK || len(sr.FailedAssertions) > 0 {
			ok = false
		}
		if x.slo != nil && sr.Stream != nil && !x.slo.Met(sr.Stream) {
			good = false
		}

		if staged {
			sa := x.stage(stage)
			sa.outcomes[outcome]++
			if sr.Stream != nil {
				sa.stream.add(sr.Stream)
			}
		}
		if staged && stage.Warmup {
			continue
		}

		se, exists := x.steps[sr.StepID]
		if !exists {
			se = &stepExt{outcomes: make(map[string]int64)}
			x.steps[sr.StepID] = se
		}
		se.outcomes[outcome]++
		if sr.Stream != nil {
			se.stream.add(sr.Stream)
			x.llmAll.Add(sr.Stream.LLM)
		}
	}
	good = good && ok

	if staged {
		x.active = true
		sa := x.stage(stage)
		sa.iterations++
		if ok {
			sa.ok++
		}
		if good {
			sa.good++
		}
		if stage.Warmup {
			return
		}
	}
	x.sched.Add(scr)
	x.iterations++
	if good {
		x.good++
	}
}

func (x *extended) stage(st types.LoadStage) *stageAgg {
	sa, ok := x.stages[st.Index]
	if !ok {
		sa = &stageAgg{stage: st, outcomes: make(map[string]int64)}
		x.stages[st.Index] = sa
	}
	return sa
}

func perc(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return round1(float64(part) * 100 / float64(whole))
}

// finalize writes the extended summaries into the result.
func (x *extended) finalize(r *Result) {
	if x == nil || !(x.active || x.slo != nil) {
		return
	}
	for id, se := range x.steps {
		step, ok := r.StepResults[id]
		if !ok {
			continue
		}
		step.Outcomes = se.outcomes
		step.Stream = se.stream.summary()
	}
	r.LLM = x.llmAll.Summary()
	r.Schedule = x.sched.Summary()
	if x.slo != nil {
		r.Goodput = &Goodput{SLO: *x.slo, Iterations: x.iterations, Good: x.good, Perc: perc(x.good, x.iterations)}
		if x.slo.GoodputPerc > 0 && r.Goodput.Perc < x.slo.GoodputPerc {
			r.TestStatus = "failed"
			r.TestFailedAssertions = append(r.TestFailedAssertions, assertion.FailedRule{
				Rule:        fmt.Sprintf("goodput_perc >= %g", x.slo.GoodputPerc),
				ReceivedMap: map[string]interface{}{"goodput_perc": r.Goodput.Perc},
			})
		}
	}
	if len(x.stages) > 0 {
		idx := make([]int, 0, len(x.stages))
		for i := range x.stages {
			idx = append(idx, i)
		}
		sort.Ints(idx)
		r.Stages = make([]*StageSummary, 0, len(idx))
		for _, i := range idx {
			sa := x.stages[i]
			ss := &StageSummary{
				Index:      sa.stage.Index,
				Name:       sa.stage.Name,
				Warmup:     sa.stage.Warmup,
				Iterations: sa.iterations,
				OK:         sa.ok,
				Outcomes:   sa.outcomes,
				Stream:     sa.stream.summary(),
			}
			if x.slo != nil {
				g := sa.good
				p := perc(sa.good, sa.iterations)
				ss.Good, ss.GoodputPerc = &g, &p
			}
			r.Stages = append(r.Stages, ss)
		}
	}
}
