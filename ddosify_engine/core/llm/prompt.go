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
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/tidwall/gjson"
	"go.ddosify.com/ddosify/v2/core/llm/tokenizer"
)

// A prefix cache (vLLM's automatic prefix caching, SGLang's radix cache) reuses the KV of the longest
// prefix it has seen. A marker that differs in the first tokens leaves nothing to reuse after it.
// The marker follows AIPerf's "[rid:<hex>]" form, with a run nonce and a counter: unique within a run,
// and a different run starts from another nonce, so it does not hit the cache the last run left.

var (
	runNonce   uint32
	markerSeq  uint32
	nonceOnce  sync.Once
	markerHead = "[rid:"
)

func initNonce() {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		runNonce = mrand.Uint32()
		return
	}
	runNonce = binary.BigEndian.Uint32(b[:])
}

// NewMarker returns a cache-busting marker, followed by a blank line.
func NewMarker() string {
	nonceOnce.Do(initNonce)
	n := atomic.AddUint32(&markerSeq, 1)
	return fmt.Sprintf("%s%08x%08x]\n\n", markerHead, runNonce, n)
}

// Words that are one token each with a leading space in the Llama 3 and Qwen vocabularies.
var prefixWords = strings.Fields(`time year people way day man thing woman life child world school state
family student group country problem hand part place case week company system program question work
government number night point home water room mother area money story fact month lot right study book
eye job word business issue side kind head house service friend father power hour game line end member
law car city community name president team minute idea kid body information back parent face others
level office door health person art war history party result change morning reason research girl guy
moment air teacher force education`)

type prefixKey struct {
	tok *tokenizer.Tokenizer
	n   int
}

var (
	prefixMu    sync.Mutex
	prefixCache = map[prefixKey]sharedPrefix{}
)

type sharedPrefix struct {
	text   string
	tokens int  // as counted, or n when no tokenizer
	exact  bool // counted with the tokenizer
}

// SharedPrefix returns the same text of n tokens on every call. Without a tokenizer it is n words,
// about n tokens for a Llama 3 or Qwen vocabulary, and exact is false.
func SharedPrefix(tok *tokenizer.Tokenizer, n int) (text string, tokens int, exact bool) {
	if n <= 0 {
		return "", 0, true
	}
	prefixMu.Lock()
	defer prefixMu.Unlock()
	key := prefixKey{tok, n}
	if p, ok := prefixCache[key]; ok {
		return p.text, p.tokens, p.exact
	}
	rnd := mrand.New(mrand.NewSource(1))
	var p sharedPrefix
	if tok == nil {
		words := make([]string, n)
		for i := range words {
			words[i] = prefixWords[rnd.Intn(len(prefixWords))]
		}
		p = sharedPrefix{text: strings.Join(words, " "), tokens: n}
	} else {
		p = exactPrefix(tok, n, rnd)
	}
	prefixCache[key] = p
	return p.text, p.tokens, p.exact
}

// exactPrefix finds by bisection the most words that stay within n tokens, then pads to exactly n.
// Counting the whole text each time keeps it right for vocabularies that merge across words.
func exactPrefix(tok *tokenizer.Tokenizer, n int, rnd *mrand.Rand) sharedPrefix {
	words := make([]string, 2*n+8)
	for i := range words {
		words[i] = prefixWords[rnd.Intn(len(prefixWords))]
	}
	text := func(k int) string { return strings.Join(words[:k], " ") }
	lo, hi := 0, len(words)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if tok.Count(text(mid)) <= n {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	best := text(lo)
	count := tok.Count(best)
	for count < n {
		grown := false
		for _, pad := range []string{" a", " I", ".", "a", " " + words[lo]} {
			if c := tok.Count(best + pad); c > count && c <= n {
				best, count, grown = best+pad, c, true
				break
			}
		}
		if !grown {
			break
		}
	}
	return sharedPrefix{text: best, tokens: count, exact: true}
}

// spliceString replaces the JSON string at path in body with value, keeping the rest of the body byte for byte.
func spliceString(body []byte, path string, edit func(old string) string) ([]byte, string, error) {
	r := gjson.GetBytes(body, path)
	if !r.Exists() {
		return body, "", fmt.Errorf("prompt_path %q is not in the request body", path)
	}
	if r.Type != gjson.String {
		return body, "", fmt.Errorf("prompt_path %q is not a string in the request body", path)
	}
	if r.Index <= 0 || r.Index+len(r.Raw) > len(body) || string(body[r.Index:r.Index+len(r.Raw)]) != r.Raw {
		return body, "", fmt.Errorf("prompt_path %q could not be located in the request body", path)
	}
	value := edit(r.Str)
	enc, err := jsonString(value)
	if err != nil {
		return body, "", err
	}
	out := make([]byte, 0, len(body)+len(enc)-len(r.Raw))
	out = append(out, body[:r.Index]...)
	out = append(out, enc...)
	out = append(out, body[r.Index+len(r.Raw):]...)
	return out, value, nil
}

// jsonString encodes s as a JSON string without escaping <, > and &.
func jsonString(s string) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}
