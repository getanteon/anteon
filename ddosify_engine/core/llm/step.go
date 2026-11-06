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
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tidwall/gjson"
	"go.ddosify.com/ddosify/v2/core/llm/tokenizer"
	"go.ddosify.com/ddosify/v2/core/types"
)

// Variables a session carries from one step to the next, in the iteration's envs.
const (
	VarSession = "_llm_session"
	VarHistory = "_llm_history"
	VarMarker  = "_llm_marker"
)

const (
	// The answer text kept per request for counting and captures; beyond it the count is a lower bound.
	maxTextBytes = 4 << 20

	// Content gaps kept per request for ITL; a reservoir sample beyond.
	maxITLSamples = 4096
)

var (
	tokMu    sync.Mutex
	tokCache = map[string]*tokenizer.Tokenizer{}
)

// LoadTokenizer reads a tokenizer.json once per path.
func LoadTokenizer(path string) (*tokenizer.Tokenizer, error) {
	tokMu.Lock()
	defer tokMu.Unlock()
	if t, ok := tokCache[path]; ok {
		return t, nil
	}
	t, err := tokenizer.LoadFile(path)
	if err != nil {
		return nil, err
	}
	tokCache[path] = t
	return t, nil
}

// Step is a step's llm config, ready to use. Safe for concurrent use.
type Step struct {
	conf         *types.LLMConf
	tok          *tokenizer.Tokenizer
	prefix       string
	prefixTokens int
	seq          uint64
	input        *inputText
}

// NewStep loads the tokenizer and builds the shared prefix.
func NewStep(conf *types.LLMConf) (*Step, error) {
	s := &Step{conf: conf}
	if conf.Tokenizer != "" {
		t, err := LoadTokenizer(conf.Tokenizer)
		if err != nil {
			return nil, err
		}
		s.tok = t
	}
	if conf.SharedPrefixTokens > 0 {
		s.prefix, s.prefixTokens, _ = SharedPrefix(s.tok, conf.SharedPrefixTokens)
	}
	if c := conf.Chat; c != nil && c.InputTokens > 0 {
		text, err := inputSource(c)
		if err != nil {
			return nil, err
		}
		in, err := newInputText(s.tok, c.InputTokens, c.InputInstruction, text, c.Seed)
		if err != nil {
			return nil, err
		}
		s.input = in
	}
	return s, nil
}

// Prepared is one request's body and the observer of its answer.
type Prepared struct {
	Body []byte

	// Variables to put into the iteration's envs, whatever the answer
	Vars map[string]interface{}

	Observer *Observer
}

// Prepare builds or edits the request body: the OpenAI preset's chat body, or the cache-busting
// marker and shared prefix put at the start of the string at prompt_path.
func (s *Step) Prepare(body []byte, envs map[string]interface{}) (*Prepared, error) {
	p := &Prepared{Body: body, Vars: map[string]interface{}{}}
	obs := &Observer{step: s, captures: s.conf.Capture}
	p.Observer = obs

	switch {
	case s.conf.Chat != nil:
		b, err := s.prepareChat(envs, p, obs)
		if err != nil {
			return nil, err
		}
		p.Body = b
	case s.conf.PromptPath != "":
		if s.conf.CacheBust || s.prefix != "" {
			marker := ""
			if s.conf.CacheBust {
				marker = NewMarker()
				obs.res.CacheBusted = true
			}
			obs.res.PrefixTokens = s.prefixTokens
			b, prompt, err := spliceString(body, s.conf.PromptPath, func(old string) string {
				return joinPrefix(s.prefix, marker+old)
			})
			if err != nil {
				return nil, err
			}
			p.Body = b
			obs.prompt = prompt
		} else {
			obs.prompt = gjson.GetBytes(body, s.conf.PromptPath).String()
		}
	}
	return p, nil
}

func joinPrefix(prefix, rest string) string {
	if prefix == "" {
		return rest
	}
	return prefix + "\n\n" + rest
}

// Observer watches one stream's events. Not safe for concurrent use; one per request.
type Observer struct {
	step     *Step
	prompt   string
	captures map[string]string
	vars     map[string]interface{}

	text        strings.Builder
	usageSeen   bool
	usageOutput int
	usageInput  int
	gapsSeen    int64
	res         types.LLMResult

	// for the OpenAI preset's history
	history []ChatMessage
	user    string
	rnd     *rand.Rand
}

// Event takes one SSE event's data, at its time from the moment the request was sent.
func (o *Observer) Event(data string, at time.Duration) {
	var text string
	if len(o.step.conf.TextPaths) > 0 {
		text = customText(data, o.step.conf.TextPaths)
	} else {
		text = eventText(data)
	}
	if text != "" {
		r := &o.res
		if !r.HasContent {
			r.HasContent = true
			r.FirstContent = at
		} else {
			o.addGap(at - r.LastContent)
		}
		r.LastContent = at
		r.ContentEvents++
		r.TextBytes += len(text)
		if room := maxTextBytes - o.text.Len(); room > 0 {
			if len(text) > room {
				text = text[:room]
				r.TextTruncated = true
			}
			o.text.WriteString(text)
		} else {
			r.TextTruncated = true
		}
	}
	if out, in, ok := eventUsage(data, o.step.conf.Usage); ok {
		o.usageSeen = true
		o.usageOutput = out
		if in > 0 {
			o.usageInput = in
		}
	}
	for name, path := range o.captures {
		if _, done := o.vars[name]; done {
			continue
		}
		if r := gjson.Get(data, path); r.Exists() {
			if o.vars == nil {
				o.vars = map[string]interface{}{}
			}
			o.vars[name] = r.Value()
		}
	}
}

func (o *Observer) addGap(d time.Duration) {
	o.gapsSeen++
	if len(o.res.ITL) < maxITLSamples {
		o.res.ITL = append(o.res.ITL, d)
		return
	}
	if o.rnd == nil {
		o.rnd = rand.New(rand.NewSource(int64(atomic.AddUint64(&o.step.seq, 1))))
	}
	if j := o.rnd.Int63n(o.gapsSeen); j < maxITLSamples {
		o.res.ITL[j] = d
	}
}

// Text is the answer read so far.
func (o *Observer) Text() string { return o.text.String() }

// Finish counts the output and returns the request's token metrics. sentAt is when the request was sent.
func (o *Observer) Finish(sentAt time.Time) *types.LLMResult {
	r := &o.res
	r.SentAt = sentAt
	r.HasUsage = o.usageSeen
	if o.usageSeen {
		r.UsageOutput, r.UsageInput = o.usageOutput, o.usageInput
	}

	source := o.step.conf.TokenSource
	if source == "" {
		source = types.TokenSourceAuto
	}
	switch {
	case (source == types.TokenSourceAuto || source == types.TokenSourceUsage) && o.usageSeen:
		r.Output, r.OutputSource = o.usageOutput, types.TokenSourceUsage
	case (source == types.TokenSourceAuto || source == types.TokenSourceTokenizer) && o.step.tok != nil:
		r.Output, r.OutputSource = o.step.tok.Count(o.text.String()), types.TokenSourceTokenizer
	default:
		r.Output, r.OutputSource = r.ContentEvents, types.TokenSourceChunks
	}

	switch {
	case o.usageSeen && o.usageInput > 0 && source != types.TokenSourceTokenizer:
		r.Input, r.InputSource = o.usageInput, types.TokenSourceUsage
	case o.step.tok != nil && o.prompt != "":
		r.Input, r.InputSource = o.step.tok.Count(o.prompt), types.TokenSourceTokenizer
	}

	if r.HasContent && r.Output >= 2 {
		r.TPOT = (r.LastContent - r.FirstContent) / time.Duration(r.Output-1)
		r.HasTPOT = true
	}
	if r.HasContent && r.LastContent > 0 {
		r.OutputPerSec = float64(r.Output) / r.LastContent.Seconds()
	}
	return r
}

// Vars returns the variables the answer gives the next step: captures, the text, the session's history.
func (o *Observer) Vars() map[string]interface{} {
	out := make(map[string]interface{}, len(o.vars)+2)
	for k, v := range o.vars {
		out[k] = v
	}
	if name := o.step.conf.CaptureText; name != "" {
		out[name] = o.text.String()
	}
	if c := o.step.conf.Chat; c != nil && c.CarryHistory && o.user != "" {
		h := make([]ChatMessage, 0, len(o.history)+2)
		h = append(h, o.history...)
		if answer := o.text.String(); answer != "" {
			h = append(h, ChatMessage{Role: "user", Content: o.user}, ChatMessage{Role: "assistant", Content: answer})
		}
		out[VarHistory] = h
	}
	return out
}
