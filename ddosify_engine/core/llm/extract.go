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

// Package llm measures streamed LLM answers in tokens and prepares their prompts: usage and text
// extraction, token counting, cache-busting and shared prefixes, the OpenAI preset, multi-turn
// sessions, and the token and scheduling summaries of the report.
package llm

import (
	"strings"

	"github.com/tidwall/gjson"
	"go.ddosify.com/ddosify/v2/core/types"
)

// eventText returns the output text an event carries: an OpenAI chat or completions chunk, an AI SDK
// UI message stream text or reasoning delta, an Anthropic content_block_delta or a Responses API delta.
func eventText(data string) string {
	if len(data) == 0 || data[0] != '{' {
		return ""
	}
	if t := gjson.Get(data, "type"); t.Exists() {
		switch t.Str {
		case "text-delta", "reasoning-delta":
			if d := gjson.Get(data, "delta"); d.Type == gjson.String {
				return d.Str
			}
			return gjson.Get(data, "textDelta").Str
		case "content_block_delta":
			d := gjson.Get(data, "delta")
			return d.Get("text").Str + d.Get("thinking").Str
		case "response.output_text.delta", "response.reasoning_text.delta", "response.reasoning_summary_text.delta":
			return gjson.Get(data, "delta").Str
		}
	}
	choice := gjson.Get(data, "choices.0")
	if !choice.Exists() {
		return ""
	}
	if delta := choice.Get("delta"); delta.Exists() {
		// vLLM has written the reasoning as reasoning_content, then as reasoning, and for a while both
		reasoning := delta.Get("reasoning").Str
		if reasoning == "" {
			reasoning = delta.Get("reasoning_content").Str
		}
		return delta.Get("content").Str + reasoning
	}
	return choice.Get("text").Str
}

func customText(data string, paths []string) string {
	if len(data) == 0 || data[0] != '{' {
		return ""
	}
	var b strings.Builder
	for _, p := range paths {
		if r := gjson.Get(data, p); r.Type == gjson.String {
			b.WriteString(r.Str)
		}
	}
	return b.String()
}

// usagePaths are tried in order; the first whose output count is a number wins.
var usagePaths = []types.LLMUsagePaths{
	{Output: "usage.completion_tokens", Input: "usage.prompt_tokens"},                          // OpenAI chat and completions, vLLM, LiteLLM
	{Output: "response.usage.output_tokens", Input: "response.usage.input_tokens"},             // OpenAI Responses, response.completed
	{Output: "messageMetadata.usage.outputTokens", Input: "messageMetadata.usage.inputTokens"}, // AI SDK UI message stream metadata
	{Output: "messageMetadata.totalUsage.outputTokens", Input: "messageMetadata.totalUsage.inputTokens"},
	{Output: "usage.outputTokens", Input: "usage.inputTokens"},
	{Output: "usage.completionTokens", Input: "usage.promptTokens"},
	{Output: "usage.output_tokens", Input: "usage.input_tokens"},                 // Anthropic message_delta
	{Output: "message.usage.output_tokens", Input: "message.usage.input_tokens"}, // Anthropic message_start
}

// eventUsage returns the output and input token counts an event reports. ok is false when it reports none.
func eventUsage(data string, custom *types.LLMUsagePaths) (output, input int, ok bool) {
	if len(data) == 0 || data[0] != '{' {
		return 0, 0, false
	}
	if custom != nil {
		return usageAt(data, *custom)
	}
	if !strings.Contains(data, "sage") {
		return 0, 0, false
	}
	for _, p := range usagePaths {
		if output, input, ok = usageAt(data, p); ok {
			return
		}
	}
	return 0, 0, false
}

func usageAt(data string, p types.LLMUsagePaths) (output, input int, ok bool) {
	r := gjson.Get(data, p.Output)
	if r.Type != gjson.Number {
		return 0, 0, false
	}
	output = int(r.Int())
	if p.Input != "" {
		if in := gjson.Get(data, p.Input); in.Type == gjson.Number {
			input = int(in.Int())
		}
	}
	return output, input, true
}
