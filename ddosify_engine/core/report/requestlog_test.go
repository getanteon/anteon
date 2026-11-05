package report

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

// runStdoutJson feeds the iterations to a stdout-json report and returns what it wrote, line by line.
func runStdoutJson(t *testing.T, requestLog bool, iterations ...*types.ScenarioResult) []map[string]interface{} {
	t.Helper()
	var buf bytes.Buffer
	realOut, realPrint := out, printJson
	out = &buf
	printJson = func(j []byte) { buf.Write(append(j, '\n')) }
	defer func() { out, printJson = realOut, realPrint }()

	sj := &stdoutJson{}
	sj.Init(false, 1)
	if requestLog {
		sj.SetRequestLog(true)
	}
	input := make(chan *types.ScenarioResult, len(iterations))
	for _, it := range iterations {
		input <- it
	}
	close(input)
	go sj.Start(input, nil)
	<-sj.DoneChan()

	var lines []map[string]interface{}
	sc := bufio.NewScanner(&buf)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var m map[string]interface{}
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("not one JSON object a line: %v: %s", err, sc.Text())
		}
		lines = append(lines, m)
	}
	return lines
}

func TestRequestLog(t *testing.T) {
	t0 := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	load := types.LoadStage{Index: 1, Name: "1/s"}
	ok := llmIteration(load, t0, 2*time.Millisecond, 101)
	ok.Iteration = 7
	ok.StepResults[0].StepName = `Turn "1"` + "\n"
	ok.StepResults[0].Duration = 2 * time.Second
	ok.StepResults[0].Stream.LLM.Input = 42

	refused := iteration(types.LoadStage{Index: 0, Name: "warm-up", Warmup: true},
		&types.ScenarioStepResult{StepID: 1, StatusCode: 503, RequestTime: t0, Duration: 5 * time.Millisecond},
		&types.ScenarioStepResult{StepID: 2, StepName: "no stream", StatusCode: 200, RequestTime: t0.Add(time.Second), Duration: time.Millisecond})
	refused.Iteration = 8

	lines := runStdoutJson(t, true, ok, refused)
	if len(lines) != 4 {
		t.Fatalf("expected 3 request lines and the report, got %d", len(lines))
	}
	for _, l := range lines[:3] {
		if l["type"] != "request" {
			t.Fatalf("expected the request lines first, got %v", l)
		}
	}
	if lines[3]["type"] != "report" {
		t.Errorf("the report should be the last line: %v", lines[3])
	}

	r := lines[0]
	want := map[string]interface{}{
		"type": "request", "stage": "1/s", "stage_index": 1.0, "warmup": false, "iteration": 7.0,
		"step_id": 1.0, "step_name": "Turn \"1\"\n", "status_code": 200.0, "outcome": OutcomeOK,
		"first_token_ms": 300.0, "total_ms": 2000.0, "output_tokens": 101.0, "input_tokens": 42.0,
		"intended_start": t0.Format(time.RFC3339Nano),
		"sent":           t0.Add(2 * time.Millisecond).Format(time.RFC3339Nano),
		"time":           t0.Add(2*time.Millisecond + 2*time.Second).Format(time.RFC3339Nano),
	}
	for k, v := range want {
		if r[k] != v {
			t.Errorf("%s: expected %v, got %v", k, v, r[k])
		}
	}
	if len(r) != len(want) {
		t.Errorf("unexpected fields: %v", r)
	}

	r = lines[1]
	if r["outcome"] != OutcomeUnavailable || r["status_code"] != 503.0 || r["warmup"] != true || r["stage"] != "warm-up" || r["iteration"] != 8.0 {
		t.Errorf("the 503: %v", r)
	}
	for _, k := range []string{"first_token_ms", "output_tokens", "input_tokens", "intended_start"} {
		if _, has := r[k]; has {
			t.Errorf("%s does not apply to a plain request: %v", k, r)
		}
	}
	if lines[2]["step_id"] != 2.0 || lines[2]["outcome"] != OutcomeOK {
		t.Errorf("the second step: %v", lines[2])
	}
}

func TestRequestLogOff(t *testing.T) {
	lines := runStdoutJson(t, false, llmIteration(types.LoadStage{Index: 1, Name: "x"}, time.Now(), 0, 11))
	if len(lines) != 1 || lines[0]["type"] != "report" {
		t.Errorf("without request_log only the report is written: %v", lines)
	}
}

func TestRequestLogChunksAndInterrupted(t *testing.T) {
	res := llmIteration(types.LoadStage{}, time.Now(), 0, 11)
	res.StepResults[0].Stream.LLM.OutputSource = types.TokenSourceChunks
	cut := iteration(types.LoadStage{}, &types.ScenarioStepResult{StepID: 1, RequestTime: time.Now(),
		Err: types.RequestError{Type: types.ErrorIntented, Reason: types.ReasonCtxCanceled}})
	cut.Others[types.OthersInterrupted] = true
	lines := runStdoutJson(t, true, res, cut)
	if _, has := lines[0]["output_tokens"]; has || lines[0]["output_chunks"] != 11.0 {
		t.Errorf("chunks are not tokens: %v", lines[0])
	}
	if lines[1]["outcome"] != OutcomeInterrupted {
		t.Errorf("an interrupted request: %v", lines[1])
	}
}

func TestRequestLogAllocations(t *testing.T) {
	l := &requestLog{buf: make([]byte, 0, 512)}
	res := llmIteration(types.LoadStage{Index: 1, Name: "1/s"}, time.Now(), time.Millisecond, 101)
	l.write(io.Discard, res)
	if n := testing.AllocsPerRun(100, func() { l.write(io.Discard, res) }); n != 0 {
		t.Errorf("a request line should not allocate, got %v allocations", n)
	}
}

func TestAppendJSONString(t *testing.T) {
	for _, s := range []string{"", "plain", `q"uo\te`, "tab\tnew\nline\r\x01\x1f", "Türkçe ğüşiöç 日本 🙂", "bad \xff utf8", "<html>&"} {
		var got string
		b := appendJSONString(nil, s)
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("%q: %v: %s", s, err, b)
		}
		want := strings.ToValidUTF8(s, "�")
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	}
}
