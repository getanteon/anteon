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
			section{prioLatency, m.latency(w)},
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
		m.kv("started", num(s.Iterations)),
	}
	return append(flow(first, w), flow(second, w)...)
}

// padRight and padLeft pad by display width, so styled or wide text keeps its column.
func padRight(str string, w int) string {
	return str + strings.Repeat(" ", max(0, w-lipgloss.Width(str)))
}

func padLeft(str string, w int) string {
	return strings.Repeat(" ", max(0, w-lipgloss.Width(str))) + str
}

const (
	labelCol = 12
	valueCol = 10
	minAllW  = labelCol + 5*valueCol
)

func cell(l Latency, v float64) string {
	if l.Count == 0 {
		return "-"
	}
	return msf(v)
}

// latency is a table of fixed-width, right-aligned columns: the current stage's p50, p90 and p99,
// then the whole run's p50 and p99 when the test has stages and the terminal has room.
func (m *Model) latency(w int) []string {
	s := m.last
	itl := "ITL"
	if s.ITLIsEventGap {
		itl = "event gap"
	}
	all := s.All != nil && w >= minAllW
	title := "latency ms"
	if s.All != nil {
		title = "this stage"
	}
	headCells := []string{"p50", "p90", "p99"}
	if all {
		headCells = append(headCells, "all p50", "all p99")
	}
	head := padRight(title, labelCol)
	for _, c := range headCells {
		head += padLeft(c, valueCol)
	}
	row := func(name string, l Latency, a *Latency) string {
		line := padRight(name, labelCol)
		for _, v := range []float64{l.P50, l.P90, l.P99} {
			line += padLeft(cell(l, v), valueCol)
		}
		if all {
			if a == nil {
				line += padLeft("", 2*valueCol)
			} else {
				line += padLeft(cell(*a, a.P50), valueCol) + padLeft(cell(*a, a.P99), valueCol)
			}
		}
		return m.st.value.Render(strings.TrimRight(line, " "))
	}
	var sc Scope
	if s.All != nil {
		sc = *s.All
	}
	ref := func(l Latency) *Latency {
		if s.All == nil {
			return nil
		}
		return &l
	}
	lines := []string{m.st.label.Render(head), row("TTFT", s.TTFT, ref(sc.TTFT))}
	if t := s.Tokens; t != nil && t.Unit == "tokens" {
		lines = append(lines, row("TPOT", t.TPOT, ref(sc.TPOT)))
	}
	lines = append(lines, row(itl, s.ITL, ref(sc.ITL)), row("E2E", s.E2E, ref(sc.E2E)))
	if s.SchedulingLag.Count > 0 {
		lines = append(lines, row("sched lag", s.SchedulingLag, nil))
	}
	return append(lines, m.tokens()...)
}

// tokens is the output line: in tokens when counted in tokens, else in chunks, never mixed up.
func (m *Model) tokens() []string {
	t := m.last.Tokens
	if t == nil {
		return nil
	}
	unit := t.Unit
	per := "-"
	if t.Output.Count > 0 {
		per = num(int64(math.Round(t.Output.P50)))
	}
	parts := []string{
		m.kv("output", fmt.Sprintf("%s %s/s", rate(t.PerSec), unit)),
		m.kv("per answer p50", per+" "+unit),
	}
	if pc := m.last.PrefixCache; pc != "" {
		parts = append(parts, m.kv("prefix cache", pc))
	}
	lines := []string{strings.Join(parts, "   ")}
	if unit != "tokens" {
		lines = append(lines, m.st.label.Render("chunks are content events; give a tokenizer to count tokens"))
	}
	return lines
}

func (m *Model) goodputText(g Goodput) string {
	if g.Total == 0 {
		return "-"
	}
	return fmt.Sprintf("%s (%s of %s)", pct(g.Perc()), num(g.Good), num(g.Total))
}

// goodput shows the current stage's and, with stages, the run's; the target is checked on the run.
func (m *Model) goodput(w int) []string {
	s := m.last
	run := s.Goodput
	if s.All != nil {
		run = s.All.Goodput
	}
	if len(m.thresholds) == 0 && run.Total == 0 && s.Goodput.Total == 0 {
		return []string{m.kv("goodput", "-") + "  " + m.st.label.Render("no SLO set")}
	}
	first := []string{m.st.label.Render("goodput")}
	if s.All != nil {
		first = append(first, m.kv("this stage", m.goodputText(s.Goodput)), m.kv("all", m.goodputText(run)))
	} else {
		if run.Total == 0 {
			first = append(first, m.st.value.Render("-"), m.st.label.Render("no iteration finished yet"))
		} else {
			first = append(first, m.st.value.Render(pct(run.Perc())),
				m.st.label.Render(fmt.Sprintf("%s of %s iterations met the SLO", num(run.Good), num(run.Total))))
		}
	}
	if m.target > 0 && run.Total > 0 {
		t := "target " + pct(m.target)
		if run.Perc() >= m.target {
			first = append(first, m.st.good.Render(t+" met"))
		} else {
			first = append(first, m.st.bad.Render("! "+t+" missed"))
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

type outcomeItem struct {
	label  string
	v      int64
	alarm  bool // 429 and 503: a "!" and the share, so the warning does not rest on colour
	always bool
}

func items(o Outcomes) []outcomeItem {
	return []outcomeItem{
		{"ok", o.OK, false, true},
		{"429", o.RateLimited, true, true},
		{"503", o.Unavailable, true, true},
		{"5xx", o.ServerError, false, true},
		{"timeout", o.Timeout, false, true},
		{"cut", o.StreamCut, false, true},
		{"4xx", o.ClientError, false, false},
		{"network", o.NetworkError, false, false},
		{"error event", o.StreamError, false, false},
		{"interrupted", o.Interrupted, false, false},
	}
}

// outcomeChunks renders the outcomes; show[i] false leaves item i out.
func (m *Model) outcomeChunks(o Outcomes, show []bool) []string {
	total := o.Total()
	var out []string
	for i, it := range items(o) {
		if !show[i] {
			continue
		}
		switch {
		case it.v == 0 || (it.label == "ok"):
			out = append(out, m.kv(it.label, num(it.v)))
		case it.alarm:
			share := ""
			if total > 0 {
				share = " (" + pct(float64(it.v)*100/float64(total)) + ")"
			}
			out = append(out, m.st.bad.Render(fmt.Sprintf("! %s %s%s", it.label, num(it.v), share)))
		default:
			out = append(out, m.st.warn.Render(fmt.Sprintf("%s %s", it.label, num(it.v))))
		}
	}
	return out
}

func shown(scopes ...Outcomes) []bool {
	show := make([]bool, len(items(Outcomes{})))
	for _, o := range scopes {
		for i, it := range items(o) {
			show[i] = show[i] || it.always || it.v > 0
		}
	}
	return show
}

// outcomes is one line for a test without stages, else one for this stage and one for the run,
// their items in shared columns when both fit on a line.
func (m *Model) outcomes(w int) []string {
	s := m.last
	if s.All == nil {
		show := shown(s.Outcomes)
		return flow(append([]string{m.st.label.Render("outcomes")}, m.outcomeChunks(s.Outcomes, show)...), w)
	}
	show := shown(s.Outcomes, s.All.Outcomes)
	stage, all := m.outcomeChunks(s.Outcomes, show), m.outcomeChunks(s.All.Outcomes, show)
	const gap = "   "
	widths := make([]int, len(stage))
	lineW := 0
	for i := range stage {
		widths[i] = max(lipgloss.Width(stage[i]), lipgloss.Width(all[i]))
		lineW += widths[i] + len(gap)
	}
	prefixW, stageLabel, allLabel := 21, "outcomes this stage", "         all"
	if w < 100 {
		prefixW, stageLabel, allLabel = 7, "stage", "all"
	}
	if prefixW+lineW-len(gap) <= w {
		row := func(label string, chunks []string) string {
			line := m.st.label.Render(padRight(label, prefixW))
			for i, c := range chunks {
				if i > 0 {
					line += gap
				}
				line += padRight(c, widths[i])
			}
			return strings.TrimRight(line, " ")
		}
		out := []string{row(stageLabel, stage), row(allLabel, all)}
		if w < 100 {
			out = append([]string{m.st.label.Render("outcomes")}, out...)
		}
		return out
	}
	line := func(label string, chunks []string) []string {
		out := flow(chunks, max(20, w-prefixW))
		for i := range out {
			if i == 0 {
				out[i] = m.st.label.Render(padRight(label, prefixW)) + out[i]
			} else {
				out[i] = strings.Repeat(" ", prefixW) + out[i]
			}
		}
		return out
	}
	out := line(stageLabel, stage)
	if w < 100 {
		out = append([]string{m.st.label.Render("outcomes")}, out...)
	}
	return append(out, line(allLabel, all)...)
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
		case i == cur && m.finished && p.Concurrency > 0:
			if s.Stage.Planned > 0 {
				frac = math.Min(1, float64(s.Stage.Done)/float64(s.Stage.Planned))
			}
			status = fmt.Sprintf("stopped at %.0f%%", frac*100)
		case i == cur && m.finished:
			if p.Duration > 0 {
				frac = math.Min(1, float64(s.Stage.Elapsed)/float64(p.Duration))
			}
			status = fmt.Sprintf("stopped at %.0f%%", frac*100)
		case i == cur && p.Concurrency > 0:
			marker = ">"
			if s.Stage.Planned > 0 {
				frac = math.Min(1, float64(s.Stage.Done)/float64(s.Stage.Planned))
			}
			status = fmt.Sprintf("%3.0f%%  %d / %d in %s", frac*100, s.Stage.Done, s.Stage.Planned, clock(s.Stage.Elapsed))
		case i == cur:
			marker = ">"
			if p.Duration > 0 {
				frac = math.Min(1, float64(s.Stage.Elapsed)/float64(p.Duration))
			}
			status = fmt.Sprintf("%3.0f%%  %s / %s", frac*100, clock(s.Stage.Elapsed), clock(p.Duration))
		default:
			status = "queued"
		}
		row := fmt.Sprintf("%s %2d ", marker, i+1) + padRight(ansi.Truncate(stageName(p), nameW, ""), nameW)
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
	switch {
	case n == "" && p.Concurrency > 0:
		n = fmt.Sprintf("%d at concurrency %d", p.Count, p.Concurrency)
	case n == "":
		n = fmt.Sprintf("%d in %s", p.Count, clock(p.Duration))
	}
	if p.Warmup && !strings.Contains(strings.ToLower(n), "warm") {
		n += " (warm-up)"
	}
	return n
}

// sparks draw the history in what the label and the value leave, the value ending at the right edge.
func (m *Model) sparks(w int) []string {
	const labelW, valueW = 10, 12
	sw := min(w-labelW-1-valueW, historyLen)
	line := func(label string, r *ring, value string) string {
		return m.st.label.Render(padRight(label, labelW)) +
			m.st.good.Render(spark(r.last(sw), sw)) + " " + m.st.value.Render(padLeft(value, w-labelW-sw-1))
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

// spark draws values from 0 to their maximum, newest on the right. NaN, a snapshot with no
// samples, is a gap rather than a zero.
func spark(vals []float64, w int) string {
	top := 0.0
	for _, v := range vals {
		if !math.IsNaN(v) {
			top = math.Max(top, v)
		}
	}
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", max(0, w-len(vals))))
	for _, v := range vals {
		if math.IsNaN(v) {
			b.WriteByte(' ')
			continue
		}
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
