package config

import (
	"os"
	"testing"

	"go.ddosify.com/ddosify/v2/core/types"
)

func TestLLMConfig(t *testing.T) {
	c, err := NewConfigReader([]byte(`{
		"iteration_count": 10, "duration": 5,
		"steps": [{"id": 1, "url": "https://example.com/chat", "method": "POST", "payload": "{\"q\":\"hi\"}",
			"llm": {"prompt_path": "q", "cache_bust": true, "shared_prefix_tokens": 64, "token_source": "chunks",
			        "capture_text": "answer", "text_paths": ["delta"], "usage": {"output": "u.out", "input": "u.in"}}}],
		"slo": {"tpot_ms": 50}
	}`), ConfigTypeJson)
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.CreateHammer()
	if err != nil {
		t.Fatal(err)
	}
	st := h.Scenario.Steps[0]
	if st.Stream == nil || st.LLM == nil || st.LLM.PromptPath != "q" || !st.LLM.CacheBust || st.LLM.SharedPrefixTokens != 64 ||
		st.LLM.TokenSource != types.TokenSourceChunks || st.LLM.CaptureText != "answer" || st.LLM.Usage.Output != "u.out" || h.SLO.TPOTMs != 50 {
		t.Errorf("step %+v llm %+v", st, st.LLM)
	}
	if err := h.Validate(); err != nil {
		t.Error(err)
	}

	c, _ = NewConfigReader([]byte(`{"llm": {"url": "http://localhost:8000/v1/chat/completions", "prompts": ["a"], "turns": 2},
		"steps": [{"id": 1, "url": "https://example.com"}]}`), ConfigTypeJson)
	if _, err := c.CreateHammer(); err == nil {
		t.Error("steps and a preset together must be refused")
	}

	c, _ = NewConfigReader([]byte(`{"llm": {"url": "http://localhost:8000/v1/chat/completions", "prompts": ["a"], "turns": 2, "model": "m"}}`), ConfigTypeJson)
	h, err = c.CreateHammer()
	if err != nil || len(h.Scenario.Steps) != 2 || h.Scenario.Steps[1].LLM.Chat.Turn != 1 {
		t.Fatalf("preset: %v %+v", err, h.Scenario.Steps)
	}
	if err := h.Validate(); err != nil {
		t.Error(err)
	}

	c, _ = NewConfigReader([]byte(`{"steps": [{"id": 1, "url": "https://example.com", "llm": {"cache_bust": true}}]}`), ConfigTypeJson)
	h, _ = c.CreateHammer()
	if err := h.Validate(); err == nil {
		t.Error("cache_bust without prompt_path must be refused")
	}
}

func TestLLMExamples(t *testing.T) {
	for _, k := range []string{"OPENAI_BASE_URL", "OPENAI_API_KEY", "GURUBASE_URL", "GURU_ID", "GURUBASE_API_KEY"} {
		t.Setenv(k, "http://localhost:1")
	}
	wd, _ := os.Getwd()
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	for _, f := range []string{"config_examples/llm_openai.json", "config_examples/llm_sse_session.json"} {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		c, err := NewConfigReader(raw, ConfigTypeJson)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		h, err := c.CreateHammer()
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if err := h.Validate(); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
}

func TestSessionAndScheduleFields(t *testing.T) {
	read := func(conf string) types.Hammer {
		t.Helper()
		c, err := NewConfigReader([]byte(conf), ConfigTypeJson)
		if err != nil {
			t.Fatal(err)
		}
		h, err := c.CreateHammer()
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	h := read(`{"steps": [{"id": 1, "url": "https://example.com"}]}`)
	if h.Seed != types.DefaultSeed || h.RequestLog || h.StopOnFailure || h.Scenario.Steps[0].SkipIfEmpty != "" {
		t.Errorf("defaults: %+v", h)
	}
	h = read(`{"seed": 0, "load_spread": "Poisson", "request_log": true, "stop_on_failure": true, "env": {"T2": ""},
		"steps": [{"id": 1, "url": "https://example.com", "skip_if_empty": "{{T2}}"}]}`)
	if h.Seed != 0 || h.LoadSpread != types.LoadSpreadPoisson || !h.RequestLog || !h.StopOnFailure || h.Scenario.Steps[0].SkipIfEmpty != "{{T2}}" {
		t.Errorf("fields: %+v", h)
	}
	if err := h.Validate(); err != nil {
		t.Error(err)
	}
	h = read(`{"steps": [{"id": 1, "url": "https://example.com", "skip_if_empty": "{{UNDEFINED}}"}]}`)
	if err := h.Validate(); err == nil {
		t.Error("skip_if_empty with an undefined variable should be refused")
	}
}
