package htmlreport

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"go.ddosify.com/ddosify/v2/core/llm"
	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/types"
)

// stage builds a closed-loop stage: c in flight, answers at user tok/s each, the stage at sys tok/s,
// first token p99 ttft99 ms and time per output token p99 tpot99 ms.
func stage(i, c int, user, sys, ttft99, tpot99 float64) *report.StageSummary {
	return &report.StageSummary{Index: i, Name: "", Concurrency: c, Iterations: 40, OK: 40,
		Stream: &report.StreamSummary{Count: 40,
			FirstToken: &report.Dist{Count: 40, P50: ttft99 / 2, P99: ttft99},
			LLM: &llm.Summary{
				Tokens: &llm.OutputStats{Unit: "tokens", Requests: 40, Total: 40 * 1024, SystemPerSec: sys,
					Output:       &llm.Dist{Count: 40, P50: 1024},
					TimePer:      &llm.Dist{Count: 40, P50: tpot99 * 0.8, P99: tpot99},
					OutputPerSec: &llm.Dist{Count: 40, P50: user}},
				InputTokens: &llm.Dist{Count: 40, P50: 1024},
			}}}
}

func run(stages ...*report.StageSummary) Report {
	return Report{Title: "t", Stages: stages}
}

func TestParetoPointsAndCapacity(t *testing.T) {
	warm := stage(0, 1, 90, 90, 100, 11)
	warm.Warmup = true
	r := run(warm,
		stage(1, 1, 90, 90, 120, 11),
		stage(2, 8, 60, 480, 400, 16),
		stage(3, 32, 35, 1100, 900, 28),
		stage(4, 64, 18, 1150, 2600, 55))
	r.Capacity = &types.Capacity{GPUs: 2}
	s := ParetoOf(r, "A", 0)
	if s == nil || len(s.Points) != 4 || s.GPUs != 2 {
		t.Fatalf("series %+v", s)
	}
	if p := s.Points[1]; p.Stage != "c=8" || *p.PerGPU != 240 || *p.PerUser != 60 || *p.TTFT99 != 400 || *p.TPOT99 != 16 || *p.Input50 != 1024 {
		t.Errorf("point %+v", p)
	}
	conversational := capacity(s, types.DefaultCapacityTiers[0])
	interactive := capacity(s, types.DefaultCapacityTiers[1])
	if conversational == nil || conversational.Concurrency != 32 {
		t.Errorf("conversational tier: %+v, want c=32 (c=64 misses TTFT 2 s)", conversational)
	}
	if interactive == nil || interactive.Concurrency != 8 {
		t.Errorf("interactive tier: %+v, want c=8 (c=32 misses TTFT 500 ms)", interactive)
	}
	if p := capacity(s, types.CapacityTier{Name: "x", FirstTokenMs: 50}); p != nil {
		t.Errorf("no stage meets 50 ms, got %+v", p)
	}
	// an override of the GPUs wins over the config's
	if s2 := ParetoOf(r, "A", 4); *s2.Points[0].PerGPU != 22.5 {
		t.Errorf("per GPU with 4: %v", *s2.Points[0].PerGPU)
	}
}

func TestParetoSkipsRunsWithoutCountedOutput(t *testing.T) {
	plain := &report.StageSummary{Index: 0, Iterations: 3, Concurrency: 2}
	if s := ParetoOf(run(plain), "x", 0); s != nil {
		t.Errorf("a stage without streamed output makes no point: %+v", s)
	}
}

func TestSingleRunShowsCapacity(t *testing.T) {
	var b bytes.Buffer
	if err := Write(&b, run(stage(0, 1, 90, 90, 120, 11), stage(1, 16, 50, 700, 600, 20))); err != nil {
		t.Fatal(err)
	}
	doc := b.String()
	for _, want := range []string{`id="pareto"`, "Capacity", "conversational", "interactive", "TTFT p99 &lt;= 2,000 ms", "c=16", "tok/s per GPU"} {
		if !strings.Contains(doc, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	// an open-loop run of one stage has no trade-off to draw
	b.Reset()
	one := stage(0, 0, 90, 90, 120, 11)
	one.Concurrency = 0
	if err := Write(&b, run(one)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), `id="pareto"`) {
		t.Error("one open-loop stage must not draw the trade-off")
	}
}

func TestWriteCompare(t *testing.T) {
	a := run(stage(0, 1, 90, 90, 120, 11), stage(1, 16, 50, 700, 600, 20))
	b := run(stage(0, 1, 120, 120, 100, 8), stage(1, 16, 70, 950, 450, 14))
	var out bytes.Buffer
	if err := WriteCompare(&out, "A against B", []Labeled{{"model A", a}, {"model B", b}}, 0, nil); err != nil {
		t.Fatal(err)
	}
	doc := out.String()
	for _, want := range []string{"A against B", "2 runs compared", "model A", "model B", `id="pareto"`, `"label":"model B"`} {
		if !strings.Contains(doc, want) {
			t.Errorf("comparison lacks %q", want)
		}
	}
	if strings.Contains(doc, `id="outcomes"`) || strings.Contains(doc, `id="load-section"`) {
		t.Error("a comparison shows no single run's sections")
	}
	doc = strings.ReplaceAll(doc, `"http://www.w3.org/2000/svg"`, "")
	if loc := regexp.MustCompile(`https?://`).FindStringIndex(doc); loc != nil {
		t.Errorf("the comparison loads %q", doc[loc[0]:min(len(doc), loc[1]+30)])
	}
	if err := WriteCompare(&out, "", []Labeled{{"empty", run(&report.StageSummary{})}}, 0, nil); err == nil {
		t.Error("a run with no counted output can not be compared")
	}
}

// Open-loop stages, such as MLPerf's Server scenario's Poisson arrivals: the capacity is the highest
// arrival rate within the tier, with its requests in flight by Little's law.
func TestCapacityOfOpenLoopStages(t *testing.T) {
	mk := func(i int, ttft99, tpot99, meanTotalMs float64) *report.StageSummary {
		s := stage(i, 0, 40, 400, ttft99, tpot99)
		s.Name = []string{"1/s", "2/s", "4/s"}[i]
		s.Stream.Total = &report.Dist{Count: 40, Mean: meanTotalMs}
		return s
	}
	r := run(mk(0, 300, 20, 9000), mk(1, 450, 28, 10000), mk(2, 1500, 60, 14000))
	r.Config.Stages = []StageConfig{{Name: "1/s", DurationS: 60, Count: 60}, {Name: "2/s", DurationS: 60, Count: 120}, {Name: "4/s", DurationS: 60, Count: 240}}
	s := ParetoOf(r, "A", 0)
	inter, conv := capacity(s, types.DefaultCapacityTiers[1]), capacity(s, types.DefaultCapacityTiers[0])
	if inter == nil || inter.Stage != "2/s" || *inter.Rate != 2 || *inter.InFlight != 20 {
		t.Errorf("interactive: %+v", inter)
	}
	if conv == nil || conv.Stage != "4/s" || *conv.Rate != 4 || *conv.InFlight != 56 {
		t.Errorf("conversational: %+v", conv)
	}
	v := buildPareto([]*ParetoSeries{s}, types.DefaultCapacityTiers)
	if c := v.Rows[0].Cells[0]; c.Value != "4.00/s" || !strings.Contains(c.Note, "about 56.0 in flight") {
		t.Errorf("cell %+v", c)
	}
}
