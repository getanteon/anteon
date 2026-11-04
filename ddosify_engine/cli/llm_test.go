package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	_ "go.ddosify.com/ddosify/v2/core/llm"
)

// fakeOpenAI streams an OpenAI chat completion of five one-word chunks and its usage.
func fakeOpenAI(t *testing.T, models *atomic.Value) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		b, _ := io.ReadAll(r.Body)
		if json.Unmarshal(b, &body) != nil || body["stream"] != true {
			http.Error(w, "bad body", 400)
			return
		}
		models.Store(body["model"])
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 0; i < 5; i++ {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"w%d \"}}]}\n\n", i)
			w.(http.Flusher).Flush()
		}
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":5}}\n\ndata: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPresetFromFlags(t *testing.T) {
	var model atomic.Value
	srv := fakeOpenAI(t, &model)
	ds := filepath.Join(t.TempDir(), "prompts.txt")
	os.WriteFile(ds, []byte("Merhaba, nasılsın?\nWhat is TPOT?\n"), 0o600)

	code, out, errOut := run(t, "run", "--preset", "openai", "--url", srv.URL+"/v1/chat/completions", "--model", "m-1",
		"--dataset", ds, "--max-tokens", "16", "-n", "2", "-d", "1", "--json")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var res struct {
		LLM struct {
			Tokens *struct {
				Unit  string
				Total int64
			} `json:"tokens"`
		} `json:"llm"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil || res.LLM.Tokens == nil || res.LLM.Tokens.Unit != "tokens" || res.LLM.Tokens.Total != 10 {
		t.Errorf("usage counts the tokens: %v %s", err, out)
	}
	if model.Load() != "m-1" {
		t.Errorf("--model reaches the body, got %v", model.Load())
	}
	if code, _, _ := run(t, "run", "--preset", "openai", "--model", "m"); code != ExitUsage {
		t.Errorf("the preset without a url is a usage error, got %d", code)
	}
}

func TestTokenizerFlag(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	cfg := writeConfig(t, fmt.Sprintf(`{"iteration_count": 1, "duration": 1, "steps":[{"id":1,"url":"%s","stream":{"type":"sse"},
		"llm":{"token_source":"tokenizer"}}]}`, srv.URL))
	if code, _, errOut := run(t, "run", "--config", cfg); code != ExitUsage || !strings.Contains(errOut, "tokenizer") {
		t.Errorf("token_source tokenizer without a tokenizer is refused: %d %s", code, errOut)
	}
	if code, _, errOut := run(t, "run", "--config", cfg, "--tokenizer", "/does/not/exist.json"); code != ExitUsage {
		t.Errorf("a missing tokenizer file is a usage error: %d %s", code, errOut)
	}
	tok := os.Getenv("DDOSIFY_TEST_TOKENIZER")
	if tok == "" {
		t.Skip("NOTICE: DDOSIFY_TEST_TOKENIZER is not set, a real tokenizer is not loaded")
	}
	t.Setenv("DDOSIFY_TOKENIZER", tok)
	code, out, errOut := run(t, "run", "--config", cfg, "--json")
	if code != ExitOK || !strings.Contains(out, `"output_source":"tokenizer"`) && !strings.Contains(out, `"tokenizer":1`) {
		t.Errorf("DDOSIFY_TOKENIZER counts with the tokenizer: %d %s %s", code, out, errOut)
	}
}
