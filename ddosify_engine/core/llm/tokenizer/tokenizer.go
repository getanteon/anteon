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

// Package tokenizer counts tokens with a Hugging Face tokenizer.json BPE model, in pure Go.
// It covers byte-level BPE (Llama 3, Qwen 2 and 3, GPT-2 style) and SentencePiece style BPE
// with byte fallback (Llama 2, Mistral). Encode is encode(text, add_special_tokens=False).
package tokenizer

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Words longer than this are not cached; at most cacheCap words are.
const (
	cacheMaxWordBytes = 64
	cacheCap          = 200000
)

type addedToken struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

type file struct {
	AddedTokens  []addedToken    `json:"added_tokens"`
	Normalizer   json.RawMessage `json:"normalizer"`
	PreTokenizer json.RawMessage `json:"pre_tokenizer"`
	Model        bpeModel        `json:"model"`
}

// Tokenizer is safe for concurrent use.
type Tokenizer struct {
	normalize normalizer
	pre       preTokenizer
	model     *bpe
	added     map[byte][]addedToken // by first byte, longest first
	vocabSize int

	cache  sync.Map
	cached int64
}

// LoadFile reads a tokenizer.json.
func LoadFile(path string) (*Tokenizer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tokenizer: %v", err)
	}
	t, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("tokenizer %s: %v", path, err)
	}
	return t, nil
}

// Parse reads the contents of a tokenizer.json.
func Parse(data []byte) (*Tokenizer, error) {
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	model, err := newBPE(&f.Model)
	if err != nil {
		return nil, err
	}
	t := &Tokenizer{model: model, added: make(map[byte][]addedToken), vocabSize: len(f.Model.Vocab)}
	if t.normalize, err = parseNormalizer(f.Normalizer); err != nil {
		return nil, err
	}
	if t.pre, err = parsePreTokenizer(f.PreTokenizer); err != nil {
		return nil, err
	}
	for _, a := range f.AddedTokens {
		if a.Content == "" {
			continue
		}
		t.added[a.Content[0]] = append(t.added[a.Content[0]], a)
		if a.ID >= t.vocabSize {
			t.vocabSize = a.ID + 1
		}
	}
	for k := range t.added {
		list := t.added[k]
		sort.SliceStable(list, func(i, j int) bool { return len(list[i].Content) > len(list[j].Content) })
	}
	return t, nil
}

// VocabSize is the number of ids, added tokens included.
func (t *Tokenizer) VocabSize() int { return t.vocabSize }

// Count returns the number of tokens in text.
func (t *Tokenizer) Count(text string) int {
	return len(t.Encode(text))
}

// Encode returns the ids of text, without special tokens added around it.
// A character a vocab without an unk token can not encode counts as one id, -1.
func (t *Tokenizer) Encode(text string) []int {
	out := make([]int, 0, len(text)/3+1)
	segStart := 0
	for i := 0; i < len(text); {
		if cands := t.added[text[i]]; cands != nil {
			matched := false
			for _, a := range cands {
				if strings.HasPrefix(text[i:], a.Content) {
					out = t.encodeSegment(text[segStart:i], segStart == 0, out)
					out = append(out, a.ID)
					i += len(a.Content)
					segStart = i
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}
		i++
	}
	return t.encodeSegment(text[segStart:], segStart == 0, out)
}

func (t *Tokenizer) encodeSegment(s string, first bool, out []int) []int {
	if s == "" {
		return out
	}
	if t.normalize != nil {
		s = t.normalize(s)
	}
	pieces := []piece{{text: s, first: first}}
	if t.pre != nil {
		pieces = t.pre(pieces)
	}
	for _, p := range pieces {
		out = t.encodeWord(p.text, out)
	}
	return out
}

func (t *Tokenizer) encodeWord(w string, out []int) []int {
	if len(w) > cacheMaxWordBytes {
		return t.model.encodeWord(w, out)
	}
	if v, ok := t.cache.Load(w); ok {
		return append(out, v.([]int)...)
	}
	ids := t.model.encodeWord(w, nil)
	if atomic.LoadInt64(&t.cached) < cacheCap {
		if _, loaded := t.cache.LoadOrStore(w, ids); !loaded {
			atomic.AddInt64(&t.cached, 1)
		}
	}
	return append(out, ids...)
}
