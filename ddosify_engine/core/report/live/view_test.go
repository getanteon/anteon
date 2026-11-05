package live

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

var update = flag.Bool("update", false, "rewrite the golden files")

func newTestModel(w, h int, noColor bool) *Model {
	m := NewModel(make(chan Snapshot), ModelOptions{Thresholds: testThresholds, GoodputTarget: 98, NoColor: noColor})
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	if string(want) != got {
		t.Errorf("%s differs from the golden file\n--- got\n%s\n--- want\n%s", name, got, want)
	}
}

func checkFits(t *testing.T, view string, w, h int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if h > 0 && len(lines) > h {
		t.Errorf("%d lines in a terminal of %d", len(lines), h)
	}
	for i, l := range lines {
		if lw := ansi.StringWidth(l); lw > w {
			t.Errorf("line %d is %d wide in a terminal of %d: %q", i, lw, w, l)
		}
	}
}

func TestGoldenViews(t *testing.T) {
	cases := []struct {
		name string
		w, h int
	}{
		{"wide_100x30", 100, 30},
		{"medium_72x24", 72, 24},
		{"narrow_44x20", 44, 20},
		{"short_100x12", 100, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newTestModel(c.w, c.h, true)
			feed(m, 40)
			v := m.Render()
			checkFits(t, v, c.w, c.h)
			if strings.Contains(v, "\x1b") {
				t.Error("an escape sequence with colour off")
			}
			golden(t, c.name, v)
		})
	}
}

func TestGoldenUnstaged(t *testing.T) {
	m := newTestModel(100, 30, true)
	for i := 0; i < 40; i++ {
		m.Update(snapshotMsg(unstaged(i)))
	}
	v := m.Render()
	checkFits(t, v, 100, 30)
	if strings.Contains(v, "this stage") || strings.Contains(v, "all p50") {
		t.Error("a test without stages labels a stage scope")
	}
	golden(t, "unstaged_100x30", v)
}

func TestGoldenStopping(t *testing.T) {
	m := newTestModel(100, 30, true)
	feed(m, 40)
	m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	golden(t, "stopping_100x30", m.Render())
}

func TestGoldenWaiting(t *testing.T) {
	m := newTestModel(100, 30, true)
	golden(t, "waiting_100x30", m.Render())
}

func TestColourViewFitsAndStyles(t *testing.T) {
	m := newTestModel(100, 30, false)
	feed(m, 40)
	v := m.Render()
	if !strings.Contains(v, "\x1b[") {
		t.Fatal("no styling with colour on")
	}
	checkFits(t, v, 100, 30)
	// The same words carry the meaning with or without colour.
	plain := newTestModel(100, 30, true)
	feed(plain, 40)
	bars := regexp.MustCompile(`[#█░-]{8,}`)
	stripped := bars.ReplaceAllString(ansi.Strip(v), "BAR")
	if want := bars.ReplaceAllString(plain.Render(), "BAR"); stripped != want {
		t.Errorf("colour changes the text\n--- colour, stripped\n%s\n--- plain\n%s", stripped, want)
	}
}

func TestAlarmsCarryText(t *testing.T) {
	m := newTestModel(100, 30, true)
	feed(m, 1)
	v := m.Render()
	for _, want := range []string{"! 429 31", "! 503 4", "! target 98.0% missed", "first token <= 10,000 ms", "total <= 30,000 ms"} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q", want)
		}
	}
}

func TestNarrowDropsPanels(t *testing.T) {
	wide := newTestModel(100, 30, true)
	feed(wide, 10)
	narrow := newTestModel(44, 30, true)
	feed(narrow, 10)
	if !strings.Contains(wide.Render(), "p95 TTFT") {
		t.Error("wide view lacks the sparklines")
	}
	nv := narrow.Render()
	if strings.Contains(nv, "p95 TTFT") {
		t.Error("narrow view keeps the sparklines")
	}
	if strings.Contains(nv, "####") {
		t.Error("narrow view keeps the stage bars")
	}
	for _, want := range []string{"TTFT", "goodput", "429"} {
		if !strings.Contains(nv, want) {
			t.Errorf("narrow view lacks %q", want)
		}
	}
}

func TestShortDropsLeastNeeded(t *testing.T) {
	m := newTestModel(100, 10, true)
	feed(m, 10)
	v := m.Render()
	checkFits(t, v, 100, 10)
	for _, want := range []string{"ddosify", "TTFT", "q stop"} {
		if !strings.Contains(v, want) {
			t.Errorf("short view lacks %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "p95 TTFT") {
		t.Error("short view keeps the sparklines over the core panels")
	}
}

func TestEventGapLabel(t *testing.T) {
	m := newTestModel(100, 30, true)
	s := synthetic(0)
	s.ITLIsEventGap = true
	m.Update(snapshotMsg(s))
	if !strings.Contains(m.Render(), "event gap") {
		t.Error("ITL is not labelled as the event gap")
	}
}

func TestManyStagesWindow(t *testing.T) {
	m := newTestModel(100, 40, true)
	s := synthetic(0)
	s.Stages = nil
	for i := 0; i < 20; i++ {
		s.Stages = append(s.Stages, StagePlan{Duration: 60e9, Count: int64(i + 1)})
	}
	s.Stage.Index, s.Stage.Count = 10, 20
	m.Update(snapshotMsg(s))
	v := m.Render()
	if !strings.Contains(v, "earlier") || !strings.Contains(v, "more") {
		t.Errorf("twenty stages are not windowed:\n%s", v)
	}
	if n := len(regexp.MustCompile(`(?m)^[> ] +\d+ `).FindAllString(v, -1)); n != maxStageRows {
		t.Errorf("%d stage rows, want %d", n, maxStageRows)
	}
}

func TestSpark(t *testing.T) {
	if got := spark([]float64{0, 1, 2, 4}, 6); got != "  ▁▃▅█" {
		t.Errorf("spark = %q", got)
	}
	if got := spark(nil, 3); got != "   " {
		t.Errorf("empty spark = %q", got)
	}
}

func TestFormat(t *testing.T) {
	cases := map[string]string{
		num(0): "0", num(999): "999", num(1000): "1,000", num(1234567): "1,234,567", num(-1200): "-1,200",
		msf(4.24): "4.2", msf(1204.4): "1,204", clock(3*3600e9 + 5e9): "3:00:05", clock(65e9): "1:05",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
