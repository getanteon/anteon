package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

func TestMarkersAreUnique(t *testing.T) {
	const workers, each = 32, 5000
	var mu sync.Mutex
	seen := make(map[string]bool, workers*each)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := make([]string, 0, each)
			for i := 0; i < each; i++ {
				local = append(local, NewMarker())
			}
			mu.Lock()
			for _, m := range local {
				if seen[m] {
					t.Errorf("duplicate marker %q", m)
				}
				seen[m] = true
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(seen) != workers*each {
		t.Fatalf("%d markers, want %d", len(seen), workers*each)
	}
	m := NewMarker()
	if !strings.HasPrefix(m, "[rid:") || !strings.HasSuffix(m, "]\n\n") || len(m) != len("[rid:]\n\n")+16 {
		t.Errorf("marker %q", m)
	}
}

func TestSpliceKeepsTheRestOfTheBody(t *testing.T) {
	body := []byte(`{"id":"c1","message":{"role":"user","parts":[{"type":"text","text":"Limit <5> & \"quote\"?"}]},"trigger":"submit-message"}`)
	out, prompt, err := spliceString(body, "message.parts.0.text", func(old string) string { return "X " + old })
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"c1","message":{"role":"user","parts":[{"type":"text","text":"X Limit <5> & \"quote\"?"}]},"trigger":"submit-message"}`
	if string(out) != want || prompt != `X Limit <5> & "quote"?` {
		t.Errorf("got %s / %q", out, prompt)
	}
	if _, _, err := spliceString(body, "message.nope", func(s string) string { return s }); err == nil {
		t.Error("a missing path must be an error")
	}
	if _, _, err := spliceString(body, "message.parts", func(s string) string { return s }); err == nil {
		t.Error("a path that is not a string must be an error")
	}
}

func TestCacheBustPerRequest(t *testing.T) {
	s, err := NewStep(&types.LLMConf{PromptPath: "q", CacheBust: true})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		p, err := s.Prepare([]byte(`{"q":"same question"}`), nil)
		if err != nil {
			t.Fatal(err)
		}
		var b struct{ Q string }
		if err := json.Unmarshal(p.Body, &b); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(b.Q, "[rid:") || !strings.HasSuffix(b.Q, "]\n\nsame question") || seen[b.Q] {
			t.Fatalf("prompt %q", b.Q)
		}
		seen[b.Q] = true
	}
}

func TestSharedPrefixLength(t *testing.T) {
	// without a tokenizer: n words, the same on every call
	text, n, exact := SharedPrefix(nil, 50)
	if len(strings.Fields(text)) != 50 || n != 50 || exact {
		t.Errorf("approximate prefix: %d words, %d, %v", len(strings.Fields(text)), n, exact)
	}
	again, _, _ := SharedPrefix(nil, 50)
	if again != text {
		t.Error("the prefix must be the same on every call")
	}

	dir := os.Getenv("DDOSIFY_TOKENIZERS_DIR")
	if dir == "" {
		t.Skip("NOTICE: DDOSIFY_TOKENIZERS_DIR is not set, the exact prefix length is not checked on a real vocabulary")
	}
	for _, name := range []string{"Qwen_Qwen3-0.6B", "unsloth_Llama-3.2-1B-Instruct", "TinyLlama_TinyLlama-1.1B-Chat-v1.0"} {
		path := filepath.Join(dir, name+".json")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		tok, err := LoadTokenizer(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []int{1, 16, 512, 2048} {
			start := time.Now()
			text, n, exact := SharedPrefix(tok, want)
			if !exact || n != want || tok.Count(text) != want {
				t.Errorf("%s: prefix of %d tokens counts %d (reported %d)", name, want, tok.Count(text), n)
			}
			if d := time.Since(start); d > 5*time.Second {
				t.Errorf("%s: %d tokens took %v", name, want, d)
			}
		}
	}
}

func TestPrefixAndMarkerOrder(t *testing.T) {
	s, err := NewStep(&types.LLMConf{PromptPath: "q", CacheBust: true, SharedPrefixTokens: 8})
	if err != nil {
		t.Fatal(err)
	}
	p1, _ := s.Prepare([]byte(`{"q":"a"}`), nil)
	p2, _ := s.Prepare([]byte(`{"q":"b"}`), nil)
	var b1, b2 struct{ Q string }
	json.Unmarshal(p1.Body, &b1)
	json.Unmarshal(p2.Body, &b2)
	prefix, _, _ := SharedPrefix(nil, 8)
	// the shared prefix comes first so it stays cacheable; the marker after it makes the rest unique
	for _, q := range []string{b1.Q, b2.Q} {
		if !strings.HasPrefix(q, prefix+"\n\n[rid:") {
			t.Errorf("prompt %q does not start with the prefix then a marker", q)
		}
	}
	if b1.Q[:len(prefix)] != b2.Q[:len(prefix)] {
		t.Error("prefixes differ")
	}
}
