package llm

import (
	"bufio"
	"os"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

// events reads the data of each event of an SSE fixture, the done marker left out.
func events(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if d := strings.TrimPrefix(sc.Text(), "data: "); d != sc.Text() && d != "[DONE]" {
			out = append(out, d)
		}
	}
	return out
}

// feed gives the observer one event every 10ms, the first at 100ms.
func feed(o *Observer, evs []string) {
	for i, e := range evs {
		o.Event(e, 100*time.Millisecond+time.Duration(i)*10*time.Millisecond)
	}
}

func newObserver(t *testing.T, conf *types.LLMConf) *Observer {
	t.Helper()
	s, err := NewStep(conf)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Prepare(nil, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	return p.Observer
}

func TestUsageFromRecordedStreams(t *testing.T) {
	cases := []struct {
		file          string
		output, input int
		text          string
		contentEvents int
	}{
		{"openai_chat.sse", 12, 12, "The user greets.Hello! How can I help?", 4},
		{"aisdk_gurubase.sse", 412, 6210, "The standard retail overdraft limit is 5,000 lira [p1].", 3},
		{"anthropic.sse", 15, 25, "Hello there", 2},
		{"responses.sse", 2, 9, "Hi you", 2},
	}
	for _, c := range cases {
		o := newObserver(t, &types.LLMConf{})
		feed(o, events(t, "testdata/"+c.file))
		r := o.Finish(time.Now())
		if r.OutputSource != types.TokenSourceUsage || r.Output != c.output || r.Input != c.input || r.InputSource != types.TokenSourceUsage {
			t.Errorf("%s: output %d (%s), input %d (%s); want %d and %d from usage", c.file, r.Output, r.OutputSource, r.Input, r.InputSource, c.output, c.input)
		}
		if o.Text() != c.text || r.ContentEvents != c.contentEvents {
			t.Errorf("%s: text %q in %d events, want %q in %d", c.file, o.Text(), r.ContentEvents, c.text, c.contentEvents)
		}
	}
}

func TestTPOT(t *testing.T) {
	o := newObserver(t, &types.LLMConf{})
	// content at 100, 150 and 400ms; usage says 11 tokens: (400 - 100) / (11 - 1) = 30ms
	o.Event(`{"choices":[{"delta":{"content":"a"}}]}`, 100*time.Millisecond)
	o.Event(`{"choices":[{"delta":{"content":"b"}}]}`, 150*time.Millisecond)
	o.Event(`{"choices":[{"delta":{"content":"c"}}]}`, 400*time.Millisecond)
	o.Event(`{"choices":[],"usage":{"prompt_tokens":5,"completion_tokens":11}}`, 410*time.Millisecond)
	r := o.Finish(time.Now())
	if !r.HasTPOT || r.TPOT != 30*time.Millisecond {
		t.Errorf("TPOT %v, want 30ms", r.TPOT)
	}
	if r.FirstContent != 100*time.Millisecond || r.LastContent != 400*time.Millisecond {
		t.Errorf("first %v last %v", r.FirstContent, r.LastContent)
	}
	if want := 11 / 0.4; r.OutputPerSec < want-0.001 || r.OutputPerSec > want+0.001 {
		t.Errorf("output per second %v, want %v", r.OutputPerSec, want)
	}
	if len(r.ITL) != 2 || r.ITL[0] != 50*time.Millisecond || r.ITL[1] != 250*time.Millisecond {
		t.Errorf("ITL %v, want [50ms 250ms]", r.ITL)
	}

	// one token has no TPOT
	o = newObserver(t, &types.LLMConf{})
	o.Event(`{"choices":[{"delta":{"content":"a"}}]}`, 100*time.Millisecond)
	o.Event(`{"choices":[],"usage":{"completion_tokens":1}}`, 110*time.Millisecond)
	if r = o.Finish(time.Now()); r.HasTPOT {
		t.Errorf("one token must not have a TPOT: %v", r.TPOT)
	}
}

const tinyTok = `{"added_tokens": [], "normalizer": null,
 "pre_tokenizer": {"type": "ByteLevel", "add_prefix_space": false, "use_regex": true},
 "model": {"type": "BPE", "vocab": {"a": 0, "b": 1, "Ġ": 2, "Ġa": 3, "ab": 4}, "merges": ["Ġ a", "a b"]}}`

func writeTok(t *testing.T) string {
	t.Helper()
	p := t.TempDir() + "/tokenizer.json"
	if err := os.WriteFile(p, []byte(tinyTok), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSourceFallbacks(t *testing.T) {
	stream := []string{
		`{"type":"text-delta","delta":"ab"}`,
		`{"type":"text-delta","delta":" a"}`,
		`{"type":"text-delta","delta":" ab"}`,
	}
	tok := writeTok(t)

	// no usage, no tokenizer: chunks, never tokens
	o := newObserver(t, &types.LLMConf{})
	feed(o, stream)
	r := o.Finish(time.Now())
	if r.OutputSource != types.TokenSourceChunks || r.Output != 3 || r.Tokens() {
		t.Errorf("want 3 chunks, got %d %s", r.Output, r.OutputSource)
	}
	if r.HasTPOT && r.Tokens() {
		t.Error("chunks must not give a TPOT in tokens")
	}

	// a tokenizer: "ab", " a", " a", "b" = ab, Ġa, Ġa, b
	o = newObserver(t, &types.LLMConf{Tokenizer: tok})
	feed(o, stream)
	if r = o.Finish(time.Now()); r.OutputSource != types.TokenSourceTokenizer || r.Output != 4 {
		t.Errorf("want 4 tokens from the tokenizer, got %d %s", r.Output, r.OutputSource)
	}

	// usage wins over the tokenizer in auto, and loses when the tokenizer is asked for
	withUsage := append(append([]string{}, stream...), `{"type":"finish","messageMetadata":{"usage":{"inputTokens":7,"outputTokens":40}}}`)
	o = newObserver(t, &types.LLMConf{Tokenizer: tok})
	feed(o, withUsage)
	if r = o.Finish(time.Now()); r.OutputSource != types.TokenSourceUsage || r.Output != 40 {
		t.Errorf("auto: want 40 from usage, got %d %s", r.Output, r.OutputSource)
	}
	o = newObserver(t, &types.LLMConf{Tokenizer: tok, TokenSource: types.TokenSourceTokenizer})
	feed(o, withUsage)
	if r = o.Finish(time.Now()); r.OutputSource != types.TokenSourceTokenizer || r.Output != 4 || r.UsageOutput != 40 {
		t.Errorf("tokenizer: want 4, usage kept as 40; got %d %s %d", r.Output, r.OutputSource, r.UsageOutput)
	}

	// custom paths
	o = newObserver(t, &types.LLMConf{TextPaths: []string{"t"}, Usage: &types.LLMUsagePaths{Output: "u.o"}})
	o.Event(`{"t":"x"}`, time.Millisecond)
	o.Event(`{"u":{"o":9}}`, 2*time.Millisecond)
	if r = o.Finish(time.Now()); r.Output != 9 || r.ContentEvents != 1 {
		t.Errorf("custom paths: %d tokens, %d events", r.Output, r.ContentEvents)
	}
}

func TestCapturesAndText(t *testing.T) {
	o := newObserver(t, &types.LLMConf{CaptureText: "answer", Capture: map[string]string{"chat": "messageMetadata.chatId"}})
	feed(o, events(t, "testdata/aisdk_gurubase.sse"))
	o.Finish(time.Now())
	v := o.Vars()
	if v["chat"] != "b6Q0" || v["answer"] != "The standard retail overdraft limit is 5,000 lira [p1]." {
		t.Errorf("vars %v", v)
	}
}

func TestTokenizerInputCount(t *testing.T) {
	tok := writeTok(t)
	s, err := NewStep(&types.LLMConf{Tokenizer: tok, PromptPath: "q"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Prepare([]byte(`{"q":"ab ab"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	p.Observer.Event(`{"type":"text-delta","delta":"a"}`, time.Millisecond)
	r := p.Observer.Finish(time.Now())
	if r.Input != 3 || r.InputSource != types.TokenSourceTokenizer {
		t.Errorf("input %d %s, want 3 from the tokenizer", r.Input, r.InputSource)
	}
}
