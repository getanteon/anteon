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

// Package live publishes a running test to observers: a Plan before the first iteration,
// a Snapshot of the aggregated metrics every Interval, and the Final report.
// A TUI or an HTML report registers a Factory under a name in init(); the CLI builds the
// observers the user asked for and passes them to the engine.
package live

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/types"
)

// Interval between two snapshots.
const Interval = 250 * time.Millisecond

// Observer receives one Start, then Update every Interval, then one Finish.
// All calls come from one goroutine. Update must return quickly: a slow Update makes the
// next snapshots be skipped, not queued. Finish may block, for example until the user
// closes a TUI; the CLI exits after it returns.
type Observer interface {
	Start(p Plan)
	Update(s Snapshot)
	Finish(f Final)
}

// Options are what the CLI knows when it builds an observer.
type Options struct {
	Stdout, Stderr io.Writer

	// Color is false under NO_COLOR, TERM=dumb, --no-color, or when stdout is not a terminal.
	Color bool

	// Interactive is true when stdout and stdin are terminals.
	Interactive bool

	// Arg is the observer's argument: the path of --html, or what follows "=" in --observer name=arg.
	Arg string

	// Stop ends the test gracefully, as CTRL+C does: no new iterations, the running ones finish,
	// then Finish is called. Safe to call more than once and from any goroutine.
	Stop func()
}

// Factory builds an observer.
type Factory func(o Options) (Observer, error)

// Failer is implemented by an observer whose Finish can fail, such as a file write. The CLI exits 1
// when Err is not nil after the run.
type Failer interface {
	Err() error
}

// TerminalOwner is implemented by an observer that draws on the terminal, such as a TUI.
// While one runs, the CLI prints no progress lines and prints the plain summary after Finish.
type TerminalOwner interface {
	OwnsTerminal() bool
}

var (
	regMu     sync.RWMutex
	factories = map[string]Factory{}
)

// Register makes an observer available to the CLI under name: "tui" for the live view on a
// terminal, "ndjson" for snapshots when stdout is not a terminal or with --json, "html" for
// --html <path>, anything else for --observer name[=arg]. A nil Factory unregisters name.
func Register(name string, f Factory) {
	regMu.Lock()
	defer regMu.Unlock()
	if f == nil {
		delete(factories, name)
		return
	}
	factories[name] = f
}

func Lookup(name string) (Factory, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

func Names() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	names := make([]string, 0, len(factories))
	for n := range factories {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Build looks up and builds the named observer.
func Build(name string, o Options) (Observer, error) {
	f, ok := Lookup(name)
	if !ok {
		return nil, fmt.Errorf("observer %q is not in this build, available: %v", name, Names())
	}
	return f(o)
}

// Stage is a manual_load stage, or the whole test when there are none.
type Stage struct {
	Index  int           `json:"index"`
	Name   string        `json:"name"`
	Warmup bool          `json:"warmup"`
	Start  time.Duration `json:"start_ns"`
	End    time.Duration `json:"end_ns"`

	// Planned iterations in the stage
	Iterations int `json:"iterations"`
}

// StepInfo names a scenario step.
type StepInfo struct {
	ID       uint16 `json:"id"`
	Name     string `json:"name"`
	Method   string `json:"method"`
	URL      string `json:"url"`
	Streamed bool   `json:"streamed"`

	// The llm config's prompt changes: a unique marker per prompt, or a shared prefix of this many tokens
	CacheBust          bool   `json:"cache_bust,omitempty"`
	SharedPrefixTokens int    `json:"shared_prefix_tokens,omitempty"`
	TokenSource        string `json:"token_source,omitempty"`
}

// Plan is the test as configured.
type Plan struct {
	Duration   time.Duration `json:"duration_ns"`
	Iterations int           `json:"iterations"`
	Stages     []Stage       `json:"stages"`
	Steps      []StepInfo    `json:"steps"`
	LoadType   string        `json:"load_type"`
	EngineMode string        `json:"engine_mode"`

	// Planned iterations per 100 ms tick, from which TargetRPS is read
	Schedule []int `json:"schedule"`

	// The thresholds, and the goodput target in SLO.GoodputPerc; nil without an slo
	SLO *types.SLO `json:"slo,omitempty"`
}

// Percentiles in milliseconds, or in units for an Extra metric such as output tokens.
type Percentiles struct {
	Count int64   `json:"count"`
	Mean  float64 `json:"mean"`
	P50   float64 `json:"p50"`
	P90   float64 `json:"p90"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Max   float64 `json:"max"`
}

// Goodput counts the iterations whose every step was ok and whose streams met the SLO.
type Goodput struct {
	Good       int64   `json:"good"`
	Iterations int64   `json:"iterations"`
	Perc       float64 `json:"perc"`
}

// Metrics aggregate the iterations that completed in a span of the test.
type Metrics struct {
	Iterations int64 `json:"iterations"`

	// Step results by report.Outcome* key: ok, rate_limited_429, unavailable_503, ...
	Outcomes map[string]int64 `json:"outcomes"`

	// Step results by HTTP status; 0 counts those that got no response
	StatusCodes map[int]int64 `json:"status_codes"`

	// Over streamed step results
	TTFB Percentiles `json:"ttfb_ms"`
	TTFT Percentiles `json:"ttft_ms"` // the first token
	ITL  Percentiles `json:"itl_ms"`  // each stream's median gap between events

	// Wall time of the iterations whose every step was ok, all steps
	E2E Percentiles `json:"e2e_ms"`

	// Set when an SLO is configured
	Goodput *Goodput `json:"goodput,omitempty"`

	// Metrics stream observers returned, such as "output_tokens"
	Extra map[string]Percentiles `json:"extra,omitempty"`

	// Answers' output, set when a step has an llm config
	Tokens *Tokens `json:"tokens,omitempty"`

	// The first request's send time minus the time the schedule meant the iteration to start, ms.
	// A high p99 means the client, not the server, fell behind.
	SchedulingLag Percentiles `json:"scheduling_lag_ms"`
}

// Tokens is the output of the answers, counted in tokens (the server's usage or a tokenizer) or,
// when no answer could be, in chunks: content events. Unit says which; a chunk is never a token.
type Tokens struct {
	Unit    string `json:"unit"`
	Answers int64  `json:"answers"`

	// Output per answer, in Unit
	Output Percentiles `json:"output"`

	// Time per output token, ms; only with Unit "tokens"
	TPOT Percentiles `json:"tpot_ms"`

	// Gaps between content events, ms, all answers pooled
	ITL Percentiles `json:"itl_ms"`

	// All output over the time from the first send to the last answer's end, in Unit a second
	PerSec float64 `json:"per_sec"`

	// Answers whose prompt got a cache-busting marker, and the shared prefix's tokens
	CacheBusted  int64 `json:"cache_busted,omitempty"`
	PrefixTokens int   `json:"prefix_tokens,omitempty"`
}

// Snapshot is the state of the test at one moment.
type Snapshot struct {
	Seq     int64         `json:"seq"`
	At      time.Time     `json:"at"`
	Elapsed time.Duration `json:"elapsed_ns"`

	// The stage running now; Index -1 when manual_load has none
	Stage Stage `json:"stage"`

	// Iterations a second the plan asks for now, and started over the last second
	TargetRPS   float64 `json:"target_rps"`
	AchievedRPS float64 `json:"achieved_rps"`

	Started   int64 `json:"started"`
	Completed int64 `json:"completed"`
	InFlight  int64 `json:"in_flight"`

	// Since the current stage began, and since the test began with warm-up stages left out
	Current Metrics `json:"current"`
	Total   Metrics `json:"total"`
}

// Final is the end of the test.
type Final struct {
	// "done", "stopped" (interrupted) or "aborted" (a success criterion with abort)
	Status  string        `json:"status"`
	Elapsed time.Duration `json:"elapsed_ns"`

	// False when a success criterion or the SLO's goodput failed
	Passed bool `json:"passed"`

	// The report the stdout and JSON outputs print; nil if the output keeps none
	Result *report.Result `json:"result"`

	// The last snapshot
	Last Snapshot `json:"last"`
}
