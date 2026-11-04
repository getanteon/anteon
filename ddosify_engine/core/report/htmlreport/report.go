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

// Package htmlreport writes the end-of-test report as one self-contained HTML file:
// inline CSS, SVG and script, no network. It holds numbers and step names only,
// never a URL, header, body, token or assertion text, because the file may leave the building.
package htmlreport

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/report/live"
	"go.ddosify.com/ddosify/v2/core/types"
)

// Report is everything the HTML file shows.
type Report struct {
	Title       string
	GeneratedAt time.Time
	Status      string // "success" or "failed"
	Stopped     bool   // stopped early from the keyboard or a signal
	Duration    time.Duration

	FailedCriteria int

	Config   Config
	Steps    []Step
	Stages   []*report.StageSummary
	Goodput  *report.Goodput
	Outcomes map[string]int64 // whole test, warm-up excluded

	Series        []Point
	ITLIsEventGap bool
}

// Config is the shape of the load, without anything a request carries.
type Config struct {
	LoadType   string
	LoadSpread string
	Iterations int
	DurationS  int
	Stages     []StageConfig
	SLO        []live.Threshold
	Steps      []StepConfig
}

type StageConfig struct {
	Name      string
	Warmup    bool
	DurationS int
	Count     int
}

type StepConfig struct {
	ID       uint16
	Name     string
	Stream   bool
	TimeoutS int
}

// Step is one step's results.
type Step struct {
	ID       uint16
	Name     string
	Success  int64
	Failed   int64
	Outcomes map[string]int64
	Stream   *report.StreamSummary
}

// ConfigFromHammer keeps the numbers and step names of a test and drops the rest:
// URLs, headers, bodies, auth, cookies, captures, assertions and test data never reach the report.
func ConfigFromHammer(h *types.Hammer) Config {
	if h == nil {
		return Config{}
	}
	c := Config{
		LoadType:   h.LoadType,
		LoadSpread: h.LoadSpread,
		Iterations: h.IterationCount,
		DurationS:  h.TestDuration,
		SLO:        live.ThresholdsFromSLO(h.SLO),
	}
	for _, s := range h.TimeRunCountMap {
		c.Stages = append(c.Stages, StageConfig{Name: s.Name, Warmup: s.Warmup, DurationS: s.Duration, Count: s.Count})
	}
	for _, s := range h.Scenario.Steps {
		c.Steps = append(c.Steps, StepConfig{ID: s.ID, Name: s.Name, Stream: s.Stream != nil, TimeoutS: s.Timeout})
	}
	return c
}

// FromResult fills the results from the engine's final report.
func FromResult(r *report.Result, cfg Config, rec *Recorder) Report {
	rep := Report{
		Title:       "Ddosify load test report",
		GeneratedAt: time.Now(),
		Config:      cfg,
	}
	if rec != nil {
		rep.Series = rec.Points()
		rep.ITLIsEventGap = rec.EventGap()
		if n := len(rep.Series); n > 0 {
			rep.Duration = time.Duration(rep.Series[n-1].T * float64(time.Second))
		}
	}
	if r == nil {
		return rep
	}
	rep.Status = r.TestStatus
	rep.FailedCriteria = len(r.TestFailedAssertions)
	rep.Goodput = r.Goodput
	rep.Stages = r.Stages

	ids := make([]int, 0, len(r.StepResults))
	for id := range r.StepResults {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	for _, id := range ids {
		s := r.StepResults[uint16(id)]
		rep.Steps = append(rep.Steps, Step{
			ID:       uint16(id),
			Name:     s.Name,
			Success:  s.SuccessCount,
			Failed:   s.Fail.Count,
			Outcomes: s.Outcomes,
			Stream:   s.Stream,
		})
		for k, v := range s.Outcomes {
			if rep.Outcomes == nil {
				rep.Outcomes = make(map[string]int64)
			}
			rep.Outcomes[k] += v
		}
	}
	return rep
}

// WriteFile writes the report to path, readable by its owner only.
func WriteFile(path string, r Report) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := Write(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Write renders the report.
func Write(w io.Writer, r Report) error {
	if r.GeneratedAt.IsZero() {
		r.GeneratedAt = time.Now()
	}
	if r.Title == "" {
		r.Title = "Ddosify load test report"
	}
	v := buildView(r)
	if err := page.Execute(w, v); err != nil {
		return fmt.Errorf("html report: %w", err)
	}
	return nil
}
