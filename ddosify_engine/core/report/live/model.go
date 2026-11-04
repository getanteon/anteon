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
	"math"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
)

// History kept for the sparklines: wider than any terminal row, fixed in size.
const historyLen = 240

type snapshotMsg Snapshot

type closedMsg struct{}

// stopRequestMsg is a stop asked by a key or a signal.
type stopRequestMsg struct{}

type keyMap struct {
	Stop  key.Binding
	Force key.Binding
}

var keys = keyMap{
	Stop:  key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "stop and write the report")),
	Force: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "again to leave now")),
}

// Model is the Bubble Tea model of the live view.
type Model struct {
	snaps <-chan Snapshot
	stop  func()

	title      string
	thresholds []Threshold
	target     float64 // goodput target in percent, 0 when none

	last Snapshot
	have bool

	rps  *ring
	ttft *ring

	width, height int
	noColor       bool
	st            styles
	bar           progress.Model

	stopping bool // stop asked, waiting for the engine to finish
	finished bool // snapshot channel closed
	forced   bool // left before the engine finished
}

// ModelOptions configure a Model. Run fills them from Options.
type ModelOptions struct {
	Title         string
	Thresholds    []Threshold
	GoodputTarget float64
	NoColor       bool
	Stop          func()
	Width, Height int
}

// NewModel builds the model that reads snaps until the channel closes.
func NewModel(snaps <-chan Snapshot, o ModelOptions) *Model {
	title := o.Title
	if title == "" {
		title = "ddosify"
	}
	m := &Model{
		snaps:      snaps,
		stop:       o.Stop,
		title:      title,
		thresholds: o.Thresholds,
		target:     o.GoodputTarget,
		rps:        newRing(historyLen),
		ttft:       newRing(historyLen),
		width:      o.Width,
		height:     o.Height,
		noColor:    o.NoColor,
		st:         newStyles(o.NoColor),
	}
	m.bar = newBar(o.NoColor)
	return m
}

func (m *Model) Init() tea.Cmd {
	return m.wait()
}

func (m *Model) wait() tea.Cmd {
	ch := m.snaps
	return func() tea.Msg {
		s, ok := <-ch
		if !ok {
			return closedMsg{}
		}
		return snapshotMsg(s)
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyPressMsg:
		if m.stopping && key.Matches(msg, keys.Force) {
			m.forced = true
			return m, tea.Quit
		}
		if key.Matches(msg, keys.Stop) {
			return m, m.requestStop()
		}
		return m, nil

	case stopRequestMsg:
		return m, m.requestStop()

	case snapshotMsg:
		m.observe(Snapshot(msg))
		return m, m.wait()

	case closedMsg:
		m.finished = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) observe(s Snapshot) {
	m.last = s
	m.have = true
	m.rps.push(s.AchievedRPS)
	if s.TTFT.Count > 0 {
		m.ttft.push(s.TTFT.P95)
	} else {
		m.ttft.push(math.NaN())
	}
}

func (m *Model) requestStop() tea.Cmd {
	if m.stopping {
		return nil
	}
	m.stopping = true
	stop := m.stop
	if stop == nil {
		return nil
	}
	return func() tea.Msg {
		stop()
		return nil
	}
}

// filter turns the quit and interrupt that signals send into a stop request,
// so the engine finishes and the final report is still written.
func filter(tm tea.Model, msg tea.Msg) tea.Msg {
	m, ok := tm.(*Model)
	if !ok {
		return msg
	}
	switch msg.(type) {
	case tea.QuitMsg:
		if m.finished || m.forced {
			return msg
		}
		return stopRequestMsg{}
	case tea.InterruptMsg:
		return stopRequestMsg{}
	}
	return msg
}

// Stopping tells whether a stop was asked.
func (m *Model) Stopping() bool { return m.stopping }

// Forced tells whether the view was left before the engine finished.
func (m *Model) Forced() bool { return m.forced }

// Last returns the latest snapshot.
func (m *Model) Last() (Snapshot, bool) { return m.last, m.have }
