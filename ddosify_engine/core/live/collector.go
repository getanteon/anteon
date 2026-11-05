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
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/types"
)

// Samples kept per live metric; a uniform reservoir beyond. The final report keeps more.
const liveSamples = 20000

const tick = 100 * time.Millisecond

type reservoir struct {
	n       int64
	sum     float64
	max     float64
	samples []float64
	rnd     *rand.Rand
}

func (r *reservoir) add(v float64) {
	r.n++
	r.sum += v
	if v > r.max {
		r.max = v
	}
	if len(r.samples) < liveSamples {
		r.samples = append(r.samples, v)
		return
	}
	if r.rnd == nil {
		r.rnd = rand.New(rand.NewSource(1))
	}
	if j := r.rnd.Int63n(r.n); j < liveSamples {
		r.samples[j] = v
	}
}

func rank(sorted []float64, p float64) float64 {
	i := int(math.Ceil(p*float64(len(sorted)))) - 1
	if i < 0 {
		i = 0
	}
	return sorted[i]
}

func (r *reservoir) percentiles() Percentiles {
	if r == nil || r.n == 0 {
		return Percentiles{}
	}
	s := make([]float64, len(r.samples))
	copy(s, r.samples)
	sort.Float64s(s)
	return Percentiles{Count: r.n, Mean: r.sum / float64(r.n), P50: rank(s, .5), P90: rank(s, .9), P95: rank(s, .95), P99: rank(s, .99), Max: r.max}
}

type agg struct {
	iterations, good int64
	outcomes         map[string]int64
	codes            map[int]int64
	ttfb, ttft, itl  reservoir
	e2e              reservoir
	extra            map[string]*reservoir
	lag              reservoir
	tok, chunk       outputAgg
	itl2             reservoir
	cacheBusted      int64
	prefixTokens     int
	answers          int64
}

// outputAgg is the answers counted in one unit.
type outputAgg struct {
	n                 int64
	total             int64
	output, tpot      reservoir
	firstSent, lastAt time.Time
}

func (o *outputAgg) add(r *types.LLMResult) {
	o.n++
	o.total += int64(r.Output)
	o.output.add(float64(r.Output))
	if r.HasTPOT {
		o.tpot.add(ms(r.TPOT))
	}
	if !r.SentAt.IsZero() {
		if o.firstSent.IsZero() || r.SentAt.Before(o.firstSent) {
			o.firstSent = r.SentAt
		}
		if end := r.SentAt.Add(r.LastContent); end.After(o.lastAt) {
			o.lastAt = end
		}
	}
}

func (o *outputAgg) perSec() float64 {
	if w := o.lastAt.Sub(o.firstSent).Seconds(); w > 0 {
		return float64(o.total) / w
	}
	return 0
}

func (a *agg) addLLM(r *types.LLMResult) {
	a.answers++
	if r.Tokens() {
		a.tok.add(r)
	} else {
		a.chunk.add(r)
	}
	for _, g := range r.ITL {
		a.itl2.add(ms(g))
	}
	if r.CacheBusted {
		a.cacheBusted++
	}
	if r.PrefixTokens > a.prefixTokens {
		a.prefixTokens = r.PrefixTokens
	}
}

// tokens reports in tokens when any answer was counted in tokens, else in chunks.
func (a *agg) tokens() *Tokens {
	if a.answers == 0 {
		return nil
	}
	t := &Tokens{Answers: a.answers, ITL: a.itl2.percentiles(), CacheBusted: a.cacheBusted, PrefixTokens: a.prefixTokens}
	if a.tok.n > 0 {
		t.Unit, t.Output, t.TPOT, t.PerSec = "tokens", a.tok.output.percentiles(), a.tok.tpot.percentiles(), a.tok.perSec()
	} else {
		t.Unit, t.Output, t.PerSec = "chunks", a.chunk.output.percentiles(), a.chunk.perSec()
	}
	return t
}

func newAgg() *agg {
	return &agg{outcomes: map[string]int64{}, codes: map[int]int64{}, extra: map[string]*reservoir{}}
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// addInterrupted counts an iteration cut in flight: outcomes only, no latency, not in goodput.
func (a *agg) addInterrupted(res *types.ScenarioResult) {
	for _, sr := range res.StepResults {
		a.outcomes[report.OutcomeOf(sr)]++
		a.codes[sr.StatusCode]++
	}
}

func (a *agg) add(res *types.ScenarioResult, e2e time.Duration, slo *types.SLO) {
	a.iterations++
	good, ok := true, true
	for _, sr := range res.StepResults {
		o := report.OutcomeOf(sr)
		a.outcomes[o]++
		a.codes[sr.StatusCode]++
		if o != report.OutcomeOK {
			ok = false
		}
		if o != report.OutcomeOK || len(sr.FailedAssertions) > 0 {
			good = false
		}
		st := sr.Stream
		if st == nil {
			continue
		}
		if slo != nil && !slo.Met(st) {
			good = false
		}
		if !st.Clean() || st.ErrorEvents > 0 {
			continue // its outcome counts it; latencies are of successful streams
		}
		if st.LLM != nil {
			a.addLLM(st.LLM)
		}
		a.ttfb.add(ms(st.TTFB))
		if st.HasFirstToken {
			a.ttft.add(ms(st.FirstToken))
		}
		if st.Events > 1 {
			a.itl.add(ms(st.GapP50))
		}
		for k, v := range st.Extra {
			r, ok := a.extra[k]
			if !ok {
				r = &reservoir{}
				a.extra[k] = r
			}
			r.add(v)
		}
	}
	// Like the stream latencies, E2E is of iterations that succeeded: a 429 answered in 2 ms is an
	// outcome, and as a latency it would pull a stage's percentiles toward zero.
	if ok {
		a.e2e.add(ms(e2e))
	}
	if intended, ok := res.Others[types.OthersIntendedStart].(time.Time); ok && !intended.IsZero() && len(res.StepResults) > 0 {
		if sent := res.StepResults[0].RequestTime; !sent.IsZero() {
			lag := sent.Sub(intended)
			if lag < 0 {
				lag = 0
			}
			a.lag.add(ms(lag))
		}
	}
	if good {
		a.good++
	}
}

func copyMap[K comparable](m map[K]int64) map[K]int64 {
	out := make(map[K]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (a *agg) metrics(withGoodput bool) Metrics {
	m := Metrics{Iterations: a.iterations, Outcomes: copyMap(a.outcomes), StatusCodes: copyMap(a.codes),
		TTFB: a.ttfb.percentiles(), TTFT: a.ttft.percentiles(), ITL: a.itl.percentiles(), E2E: a.e2e.percentiles(),
		Tokens: a.tokens(), SchedulingLag: a.lag.percentiles()}
	if withGoodput {
		g := &Goodput{Good: a.good, Iterations: a.iterations}
		if a.iterations > 0 {
			g.Perc = float64(a.good) * 100 / float64(a.iterations)
		}
		m.Goodput = g
	}
	if len(a.extra) > 0 {
		m.Extra = make(map[string]Percentiles, len(a.extra))
		for k, r := range a.extra {
			m.Extra[k] = r.percentiles()
		}
	}
	return m
}

// Collector aggregates what the engine reports and publishes it to observers. Safe for concurrent use.
type Collector struct {
	plan      Plan
	slo       *types.SLO
	observers []Observer

	mu        sync.Mutex
	begin     time.Time
	started   int64
	completed int64
	starts    [10]int64 // per tick, for the last second
	lastTick  int64
	stages    map[int]*agg
	total     *agg
	seq       int64

	stop chan struct{}
	done chan struct{}
	once sync.Once
}

func NewCollector(plan Plan, slo *types.SLO, observers []Observer) *Collector {
	return &Collector{plan: plan, slo: slo, observers: observers, stages: map[int]*agg{}, total: newAgg(),
		stop: make(chan struct{}), done: make(chan struct{})}
}

// Start calls every observer's Start and begins the snapshots.
func (c *Collector) Start() {
	c.mu.Lock()
	c.begin = time.Now()
	c.mu.Unlock()
	for _, o := range c.observers {
		o.Start(c.plan)
	}
	go c.loop()
}

func (c *Collector) loop() {
	defer close(c.done)
	t := time.NewTicker(Interval)
	defer t.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-t.C:
			s := c.Snapshot()
			for _, o := range c.observers {
				o.Update(s)
			}
		}
	}
}

// advance moves the per-tick start counters to now; the caller holds mu.
func (c *Collector) advance(now time.Time) int64 {
	t := int64(now.Sub(c.begin) / tick)
	for c.lastTick < t {
		c.lastTick++
		c.starts[c.lastTick%10] = 0
	}
	return t
}

func (c *Collector) IterationStarted() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.started++
	t := c.advance(time.Now())
	c.starts[t%10]++
}

// IterationDone records one iteration. stage is the stage it started in, nil without stages.
func (c *Collector) IterationDone(res *types.ScenarioResult, stage *types.LoadStage, e2e time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.completed++
	if res == nil { // the requesters could not be built
		return
	}
	if res.Interrupted() {
		if stage != nil {
			c.stageAgg(stage.Index).addInterrupted(res)
		}
		if stage == nil || !stage.Warmup {
			c.total.addInterrupted(res)
		}
		return
	}
	if stage != nil {
		a := c.stageAgg(stage.Index)
		a.add(res, e2e, c.slo)
		if stage.Warmup {
			return
		}
	}
	c.total.add(res, e2e, c.slo)
}

func (c *Collector) stageAgg(index int) *agg {
	a, ok := c.stages[index]
	if !ok {
		a = newAgg()
		c.stages[index] = a
	}
	return a
}

func (c *Collector) stageAt(elapsed time.Duration) Stage {
	for _, s := range c.plan.Stages {
		if elapsed >= s.Start && elapsed < s.End {
			return s
		}
	}
	if n := len(c.plan.Stages); n > 0 && elapsed >= c.plan.Stages[n-1].End {
		return c.plan.Stages[n-1]
	}
	return Stage{Index: -1, End: c.plan.Duration, Iterations: c.plan.Iterations}
}

// Snapshot returns the state now.
func (c *Collector) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	t := c.advance(now)
	elapsed := now.Sub(c.begin)
	c.seq++

	var achieved int64
	for _, v := range c.starts {
		achieved += v
	}
	var target int
	for i := t - 9; i <= t; i++ {
		if i >= 0 && int(i) < len(c.plan.Schedule) {
			target += c.plan.Schedule[i]
		}
	}

	st := c.stageAt(elapsed)
	withGoodput := c.slo != nil
	s := Snapshot{Seq: c.seq, At: now, Elapsed: elapsed, Stage: st, TargetRPS: float64(target),
		AchievedRPS: float64(achieved), Started: c.started, Completed: c.completed,
		InFlight: c.started - c.completed, Total: c.total.metrics(withGoodput)}
	if a, ok := c.stages[st.Index]; ok && st.Index >= 0 {
		s.Current = a.metrics(withGoodput)
	} else if st.Index < 0 {
		s.Current = s.Total
	} else {
		s.Current = newAgg().metrics(withGoodput)
	}
	return s
}

// Finish stops the snapshots and hands every observer the final report, once.
func (c *Collector) Finish(status string, passed bool, result *report.Result) {
	c.once.Do(func() {
		close(c.stop)
		<-c.done
		last := c.Snapshot()
		f := Final{Status: status, Elapsed: last.Elapsed, Passed: passed, Result: result, Last: last}
		for _, o := range c.observers {
			o.Finish(f)
		}
	})
}
