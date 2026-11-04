package llm

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

type chatBody struct {
	Model         string                 `json:"model"`
	Messages      []ChatMessage          `json:"messages"`
	Stream        bool                   `json:"stream"`
	MaxTokens     int                    `json:"max_tokens"`
	StreamOptions map[string]interface{} `json:"stream_options"`
	IgnoreEOS     bool                   `json:"ignore_eos"`
	Seed          float64                `json:"seed"`
}

func presetSteps(t *testing.T, p Preset) []types.ScenarioStep {
	t.Helper()
	steps, err := p.Steps()
	if err != nil {
		t.Fatal(err)
	}
	return steps
}

// runTurn prepares the turn's body, feeds it an answer and carries its variables into envs.
func runTurn(t *testing.T, st types.ScenarioStep, envs map[string]interface{}, answer string) chatBody {
	t.Helper()
	s, err := NewStep(st.LLM)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Prepare(nil, envs)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range p.Vars {
		envs[k] = v
	}
	var b chatBody
	if err := json.Unmarshal(p.Body, &b); err != nil {
		t.Fatalf("%s: %v", p.Body, err)
	}
	p.Observer.Event(`{"choices":[{"delta":{"content":`+string(mustJSON(answer))+`}}]}`, time.Millisecond)
	p.Observer.Finish(time.Now())
	for k, v := range p.Observer.Vars() {
		envs[k] = v
	}
	return b
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestChatSessionCarriesHistory(t *testing.T) {
	steps := presetSteps(t, Preset{
		URL: "http://x/v1/chat/completions", Model: "m", MaxTokens: 64, IgnoreEOS: true,
		Prompts:   []string{"first question"},
		Dataset:   nil,
		Turns:     3,
		ThinkTime: "100-200",
		CacheBust: true,
		ExtraBody: map[string]interface{}{"seed": 7},
	})
	if len(steps) != 3 || steps[0].Sleep != "100-200" || steps[2].Sleep != "" || steps[0].Timeout != defaultPresetTimeout {
		t.Fatalf("steps %+v", steps)
	}
	envs := map[string]interface{}{}
	b1 := runTurn(t, steps[0], envs, "answer one")
	b2 := runTurn(t, steps[1], envs, "answer two")
	b3 := runTurn(t, steps[2], envs, "answer three")

	if b1.Model != "m" || !b1.Stream || b1.MaxTokens != 64 || b1.StreamOptions["include_usage"] != true || !b1.IgnoreEOS || b1.Seed != 7 {
		t.Errorf("body %+v", b1)
	}
	if len(b1.Messages) != 1 || !strings.HasPrefix(b1.Messages[0].Content, "[rid:") || !strings.HasSuffix(b1.Messages[0].Content, "first question") {
		t.Fatalf("turn 1 messages %+v", b1.Messages)
	}
	first := b1.Messages[0]
	// one prompt only, so the second turn borrows the first turn of the next session, the same prompt
	want2 := []ChatMessage{first, {"assistant", "answer one"}, {"user", "first question"}}
	if !reflect.DeepEqual(b2.Messages, want2) {
		t.Errorf("turn 2 messages %+v, want %+v", b2.Messages, want2)
	}
	if len(b3.Messages) != 5 || b3.Messages[0] != first || b3.Messages[3] != (ChatMessage{"assistant", "answer two"}) {
		t.Errorf("turn 3 messages %+v", b3.Messages)
	}

	// a new session, a new marker
	other := runTurn(t, steps[0], map[string]interface{}{}, "x")
	if other.Messages[0].Content == first.Content {
		t.Error("two sessions share a marker")
	}
}

func TestChatSessionsFromDataset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.jsonl")
	os.WriteFile(path, []byte(`{"turns":["s1 t1","s1 t2"]}
{"messages":[{"role":"system","content":"sys"},{"role":"user","content":"s2 t1"},{"role":"assistant","content":"a"},{"role":"user","content":"s2 t2"}]}
`), 0o600)
	steps := presetSteps(t, Preset{URL: "http://x", Dataset: &Dataset{Path: path}, Turns: 2, SystemPrompt: "Be brief.", SharedPrefixTokens: 4})
	for _, want := range [][]string{{"s1 t1", "s1 t2"}, {"s2 t1", "s2 t2"}, {"s1 t1", "s1 t2"}} {
		envs := map[string]interface{}{}
		b1 := runTurn(t, steps[0], envs, "a1")
		b2 := runTurn(t, steps[1], envs, "a2")
		prefix, _, _ := SharedPrefix(nil, 4)
		if b1.Messages[0] != (ChatMessage{"system", prefix + "\n\nBe brief."}) {
			t.Errorf("system %+v", b1.Messages[0])
		}
		if b1.Messages[1].Content != want[0] || b2.Messages[len(b2.Messages)-1].Content != want[1] {
			t.Errorf("session turns %q, %q; want %v", b1.Messages[1].Content, b2.Messages[len(b2.Messages)-1].Content, want)
		}
	}
}

func TestChatBodyEscapesNothingItNeedNot(t *testing.T) {
	steps := presetSteps(t, Preset{URL: "http://x", Prompts: []string{"a <b> & c, Türkçe: şğı"}})
	s, _ := NewStep(steps[0].LLM)
	p, _ := s.Prepare(nil, map[string]interface{}{})
	if !bytes.Contains(p.Body, []byte(`"a <b> & c, Türkçe: şğı"`)) {
		t.Errorf("body %s", p.Body)
	}
}

func TestDatasets(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(content), 0o600)
		return p
	}
	cases := []struct {
		d    Dataset
		want [][]string
	}{
		{Dataset{Path: write("a.txt", "one\n\ntwo\r\n")}, [][]string{{"one"}, {"two"}}},
		{Dataset{Path: write("b.jsonl", `{"prompt":"p1"}`+"\n"+`{"conversations":[{"from":"human","value":"h1"},{"from":"gpt","value":"g"},{"from":"human","value":"h2"}]}`+"\n")},
			[][]string{{"p1"}, {"h1", "h2"}}},
		{Dataset{Path: write("c.jsonl", `{"q":"x","prompt":"ignored"}`+"\n"), Field: "q"}, [][]string{{"x"}}},
		{Dataset{Path: write("d.csv", "id,question,follow_up\n1,\"Limit, please?\",And fees?\n2,Hello,\n"), Fields: []string{"question", "follow_up"}},
			[][]string{{"Limit, please?", "And fees?"}, {"Hello"}}},
		{Dataset{Path: write("e.csv", "x,first\ny,second\n"), Field: "1"}, [][]string{{"first"}, {"second"}}},
		{Dataset{Path: write("f.txt", "1\n2\n3\n"), Limit: 2}, [][]string{{"1"}, {"2"}}},
	}
	for _, c := range cases {
		got, err := c.d.Load()
		if err != nil {
			t.Errorf("%s: %v", c.d.Path, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %q, want %q", c.d.Path, got, c.want)
		}
	}
	if _, err := (&Dataset{Path: write("bad.jsonl", "{}\n")}).Load(); err == nil {
		t.Error("a row with no prompt must be refused")
	}
}

func TestAggSeparatesTokensAndChunks(t *testing.T) {
	t0 := time.Now()
	var a Agg
	a.Add(&types.LLMResult{SentAt: t0, Output: 100, OutputSource: types.TokenSourceUsage, HasContent: true,
		FirstContent: 100 * time.Millisecond, LastContent: time.Second, TPOT: 9 * time.Millisecond, HasTPOT: true, OutputPerSec: 100,
		ITL: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}, Input: 50, InputSource: types.TokenSourceUsage})
	a.Add(&types.LLMResult{SentAt: t0.Add(time.Second), Output: 300, OutputSource: types.TokenSourceTokenizer, HasContent: true,
		FirstContent: 200 * time.Millisecond, LastContent: time.Second, TPOT: 3 * time.Millisecond, HasTPOT: true, OutputPerSec: 300})
	a.Add(&types.LLMResult{SentAt: t0, Output: 7, OutputSource: types.TokenSourceChunks, HasContent: true, LastContent: time.Second, TPOT: 150 * time.Millisecond, HasTPOT: true, OutputPerSec: 7})
	a.Add(&types.LLMResult{SentAt: t0, OutputSource: types.TokenSourceChunks})
	a.Add(nil)
	s := a.Summary()
	if s.Tokens == nil || s.Tokens.Requests != 2 || s.Tokens.Total != 400 || s.Tokens.Unit != "tokens" {
		t.Fatalf("tokens %+v", s.Tokens)
	}
	// 400 tokens from the first send to the last answer's end, 2 seconds
	if s.Tokens.SystemPerSec != 200 || s.Tokens.WindowSeconds != 2 {
		t.Errorf("system rate %v over %v", s.Tokens.SystemPerSec, s.Tokens.WindowSeconds)
	}
	if s.Tokens.TimePer.P50 != 3 || s.Tokens.TimePer.Max != 9 {
		t.Errorf("tpot %+v", s.Tokens.TimePer)
	}
	if s.Chunks == nil || s.Chunks.Requests != 2 || s.Chunks.Total != 7 || s.Chunks.Unit != "chunks" || s.NoContent != 1 {
		t.Errorf("chunks %+v, no content %d", s.Chunks, s.NoContent)
	}
	if s.ITL.Count != 2 || s.InputTokens.Count != 1 {
		t.Errorf("itl %+v input %+v", s.ITL, s.InputTokens)
	}
	var b bytes.Buffer
	WriteText(&b, s)
	WriteSystem(&b, s, nil)
	out := b.String()
	for _, want := range []string{"TPOT (ms)", "Output Chunks", "Time/Chunk (ms)", "200.0 tokens/s", "chunks/s", "not tokens"} {
		if !strings.Contains(out, want) {
			t.Errorf("report misses %q:\n%s", want, out)
		}
	}
}

func TestScheduleLag(t *testing.T) {
	var a ScheduleAgg
	t0 := time.Now()
	for i := 0; i < 99; i++ {
		a.Add(&types.ScenarioResult{Others: map[string]interface{}{types.OthersIntendedStart: t0},
			StepResults: []*types.ScenarioStepResult{{RequestTime: t0.Add(time.Millisecond)}}})
	}
	if s := a.Summary(); s.Saturated || s.Late != 0 || s.Lag.P99 != 1 {
		t.Errorf("on time: %+v %+v", s, s.Lag)
	}
	for i := 0; i < 5; i++ {
		a.Add(&types.ScenarioResult{Others: map[string]interface{}{types.OthersIntendedStart: t0},
			StepResults: []*types.ScenarioStepResult{{RequestTime: t0.Add(time.Second)}}})
	}
	s := a.Summary()
	if !s.Saturated || s.Late != 5 || s.Iterations != 104 {
		t.Errorf("late: %+v", s)
	}
	var b bytes.Buffer
	WriteSchedule(&b, s)
	if !strings.Contains(b.String(), "WARNING: the client could not keep the schedule") {
		t.Errorf("no warning: %s", b.String())
	}
	a.Add(&types.ScenarioResult{}) // no schedule, ignored
}
