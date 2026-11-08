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
	"io"
	"os"

	elive "go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/report/live"
	"go.ddosify.com/ddosify/v2/core/types"
)

func init() {
	elive.Register("html", newObserver)
	elive.LogReport = writeFromLog
	elive.CompareLogs = compareLogs
}

// withOptions puts the options over what the run's own config said.
func withOptions(rep *Report, o elive.ReportOptions) {
	if o.Title != "" {
		rep.Title = o.Title
	}
	if o.GPUs > 0 || o.Tiers != nil {
		c := types.Capacity{}
		if rep.Capacity != nil {
			c = *rep.Capacity
		}
		if o.GPUs > 0 {
			c.GPUs = o.GPUs
		}
		if o.Tiers != nil {
			c.Tiers = o.Tiers
		}
		rep.Capacity = &c
	}
}

func writeFromLog(in io.Reader, o elive.ReportOptions, open func() (io.Writer, error)) error {
	rep, err := ReadLog(in)
	if err != nil {
		return err
	}
	withOptions(&rep, o)
	if len(o.Resources) > 0 {
		if rep.Resources, err = LoadResources(rep, o.Resources, o.ResourcesPer, o.Stderr); err != nil {
			return err
		}
	}
	w, err := open()
	if err != nil {
		return err
	}
	return Write(w, rep)
}

func compareLogs(logs []elive.LabeledLog, o elive.ReportOptions, open func() (io.Writer, error)) error {
	runs := make([]Labeled, 0, len(logs))
	for _, l := range logs {
		rep, err := ReadLog(l.In)
		if err != nil {
			return fmt.Errorf("%s: %w", l.Label, err)
		}
		withOptions(&rep, elive.ReportOptions{GPUs: o.GPUs, Tiers: o.Tiers})
		var files []elive.ResourceLog
		for _, f := range o.Resources {
			if f.Run == l.Label {
				files = append(files, f)
			}
		}
		if len(files) > 0 {
			if rep.Resources, err = LoadResources(rep, files, o.ResourcesPer, o.Stderr); err != nil {
				return fmt.Errorf("%s: %w", l.Label, err)
			}
		}
		runs = append(runs, Labeled{Label: l.Label, Report: rep})
	}
	var tiers []types.CapacityTier
	if o.Tiers != nil {
		tiers = o.Tiers
	}
	w, err := open()
	if err != nil {
		return err
	}
	return WriteCompare(w, o.Title, runs, o.GPUs, tiers)
}

type observer struct {
	o    elive.Options
	plan elive.Plan
	rec  *Recorder
	f    *os.File
	err  error
}

// newObserver opens the file now, so a path that can not be written fails before the test runs.
func newObserver(o elive.Options) (elive.Observer, error) {
	if o.Arg == "" {
		return nil, fmt.Errorf("--html needs a file path")
	}
	f, err := os.OpenFile(o.Arg, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("--html: %w", err)
	}
	return &observer{o: o, rec: NewRecorder(0), f: f}, nil
}

// Err is the write's error, if the report could not be written.
func (h *observer) Err() error { return h.err }

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
	rep.PrefixCache = live.PrefixCache(h.plan)
	rep.Duration = f.Elapsed
	rep.Stopped = f.Status != "done"
	if f.Result == nil {
		rep.Status = map[bool]string{true: "success", false: "failed"}[f.Passed]
	}
	err := Write(h.f, rep)
	if cerr := h.f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		h.err = fmt.Errorf("html report %s: %w", h.o.Arg, err)
		fmt.Fprintln(h.o.Stderr, h.err)
		return
	}
	fmt.Fprintf(h.o.Stderr, "HTML report written to %s\n", h.o.Arg)
}

// ConfigFromPlan keeps the plan's numbers and step names; the step URL and method stay out.
func ConfigFromPlan(p elive.Plan) Config {
	c := Config{LoadType: p.LoadType, Iterations: p.Iterations, DurationS: int(p.Duration.Seconds())}
	for _, s := range p.Stages {
		d := int((s.End - s.Start).Seconds())
		if s.Concurrency > 0 {
			d = int(s.Cap.Seconds())
		}
		c.Stages = append(c.Stages, StageConfig{Name: s.Name, Warmup: s.Warmup, DurationS: d, Count: s.Iterations, Concurrency: s.Concurrency})
	}
	for _, s := range p.Steps {
		c.Steps = append(c.Steps, StepConfig{ID: s.ID, Name: s.Name, Stream: s.Streamed})
	}
	return c
}
