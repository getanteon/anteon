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

// The engine's observers, on core/live.

package live

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	elive "go.ddosify.com/ddosify/v2/core/live"
)

func init() {
	elive.Register("tui", newTUIObserver)
	elive.Register("ndjson", newNDJSONObserver)
}

func pc(x elive.Percentiles) Latency {
	return Latency{Count: x.Count, P50: x.P50, P90: x.P90, P95: x.P95, P99: x.P99}
}

func scopeOf(m elive.Metrics) *Scope {
	sc := &Scope{TTFT: pc(m.TTFT), ITL: pc(m.ITL), E2E: pc(m.E2E), Outcomes: OutcomesFromMap(m.Outcomes)}
	if m.Goodput != nil {
		sc.Goodput = Goodput{Good: m.Goodput.Good, Total: m.Goodput.Iterations}
	}
	if t := m.Tokens; t != nil {
		sc.TPOT = pc(t.TPOT)
		if t.ITL.Count > 0 {
			sc.ITL = pc(t.ITL)
		}
	}
	return sc
}

// FromEngine converts the engine's snapshot. With stages, latency, goodput and outcomes are the
// current stage's, so each step of a stepped load reads on its own, and All holds the run's.
func FromEngine(p elive.Plan, s elive.Snapshot) Snapshot {
	m := s.Current
	out := Snapshot{
		At:            s.At,
		Elapsed:       s.Elapsed,
		Duration:      p.Duration,
		TargetRPS:     s.TargetRPS,
		AchievedRPS:   s.AchievedRPS,
		InFlight:      s.InFlight,
		Iterations:    s.Started,
		TTFT:          pc(m.TTFT),
		ITL:           pc(m.ITL),
		E2E:           pc(m.E2E),
		ITLIsEventGap: true,
		Outcomes:      OutcomesFromMap(m.Outcomes),
	}
	if m.Goodput != nil {
		out.Goodput = Goodput{Good: m.Goodput.Good, Total: m.Goodput.Iterations}
	}
	out.SchedulingLag = pc(m.SchedulingLag)
	if t := m.Tokens; t != nil {
		out.Tokens = &Tokens{Unit: t.Unit, Output: pc(t.Output), TPOT: pc(t.TPOT), ITL: pc(t.ITL), PerSec: t.PerSec}
		if t.ITL.Count > 0 {
			out.ITL, out.ITLIsEventGap = pc(t.ITL), t.Unit != "tokens"
		}
	}
	out.PrefixCache = PrefixCache(p)
	for _, st := range p.Stages {
		out.Stages = append(out.Stages, StagePlan{Name: st.Name, Warmup: st.Warmup, Duration: st.End - st.Start, Count: int64(st.Iterations)})
	}
	if len(p.Stages) > 0 {
		out.All = scopeOf(s.Total)
	}
	if s.Stage.Index >= 0 && len(p.Stages) > 0 {
		out.Stage = StageRef{
			Index:    s.Stage.Index,
			Count:    len(p.Stages),
			Name:     s.Stage.Name,
			Warmup:   s.Stage.Warmup,
			Elapsed:  max(0, s.Elapsed-s.Stage.Start),
			Duration: s.Stage.End - s.Stage.Start,
		}
	}
	return out
}

func goodputTarget(p elive.Plan) float64 {
	if p.SLO == nil {
		return 0
	}
	return p.SLO.GoodputPerc
}

// PrefixCache describes what the llm steps do to the prompt's prefix cache.
func PrefixCache(p elive.Plan) string {
	for _, s := range p.Steps {
		switch {
		case s.CacheBust:
			return "busted"
		case s.SharedPrefixTokens > 0:
			return fmt.Sprintf("shared %d tokens", s.SharedPrefixTokens)
		}
	}
	return ""
}

type tuiObserver struct {
	o    elive.Options
	plan elive.Plan
	ch   chan Snapshot
	done chan struct{}
}

func newTUIObserver(o elive.Options) (elive.Observer, error) {
	return &tuiObserver{o: o}, nil
}

func (t *tuiObserver) OwnsTerminal() bool { return true }

func (t *tuiObserver) Start(p elive.Plan) {
	t.plan = p
	t.ch = make(chan Snapshot, 1)
	t.done = make(chan struct{})
	go func() {
		defer close(t.done)
		_, _ = RunTUI(context.Background(), t.ch, Options{
			Out:           t.o.Stdout,
			In:            os.Stdin,
			NoColor:       !t.o.Color,
			Stop:          t.o.Stop,
			Thresholds:    ThresholdsFromSLO(p.SLO),
			GoodputTarget: goodputTarget(p),
		})
	}()
}

// Update never blocks: a snapshot the view has not taken yet is replaced by the newer one.
func (t *tuiObserver) Update(s elive.Snapshot) {
	snap := FromEngine(t.plan, s)
	select {
	case t.ch <- snap:
		return
	default:
	}
	select {
	case <-t.ch:
	default:
	}
	select {
	case t.ch <- snap:
	default:
	}
}

func (t *tuiObserver) Finish(f elive.Final) {
	if t.ch == nil {
		return
	}
	t.Update(f.Last)
	close(t.ch)
	<-t.done
}

type ndjsonObserver struct {
	o     elive.Options
	plan  elive.Plan
	mu    sync.Mutex
	last  time.Time
	every time.Duration
}

func newNDJSONObserver(o elive.Options) (elive.Observer, error) {
	every := DefaultJSONEvery
	if o.Arg != "" {
		d, err := time.ParseDuration(o.Arg)
		if err != nil {
			return nil, err
		}
		every = d
	}
	return &ndjsonObserver{o: o, every: every}, nil
}

func (n *ndjsonObserver) Start(p elive.Plan) { n.plan = p }

func (n *ndjsonObserver) Update(s elive.Snapshot) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.last.IsZero() && time.Since(n.last) < n.every {
		return
	}
	n.last = time.Now()
	_ = WriteNDJSON(n.o.Stderr, FromEngine(n.plan, s))
}

func (n *ndjsonObserver) Finish(f elive.Final) {
	n.mu.Lock()
	defer n.mu.Unlock()
	_ = WriteNDJSON(n.o.Stderr, FromEngine(n.plan, f.Last))
}
