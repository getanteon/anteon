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
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	defaultWidth  = 100
	maxStageRows  = 8
	minSparkWidth = 50
	minBarWidth   = 60
)

// Lower number, kept longer when the terminal is short.
const (
	prioHeader = iota
	prioFooter
	prioLatency
	prioOutcomes
	prioGoodput
	prioStages
	prioSparks
)

type section struct {
	prio  int
	lines []string
}

func (m *Model) View() tea.View {
	return tea.NewView(m.Render())
}

// Render draws the view at the model's size.
func (m *Model) Render() string {
	w := m.width
	if w <= 0 {
		w = defaultWidth
	}
	secs := []section{{prioHeader, m.header(w)}}
	if m.have {
		secs = append(secs,
			section{prioLatency, m.latency()},
			section{prioGoodput, m.goodput(w)},
			section{prioOutcomes, m.outcomes(w)},
		)
		if st := m.stages(w); len(st) > 0 {
			secs = append(secs, section{prioStages, st})
		}
		if w >= minSparkWidth {
			secs = append(secs, section{prioSparks, m.sparks(w)})
		}
	}
	secs = append(secs, section{prioFooter, m.footer(w)})

	lines := fit(secs, m.height)
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, w, "")
	}
	return strings.Join(lines, "\n")
}

// fit drops the blank lines between sections, then the least needed sections, until the view fits.
func fit(secs []section, h int) []string {
	join := func(blank bool) []string {
		var out []string
		for i, s := range secs {
			if blank && i > 0 {
				out = append(out, "")
			}
			out = append(out, s.lines...)
		}
		return out
	}
	if h <= 0 {
		return join(true)
	}
	if out := join(true); len(out) <= h {
		return out
	}
	for {
		out := join(false)
		if len(out) <= h || len(secs) <= 2 {
			if len(out) > h {
				out = out[:h]
			}
			return out
		}
		drop := 0
		for i, s := range secs {
			if s.prio > secs[drop].prio {
				drop = i
			}
		}
		secs = append(secs[:drop:drop], secs[drop+1:]...)
	}
}

// flow lays chunks out in lines no wider than w.
func flow(chunks []string, w int) []string {
	const gap = "   "
	var lines []string
	cur := ""
	for _, c := range chunks {
		if c == "" {
			continue
		}
		if cur == "" {
			cur = c
			continue
		}
		if lipgloss.Width(cur)+len(gap)+lipgloss.Width(c) > w {
			lines = append(lines, cur)
			cur = c
			continue
		}
		cur += gap + c
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func (m *Model) kv(label, value string) string {
	return m.st.label.Render(label) + " " + m.st.value.Render(value)
}

func (m *Model) header(w int) []string {
	s := m.last
	first := []string{m.st.title.Render(m.title)}
	if !m.have {
		first = append(first, m.st.label.Render("waiting for the first snapshot"))
		return flow(first, w)
	}
	if s.Stage.Count > 0 {
		stage := fmt.Sprintf("%d/%d", s.Stage.Index+1, s.Stage.Count)
		if s.Stage.Name != "" {
			stage += " " + s.Stage.Name
		}
		if s.Stage.Warmup {
			stage += " (warm-up)"
		}
		first = append(first, m.kv("stage", stage))
	}
	first = append(first, m.kv("elapsed", clock(s.Elapsed)))
	if s.Duration > 0 {
		first = append(first, m.kv("remaining", clock(max(0, s.Duration-s.Elapsed))))
	}
	second := []string{
		m.kv("rps target", rate(s.TargetRPS)),
		m.kv("achieved", rate(s.AchievedRPS)),
		m.kv("in flight", num(s.InFlight)),
		m.kv("iterations", num(s.Iterations)),
	}
	return append(flow(first, w), flow(second, w)...)
}

func (m *Model) latency() []string {
	s := m.last
	itl := "ITL"
	if s.ITLIsEventGap {
		itl = "event gap"
	}
	row := func(name string, l Latency) string {
		vals := []float64{l.P50, l.P90, l.P99}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("%-10s", name))
		for _, v := range vals {
			cell := "-"
			if l.Count > 0 {
				cell = msf(v)
			}
			b.WriteString(fmt.Sprintf("%10s", cell))
		}
		return m.st.value.Render(b.String())
	}
	head := m.st.label.Render(fmt.Sprintf("%-10s%10s%10s%10s", "latency ms", "p50", "p90", "p99"))
	return []string{head, row("TTFT", s.TTFT), row(itl, s.ITL), row("E2E", s.E2E)}
}

func (m *Model) goodput(w int) []string {
	g := m.last.Goodput
	var first []string
	switch {
	case len(m.thresholds) == 0 && g.Total == 0:
		return []string{m.kv("goodput", "-") + "  " + m.st.label.Render("no SLO set")}
	case g.Total == 0:
		first = append(first, m.kv("goodput", "-"), m.st.label.Render("no iteration finished yet"))
	default:
		first = append(first,
			m.kv("goodput", pct(g.Perc())),
			m.st.label.Render(fmt.Sprintf("%s of %s iterations met the SLO", num(g.Good), num(g.Total))),
		)
		if m.target > 0 {
			t := "target " + pct(m.target)
			if g.Perc() >= m.target {
				first = append(first, m.st.good.Render(t+" met"))
			} else {
				first = append(first, m.st.bad.Render("! "+t+" missed"))
			}
		}
	}
	lines := flow(first, w)
	if len(m.thresholds) > 0 {
		slo := []string{m.st.label.Render("slo")}
		for i, t := range m.thresholds {
			c := fmt.Sprintf("%s <= %s ms", t.Name, num(t.Ms))
			if i < len(m.thresholds)-1 {
				c += ","
			}
			slo = append(slo, c)
		}
		lines = append(lines, flowTight(slo, w)...)
	}
	return lines
}

// flowTight is flow with single spaces, for one sentence.
func flowTight(chunks []string, w int) []string {
	var lines []string
	cur := ""
	for _, c := range chunks {
		if cur != "" && lipgloss.Width(cur)+1+lipgloss.Width(c) > w {
			lines = append(lines, cur)
			cur = "    " + c
			continue
		}
		if cur != "" {
			cur += " "
		}
		cur += c
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func (m *Model) outcomes(w int) []string {
	o := m.last.Outcomes
	total := o.Total()
	share := func(v int64) string {
		if total == 0 || v == 0 {
			return ""
		}
		return " (" + pct(float64(v)*100/float64(total)) + ")"
	}
	// 429 and 503 carry a "!" when present, so the warning does not rest on colour.
	alarm := func(label string, v int64) string {
		if v == 0 {
			return m.kv(label, "0")
		}
		return m.st.bad.Render(fmt.Sprintf("! %s %s%s", label, num(v), share(v)))
	}
	plain := func(label string, v int64, always bool) string {
		if v == 0 && !always {
			return ""
		}
		if v == 0 {
			return m.kv(label, "0")
		}
		return m.st.warn.Render(fmt.Sprintf("%s %s%s", label, num(v), share(v)))
	}
	chunks := []string{
		m.st.label.Render("outcomes"),
		m.kv("ok", num(o.OK)),
		alarm("429", o.RateLimited),
		alarm("503", o.Unavailable),
		plain("5xx", o.ServerError, true),
		plain("timeout", o.Timeout, true),
		plain("cut", o.StreamCut, true),
		plain("4xx", o.ClientError, false),
		plain("network", o.NetworkError, false),
		plain("error event", o.StreamError, false),
	}
	return flow(chunks, w)
}

func (m *Model) stages(w int) []string {
	s := m.last
	plans := s.Stages
	if len(plans) == 0 {
		return nil
	}
	cur := s.Stage.Index
	from, to := 0, len(plans)
	if len(plans) > maxStageRows {
		from = max(0, min(cur-maxStageRows/2, len(plans)-maxStageRows))
		to = from + maxStageRows
	}
	nameW := 0
	for _, p := range plans[from:to] {
		nameW = max(nameW, lipgloss.Width(stageName(p)))
	}
	nameW = min(nameW, 24)

	lines := []string{m.st.label.Render("stages")}
	if from > 0 {
		lines = append(lines, m.st.dim.Render(fmt.Sprintf("  ... %d earlier", from)))
	}
	for i := from; i < to; i++ {
		p := plans[i]
		var frac float64
		var status string
		marker := " "
		switch {
		case i < cur || (i == cur && m.finished && !m.stopping):
			frac, status = 1, "done"
		case i == cur && m.finished:
			if p.Duration > 0 {
				frac = math.Min(1, float64(s.Stage.Elapsed)/float64(p.Duration))
			}
			status = fmt.Sprintf("stopped at %.0f%%", frac*100)
		case i == cur:
			marker = ">"
			if p.Duration > 0 {
				frac = math.Min(1, float64(s.Stage.Elapsed)/float64(p.Duration))
			}
			status = fmt.Sprintf("%3.0f%%  %s / %s", frac*100, clock(s.Stage.Elapsed), clock(p.Duration))
		default:
			status = "queued"
		}
		name := ansi.Truncate(stageName(p), nameW, "")
		row := fmt.Sprintf("%s %2d %-*s", marker, i+1, nameW, name)
		if w >= minBarWidth {
			barW := min(30, w-lipgloss.Width(row)-24)
			if barW >= 8 {
				m.bar.SetWidth(barW)
				row += " " + m.bar.ViewAs(frac)
			}
		}
		row += " " + status
		if i == cur && !m.finished {
			row = m.st.value.Render(row)
		} else if i > cur {
			row = m.st.dim.Render(row)
		}
		lines = append(lines, row)
	}
	if to < len(plans) {
		lines = append(lines, m.st.dim.Render(fmt.Sprintf("  ... %d more", len(plans)-to)))
	}
	return lines
}

func stageName(p StagePlan) string {
	n := p.Name
	if n == "" {
		n = fmt.Sprintf("%d in %s", p.Count, clock(p.Duration))
	}
	if p.Warmup && !strings.Contains(strings.ToLower(n), "warm") {
		n += " (warm-up)"
	}
	return n
}

func (m *Model) sparks(w int) []string {
	const labelW, valueW = 10, 12
	sw := min(w-labelW-valueW-2, historyLen)
	line := func(label string, r *ring, value string) string {
		return m.st.label.Render(fmt.Sprintf("%-*s", labelW, label)) +
			m.st.good.Render(spark(r.last(sw), sw)) + " " + m.st.value.Render(fmt.Sprintf("%*s", valueW, value))
	}
	s := m.last
	ttft := "-"
	if s.TTFT.Count > 0 {
		ttft = msf(s.TTFT.P95) + " ms"
	}
	return []string{
		line("rps", m.rps, rate(s.AchievedRPS)+"/s"),
		line("p95 TTFT", m.ttft, ttft),
	}
}

var sparkLevels = []rune("▁▂▃▄▅▆▇█")

// spark draws values from 0 to their maximum, newest on the right.
func spark(vals []float64, w int) string {
	top := 0.0
	for _, v := range vals {
		top = math.Max(top, v)
	}
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", max(0, w-len(vals))))
	for _, v := range vals {
		i := 0
		if top > 0 {
			i = int(math.Round(v / top * float64(len(sparkLevels)-1)))
		}
		b.WriteRune(sparkLevels[max(0, min(i, len(sparkLevels)-1))])
	}
	return b.String()
}

func (m *Model) footer(w int) []string {
	switch {
	case m.finished:
		return []string{m.st.label.Render("test finished, writing the report")}
	case m.stopping:
		return flow([]string{
			m.st.warn.Render("stopping: waiting for requests in flight, then the report"),
			m.st.label.Render(keys.Force.Help().Key + " " + keys.Force.Help().Desc),
		}, w)
	}
	return []string{m.st.label.Render(keys.Stop.Help().Key + " " + keys.Stop.Help().Desc)}
}

func clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	t := int64(d / time.Second)
	h, mnt, sec := t/3600, (t/60)%60, t%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, mnt, sec)
	}
	return fmt.Sprintf("%d:%02d", mnt, sec)
}

func num(v int64) string {
	s := fmt.Sprintf("%d", v)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func msf(v float64) string {
	if v < 10 {
		return fmt.Sprintf("%.1f", v)
	}
	return num(int64(math.Round(v)))
}

func rate(v float64) string {
	if v < 100 {
		return fmt.Sprintf("%.2f", v)
	}
	return num(int64(math.Round(v)))
}

func pct(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}
