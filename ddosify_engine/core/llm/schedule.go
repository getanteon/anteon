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

package llm

import (
	"fmt"
	"io"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

// Ddosify starts iterations on a schedule whatever the earlier ones are doing, an open model. When the
// client itself is saturated (CPU, GC, a full connection pool) the first request goes out late, and the
// latency the server is blamed for leaves out the wait: coordinated omission. The lag is the first
// request's send time minus the time the schedule meant it to start.

// LateThreshold is the p99 lag above which the report warns that the client could not keep the schedule.
const LateThreshold = 100 * time.Millisecond

// ScheduleSummary is the scheduling lag of the iterations, in milliseconds.
type ScheduleSummary struct {
	Iterations int64 `json:"iterations"`
	Lag        *Dist `json:"lag_ms"`
	Late       int64 `json:"late_over_100ms"`
	Saturated  bool  `json:"client_saturated"`
}

// ScheduleAgg aggregates the lag. Not safe for concurrent use.
type ScheduleAgg struct {
	lag  sampler
	late int64
}

// Add takes one iteration; one with no intended start or no request is ignored.
func (a *ScheduleAgg) Add(scr *types.ScenarioResult) {
	if scr == nil || scr.Others == nil || len(scr.StepResults) == 0 {
		return
	}
	intended, ok := scr.Others[types.OthersIntendedStart].(time.Time)
	if !ok || intended.IsZero() {
		return
	}
	sent := scr.StepResults[0].RequestTime
	if sent.IsZero() {
		return
	}
	lag := sent.Sub(intended)
	if lag < 0 {
		lag = 0
	}
	if lag > LateThreshold {
		a.late++
	}
	a.lag.add(ms(lag))
}

// Summary is nil when nothing was added.
func (a *ScheduleAgg) Summary() *ScheduleSummary {
	if a == nil || a.lag.n == 0 {
		return nil
	}
	d := a.lag.dist(2)
	return &ScheduleSummary{Iterations: a.lag.n, Lag: d, Late: a.late, Saturated: d.P99 > ms(LateThreshold)}
}

// WriteSchedule prints the lag, with a warning when the client fell behind.
func WriteSchedule(w io.Writer, s *ScheduleSummary) {
	if s == nil {
		return
	}
	fmt.Fprintf(w, "Scheduling lag (ms):\tp50 %.2f, p99 %.2f, max %.2f over %d iterations; %d started over 100ms late\n",
		s.Lag.P50, s.Lag.P99, s.Lag.Max, s.Iterations, s.Late)
	if s.Saturated {
		fmt.Fprintf(w, "WARNING: the client could not keep the schedule (p99 lag %.0fms): the arrival rate was lower than asked and latencies leave the wait out. Use a bigger machine or fewer iterations.\n", s.Lag.P99)
	}
}
