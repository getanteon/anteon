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
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
)

// ChatMessage is an OpenAI chat message.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var sessionSeq int64

func (s *Step) pickSession(envs map[string]interface{}, p *Prepared) int {
	c := s.conf.Chat
	if v, ok := envs[VarSession].(int); ok && v >= 0 && v < len(c.Sessions) {
		return v
	}
	var idx int
	if c.Random {
		idx = rand.Intn(len(c.Sessions))
	} else {
		idx = int((atomic.AddInt64(&sessionSeq, 1) - 1) % int64(len(c.Sessions)))
	}
	p.Vars[VarSession] = idx
	return idx
}

// turnPrompt is the session's turn; a session shorter than the turn borrows the first turn of a later one.
func turnPrompt(sessions [][]string, idx, turn int) string {
	if turns := sessions[idx]; turn < len(turns) {
		return turns[turn]
	}
	return sessions[(idx+turn)%len(sessions)][0]
}

func (s *Step) prepareChat(envs map[string]interface{}, p *Prepared, obs *Observer) ([]byte, error) {
	c := s.conf.Chat
	if len(c.Sessions) == 0 {
		return nil, fmt.Errorf("llm preset has no prompts")
	}
	idx := s.pickSession(envs, p)
	user := turnPrompt(c.Sessions, idx, c.Turn)

	var history []ChatMessage
	if c.CarryHistory {
		if h, ok := envs[VarHistory].([]ChatMessage); ok {
			history = h
		}
	}

	if s.conf.CacheBust {
		obs.res.CacheBusted = true
		if len(history) == 0 {
			// the first turn of the session: later turns resend it, so a session reuses its own prefix
			marker, _ := envs[VarMarker].(string)
			if marker == "" {
				marker = NewMarker()
				p.Vars[VarMarker] = marker
			}
			user = marker + user
		}
	}
	obs.user = user
	obs.history = history
	obs.res.PrefixTokens = s.prefixTokens

	messages := make([]ChatMessage, 0, len(history)+2)
	system := c.SystemPrompt
	if s.prefix != "" {
		system = s.prefix
		if c.SystemPrompt != "" {
			system = joinPrefix(s.prefix, c.SystemPrompt)
		}
	}
	if system != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: system})
	}
	messages = append(messages, history...)
	messages = append(messages, ChatMessage{Role: "user", Content: user})

	var prompt strings.Builder
	for _, m := range messages {
		prompt.WriteString(m.Content)
		prompt.WriteByte('\n')
	}
	obs.prompt = prompt.String()

	body := map[string]interface{}{
		"messages": messages,
		"stream":   true,
	}
	if c.Model != "" {
		body["model"] = c.Model
	}
	if c.MaxTokens > 0 {
		field := c.MaxTokensField
		if field == "" {
			field = "max_tokens"
		}
		body[field] = c.MaxTokens
	}
	if c.Temperature != nil {
		body["temperature"] = *c.Temperature
	}
	if c.IgnoreEOS {
		body["ignore_eos"] = true
	}
	if c.IncludeUsage {
		body["stream_options"] = map[string]interface{}{"include_usage": true}
	}
	for k, v := range c.ExtraBody {
		body[k] = v
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(body); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}
