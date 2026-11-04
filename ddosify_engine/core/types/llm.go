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

package types

import (
	"fmt"
	"time"
)

// Where a request's output count came from
const (
	TokenSourceAuto      = "auto"
	TokenSourceUsage     = "usage"
	TokenSourceTokenizer = "tokenizer"
	TokenSourceChunks    = "chunks"
)

// LLMUsagePaths are gjson paths to the token counts a server reports in its stream.
type LLMUsagePaths struct {
	Output string `json:"output"`
	Input  string `json:"input"`
}

// LLMConf is a step's "llm" config: how to count its output, and what to do to its prompt.
type LLMConf struct {
	// tokenizer.json of the served model, for counting when the stream carries no usage
	Tokenizer string `json:"tokenizer"`

	// auto (usage, else tokenizer, else chunks), usage, tokenizer or chunks
	TokenSource string `json:"token_source"`

	// gjson paths to an event's output text. Empty knows OpenAI, the AI SDK, Anthropic and the Responses API.
	TextPaths []string `json:"text_paths"`

	// gjson paths to the usage. Empty knows the same formats.
	Usage *LLMUsagePaths `json:"usage"`

	// gjson path to the prompt string in the request body, for cache_bust, shared_prefix_tokens and input tokens
	PromptPath string `json:"prompt_path"`

	// A unique marker at the start of every prompt, so no request reuses another's prefix cache
	CacheBust bool `json:"cache_bust"`

	// A fixed prefix of this many tokens before every prompt, shared by all requests
	SharedPrefixTokens int `json:"shared_prefix_tokens"`

	// Variable that receives the answer's text, for the next step
	CaptureText string `json:"capture_text"`

	// Variable name to a gjson path; the first event holding the path gives the value
	Capture map[string]string `json:"capture"`

	// Set by the OpenAI preset: the request body is built at send time
	Chat *ChatRequest `json:"-"`
}

// ChatRequest builds an OpenAI chat completions body for one turn of a session.
type ChatRequest struct {
	Model     string
	MaxTokens int
	// max_tokens by default; max_completion_tokens for an OpenAI reasoning model
	MaxTokensField string
	Temperature    *float64
	IgnoreEOS      bool
	IncludeUsage   bool
	SystemPrompt   string
	ExtraBody      map[string]interface{}

	// Sessions, each a list of user turns
	Sessions [][]string
	Random   bool

	// 0 based turn of this step in the session
	Turn int

	// The previous turns' messages are sent with this one
	CarryHistory bool
}

func (c *LLMConf) validate() error {
	switch c.TokenSource {
	case "", TokenSourceAuto, TokenSourceUsage, TokenSourceTokenizer, TokenSourceChunks:
	default:
		return fmt.Errorf("unsupported llm token_source: %q, supported: auto, usage, tokenizer, chunks", c.TokenSource)
	}
	if c.TokenSource == TokenSourceTokenizer && c.Tokenizer == "" {
		return fmt.Errorf("llm token_source tokenizer needs a tokenizer")
	}
	if c.SharedPrefixTokens < 0 {
		return fmt.Errorf("llm shared_prefix_tokens can not be negative")
	}
	if (c.CacheBust || c.SharedPrefixTokens > 0) && c.PromptPath == "" && c.Chat == nil {
		return fmt.Errorf("llm cache_bust and shared_prefix_tokens need prompt_path")
	}
	return nil
}

// LLMResult is what one streamed answer measured in tokens. Durations are from the moment the request was sent.
type LLMResult struct {
	SentAt time.Time `json:"sent_at"`

	// Output count and where it came from: usage, tokenizer or chunks
	Output       int    `json:"output"`
	OutputSource string `json:"output_source"`

	// Input tokens, 0 when not known; usage, or tokenizer (the prompt text, chat template excluded)
	Input       int    `json:"input,omitempty"`
	InputSource string `json:"input_source,omitempty"`

	// What the server reported, kept when another source was used
	UsageOutput int  `json:"usage_output,omitempty"`
	UsageInput  int  `json:"usage_input,omitempty"`
	HasUsage    bool `json:"has_usage"`

	// Events that carried output text
	ContentEvents int `json:"content_events"`

	// First and last event that carried output text
	FirstContent time.Duration `json:"first_content_ns"`
	LastContent  time.Duration `json:"last_content_ns"`
	HasContent   bool          `json:"has_content"`

	// (last content - first content) / (output - 1), with at least two outputs
	TPOT    time.Duration `json:"tpot_ns"`
	HasTPOT bool          `json:"has_tpot"`

	// Output per second of this request, output / last content
	OutputPerSec float64 `json:"output_per_sec"`

	// Gaps between consecutive content events, at most a few thousand
	ITL []time.Duration `json:"-"`

	TextBytes     int  `json:"text_bytes"`
	TextTruncated bool `json:"text_truncated,omitempty"`

	CacheBusted  bool `json:"cache_busted,omitempty"`
	PrefixTokens int  `json:"prefix_tokens,omitempty"`
}

// Tokens tells whether Output counts tokens rather than chunks.
func (r *LLMResult) Tokens() bool {
	return r.OutputSource == TokenSourceUsage || r.OutputSource == TokenSourceTokenizer
}

// tpot is the request's TPOT, present only when it was measured in tokens.
func (r *LLMResult) tpot() (time.Duration, bool) {
	if r == nil {
		return 0, false
	}
	return r.TPOT, r.HasTPOT && r.Tokens()
}

// OthersIntendedStart is the ScenarioResult.Others key of the time.Time the schedule meant the iteration to start.
const OthersIntendedStart = "intendedStart"

// capturedNames are the variables the step's answer sets for the next steps.
func (c *LLMConf) capturedNames() []string {
	if c == nil {
		return nil
	}
	names := make([]string, 0, len(c.Capture)+1)
	if c.CaptureText != "" {
		names = append(names, c.CaptureText)
	}
	for name := range c.Capture {
		names = append(names, name)
	}
	return names
}
