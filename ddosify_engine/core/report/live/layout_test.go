package live

import (
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func lineWith(t *testing.T, view, prefix string) string {
	t.Helper()
	for _, l := range strings.Split(view, "\n") {
		if strings.HasPrefix(ansi.Strip(l), prefix) {
			return l
		}
	}
	t.Fatalf("no line starts with %q in\n%s", prefix, view)
	return ""
}

// colOf is the display column where re first matches in the line.
func colOf(line string, re *regexp.Regexp) int {
	plain := ansi.Strip(line)
	loc := re.FindStringIndex(plain)
	if loc == nil {
		return -1
	}
	return ansi.StringWidth(plain[:loc[0]])
}

func TestLatencyColumnsAreFixedWidthWithoutTabs(t *testing.T) {
	for _, noColor := range []bool{true, false} {
		m := newTestModel(110, 34, noColor)
		feed(m, 5)
		v := m.Render()
		if strings.Contains(v, "\t") {
			t.Fatal("a tab in the view")
		}
		for _, row := range []string{"TTFT", "ITL", "E2E"} {
			l := ansi.Strip(lineWith(t, v, row))
			if w := ansi.StringWidth(l); w != labelCol+5*valueCol {
				t.Errorf("%s row is %d wide, want %d: %q", row, w, labelCol+5*valueCol, l)
			}
		}
	}
}

func TestStageBarsShareColumns(t *testing.T) {
	m := newTestModel(110, 34, true)
	s := synthetic(0)
	s.Stages = []StagePlan{
		{Name: "ısınma", Warmup: true, Duration: 60e9},
		{Name: "60/min", Duration: 60e9},
		{Name: "a much longer stage name", Duration: 60e9},
		{Name: "x", Duration: 60e9},
	}
	s.Stage.Index, s.Stage.Count = 2, 4
	m.Update(snapshotMsg(s))
	v := m.Render()
	bar := regexp.MustCompile(`[#-]{8,}`)
	starts, ends := map[int]bool{}, map[int]bool{}
	for _, l := range strings.Split(ansi.Strip(v), "\n") {
		if loc := bar.FindStringIndex(l); loc != nil && regexp.MustCompile(`^[> ] +\d `).MatchString(l) {
			starts[ansi.StringWidth(l[:loc[0]])] = true
			ends[ansi.StringWidth(l[:loc[1]])] = true
		}
	}
	if len(starts) != 1 || len(ends) != 1 {
		t.Fatalf("bars start at %v and end at %v:\n%s", starts, ends, v)
	}
}

func TestSparklinesFitWithValueAtRightEdge(t *testing.T) {
	for _, w := range []int{80, 100, 110, 300} {
		m := newTestModel(w, 40, true)
		feed(m, historyLen+10)
		v := m.Render()
		for _, prefix := range []string{"rps", "p95 TTFT"} {
			l := ansi.Strip(lineWith(t, v, prefix+"  "))
			if got := ansi.StringWidth(l); got != w {
				t.Errorf("width %d: %s line is %d wide", w, prefix, got)
			}
			if strings.HasSuffix(l, " ") {
				t.Errorf("width %d: %s value is not at the right edge: %q", w, prefix, l)
			}
		}
	}
}

func TestSparkGapWhenNoSamples(t *testing.T) {
	if got := spark([]float64{1, math.NaN(), 2}, 3); got != "▅ █" {
		t.Errorf("spark = %q", got)
	}
	m := newTestModel(100, 30, true)
	for i := 0; i < 6; i++ {
		s := synthetic(i)
		if i == 3 {
			s.TTFT = Latency{} // a snapshot with no first token yet, as at a stage's start
		}
		m.Update(snapshotMsg(s))
	}
	got := m.ttft.last(6)
	if !math.IsNaN(got[3]) {
		t.Fatalf("no-sample snapshot stored as %v", got[3])
	}
	l := ansi.Strip(lineWith(t, m.Render(), "p95 TTFT"))
	if !regexp.MustCompile(`[▁-█]{3} [▁-█]{2}`).MatchString(l) {
		t.Errorf("no gap where the snapshot had no samples: %q", l)
	}
}

func TestStageAndRunAreLabelled(t *testing.T) {
	m := newTestModel(110, 34, true)
	feed(m, 1)
	v := m.Render()
	for _, want := range []string{
		"this stage", "all p50", "all p99",
		"this stage 93.4% (1,120 of 1,199)", "all 93.3% (3,900 of 4,180)",
		"outcomes this stage  ok 1,120", "all         ok 4,100",
		"started 1,234",
	} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q:\n%s", want, v)
		}
	}
}

func TestOutcomeRowsShareColumns(t *testing.T) {
	m := newTestModel(110, 34, true)
	s := synthetic(0)
	s.Outcomes.OK = 5
	m.Update(snapshotMsg(s))
	v := m.Render()
	stage := lineWith(t, v, "outcomes this stage")
	all := lineWith(t, v, "         all")
	for _, re := range []string{`! 429`, `! 503`, `5xx`, `timeout`, `cut`} {
		r := regexp.MustCompile(re)
		if a, b := colOf(stage, r), colOf(all, r); a != b || a < 0 {
			t.Errorf("%s at column %d in the stage row and %d in the run row:\n%s\n%s", re, a, b, stage, all)
		}
	}
}
