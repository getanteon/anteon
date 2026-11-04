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
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
)

// Mode is how a run is shown.
type Mode int

const (
	ModeTUI Mode = iota + 1
	ModeNDJSON
)

const (
	DefaultFPS       = 8
	minFPS, maxFPS   = 4, 10
	DefaultJSONEvery = time.Second
)

// Options configure Run. Every field is optional.
type Options struct {
	// Stop asks the engine to stop early. Run calls it at most once, on q, Ctrl-C, SIGINT or SIGTERM.
	// The engine then finishes its iterations in flight and closes the snapshot channel.
	Stop func()

	// JSON writes NDJSON even on a TTY: the --json flag.
	JSON bool

	// NoColor turns colour off: the --no-color flag. NO_COLOR and TERM=dumb do the same.
	NoColor bool

	// The terminal. Default os.Stdout and os.Stdin.
	Out io.Writer
	In  io.Reader

	// Where NDJSON lines go. Default os.Stderr.
	Err io.Writer

	// Environment read for NO_COLOR and TERM. Default os.Environ().
	Env []string

	// Size used until the terminal reports its own, and for output that is not a terminal.
	Width, Height int

	// Redraws a second, 4 to 10. Default 8.
	FPS int

	// Least time between NDJSON lines. The last snapshot is always written. Default 1s; negative writes every one.
	JSONEvery time.Duration

	Title         string
	Thresholds    []Threshold
	GoodputTarget float64 // percent, 0 for none
}

// Result tells how the live view ended.
type Result struct {
	Mode    Mode
	Stopped bool // a stop was asked
	Forced  bool // the view was left before the engine finished; the channel is still drained
	Last    Snapshot
	Seen    bool // at least one snapshot arrived
}

// IsTerminal tells whether w, a reader or a writer, is a terminal.
func IsTerminal(w any) bool {
	f, ok := w.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(f.Fd())
}

// NoColorFromEnv tells whether NO_COLOR is set to something, or TERM is dumb.
func NoColorFromEnv(env []string) bool {
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		if (k == "NO_COLOR" && v != "") || (k == "TERM" && v == "dumb") {
			return true
		}
	}
	return false
}

func (o *Options) defaults() {
	if o.Out == nil {
		o.Out = os.Stdout
	}
	if o.In == nil {
		o.In = os.Stdin
	}
	if o.Err == nil {
		o.Err = os.Stderr
	}
	if o.Env == nil {
		o.Env = os.Environ()
	}
	if o.FPS == 0 {
		o.FPS = DefaultFPS
	}
	o.FPS = max(minFPS, min(maxFPS, o.FPS))
	if o.JSONEvery == 0 {
		o.JSONEvery = DefaultJSONEvery
	}
	o.NoColor = o.NoColor || NoColorFromEnv(o.Env)
	if o.Stop != nil {
		var once sync.Once
		stop := o.Stop
		o.Stop = func() { once.Do(stop) }
	}
}

// Run shows snapshots until the channel closes, then returns. The caller writes the final report after it.
// On a terminal it runs the TUI on Out; otherwise, or with JSON, it writes NDJSON lines to Err.
func Run(ctx context.Context, snaps <-chan Snapshot, o Options) (Result, error) {
	o.defaults()
	if o.JSON || !IsTerminal(o.Out) {
		return RunNDJSON(ctx, snaps, o)
	}
	return RunTUI(ctx, snaps, o)
}

// RunTUI runs the terminal UI whatever Out is.
func RunTUI(ctx context.Context, snaps <-chan Snapshot, o Options) (Result, error) {
	o.defaults()
	m := NewModel(snaps, ModelOptions{
		Title:         o.Title,
		Thresholds:    o.Thresholds,
		GoodputTarget: o.GoodputTarget,
		NoColor:       o.NoColor,
		Stop:          o.Stop,
	})
	opts := []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithOutput(o.Out),
		tea.WithEnvironment(o.Env),
		tea.WithFPS(o.FPS),
		tea.WithFilter(filter),
	}
	// Without a terminal on stdin, as under kubectl run -t without -i, keys cannot arrive; signals still stop.
	if IsTerminal(o.In) {
		opts = append(opts, tea.WithInput(o.In))
	} else {
		opts = append(opts, tea.WithInput(nil))
	}
	if o.Width > 0 && o.Height > 0 {
		opts = append(opts, tea.WithWindowSize(o.Width, o.Height))
	}
	if o.NoColor {
		opts = append(opts, tea.WithColorProfile(colorprofile.Ascii))
	}
	_, err := tea.NewProgram(m, opts...).Run()

	res := Result{Mode: ModeTUI, Stopped: m.Stopping(), Forced: m.Forced()}
	res.Last, res.Seen = m.Last()
	if !m.finished {
		// Never leave the engine blocked on a send nobody reads.
		go drain(snaps)
	}
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		err = ctx.Err()
	}
	return res, err
}

func drain(snaps <-chan Snapshot) {
	for range snaps {
	}
}

// RunNDJSON writes one line per JSONEvery to Err until the channel closes.
func RunNDJSON(ctx context.Context, snaps <-chan Snapshot, o Options) (Result, error) {
	o.defaults()
	res := Result{Mode: ModeNDJSON}
	var (
		lastWrite time.Time
		pending   bool
		werr      error
	)
	write := func(s Snapshot) {
		if err := WriteNDJSON(o.Err, s); err != nil && werr == nil {
			werr = err
		}
		lastWrite = time.Now()
		pending = false
	}
	for {
		select {
		case <-ctx.Done():
			go drain(snaps)
			return res, ctx.Err()
		case s, ok := <-snaps:
			if !ok {
				if pending {
					write(res.Last)
				}
				return res, werr
			}
			res.Last, res.Seen = s, true
			if o.JSONEvery < 0 || lastWrite.IsZero() || time.Since(lastWrite) >= o.JSONEvery {
				write(s)
			} else {
				pending = true
			}
		}
	}
}

// Tee forwards every snapshot to fns, then to the returned channel, which closes when in closes.
// Use it to feed the HTML report's recorder from the same stream the view reads.
func Tee(in <-chan Snapshot, fns ...func(Snapshot)) <-chan Snapshot {
	out := make(chan Snapshot, 4)
	go func() {
		defer close(out)
		for s := range in {
			for _, fn := range fns {
				fn(s)
			}
			out <- s
		}
	}()
	return out
}
