package llm

import (
	"encoding/json"
	mrand "math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.ddosify.com/ddosify/v2/core/types"
)

// corpus is a text that does not repeat itself, in the tiny tokenizer's letters.
func corpus(words int) [][]string { return [][]string{{corpusText(words)}} }

func corpusText(words int) string {
	r := mrand.New(mrand.NewSource(3))
	var b strings.Builder
	for i := range words {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString([]string{"ab", "a", "b", "aab", "ba", "bab", "aa"}[r.Intn(7)])
	}
	return b.String()
}

func TestInputTokensAreExactAndDistinct(t *testing.T) {
	tok, err := LoadTokenizer(writeTok(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{8, 64, 300} {
		in, err := newInputText(tok, n, "ab", corpusText(5000), 1)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for range 50 {
			p, c := in.next()
			if c != n || tok.Count(p) != n {
				t.Fatalf("n %d: a prompt of %d tokens (reported %d)", n, tok.Count(p), c)
			}
			if !strings.HasSuffix(p, "\n\nab") {
				t.Fatalf("the instruction must come last: %q", p[max(0, len(p)-10):])
			}
			head := p[:min(len(p), 24)]
			if n >= 64 && seen[head] { // a few words of a seven-word vocabulary repeat by chance
				t.Fatalf("n %d: two prompts start alike, %q", n, head)
			}
			seen[head] = true
		}
	}
}

func TestInputTokensRefusals(t *testing.T) {
	tok, _ := LoadTokenizer(writeTok(t))
	if _, err := newInputText(nil, 10, "", corpusText(100), 1); err == nil || !strings.Contains(err.Error(), "tokenizer") {
		t.Errorf("no tokenizer: %v", err)
	}
	if _, err := newInputText(tok, 1000, "ab", corpusText(50), 1); err == nil || !strings.Contains(err.Error(), "at least") {
		t.Errorf("a text too short: %v", err)
	}
	for _, p := range []Preset{
		{URL: "http://x", Prompts: []string{"a"}, InputTokens: 8, Turns: 2},
		{URL: "http://x", Prompts: []string{"a"}, InputTokens: 8, CacheBust: true},
	} {
		if _, err := p.Steps(); err == nil {
			t.Errorf("%+v must be refused", p)
		}
	}
}

func TestInputTokensSameWithTheSeed(t *testing.T) {
	tok, _ := LoadTokenizer(writeTok(t))
	a, _ := newInputText(tok, 32, "ab", corpusText(3000), 7)
	b, _ := newInputText(tok, 32, "ab", corpusText(3000), 7)
	c, _ := newInputText(tok, 32, "ab", corpusText(3000), 8)
	pa, _ := a.next()
	pb, _ := b.next()
	pc, _ := c.next()
	if pa != pb || pa == pc {
		t.Error("the same seed must give the same prompts, another seed others")
	}
}

func TestInputTokensInTheChatBody(t *testing.T) {
	tokPath := writeTok(t)
	s, err := NewStep(&types.LLMConf{Tokenizer: tokPath, Chat: &types.ChatRequest{Sessions: corpus(4000), InputTokens: 40, InputInstruction: "ab", MaxTokens: 5, IgnoreEOS: true}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Prepare(nil, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	body := string(p.Body)
	if !strings.Contains(body, `"ignore_eos":true`) || !strings.Contains(body, `"max_tokens":5`) || !strings.Contains(body, `\n\nab"`) {
		t.Errorf("body %s", body)
	}
}

// On a real vocabulary: exact lengths for the three usual scenarios' inputs.
func TestInputTokensRealVocabulary(t *testing.T) {
	dir := os.Getenv("DDOSIFY_TOKENIZERS_DIR")
	if dir == "" {
		t.Skip("NOTICE: DDOSIFY_TOKENIZERS_DIR is not set, exact input lengths are not checked on a real vocabulary")
	}
	path := filepath.Join(dir, "Qwen_Qwen3-0.6B.json")
	if _, err := os.Stat(path); err != nil {
		t.Skip("NOTICE: no Qwen3-0.6B tokenizer in DDOSIFY_TOKENIZERS_DIR")
	}
	tok, err := LoadTokenizer(path)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for i := range 60000 {
		b.WriteString(prefixWords[i*31%len(prefixWords)])
		b.WriteString([]string{" ", " ", ", ", ". ", " "}[i%5])
	}
	text := b.String()
	for _, n := range []int{1024, 8192} {
		in, err := newInputText(tok, n, "", text, 1)
		if err != nil {
			t.Fatal(err)
		}
		for range 5 {
			if p, c := in.next(); c != n || tok.Count(p) != n {
				t.Errorf("%d tokens wanted, got %d", n, tok.Count(p))
			}
		}
	}
}

// A corpus far larger than the budget is read as the budget, spread over the whole file, its
// spacing kept, so a file of gigabytes costs megabytes.
func TestInputCorpusIsSampledNotLoaded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corpus.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	line := "func add(a, b int) int {\n\treturn a + b\n}\n"
	for i := range 400000 { // about 17 MB
		if i == 399990 {
			f.WriteString("THE_END_OF_THE_FILE\n")
		}
		f.WriteString(line)
	}
	f.Close()
	text, err := sampleText(path, 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(text) > 1<<20+200 || len(text) < 1<<19 {
		t.Errorf("a 1 MiB budget read %d bytes", len(text))
	}
	if !strings.Contains(text, "\n\treturn a + b\n") {
		t.Error("the sample must keep the text's own lines and indentation")
	}
	again, _ := sampleText(path, 1<<20, 1)
	if again != text {
		t.Error("the same seed must sample the same text")
	}
	small := filepath.Join(t.TempDir(), "small.txt")
	os.WriteFile(small, []byte("all of it"), 0o600)
	if got, _ := sampleText(small, 1<<20, 1); got != "all of it" {
		t.Errorf("a file under the budget is read whole: %q", got)
	}

	// the preset hands the file to the step rather than loading it
	steps, err := (&Preset{URL: "http://x", Dataset: &Dataset{Path: path}, InputTokens: 64, InputCorpusBytes: 1 << 20}).Steps()
	if err != nil {
		t.Fatal(err)
	}
	c := steps[0].LLM.Chat
	if c.InputCorpus != path || c.InputCorpusBytes != 1<<20 || len(c.Sessions) != 1 {
		t.Errorf("chat %+v", c.InputCorpus)
	}
}

// extra_body reaches every request, nested objects included, beside the preset's own fields.
func TestExtraBodyFields(t *testing.T) {
	steps, err := (&Preset{URL: "http://x", Prompts: []string{"hi"}, MaxTokens: 9, IgnoreEOS: true,
		ExtraBody: map[string]interface{}{"top_p": 0.8, "top_k": 20, "presence_penalty": 1.5,
			"chat_template_kwargs": map[string]interface{}{"enable_thinking": false}}}).Steps()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStep(steps[0].LLM)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Prepare(nil, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(p.Body, &body); err != nil {
		t.Fatal(err)
	}
	kw, _ := body["chat_template_kwargs"].(map[string]interface{})
	if body["top_p"] != 0.8 || body["top_k"] != 20.0 || body["presence_penalty"] != 1.5 || kw["enable_thinking"] != false ||
		body["max_tokens"] != 9.0 || body["ignore_eos"] != true {
		t.Errorf("body %s", p.Body)
	}
}
