package htmlreport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	elive "go.ddosify.com/ddosify/v2/core/live"
)

func parseAll(t *testing.T, csv string) ([]Sample, ParseStats, error) {
	t.Helper()
	var got []Sample
	st, err := ParseSamples(strings.NewReader(csv), func(s Sample) { got = append(got, s) })
	return got, st, err
}

func TestParseSamples(t *testing.T) {
	csv := "\xef\xbb\xbftimestamp,source,metric,value\n" +
		"1762601234.5,masking-cpu-0,cpu_cores,3.9\n" +
		"2026-11-08T12:00:01Z,masking-cpu-0,cpu_pct,391.2%\n" + // percent of one core
		"2026-11-08 12:00:02.5,gateway-7c9f,mem_bytes,2120Mi\n" + // no zone is UTC, a space for the T
		"1762601234500,gpu0,gpu_util_pct,87\n" + // milliseconds
		"1762601235,gpu0,gpu_mem_bytes,2.5GiB\n" +
		"1762601235,proxy,cpu_cores,250m\n" + // kubectl's millicores
		"1762601235,proxy,queue_depth,-3\n" + // unknown: kept, sign and all
		"\n# a comment\n" +
		"not a time,gpu0,gpu_util_pct,1\n" +
		"1762601235,,cpu_cores,1\n" +
		"1762601235,gpu0,gpu_util_pct,--\n" +
		"1762601235,gpu0,gpu_util_pct\n" +
		"1762601235,gpu0,mem_bytes,-5\n" + // a known metric is never negative
		"\"1762601236\", \"quoted pod\" ,cpu_cores, 1.5\r\n"
	got, st, err := parseAll(t, csv)
	if err != nil {
		t.Fatal(err)
	}
	if st.Bad != 5 || st.FirstBad != 11 || st.Samples != 8 {
		t.Errorf("stats %+v, want 5 bad from line 11 and 8 samples", st)
	}
	want := []Sample{
		{1762601234.5, "masking-cpu-0", MetricCPUCores, 3.9},
		{float64(time.Date(2026, 11, 8, 12, 0, 1, 0, time.UTC).Unix()), "masking-cpu-0", MetricCPUCores, 3.912},
		{float64(time.Date(2026, 11, 8, 12, 0, 2, 0, time.UTC).Unix()) + 0.5, "gateway-7c9f", MetricMem, 2120 << 20},
		{1762601234.5, "gpu0", MetricGPUUtil, 87},
		{1762601235, "gpu0", MetricGPUMem, 2.5 * (1 << 30)},
		{1762601235, "proxy", MetricCPUCores, 0.25},
		{1762601235, "proxy", "queue_depth", -3},
		{1762601236, "quoted pod", MetricCPUCores, 1.5},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d samples: %+v", len(got), got)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Source != w.Source || g.Metric != w.Metric || !near(g.At, w.At) || !near(g.Value, w.Value) {
			t.Errorf("sample %d: got %+v, want %+v", i, g, w)
		}
	}
}

func TestParseSamplesHeader(t *testing.T) {
	// columns in any order
	got, _, err := parseAll(t, "value,metric,source,timestamp\n2,cpu_cores,a,1762601234\n")
	if err != nil || len(got) != 1 || got[0].Source != "a" || got[0].Value != 2 {
		t.Errorf("reordered header: %+v, %v", got, err)
	}
	// no header: the default order
	if got, _, err = parseAll(t, "1762601234,a,cpu_cores,2\n"); err != nil || len(got) != 1 {
		t.Errorf("no header: %+v, %v", got, err)
	}
	if _, _, err = parseAll(t, "timestamp,source,value\n1,a,2\n"); err == nil || !strings.Contains(err.Error(), `"metric"`) {
		t.Errorf("a header without metric: %v", err)
	}
	for _, empty := range []string{"", "timestamp,source,metric,value\n", "garbage\nmore garbage\n"} {
		if _, _, err := parseAll(t, empty); !errors.Is(err, errNoSamples) {
			t.Errorf("%q: %v, want no samples", empty, err)
		}
	}
}

// fixtureStart is when the run of testdata/run.ndjson began: its first progress line is at 0.25 s.
var fixtureStart = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

func TestRunSpanFromTheLog(t *testing.T) {
	rep := readFixture(t)
	if !rep.StartedAt.Equal(fixtureStart) {
		t.Errorf("start %v, want %v", rep.StartedAt, fixtureStart)
	}
	// the last progress line is at 655.25 s, the last answer ends at 715.652 s
	if d := rep.EndedAt.Sub(rep.StartedAt); d != 715652*time.Millisecond {
		t.Errorf("run of %v", d)
	}
	// Without progress lines the report line's started_at places the run.
	log := `{"type":"request","time":"2026-01-01T09:00:30Z","intended_start":"2026-01-01T09:00:10Z","sent":"2026-01-01T09:00:10Z","step_id":1,"step_name":"a","outcome":"ok","total_ms":20000}
{"type":"report","test_status":"success","started_at":"2026-01-01T09:00:05Z","steps":{"1":{"name":"a"}}}`
	r, err := ReadLog(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if want := fixtureStart.Add(5 * time.Second); !r.StartedAt.Equal(want) || !r.EndedAt.Equal(fixtureStart.Add(30*time.Second)) {
		t.Errorf("span %v to %v, want %v to 09:00:30", r.StartedAt, r.EndedAt, want)
	}
	// and without it, the first request's intended start
	r, _ = ReadLog(strings.NewReader(strings.Replace(log, `"started_at":"2026-01-01T09:00:05Z",`, "", 1)))
	if !r.StartedAt.Equal(fixtureStart.Add(10 * time.Second)) {
		t.Errorf("start from the requests %v", r.StartedAt)
	}
	r, _ = ReadLog(strings.NewReader(`{"type":"report","test_status":"success"}`))
	if _, err := LoadResources(r, []elive.ResourceLog{{Name: "x.csv", In: strings.NewReader("1,a,cpu_cores,1\n")}}, nil, nil); !errors.Is(err, ErrNoTimeline) {
		t.Errorf("a log with no time: %v", err)
	}
}

// fixtureCSV is a sampler's file for the fixture's run: a CPU pod at 1, 2 and 4 cores in the three
// stages, a second pod at half that, a GPU, and samples well before and after the run.
func fixtureCSV() string {
	var b strings.Builder
	b.WriteString("timestamp,source,metric,value\n")
	start := float64(fixtureStart.Unix())
	for s := -120; s <= 800; s++ {
		t := start + float64(s) + 0.5
		cores := 1.0
		switch {
		case s >= 360:
			cores = 4
		case s >= 60:
			cores = 2
		}
		fmt.Fprintf(&b, "%.1f,masking-cpu-0,cpu_pct,%g\n", t, cores*100)
		fmt.Fprintf(&b, "%.1f,gateway-7c9f,cpu_cores,%g\n", t, cores/2)
		fmt.Fprintf(&b, "%.1f,masking-cpu-0,mem_bytes,%d\n", t, (1<<30)+s*1024)
		fmt.Fprintf(&b, "%.1f,gpu0,gpu_util_pct,%d\n", t, 40+s%20)
		fmt.Fprintf(&b, "%.1f,gpu0,gpu_mem_bytes,%dMi\n", t, 20000)
	}
	b.WriteString("oops\n")
	return b.String()
}

func TestLoadResourcesPerStage(t *testing.T) {
	rep := readFixture(t)
	var stderr bytes.Buffer
	res, err := LoadResources(rep, []elive.ResourceLog{{Name: "pods.csv", In: strings.NewReader(fixtureCSV())}}, []string{"masking"}, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stages) != 3 {
		t.Fatalf("%d stages", len(res.Stages))
	}
	wantCores := []float64{1, 2, 4}
	for i, st := range res.Stages {
		if len(st.Sources) != 3 {
			t.Fatalf("stage %d: sources %+v", i, st.Sources)
		}
		gw, gpu, m := st.Sources[0], st.Sources[1], st.Sources[2] // by name
		if m.Source != "masking-cpu-0" || gw.Source != "gateway-7c9f" || gpu.Source != "gpu0" {
			t.Fatalf("stage %d: order %s %s %s", i, gw.Source, gpu.Source, m.Source)
		}
		if !near(m.CPU.Mean(), wantCores[i]) || !near(m.CPU.Max, wantCores[i]) || !near(gw.CPU.Mean(), wantCores[i]/2) {
			t.Errorf("stage %d: masking mean %v peak %v, gateway %v", i, m.CPU.Mean(), m.CPU.Max, gw.CPU.Mean())
		}
		if m.Mem.N == 0 || gpu.GPU.N == 0 || gpu.GPUMem.Max != 20000<<20 || gpu.GPU.Max > 59 || gpu.CPU.N != 0 {
			t.Errorf("stage %d: memory %+v, gpu %+v", i, m.Mem, gpu)
		}
		if st.Warmup != (i == 0) {
			t.Errorf("stage %d warm-up %v", i, st.Warmup)
		}
	}
	// the second stage, 60.25 s to 360.25 s: 300 samples of each series, each on the half second
	st := res.Stages[1]
	if st.Sources[2].CPU.N != 300 || st.OK != 87 || !near(st.Seconds, 300) {
		t.Errorf("stage 2: %d samples, %d ok, %v s", st.Sources[2].CPU.N, st.OK, st.Seconds)
	}
	if peak := st.Sources[2].Mem.Max; peak != float64((1<<30)+359*1024) {
		t.Errorf("stage 2 memory peak %v", peak)
	}
	f := res.Files[0]
	if f.Bad != 1 || f.Outside == 0 || f.Inside == 0 || f.Read != f.Inside+f.Outside {
		t.Errorf("file stats %+v", f)
	}
	// 15 s of margin on each side of 0 to 715.652 s are drawn; the rest is left out, and said once
	if msg := stderr.String(); strings.Count(msg, "\n") != 1 || !strings.Contains(msg, "pods.csv: 1 lines could not be read") || !strings.Contains(msg, "outside the run") {
		t.Errorf("stderr %q", msg)
	}
	for _, s := range res.Series {
		if s.T[0] < -15 || s.T[len(s.T)-1] > 715.652+15+1 {
			t.Errorf("%s %s drawn from %v to %v", s.Source, s.Metric, s.T[0], s.T[len(s.T)-1])
		}
	}
	if res.Series[0].Metric != MetricCPUCores || res.Series[len(res.Series)-1].Metric != MetricGPUMem {
		t.Errorf("series order: %s first, %s last", res.Series[0].Metric, res.Series[len(res.Series)-1].Metric)
	}
}

func TestLoadResourcesOutsideTheRun(t *testing.T) {
	rep := readFixture(t)
	var stderr bytes.Buffer
	res, err := LoadResources(rep, []elive.ResourceLog{{Name: "late.csv", In: strings.NewReader("1900000000,a,cpu_cores,1\n")}}, nil, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Series) != 0 || !strings.Contains(stderr.String(), "none of its 1 samples falls within the run") {
		t.Errorf("series %d, stderr %q", len(res.Series), stderr.String())
	}
	if _, err := LoadResources(rep, []elive.ResourceLog{{Name: "empty.csv", In: strings.NewReader("")}}, nil, nil); err == nil || !strings.Contains(err.Error(), "empty.csv") {
		t.Errorf("an empty file: %v", err)
	}
}

func TestResourcesAreBounded(t *testing.T) {
	// a ten hour soak sampled every second: each series keeps at most DefaultMaxPoints points
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rep := Report{StartedAt: start, EndedAt: start.Add(10 * time.Hour)}
	acc := newResAcc(float64(start.Unix()), float64(rep.EndedAt.Unix()), spansOf(rep, float64(start.Unix()), float64(rep.EndedAt.Unix())))
	for s := 0; s < 36000; s++ {
		acc.add(Sample{At: float64(start.Unix() + int64(s)), Source: "a", Metric: MetricCPUCores, Value: float64(s % 10)})
	}
	res := acc.result()
	if n := len(res.Series[0].T); n > DefaultMaxPoints || n < DefaultMaxPoints-10 {
		t.Errorf("%d points", n)
	}
	if st := res.Stages[0].Sources[0].CPU; st.N != 36000 || st.Max != 9 || !near(st.Mean(), 4.5) {
		t.Errorf("whole run stats %+v", st)
	}
	// the slices keep the mean
	for _, v := range res.Series[0].V {
		if v == nil || math.Abs(*v-4.5) > 0.5 {
			t.Fatalf("a slice's mean %v", v)
		}
	}
	// more series than the bound are left out, not kept
	for i := 0; i < maxResourceSeries+5; i++ {
		acc.add(Sample{At: float64(start.Unix()), Source: fmt.Sprint("pod", i), Metric: MetricMem, Value: 1})
	}
	if n := len(acc.series); n != maxResourceSeries {
		t.Errorf("%d series kept", n)
	}
}

func TestResourceSeriesBreaksAtAGap(t *testing.T) {
	start := 1_000_000.0
	acc := newResAcc(start, start+600, []stageSpan{{from: start, to: start + 600}})
	for s := 0; s <= 600; s += 10 {
		if s > 200 && s < 400 { // the pod was gone
			continue
		}
		acc.add(Sample{At: start + float64(s), Source: "a", Metric: MetricCPUCores, Value: 1})
	}
	sr := acc.result().Series[0]
	nils := 0
	for i, v := range sr.V {
		if v == nil {
			nils++
			if sr.T[i] < 200 || sr.T[i] > 400 {
				t.Errorf("break at %v", sr.T[i])
			}
		}
	}
	if nils != 1 || sr.Step != 10 {
		t.Errorf("%d breaks, step %v", nils, sr.Step)
	}
}

func TestPerCore(t *testing.T) {
	if v := PerCore(600, 60, 2.5); v == nil || !near(*v, 4) {
		t.Errorf("600 ok in 60 s on 2.5 cores: %v, want 4 a second a core", deref(v))
	}
	if v := PerCore(0, 60, 2); v == nil || *v != 0 {
		t.Errorf("no ok request: %v, want 0", deref(v))
	}
	for _, c := range [][2]float64{{0, 2}, {60, 0}, {60, math.NaN()}} {
		if v := PerCore(10, c[0], c[1]); v != nil {
			t.Errorf("%v s on %v cores: %v, want none", c[0], c[1], *v)
		}
	}
	src := []ResSource{
		{Source: "masking-cpu-0", CPU: ResStat{N: 2, Sum: 6, Max: 4}},
		{Source: "masking-cpu-1", CPU: ResStat{N: 1, Sum: 2, Max: 2}},
		{Source: "gateway-7c9f", CPU: ResStat{N: 1, Sum: 1, Max: 1}},
		{Source: "gpu0", GPU: ResStat{N: 1, Sum: 90, Max: 90}}, // no CPU: never counted
	}
	if sum, n := CoresOf(src, nil); sum != 6 || n != 3 {
		t.Errorf("every CPU source: %v cores of %d", sum, n)
	}
	if sum, n := CoresOf(src, []string{"masking"}); sum != 5 || n != 2 {
		t.Errorf("masking: %v cores of %d", sum, n)
	}
	if sum, n := CoresOf(src, []string{"gateway", "cpu-1"}); sum != 3 || n != 2 {
		t.Errorf("two filters: %v cores of %d", sum, n)
	}
}

// The section is there only with samples, and a report without them is the same page as before.
func TestResourcesSection(t *testing.T) {
	rep := readFixture(t)
	rep.GeneratedAt = time.Date(2026, 11, 6, 15, 0, 0, 0, time.UTC)
	var plain, empty bytes.Buffer
	if err := Write(&plain, rep); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain.String(), `id="resources`) || strings.Contains(dataJSON(t, plain.String()), `"res`) {
		t.Error("a report without samples has a resources section")
	}
	rep2 := rep
	rep2.Resources = nil
	if err := Write(&empty, rep2); err != nil || empty.String() != plain.String() {
		t.Error("rendering is not stable")
	}

	res, err := LoadResources(rep, []elive.ResourceLog{{Name: "pods.csv", In: strings.NewReader(fixtureCSV())}}, []string{"masking"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rep.Resources = res
	var b bytes.Buffer
	if err := Write(&b, rep); err != nil {
		t.Fatal(err)
	}
	doc := b.String()
	checkBalanced(t, doc)
	for _, want := range []string{`id="resources-section"`, "CPU mean (cores)", "GPU memory peak", "masking-cpu-0", "gateway-7c9f",
		"OK requests/s per core", `the CPU sources whose name holds &#34;masking&#34;`, "19.53 GiB"} {
		if !strings.Contains(doc, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	var d chartData
	if err := json.Unmarshal([]byte(dataJSON(t, doc)), &d); err != nil {
		t.Fatal(err)
	}
	if d.Res == nil || len(d.Res.Series) != 5 || d.Res.Stages != nil || d.Res.End < 655 {
		t.Fatalf("chart data %+v", d.Res)
	}
	// Stage 2 (6/min): 87 ok requests over 300 s on the masking pod's 2 cores
	v := buildResView([]labeledRes{{res: res}})
	if r := v.Rates[1]; r.Stage != "6/min" || r.Rate != "0.29" || r.Cores != "2.00" || r.Sources != "1" || r.PerCore != "0.14" {
		t.Errorf("rate row %+v", r)
	}
	if !v.Rates[0].Warmup || !v.Rows[0].First || v.Rows[1].First {
		t.Errorf("rows %+v", v.Rows[:2])
	}
}

func TestCompareWithResources(t *testing.T) {
	a, b := readFixture(t), readFixture(t)
	res, err := LoadResources(a, []elive.ResourceLog{{Name: "a.csv", In: strings.NewReader(fixtureCSV())}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.Resources = res
	// the fixture has no counted output, so give each a closed-loop stage with tokens
	for _, r := range []*Report{&a, &b} {
		r.Stages = append(r.Stages, stage(3, 8, 30, 600, 900, 40))
	}
	var out bytes.Buffer
	if err := WriteCompare(&out, "", []Labeled{{Label: "cpu", Report: a}, {Label: "gpu", Report: b}}, 1, nil); err != nil {
		t.Fatal(err)
	}
	doc := out.String()
	if dir := os.Getenv("HTMLREPORT_SAMPLE_DIR"); dir != "" {
		_ = os.WriteFile(filepath.Join(dir, "sample-compare-resources.html"), out.Bytes(), 0o644)
	}
	checkBalanced(t, doc)
	var d chartData
	if err := json.Unmarshal([]byte(dataJSON(t, doc)), &d); err != nil {
		t.Fatal(err)
	}
	if len(d.ResRuns) != 1 || d.ResRuns[0].Label != "cpu" || len(d.ResRuns[0].Stages) == 0 {
		t.Fatalf("resource runs %+v", d.ResRuns)
	}
	if !strings.Contains(doc, `<th class="name">Run</th><th class="name">Stage</th><th class="name">Source</th>`) || !strings.Contains(doc, `<td class="run">cpu</td>`) {
		t.Error("the comparison's resource tables have no run column")
	}
	// without samples the comparison has no resources section
	out.Reset()
	b.Resources = nil
	if err := WriteCompare(&out, "", []Labeled{{Label: "x", Report: b}, {Label: "y", Report: b}}, 1, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `id="resources`) {
		t.Error("a comparison without samples has a resources section")
	}
}
