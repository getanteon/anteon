package htmlreport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/html"

	"go.ddosify.com/ddosify/v2/core/assertion"
	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/report/live"
	"go.ddosify.com/ddosify/v2/core/types"
)

var secrets = []string{
	"sk-live-HEADERSECRET", "QUERYSECRET", "BODYSECRET", "PASSSECRET", "COOKIESECRET",
	"ENVSECRET", "ASSERTSECRET", "CRITSECRET", "CSVSECRET", "llm.bank.internal", "Authorization", "Bearer",
}

func secretHammer() *types.Hammer {
	slo := &types.SLO{FirstTokenMs: 10000, TotalMs: 30000}
	return &types.Hammer{
		IterationCount: 1290,
		LoadType:       "waved",
		TestDuration:   1620,
		LoadSpread:     types.LoadSpreadEven,
		SLO:            slo,
		TimeRunCountMap: types.TimeRunCount{
			{Duration: 120, Count: 6, Name: "warm-up", Warmup: true},
			{Duration: 300, Count: 30, Name: "6/min"},
			{Duration: 300, Count: 60, Name: "12/min"},
		},
		Cookies:      []types.CustomCookie{{Name: "session", Value: "COOKIESECRET"}},
		TestDataConf: map[string]types.CsvConf{"q": {Path: "/data/CSVSECRET.csv"}},
		Scenario: types.Scenario{
			Envs: map[string]interface{}{"KEY": "ENVSECRET"},
			Steps: []types.ScenarioStep{
				{
					ID: 1, Name: "create chat", Method: "POST",
					URL:     "https://llm.bank.internal/v1/chats?api_key=QUERYSECRET",
					Headers: map[string]string{"Authorization": "Bearer sk-live-HEADERSECRET"},
					Payload: `{"title":"BODYSECRET"}`,
					Auth:    types.Auth{Type: "basic", Username: "user", Password: "PASSSECRET"},
					Timeout: 10,
				},
				{
					ID: 2, Name: "ask <question>", Method: "POST",
					URL:        "https://llm.bank.internal/v1/answer",
					Headers:    map[string]string{"Authorization": "Bearer sk-live-HEADERSECRET"},
					Payload:    `{"messages":[{"role":"user","content":"BODYSECRET"}]}`,
					Assertions: []string{`equals(headers.X, "ASSERTSECRET")`},
					Stream:     types.DefaultStreamConf(),
					Timeout:    120,
				},
			},
		},
	}
}

func dist(p50, p95 float64) *report.Dist {
	return &report.Dist{Count: 1100, Min: p50 / 3, Mean: p50 * 1.1, P50: p50, P90: p95 * 0.9, P95: p95, P99: p95 * 1.3, Max: p95 * 2}
}

func sampleResult() *report.Result {
	g1, g2 := int64(29), int64(52)
	p1, p2 := 96.7, 86.7
	stream := &report.StreamSummary{
		Count: 1100, Ends: map[string]int64{"done": 1090, "cut": 10},
		TTFB: dist(180, 420), FirstEvent: dist(240, 600), FirstToken: dist(1204, 4500), Total: dist(9870, 19000),
		GapMax: dist(300, 900), GapP50: dist(41, 60), GapP95: dist(120, 210), Events: dist(220, 400),
	}
	return &report.Result{
		TestStatus:           "failed",
		TestFailedAssertions: []assertion.FailedRule{{Rule: `p95(iteration_duration) < 1 CRITSECRET`}},
		SuccessCount:         1120,
		StepResults: map[uint16]*report.ScenarioStepResultSummary{
			2: {Name: "ask <question>", SuccessCount: 1040, Fail: report.FailVerbose{Count: 60},
				Outcomes: map[string]int64{report.OutcomeOK: 1040, report.OutcomeRateLimited: 31, report.OutcomeUnavailable: 4,
					report.OutcomeServerError: 2, report.OutcomeTimeout: 3, report.OutcomeStreamCut: 10}, Stream: stream},
			1: {Name: "create chat", SuccessCount: 1090, Outcomes: map[string]int64{report.OutcomeOK: 1090}},
		},
		Goodput: &report.Goodput{SLO: types.SLO{FirstTokenMs: 10000, TotalMs: 30000}, Iterations: 1090, Good: 1013, Perc: 92.9},
		Stages: []*report.StageSummary{
			{Index: 0, Name: "warm-up", Warmup: true, Iterations: 6, OK: 6, Outcomes: map[string]int64{report.OutcomeOK: 12}, Stream: stream},
			{Index: 1, Name: "6/min", Iterations: 30, OK: 30, Good: &g1, GoodputPerc: &p1, Outcomes: map[string]int64{report.OutcomeOK: 60}, Stream: stream},
			{Index: 2, Name: "12/min", Iterations: 60, OK: 55, Good: &g2, GoodputPerc: &p2,
				Outcomes: map[string]int64{report.OutcomeOK: 110, report.OutcomeRateLimited: 31, report.OutcomeUnavailable: 4}, Stream: stream},
		},
	}
}

func sampleRecorder(n int) *Recorder {
	rec := NewRecorder(0)
	for i := 0; i < n; i++ {
		stage := 0
		if i > n/5 {
			stage = 1
		}
		if i > n*3/5 {
			stage = 2
		}
		f := float64(i)
		rec.Add(live.Snapshot{
			Elapsed:       time.Duration(i) * 250 * time.Millisecond,
			Stage:         live.StageRef{Index: stage, Count: 3},
			TargetRPS:     []float64{0.05, 0.1, 0.2}[stage],
			AchievedRPS:   []float64{0.05, 0.1, 0.2}[stage] * (0.9 + 0.1*float64(i%7)/7),
			InFlight:      int64(stage*4 + i%3),
			TTFT:          live.Latency{Count: int64(i), P50: 900 + f, P90: 1500 + 2*f, P95: 1800 + 2*f, P99: 2500 + 3*f},
			ITL:           live.Latency{Count: int64(i), P50: 40, P90: 80, P95: 95, P99: 150},
			E2E:           live.Latency{Count: int64(i), P50: 8000 + f, P90: 12000, P95: 14000, P99: 19000},
			Goodput:       live.Goodput{Good: int64(i * 9 / 10), Total: int64(i)},
			ITLIsEventGap: true,
		})
	}
	return rec
}

func render(t *testing.T) string {
	t.Helper()
	rep := FromResult(sampleResult(), ConfigFromHammer(secretHammer()), sampleRecorder(6000))
	rep.GeneratedAt = time.Date(2026, 11, 6, 15, 0, 0, 0, time.UTC)
	var b bytes.Buffer
	if err := Write(&b, rep); err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("HTMLREPORT_SAMPLE_DIR"); dir != "" {
		_ = os.WriteFile(filepath.Join(dir, "sample-report.html"), b.Bytes(), 0o644)
	}
	return b.String()
}

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true,
	"input": true, "link": true, "meta": true, "source": true, "track": true, "wbr": true,
}

// checkBalanced fails on a tag closed out of order or left open.
func checkBalanced(t *testing.T, doc string) {
	t.Helper()
	z := html.NewTokenizer(strings.NewReader(doc))
	var stack []string
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() != io.EOF {
				t.Fatal(z.Err())
			}
			if len(stack) > 0 {
				t.Fatalf("left open: %v", stack)
			}
			return
		case html.StartTagToken:
			name, _ := z.TagName()
			if !voidElements[string(name)] {
				stack = append(stack, string(name))
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			if len(stack) == 0 || stack[len(stack)-1] != string(name) {
				t.Fatalf("</%s> closes %v", name, stack)
			}
			stack = stack[:len(stack)-1]
		}
	}
}

func TestReportIsValidHTML(t *testing.T) {
	doc := render(t)
	if !strings.HasPrefix(doc, "<!doctype html>") {
		t.Fatal("no doctype")
	}
	checkBalanced(t, doc)
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	ids := map[string]int{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			name := n.Data
			if name == "title" && n.Parent != nil && n.Parent.Data != "head" {
				name = "svg title"
			}
			counts[name]++
			for _, a := range n.Attr {
				if a.Key == "id" {
					ids[a.Val]++
				}
				if a.Key == "data-series" {
					var v map[string]any
					if err := json.Unmarshal([]byte(a.Val), &v); err != nil {
						t.Errorf("chart data is not JSON: %v", err)
					}
				}
				if a.Key == "style" && strings.Contains(a.Val, "ZgotmplZ") {
					t.Errorf("template refused a style value: %q", a.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	for _, el := range []string{"html", "head", "body", "title", "main"} {
		if counts[el] != 1 {
			t.Errorf("%d <%s>", counts[el], el)
		}
	}
	for id, n := range ids {
		if n > 1 {
			t.Errorf("id %q used %d times", id, n)
		}
	}
	if counts["svg"] < 6 {
		t.Errorf("%d svg elements, want the charts", counts["svg"])
	}
	for _, want := range []string{"Goodput", "92.9%", "429 rate limited", "503 unavailable", "6/min", "12/min",
		"Time to first token", "Event gap", "first token &lt;= 10,000 ms", "ask &lt;question&gt;", "1 success criteria failed"} {
		if !strings.Contains(doc, want) {
			t.Errorf("report lacks %q", want)
		}
	}
}

func TestReportLoadsNothing(t *testing.T) {
	doc := render(t)
	for _, re := range []string{`https?://`, `(?i)<link\b`, `(?i)\bsrc\s*=`, `(?i)@import`, `(?i)url\(`, `(?i)<iframe`, `(?i)\bhref\s*=\s*"[^#]`, `(?i)fetch\(|XMLHttpRequest|WebSocket|sendBeacon`} {
		if loc := regexp.MustCompile(re).FindStringIndex(doc); loc != nil {
			t.Errorf("%s found: %q", re, doc[max(0, loc[0]-40):min(len(doc), loc[1]+40)])
		}
	}
	if !strings.Contains(doc, `default-src 'none'`) {
		t.Error("no content security policy")
	}
}

func TestReportHoldsNoSecrets(t *testing.T) {
	doc := render(t)
	for _, s := range secrets {
		if strings.Contains(doc, s) {
			t.Errorf("report contains %q", s)
		}
	}
}

func TestRecorderIsBounded(t *testing.T) {
	rec := NewRecorder(100)
	for i := 0; i < 100000; i++ {
		rec.Add(live.Snapshot{Elapsed: time.Duration(i) * 250 * time.Millisecond})
	}
	pts := rec.Points()
	if len(pts) > 101 || len(pts) < 50 {
		t.Fatalf("%d points kept, cap 100", len(pts))
	}
	if last := pts[len(pts)-1].T; last != (time.Duration(99999) * 250 * time.Millisecond).Seconds() {
		t.Fatalf("series ends at %v, not at the last snapshot", last)
	}
	for i := 1; i < len(pts); i++ {
		if pts[i].T <= pts[i-1].T {
			t.Fatalf("series not in time order at %d", i)
		}
	}
	steps := map[float64]bool{}
	for i := 1; i < len(pts)-1; i++ {
		steps[pts[i].T-pts[i-1].T] = true
	}
	if len(steps) != 1 {
		t.Fatalf("uneven spacing: %v", steps)
	}
}

func TestRecorderRace(t *testing.T) {
	rec := NewRecorder(64)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				rec.Add(live.Snapshot{Elapsed: time.Duration(i) * time.Millisecond})
				if i%100 == 0 {
					_ = rec.Points()
				}
			}
		}()
	}
	wg.Wait()
	if n := len(rec.Points()); n == 0 || n > 65 {
		t.Fatalf("%d points", n)
	}
}

func TestEmptyReport(t *testing.T) {
	var b bytes.Buffer
	if err := Write(&b, FromResult(nil, Config{}, nil)); err != nil {
		t.Fatal(err)
	}
	checkBalanced(t, b.String())
	if !strings.Contains(b.String(), "No SLO was set") {
		t.Error("empty report does not say there was no SLO")
	}
}

func TestWriteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.html")
	if err := WriteFile(path, FromResult(sampleResult(), Config{}, nil)); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows has no Unix permission bits; Go reports every writable file as 0666 there.
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, want 0600", fi.Mode().Perm())
	}
}

func TestAxisNum(t *testing.T) {
	cases := []struct {
		v, step float64
		want    string
	}{{0, 1, "0"}, {0.05, 0.05, "0.05"}, {0.15, 0.05, "0.15"}, {6250, 6250, "6,250"}, {2.5, 2.5, "2.5"}, {25000, 6250, "25,000"}}
	for _, c := range cases {
		if got := axisNum(c.v, c.step); got != c.want {
			t.Errorf("axisNum(%v, %v) = %q, want %q", c.v, c.step, got, c.want)
		}
	}
}

func TestChartGapWhereNoSamples(t *testing.T) {
	rec := NewRecorder(0)
	for i := 0; i < 9; i++ {
		l := live.Latency{Count: int64(i + 1), P50: 1000, P90: 1100, P99: 1200}
		if i == 4 {
			l = live.Latency{} // a stage just began and nothing has finished in it
		}
		rec.Add(live.Snapshot{Elapsed: time.Duration(i) * time.Second, E2E: l, TTFT: l})
	}
	rep := FromResult(nil, Config{}, rec)
	var e2e *chart
	charts := buildCharts(rep, "ITL")
	for i := range charts {
		if charts[i].ID == "e2e" {
			e2e = &charts[i]
		}
	}
	if e2e == nil {
		t.Fatal("no E2E chart")
	}
	path := regexp.MustCompile(`<path class="line s0"[^>]* d="([^"]*)"`).FindStringSubmatch(string(e2e.SVG))
	if path == nil {
		t.Fatal("no p50 path")
	}
	if n := strings.Count(path[1], "M"); n != 2 {
		t.Errorf("p50 path has %d segments, want a break where there were no samples: %s", n, path[1])
	}
	if strings.Contains(path[1], fmt.Sprintf(" %.1f", float64(padT+plotH))) {
		t.Errorf("a point drawn at zero: %s", path[1])
	}
	var data struct {
		S []struct{ V []float64 }
	}
	if err := json.Unmarshal([]byte(e2e.Data), &data); err != nil {
		t.Fatal(err)
	}
	if data.S[0].V[4] != -1 {
		t.Errorf("hover data for the empty snapshot is %v, want the no-value marker", data.S[0].V[4])
	}
}

func TestRequestsCardSaysPerIteration(t *testing.T) {
	doc := render(t)
	if !strings.Contains(doc, ">Requests<") || !strings.Contains(doc, "2 per iteration, warm-up excluded") {
		t.Error("the requests card does not say it counts both steps")
	}
	if strings.Contains(doc, "Step results") {
		t.Error("the old label is still there")
	}
}
