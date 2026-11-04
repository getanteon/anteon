package llm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go.ddosify.com/ddosify/config"
	"go.ddosify.com/ddosify/core"
	"go.ddosify.com/ddosify/core/assertion"
	"go.ddosify.com/ddosify/core/llm"
	"go.ddosify.com/ddosify/core/types"
)

type recorder struct {
	mu      sync.Mutex
	results []*types.ScenarioResult
	done    chan bool
}

func (r *recorder) Init(bool, int) error { r.done = make(chan bool, 1); return nil }
func (r *recorder) Start(in chan *types.ScenarioResult, _ <-chan assertion.TestAssertionResult) {
	for res := range in {
		r.mu.Lock()
		r.results = append(r.results, res)
		r.mu.Unlock()
	}
	r.done <- true
}
func (r *recorder) DoneChan() <-chan bool { return r.done }

// run reads a JSON config the way the CLI does and runs it through the engine.
func run(t *testing.T, conf string) []*types.ScenarioResult {
	t.Helper()
	c, err := config.NewConfigReader([]byte(conf), config.ConfigTypeJson)
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.CreateHammer()
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Validate(); err != nil {
		t.Fatal(err)
	}
	es, err := core.InitEngineServices(h)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	rec.Init(false, 0)
	es.ReportServ = rec
	e, err := core.NewEngine(context.Background(), h, es)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Init(); err != nil {
		t.Fatal(err)
	}
	e.Start()
	return rec.results
}

type openAIRequest struct {
	Model         string            `json:"model"`
	Messages      []llm.ChatMessage `json:"messages"`
	Stream        bool              `json:"stream"`
	MaxTokens     int               `json:"max_tokens"`
	StreamOptions struct {
		IncludeUsage bool `json:"include_usage"`
	} `json:"stream_options"`
}

const fakeTokens = 8

// fakeOpenAI streams fakeTokens content chunks a few ms apart, then the usage chunk include_usage asks for.
func fakeOpenAI(t *testing.T, got *[]openAIRequest, mu *sync.Mutex) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer secret-key" {
			http.Error(w, `{"error":{"message":"unauthorized"}}`, http.StatusUnauthorized)
			return
		}
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		*got = append(*got, req)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		f := w.(http.Flusher)
		chunk := func(delta string) {
			fmt.Fprintf(w, "data: {\"id\":\"c\",\"object\":\"chat.completion.chunk\",\"model\":%q,\"choices\":[{\"index\":0,\"delta\":%s,\"finish_reason\":null}]}\n\n", req.Model, delta)
			f.Flush()
		}
		chunk(`{"role":"assistant","content":""}`)
		for i := 0; i < fakeTokens; i++ {
			time.Sleep(3 * time.Millisecond)
			chunk(fmt.Sprintf(`{"content":" tok%d"}`, i))
		}
		if req.StreamOptions.IncludeUsage {
			fmt.Fprintf(w, "data: {\"id\":\"c\",\"choices\":[],\"usage\":{\"prompt_tokens\":%d,\"completion_tokens\":%d,\"total_tokens\":%d}}\n\n",
				10*len(req.Messages), fakeTokens, 10*len(req.Messages)+fakeTokens)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

func TestOpenAIPresetAgainstFakeServer(t *testing.T) {
	var got []openAIRequest
	var mu sync.Mutex
	srv := fakeOpenAI(t, &got, &mu)
	defer srv.Close()
	t.Setenv("DDOSIFY_LLM_TEST_KEY", "secret-key")

	results := run(t, fmt.Sprintf(`{
		"iteration_count": 4, "duration": 1, "load_type": "linear",
		"llm": {
			"preset": "openai",
			"url": "%s/v1/chat/completions",
			"api_key": "{{$DDOSIFY_LLM_TEST_KEY}}",
			"model": "fake-model",
			"max_tokens": 16,
			"prompts": ["What is a load test?", "Explain TTFT."],
			"turns": 2,
			"think_time": "20",
			"cache_bust": true
		}
	}`, srv.URL))

	if len(results) != 4 || len(got) != 8 {
		t.Fatalf("%d iterations, %d requests; want 4 and 8", len(results), len(got))
	}
	markers := map[string]bool{}
	answer := strings.Repeat(" tok%d", fakeTokens)
	answer = fmt.Sprintf(answer, 0, 1, 2, 3, 4, 5, 6, 7)
	var firstTurns []llm.ChatMessage
	for _, req := range got {
		if req.Model != "fake-model" || !req.Stream || req.MaxTokens != 16 || !req.StreamOptions.IncludeUsage {
			t.Errorf("request %+v", req)
		}
		if len(req.Messages) == 1 {
			firstTurns = append(firstTurns, req.Messages[0])
			m := req.Messages[0].Content
			if !strings.HasPrefix(m, "[rid:") || markers[m[:22]] {
				t.Errorf("first turn %q lacks a unique marker", m)
			}
			markers[m[:22]] = true
		}
	}
	for _, req := range got {
		if len(req.Messages) != 3 {
			continue
		}
		found := false
		for _, f := range firstTurns {
			if req.Messages[0] == f {
				found = true
			}
		}
		if !found || req.Messages[1] != (llm.ChatMessage{Role: "assistant", Content: answer}) || req.Messages[2].Role != "user" {
			t.Errorf("second turn does not carry the first: %+v", req.Messages)
		}
	}
	if len(firstTurns) != 4 {
		t.Errorf("%d first turns, want 4", len(firstTurns))
	}

	var agg llm.Agg
	var sched llm.ScheduleAgg
	for _, res := range results {
		sched.Add(res)
		if len(res.StepResults) != 2 {
			t.Fatalf("%d steps", len(res.StepResults))
		}
		gap := res.StepResults[1].RequestTime.Sub(res.StepResults[0].RequestTime) - res.StepResults[0].Stream.Total
		if gap < 20*time.Millisecond {
			t.Errorf("think time %v, want at least 20ms", gap)
		}
		for i, sr := range res.StepResults {
			if sr.StatusCode != 200 || sr.Stream == nil || sr.Stream.LLM == nil {
				t.Fatalf("step %d: %d %+v", i, sr.StatusCode, sr.Stream)
			}
			r := sr.Stream.LLM
			agg.Add(r)
			if r.OutputSource != types.TokenSourceUsage || r.Output != fakeTokens || r.Input != 10*(1+2*i) || !r.HasTPOT || r.TPOT <= 0 {
				t.Errorf("step %d: %+v", i, r)
			}
			if !sr.Stream.HasFirstToken || sr.Stream.FirstToken != r.FirstContent {
				t.Errorf("step %d: first token %v, first content %v", i, sr.Stream.FirstToken, r.FirstContent)
			}
		}
	}
	s := agg.Summary()
	if s.Tokens == nil || s.Tokens.Total != 8*fakeTokens || s.Tokens.SystemPerSec <= 0 || s.ITL.Count != 8*(fakeTokens-1) || s.CacheBusted != 8 {
		t.Errorf("summary %+v %+v", s.Tokens, s.ITL)
	}
	if ss := sched.Summary(); ss == nil || ss.Iterations != 4 || ss.Saturated {
		t.Errorf("schedule %+v", ss)
	}
}

// A Gurubase shaped API: the server issues a chat id, keeps the history, and streams the AI SDK UI message stream.
func gurubaseLike(t *testing.T, log *[]string, mu *sync.Mutex) *httptest.Server {
	var chats int
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/chats" {
			chats++
			fmt.Fprintf(w, `{"id":"chat-%d"}`, chats)
			return
		}
		var req struct {
			Message struct {
				Parts []struct{ Text string } `json:"parts"`
			} `json:"message"`
		}
		if err := json.Unmarshal(body, &req); err != nil || len(req.Message.Parts) != 1 {
			http.Error(w, "bad body "+string(body), http.StatusBadRequest)
			return
		}
		text := req.Message.Parts[0].Text
		*log = append(*log, r.URL.Path+" "+r.Header.Get("X-Run")+" "+text)
		w.Header().Set("Content-Type", "text/event-stream")
		f := w.(http.Flusher)
		fmt.Fprintf(w, "data: {\"type\":\"start\",\"messageId\":\"run-%d\",\"messageMetadata\":{\"chatId\":%q,\"runId\":\"run-%d\"}}\n\n", len(*log), r.URL.Path, len(*log))
		fmt.Fprint(w, ": keepalive\n\n")
		f.Flush()
		for _, d := range []string{"Answer to ", "\"" + strings.TrimPrefix(text, "Q: ") + "\"", ": ab ab ab."} {
			time.Sleep(2 * time.Millisecond)
			fmt.Fprintf(w, "data: {\"type\":\"text-delta\",\"id\":\"t\",\"delta\":%s}\n\n", mustJSON(d))
			f.Flush()
		}
		fmt.Fprint(w, "data: {\"type\":\"finish\",\"messageMetadata\":{\"status\":\"ok\",\"usage\":{\"inputTokens\":6210,\"outputTokens\":412}}}\n\ndata: [DONE]\n\n")
	}))
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

const tinyTok = `{"added_tokens": [], "normalizer": null,
 "pre_tokenizer": {"type": "ByteLevel", "add_prefix_space": false, "use_regex": true},
 "model": {"type": "BPE", "vocab": {"a": 0, "b": 1, "Ġ": 2, "Ġa": 3, "ab": 4}, "merges": ["Ġ a", "a b"]}}`

func TestMultiTurnSessionAgainstSSEServer(t *testing.T) {
	var log []string
	var mu sync.Mutex
	srv := gurubaseLike(t, &log, &mu)
	defer srv.Close()
	tok := t.TempDir() + "/tokenizer.json"
	if err := os.WriteFile(tok, []byte(tinyTok), 0o600); err != nil {
		t.Fatal(err)
	}

	results := run(t, fmt.Sprintf(`{
		"iteration_count": 3, "duration": 1, "load_type": "linear", "engine_mode": "distinct-user",
		"steps": [
			{"id": 1, "name": "create chat", "url": "%[1]s/chats", "method": "POST",
			 "capture_env": {"chat_id": {"from": "body", "json_path": "id"}}},
			{"id": 2, "name": "turn 1", "url": "%[1]s/chats/{{chat_id}}/messages", "method": "POST",
			 "headers": {"Content-Type": "application/json"},
			 "payload": "{\"message\":{\"parts\":[{\"type\":\"text\",\"text\":\"Q: What is the limit?\"}]}}",
			 "stream": {"type": "sse"},
			 "llm": {"prompt_path": "message.parts.0.text", "cache_bust": true, "capture_text": "answer1",
			         "capture": {"run1": "messageMetadata.runId"}},
			 "sleep": "30"},
			{"id": 3, "name": "turn 2", "url": "%[1]s/chats/{{chat_id}}/messages", "method": "POST",
			 "headers": {"Content-Type": "application/json", "X-Run": "{{run1}}"},
			 "payload": "{\"message\":{\"parts\":[{\"type\":\"text\",\"text\":\"{{answer1}}\"}]}}",
			 "llm": {"tokenizer": %[2]q, "token_source": "tokenizer"}}
		]
	}`, srv.URL, tok))

	if len(results) != 3 {
		t.Fatalf("%d iterations", len(results))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(log) != 6 {
		t.Fatalf("server saw %d stream requests: %q", len(log), log)
	}
	for _, res := range results {
		turn1, turn2 := res.StepResults[1], res.StepResults[2]
		if turn1.Stream == nil || turn2.Stream == nil || turn1.Stream.LLM == nil || turn2.Stream.LLM == nil {
			t.Fatalf("no llm results: %+v %+v", turn1.Stream, turn2.Stream)
		}
		answer1, _ := turn1.ExtractedEnvs["answer1"].(string)
		run1, _ := turn1.ExtractedEnvs["run1"].(string)
		chat, _ := res.StepResults[0].ExtractedEnvs["chat_id"].(string)
		if !strings.HasPrefix(answer1, "Answer to \"[rid:") || !strings.HasSuffix(answer1, "What is the limit?\": ab ab ab.") || run1 == "" {
			t.Errorf("captured %q, run %q", answer1, run1)
		}
		// the second turn carried the first answer and the run id, to the same chat
		want := "/chats/" + chat + "/messages " + run1 + " " + answer1
		found := false
		for _, l := range log {
			if l == want {
				found = true
			}
		}
		if !found {
			t.Errorf("no request %q in %q", want, log)
		}
		if gap := turn2.RequestTime.Sub(turn1.RequestTime) - turn1.Stream.Total; gap < 30*time.Millisecond {
			t.Errorf("think time %v, want 30ms", gap)
		}
		// turn 1 trusts the usage the stream reports; turn 2 counts the text with the tokenizer
		if r := turn1.Stream.LLM; r.OutputSource != types.TokenSourceUsage || r.Output != 412 || !r.CacheBusted || r.ContentEvents != 3 {
			t.Errorf("turn 1 %+v", r)
		}
		if r := turn2.Stream.LLM; r.OutputSource != types.TokenSourceTokenizer || r.UsageOutput != 412 || r.Output < 10 {
			t.Errorf("turn 2 %+v", r)
		}
	}
}
