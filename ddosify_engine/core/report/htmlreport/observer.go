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

// The --html observer, on core/live.

package htmlreport

import (
	"fmt"

	elive "go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/report/live"
)

func init() {
	elive.Register("html", newObserver)
}

type observer struct {
	o    elive.Options
	plan elive.Plan
	rec  *Recorder
}

func newObserver(o elive.Options) (elive.Observer, error) {
	if o.Arg == "" {
		return nil, fmt.Errorf("--html needs a file path")
	}
	return &observer{o: o, rec: NewRecorder(0)}, nil
}

func (h *observer) Start(p elive.Plan) { h.plan = p }

func (h *observer) Update(s elive.Snapshot) { h.rec.Add(live.FromEngine(h.plan, s)) }

func (h *observer) Finish(f elive.Final) {
	h.rec.Add(live.FromEngine(h.plan, f.Last))
	cfg := ConfigFromPlan(h.plan)
	cfg.SLO = live.ThresholdsFromSLO(h.plan.SLO)
	if f.Result != nil && f.Result.Goodput != nil {
		cfg.SLO = live.ThresholdsFromSLO(&f.Result.Goodput.SLO)
	}
	rep := FromResult(f.Result, cfg, h.rec)
	rep.Duration = f.Elapsed
	rep.Stopped = f.Status != "done"
	if f.Result == nil {
		rep.Status = map[bool]string{true: "success", false: "failed"}[f.Passed]
	}
	if err := WriteFile(h.o.Arg, rep); err != nil {
		fmt.Fprintf(h.o.Stderr, "html report: %v\n", err)
		return
	}
	fmt.Fprintf(h.o.Stderr, "HTML report written to %s\n", h.o.Arg)
}

// ConfigFromPlan keeps the plan's numbers and step names; the step URL and method stay out.
func ConfigFromPlan(p elive.Plan) Config {
	c := Config{LoadType: p.LoadType, Iterations: p.Iterations, DurationS: int(p.Duration.Seconds())}
	for _, s := range p.Stages {
		c.Stages = append(c.Stages, StageConfig{Name: s.Name, Warmup: s.Warmup, DurationS: int((s.End - s.Start).Seconds()), Count: s.Iterations})
	}
	for _, s := range p.Steps {
		c.Steps = append(c.Steps, StepConfig{ID: s.ID, Name: s.Name, Stream: s.Streamed})
	}
	return c
}
