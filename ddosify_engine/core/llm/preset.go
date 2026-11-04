/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package llm

import (
	"fmt"
	"strings"

	"go.ddosify.com/ddosify/core/types"
)

const (
	PresetOpenAI = "openai"

	// An answer can take minutes under load; the step's timeout covers the whole stream.
	defaultPresetTimeout = 300
)

// Preset is the top level "llm" config: an OpenAI compatible chat completions endpoint, streamed,
// asked the prompts of a dataset, a session of Turns steps an iteration.
type Preset struct {
	Preset string `json:"preset"`
	URL    string `json:"url"`

	// Sent as "Authorization: Bearer <api_key>"; "{{$OPENAI_API_KEY}}" reads the environment
	APIKey  string            `json:"api_key"`
	Headers map[string]string `json:"headers"`

	Model          string                 `json:"model"`
	MaxTokens      int                    `json:"max_tokens"`
	MaxTokensField string                 `json:"max_tokens_field"`
	Temperature    *float64               `json:"temperature"`
	IgnoreEOS      bool                   `json:"ignore_eos"`
	IncludeUsage   *bool                  `json:"include_usage"`
	SystemPrompt   string                 `json:"system_prompt"`
	ExtraBody      map[string]interface{} `json:"extra_body"`

	Dataset *Dataset `json:"dataset"`
	Prompts []string `json:"prompts"`

	// sequential (default) or random
	Order string `json:"order"`

	// Steps of a session; each sends the turns before it. 1 by default.
	Turns int `json:"turns"`

	// Pause between turns in ms, "2000" or a range "1000-3000"
	ThinkTime string `json:"think_time"`

	// Seconds, for the whole stream
	Timeout int `json:"timeout"`

	Tokenizer          string `json:"tokenizer"`
	TokenSource        string `json:"token_source"`
	CacheBust          bool   `json:"cache_bust"`
	SharedPrefixTokens int    `json:"shared_prefix_tokens"`
}

// openAIStream marks the first token as the first chunk with content, reasoning included.
func openAIStream() *types.StreamConf {
	c := types.DefaultStreamConf()
	c.FirstToken = types.StreamMatch{Regexp: `"(?:content|reasoning_content|reasoning|text)"\s*:\s*"[^"]`}
	c.Error = types.StreamMatch{JsonPath: "error"}
	return c
}

// Steps builds the scenario: one step per turn.
func (p *Preset) Steps() ([]types.ScenarioStep, error) {
	if p.Preset != "" && p.Preset != PresetOpenAI {
		return nil, fmt.Errorf("unsupported llm preset %q, supported: %s", p.Preset, PresetOpenAI)
	}
	if p.URL == "" {
		return nil, fmt.Errorf("llm preset needs a url, such as http://localhost:8000/v1/chat/completions")
	}
	var sessions [][]string
	for _, pr := range p.Prompts {
		if strings.TrimSpace(pr) != "" {
			sessions = append(sessions, []string{pr})
		}
	}
	if p.Dataset != nil {
		s, err := p.Dataset.Load()
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s...)
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("llm preset needs a dataset or prompts")
	}
	switch strings.ToLower(p.Order) {
	case "", "sequential", "random":
	default:
		return nil, fmt.Errorf("llm order %q is not supported: sequential or random", p.Order)
	}
	turns := p.Turns
	if turns <= 0 {
		turns = 1
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = defaultPresetTimeout
	}
	includeUsage := p.IncludeUsage == nil || *p.IncludeUsage

	headers := map[string]string{"Content-Type": "application/json", "Accept": "text/event-stream"}
	if p.APIKey != "" {
		headers["Authorization"] = "Bearer " + p.APIKey
	}
	for k, v := range p.Headers {
		headers[k] = v
	}

	steps := make([]types.ScenarioStep, 0, turns)
	for t := 0; t < turns; t++ {
		h := make(map[string]string, len(headers))
		for k, v := range headers {
			h[k] = v
		}
		step := types.ScenarioStep{
			ID:      uint16(t + 1),
			Name:    fmt.Sprintf("turn %d", t+1),
			Method:  "POST",
			URL:     p.URL,
			Headers: h,
			Timeout: timeout,
			Stream:  openAIStream(),
			LLM: &types.LLMConf{
				Tokenizer:          p.Tokenizer,
				TokenSource:        p.TokenSource,
				CacheBust:          p.CacheBust,
				SharedPrefixTokens: p.SharedPrefixTokens,
				Chat: &types.ChatRequest{
					Model:          p.Model,
					MaxTokens:      p.MaxTokens,
					MaxTokensField: p.MaxTokensField,
					Temperature:    p.Temperature,
					IgnoreEOS:      p.IgnoreEOS,
					IncludeUsage:   includeUsage,
					SystemPrompt:   p.SystemPrompt,
					ExtraBody:      p.ExtraBody,
					Sessions:       sessions,
					Random:         strings.EqualFold(p.Order, "random"),
					Turn:           t,
					CarryHistory:   turns > 1,
				},
			},
		}
		if t < turns-1 {
			step.Sleep = strings.ReplaceAll(p.ThinkTime, " ", "")
		}
		steps = append(steps, step)
	}
	return steps, nil
}
