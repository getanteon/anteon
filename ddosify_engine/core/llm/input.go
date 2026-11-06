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
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync/atomic"

	"go.ddosify.com/ddosify/v2/core/llm/tokenizer"
	"go.ddosify.com/ddosify/v2/core/types"
	"go.ddosify.com/ddosify/v2/core/util"
)

// DefaultInputInstruction follows the text of an exact length input, so the model has something
// natural to do with it. It comes last, so two requests share no prefix.
const DefaultInputInstruction = "Continue the text above."

// DefaultInputCorpusBytes is how much of a plain text dataset input_tokens reads: about eight
// million tokens, a thousand distinct prompts of 8k before two start in the same place.
const DefaultInputCorpusBytes = 32 << 20

// inputText cuts prompts of an exact token count out of a text: request i gets the text from its
// own place in it onwards, as much as makes the prompt, instruction included, exactly n tokens as
// the tokenizer counts them. Real text keeps speculative decoding's acceptance what it is on real
// prompts, which random token ids do not; a different start for every request keeps each prompt
// out of the others' prefix cache. The text keeps its own spacing, so source code keeps its lines.
type inputText struct {
	tok    *tokenizer.Tokenizer
	n      int
	suffix string
	text   string
	starts []int32 // where each word of text begins
	per    float64 // tokens per word, measured on the text
	rot    int
	seq    uint64
}

// sampleText reads at most budget bytes of a text file: all of a smaller one, else budget bytes in
// 64 pieces spread evenly over the file, each moved by the seed and cut at whitespace, so a file of
// any size costs the budget in memory and the sample still spans the whole of it.
func sampleText(path string, budget int64, seed int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("llm dataset: %v", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := st.Size()
	if size <= budget {
		b, err := io.ReadAll(io.LimitReader(f, budget))
		return strings.ToValidUTF8(string(b), ""), err
	}
	const pieces = 64
	piece := budget / pieces
	stride := size / pieces
	var out strings.Builder
	out.Grow(int(budget) + 2*pieces)
	buf := make([]byte, piece)
	for i := int64(0); i < pieces; i++ {
		off := i * stride
		if slack := stride - piece; slack > 0 {
			off += int64(util.SeedMix(seed, uint64(i), 0x5a) % uint64(slack))
		}
		n, err := f.ReadAt(buf, off)
		if err != nil && err != io.EOF {
			return "", err
		}
		b := buf[:n]
		// whole words only: from after the first whitespace to the last
		if j := bytes.IndexAny(b, " \t\n\r"); j >= 0 {
			b = b[j+1:]
		}
		if j := bytes.LastIndexAny(b, " \t\n\r"); j >= 0 {
			b = b[:j]
		}
		if out.Len() > 0 {
			out.WriteString("\n\n")
		}
		out.Write(bytes.ToValidUTF8(b, nil))
	}
	return out.String(), nil
}

// inputSource is the text input_tokens cuts from: a sample of the corpus file, else the sessions'
// turns, up to the same budget.
func inputSource(c *types.ChatRequest) (string, error) {
	budget := c.InputCorpusBytes
	if budget <= 0 {
		budget = DefaultInputCorpusBytes
	}
	if c.InputCorpus != "" {
		return sampleText(c.InputCorpus, budget, c.Seed)
	}
	var b strings.Builder
	for _, sess := range c.Sessions {
		for _, t := range sess {
			if int64(b.Len()) >= budget {
				return b.String(), nil
			}
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(t)
		}
	}
	return b.String(), nil
}

func wordStarts(text string) []int32 {
	var starts []int32
	space := true
	for i := 0; i < len(text); i++ {
		c := text[i]
		isSpace := c == ' ' || c == '\t' || c == '\n' || c == '\r'
		if space && !isSpace {
			starts = append(starts, int32(i))
		}
		space = isSpace
	}
	return starts
}

func newInputText(tok *tokenizer.Tokenizer, n int, instruction string, text string, seed int64) (*inputText, error) {
	if tok == nil {
		return nil, fmt.Errorf("input_tokens needs a tokenizer to count them: set llm.tokenizer, or --tokenizer, to the served model's tokenizer.json")
	}
	if len(text) > math.MaxInt32 {
		text = text[:math.MaxInt32]
	}
	starts := wordStarts(text)
	if len(starts) == 0 {
		return nil, fmt.Errorf("input_tokens needs a text: the dataset has no words")
	}
	sampleWords := min(len(starts), 20000)
	end := len(text)
	if sampleWords < len(starts) {
		end = int(starts[sampleWords])
	}
	per := float64(tok.Count(text[:end])) / float64(sampleWords)
	if per <= 0 {
		per = 1
	}
	if have := int(float64(len(starts)) * per); have < 2*n+int(4*per*8) {
		return nil, fmt.Errorf("input_tokens %d needs a text of at least %d tokens so requests do not repeat each other; the dataset has about %d", n, 2*n, have)
	}
	if instruction == "" {
		instruction = DefaultInputInstruction
	}
	suffix := "\n\n" + instruction
	if tok.Count(strings.TrimSpace(suffix)) >= n {
		return nil, fmt.Errorf("input_tokens %d is not more than the instruction's own tokens", n)
	}
	return &inputText{tok: tok, n: n, suffix: suffix, text: text, starts: starts, per: per,
		rot: int(util.SeedMix(seed, 0x15, 0) % uint64(len(starts)))}, nil
}

// next is the prompt of the next request and its token count, n unless the text could not reach it.
func (it *inputText) next() (string, int) {
	i := atomic.AddUint64(&it.seq, 1) - 1
	span := int(float64(it.n)/it.per) + 1
	kmax := 2*span + 8
	room := len(it.starts) - kmax // starts that leave kmax words after them
	if room < 1 {
		room, kmax = 1, len(it.starts)-1
	}
	pass := int(i) * span / room
	start := (it.rot + int(i)*span + pass*7) % room
	at := func(k int) string {
		if k <= 0 {
			return strings.TrimSpace(it.suffix)
		}
		from, to := int(it.starts[start]), len(it.text)
		if start+k < len(it.starts) {
			to = int(it.starts[start+k])
		}
		return strings.TrimRight(it.text[from:to], " \t\n\r") + it.suffix
	}
	// a secant search from the measured ratio, then a bisection on what is left
	lo, hi := 0, kmax
	k := min(span, hi)
	for range 6 {
		c := it.tok.Count(at(k))
		if c == it.n {
			return at(k), c
		}
		if c < it.n {
			lo = max(lo, k)
		} else {
			hi = min(hi, k-1)
		}
		step := int(float64(it.n-c) / it.per)
		if step == 0 {
			step = 1
			if c > it.n {
				step = -1
			}
		}
		k = min(max(k+step, lo), hi)
	}
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if it.tok.Count(at(mid)) <= it.n {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	best := at(lo)
	count := it.tok.Count(best)
	body := strings.TrimSuffix(best, it.suffix)
	for count < it.n {
		grown := false
		for _, pad := range []string{" a", " I", ".", "a"} {
			if c := it.tok.Count(body + pad + it.suffix); c > count && c <= it.n {
				body, count, grown = body+pad, c, true
				break
			}
		}
		if !grown {
			break
		}
	}
	return body + it.suffix, count
}
