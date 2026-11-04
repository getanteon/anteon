package report

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func stageFixture(names ...string) []*StageSummary {
	var out []*StageSummary
	for i, n := range names {
		g := 95.0
		out = append(out, &StageSummary{
			Index: i, Name: n, Warmup: i == 0, Iterations: 12000, OK: 11000, GoodputPerc: &g,
			Outcomes: map[string]int64{OutcomeRateLimited: 1234, OutcomeUnavailable: 567, OutcomeServerError: 89,
				OutcomeTimeout: 1000, OutcomeNetworkError: 3, OutcomeStreamCut: 4},
			Stream: &StreamSummary{FirstToken: &Dist{Count: 1, P50: 12345, P95: 23456}, Total: &Dist{Count: 1, P50: 123456, P95: 234567}},
		})
	}
	return out
}

// small gives the figures of a short run, as in the demo: hundreds of iterations, seconds of latency.
func small(stages []*StageSummary) []*StageSummary {
	for _, st := range stages {
		st.Iterations, st.OK = 240, 216
		st.Outcomes = map[string]int64{OutcomeRateLimited: 14, OutcomeUnavailable: 8, OutcomeTimeout: 2}
		st.Stream = &StreamSummary{FirstToken: &Dist{Count: 1, P50: 378, P95: 1379}, Total: &Dist{Count: 1, P50: 1046, P95: 1051}}
	}
	return stages
}

func TestStageTableFitsAndAligns(t *testing.T) {
	cases := map[string][]*StageSummary{
		"short names": small(stageFixture("warm-up", "60/min", "120/min")),
		"long names":  stageFixture("warm-up of the whole system", "sixty requests a minute, steady", "ısınma sonrası"),
	}
	for name, stages := range cases {
		t.Run(name, func(t *testing.T) {
			var b strings.Builder
			printStages(&b, stages, true)
			lines := strings.Split(strings.TrimSpace(b.String()), "\n")[1:]
			if len(lines) != len(stages)+1 {
				t.Fatalf("%d lines:\n%s", len(lines), b.String())
			}
			end := -1
			for _, l := range lines {
				if strings.Contains(l, "\t") {
					t.Errorf("a tab: %q", l)
				}
				n := utf8.RuneCountInString(l)
				if n > stageTableWidth {
					t.Errorf("%d columns: %q", n, l)
				}
				if end >= 0 && n != end {
					t.Errorf("rows end at different columns, %d and %d:\n%s", end, n, b.String())
				}
				end = n
			}
			if name == "long names" && !strings.Contains(lines[0], "Other") {
				t.Errorf("the error columns did not fold:\n%s", b.String())
			}
			if name == "short names" && !strings.Contains(lines[0], "Timeout") {
				t.Errorf("folded with room to spare:\n%s", b.String())
			}
		})
	}
}

func TestStageTableOnANarrowTerminal(t *testing.T) {
	defer SetWidth(0)
	for _, width := range []int{80, 70} {
		SetWidth(width)
		var b strings.Builder
		printStages(&b, small(stageFixture("warm-up", "60/min", "120/min")), true)
		lines := strings.Split(strings.TrimSpace(b.String()), "\n")[1:]
		for _, l := range lines {
			if n := utf8.RuneCountInString(l); n > width {
				t.Errorf("width %d: %d columns: %q", width, n, l)
			}
		}
		if width == 70 {
			if len(lines) != 6 {
				t.Fatalf("want two lines a stage:\n%s", b.String())
			}
			for _, want := range []string{"1 warm-up", "429 14", "TTFT p50 378", "Total p50 1046"} {
				if !strings.Contains(b.String(), want) {
					t.Errorf("lacks %q:\n%s", want, b.String())
				}
			}
		}
	}
}
