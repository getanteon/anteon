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
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	elive "go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/report"
)

func init() {
	elive.Register("status", newStatusObserver)
}

// DefaultStatusEvery is how often the status observer writes a line while a test runs.
const DefaultStatusEvery = 10 * time.Second

// statusObserver writes what a run is doing as plain lines for a person: when it starts, every
// DefaultStatusEvery, and how it ended. It is the live view of --json, whose data goes to stdout,
// so a container's log says what is happening without anybody opening the NDJSON.
type statusObserver struct {
	w     io.Writer
	plan  elive.Plan
	every time.Duration
	mu    sync.Mutex
	last  time.Time
}

func newStatusObserver(o elive.Options) (elive.Observer, error) {
	every := DefaultStatusEvery
	if o.Arg != "" {
		d, err := time.ParseDuration(o.Arg)
		if err != nil {
			return nil, err
		}
		every = d
	}
	return &statusObserver{w: o.Stderr, every: every}, nil
}

func (s *statusObserver) Start(p elive.Plan) {
	s.plan = p
	s.last = time.Now()
	var b strings.Builder
	fmt.Fprintf(&b, "Started: %d iterations over %s", p.Iterations, clockOf(p.Duration))
	if len(p.Stages) > 1 || (len(p.Stages) == 1 && p.Stages[0].Name != "") {
		var st []string
		for _, x := range p.Stages {
			name := x.Name
			if name == "" {
				name = fmt.Sprintf("S%d", x.Index+1)
			}
			st = append(st, fmt.Sprintf("%s %d in %s", name, x.Iterations, clockOf(x.End-x.Start)))
		}
		fmt.Fprintf(&b, "; stages: %s", strings.Join(st, ", "))
	}
	var steps []string
	for _, x := range p.Steps {
		if x.Name == "" {
			steps = append(steps, fmt.Sprintf("%d", x.ID))
		} else {
			steps = append(steps, fmt.Sprintf("%d %q", x.ID, x.Name))
		}
	}
	if len(steps) > 0 {
		fmt.Fprintf(&b, "; steps: %s", strings.Join(steps, ", "))
	}
	fmt.Fprintln(s.w, b.String())
}

func (s *statusObserver) Update(snap elive.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.last) < s.every {
		return
	}
	s.last = time.Now()
	fmt.Fprintln(s.w, statusLine(s.plan, snap))
}

func (s *statusObserver) Finish(f elive.Final) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := f.Last.Total
	ok := t.Outcomes[report.OutcomeOK]
	var failed int64
	for k, v := range t.Outcomes {
		if k != report.OutcomeOK {
			failed += v
		}
	}
	verdict := "the test passed"
	if !f.Passed {
		verdict = "the test failed"
		if ok == 0 && failed > 0 {
			verdict += ": no request succeeded; the first error of each kind is above"
		}
	}
	if f.Status != "" && f.Status != "done" {
		verdict += " (" + f.Status + ")"
	}
	fmt.Fprintf(s.w, "Finished in %s: %d iterations started, requests ok %d, failed %d%s; %s\n",
		clockOf(f.Elapsed), f.Last.Started, ok, failed, failedBy(t.Outcomes), verdict)
}

func statusLine(p elive.Plan, snap elive.Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s", clockOf(snap.Elapsed))
	if p.Duration > 0 {
		fmt.Fprintf(&b, "/%s", clockOf(p.Duration))
	}
	if snap.Stage.Index >= 0 && len(p.Stages) > 1 && snap.Stage.Name != "" {
		fmt.Fprintf(&b, " %s", snap.Stage.Name)
	}
	t := snap.Total
	ok := t.Outcomes[report.OutcomeOK]
	var failed int64
	for k, v := range t.Outcomes {
		if k != report.OutcomeOK {
			failed += v
		}
	}
	fmt.Fprintf(&b, " · started %d · in flight %d · ok %d · failed %d%s", snap.Started, snap.InFlight, ok, failed, failedBy(t.Outcomes))
	if x := t.TTFT; x.Count > 0 {
		fmt.Fprintf(&b, " · first token p50 %s", msOf(x.P50))
	}
	if x := t.E2E; x.Count > 0 {
		fmt.Fprintf(&b, " · total p50 %s", msOf(x.P50))
	}
	return b.String()
}

// failedBy is " (network_error 3, unavailable_503 1)", or "" without failures.
func failedBy(m map[string]int64) string {
	var parts []string
	for k, v := range m {
		if k != report.OutcomeOK && v > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", k, v))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	sort.Strings(parts)
	return " (" + strings.Join(parts, ", ") + ")"
}

func clockOf(d time.Duration) string {
	sec := int64(d.Round(time.Second).Seconds())
	if sec >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", sec/3600, sec/60%60, sec%60)
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func msOf(ms float64) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.1f s", ms/1000)
	}
	return fmt.Sprintf("%.0f ms", ms)
}
