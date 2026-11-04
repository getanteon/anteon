//go:build v2live

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

// The engine's observers, for v2/foundation's core/live. Built with -tags v2live until that branch merges;
// then the tag goes and the import paths take /v2.

package live

import (
	"context"
	"os"
	"sync"
	"time"

	elive "go.ddosify.com/ddosify/v2/core/live"
)

func init() {
	elive.Register("tui", newTUIObserver)
	elive.Register("ndjson", newNDJSONObserver)
}

// FromEngine converts the engine's snapshot. Latency, goodput and outcomes are the current stage's,
// so each step of a stepped load reads on its own; ITL is the gap between SSE events.
func FromEngine(p elive.Plan, s elive.Snapshot) Snapshot {
	m := s.Current
	if m.Iterations == 0 && s.Total.Iterations > 0 {
		m = s.Total
	}
	pc := func(x elive.Percentiles) Latency {
		return Latency{Count: x.Count, P50: x.P50, P90: x.P90, P95: x.P95, P99: x.P99}
	}
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
	for _, st := range p.Stages {
		out.Stages = append(out.Stages, StagePlan{Name: st.Name, Warmup: st.Warmup, Duration: st.End - st.Start, Count: int64(st.Iterations)})
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

// interruptSelf stops the test the way Ctrl-C outside the TUI does: the CLI's signal context ends the engine.
func interruptSelf() {
	if p, err := os.FindProcess(os.Getpid()); err == nil {
		_ = p.Signal(os.Interrupt)
	}
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
			Out:     t.o.Stdout,
			In:      os.Stdin,
			NoColor: !t.o.Color,
			Stop:    interruptSelf,
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
