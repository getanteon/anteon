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
	"sync"

	"go.ddosify.com/ddosify/v2/core/report/live"
)

// DefaultMaxPoints bounds the time series: about 1,500 points whatever the test's length.
const DefaultMaxPoints = 1500

// Point is one recorded snapshot, numbers and the stage's name only.
type Point struct {
	T           float64 // seconds from the start
	At          int64   // Unix milliseconds, 0 when unknown
	Stage       int     // 0-based stage index, -1 when the test has no stages
	StageName   string
	Warmup      bool
	TargetRPS   float64
	AchievedRPS float64
	InFlight    int64
	Iterations  int64 // started so far, the whole run
	TTFT        live.Latency
	ITL         live.Latency
	E2E         live.Latency
	Goodput     live.Goodput
	// The current stage's counts when the test has stages, else the run's
	OK, Failed, Unavailable, RateLimited int64
}

// Recorder keeps a bounded time series of snapshots for the report's charts.
// When full it keeps every other point and records half as often, so memory is fixed.
type Recorder struct {
	mu       sync.Mutex
	max      int
	stride   int
	seen     int
	points   []Point
	eventGap bool

	// The latest snapshot, kept so the series always ends where the test did.
	tail        Point
	tailPending bool

	// When each stage began and ended, from every snapshot, not only the kept ones
	spans map[int]*Span
}

// Span is when a stage ran: seconds from the start, and Unix milliseconds when known.
type Span struct {
	Start, End     float64
	StartAt, EndAt int64
}

// NewRecorder keeps at most maxPoints points; 0 uses DefaultMaxPoints.
func NewRecorder(maxPoints int) *Recorder {
	if maxPoints <= 0 {
		maxPoints = DefaultMaxPoints
	}
	maxPoints = max(maxPoints, 4) &^ 1 // even, so a halved series keeps its stride
	return &Recorder{max: maxPoints, stride: 1, points: make([]Point, 0, maxPoints)}
}

// Add records a snapshot. Safe from any goroutine; fits live.Tee.
func (r *Recorder) Add(s live.Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventGap = s.ITLIsEventGap
	stage := -1
	if s.Stage.Count > 0 {
		stage = s.Stage.Index
	}
	o := s.Outcomes
	p := Point{
		T:           s.Elapsed.Seconds(),
		Stage:       stage,
		StageName:   s.Stage.Name,
		Warmup:      s.Stage.Warmup,
		TargetRPS:   s.TargetRPS,
		AchievedRPS: s.AchievedRPS,
		InFlight:    s.InFlight,
		Iterations:  s.Iterations,
		TTFT:        s.TTFT,
		ITL:         s.ITL,
		E2E:         s.E2E,
		Goodput:     s.Goodput,
		OK:          o.OK,
		Failed:      o.Total() - o.OK - o.Interrupted,
		Unavailable: o.Unavailable,
		RateLimited: o.RateLimited,
	}
	if !s.At.IsZero() {
		p.At = s.At.UnixMilli()
	}
	if stage >= 0 {
		if r.spans == nil {
			r.spans = map[int]*Span{}
		}
		sp := r.spans[stage]
		if sp == nil {
			sp = &Span{Start: p.T, StartAt: p.At}
			r.spans[stage] = sp
		}
		sp.End, sp.EndAt = p.T, p.At
	}
	r.seen++
	if (r.seen-1)%r.stride != 0 {
		r.tail, r.tailPending = p, true
		return
	}
	r.tailPending = false
	if len(r.points) == r.max {
		half := r.points[:0]
		for i := 0; i < len(r.points); i += 2 {
			half = append(half, r.points[i])
		}
		r.points = half
		r.stride *= 2
	}
	r.points = append(r.points, p)
}

// Points returns a copy of the series.
func (r *Recorder) Points() []Point {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append(make([]Point, 0, len(r.points)+1), r.points...)
	if r.tailPending {
		out = append(out, r.tail)
	}
	return out
}

// EventGap tells whether ITL was the gap between SSE events.
func (r *Recorder) EventGap() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.eventGap
}

// Spans returns when each stage ran, by its index.
func (r *Recorder) Spans() map[int]Span {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[int]Span, len(r.spans))
	for i, s := range r.spans {
		out[i] = *s
	}
	return out
}
