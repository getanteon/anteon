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

package tokenizer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// The normalizers and pre-tokenizers of tokenizer.json that BPE models use, after
// https://huggingface.co/docs/tokenizers/api/normalizers and .../api/pre-tokenizers.

type normalizer func(string) string

type pattern struct {
	String *string `json:"String"`
	Regex  *string `json:"Regex"`
}

type component struct {
	Type string `json:"type"`

	// Sequence
	Normalizers   []json.RawMessage `json:"normalizers"`
	Pretokenizers []json.RawMessage `json:"pretokenizers"`

	// Prepend, Replace
	Prepend string   `json:"prepend"`
	Pattern *pattern `json:"pattern"`
	Content string   `json:"content"`

	// Strip
	StripLeft  bool `json:"strip_left"`
	StripRight bool `json:"strip_right"`

	// ByteLevel, Metaspace
	AddPrefixSpace *bool  `json:"add_prefix_space"`
	UseRegex       *bool  `json:"use_regex"`
	Replacement    string `json:"replacement"`
	PrependScheme  string `json:"prepend_scheme"`
	Split          *bool  `json:"split"`

	// Split
	Behavior string `json:"behavior"`
	Invert   bool   `json:"invert"`

	// Digits
	IndividualDigits bool `json:"individual_digits"`
}

func isNull(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s == "" || s == "null"
}

func parseNormalizer(raw json.RawMessage) (normalizer, error) {
	if isNull(raw) {
		return nil, nil
	}
	var c component
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	switch c.Type {
	case "NFC":
		return norm.NFC.String, nil
	case "NFD":
		return norm.NFD.String, nil
	case "NFKC":
		return norm.NFKC.String, nil
	case "NFKD":
		return norm.NFKD.String, nil
	case "Lowercase":
		return strings.ToLower, nil
	case "Prepend":
		p := c.Prepend
		return func(s string) string {
			if s == "" {
				return s
			}
			return p + s
		}, nil
	case "Strip":
		l, r := c.StripLeft, c.StripRight
		return func(s string) string {
			if l {
				s = strings.TrimLeftFunc(s, unicode.IsSpace)
			}
			if r {
				s = strings.TrimRightFunc(s, unicode.IsSpace)
			}
			return s
		}, nil
	case "Replace":
		if c.Pattern == nil {
			return nil, fmt.Errorf("Replace normalizer without a pattern")
		}
		content := c.Content
		if c.Pattern.String != nil {
			old := *c.Pattern.String
			return func(s string) string { return strings.ReplaceAll(s, old, content) }, nil
		}
		if c.Pattern.Regex != nil {
			sr, err := compileOnig(*c.Pattern.Regex)
			if err != nil {
				return nil, err
			}
			return func(s string) string { return sr.re.ReplaceAllLiteralString(s, content) }, nil
		}
		return nil, fmt.Errorf("Replace normalizer without a pattern")
	case "Sequence":
		var seq []normalizer
		for _, r := range c.Normalizers {
			n, err := parseNormalizer(r)
			if err != nil {
				return nil, err
			}
			if n != nil {
				seq = append(seq, n)
			}
		}
		return func(s string) string {
			for _, n := range seq {
				s = n(s)
			}
			return s
		}, nil
	}
	return nil, fmt.Errorf("unsupported normalizer %q", c.Type)
}

// piece is a pre-token. first tells whether it starts at the beginning of the input.
type piece struct {
	text  string
	first bool
}

type preTokenizer func([]piece) []piece

func parsePreTokenizer(raw json.RawMessage) (preTokenizer, error) {
	if isNull(raw) {
		return nil, nil
	}
	var c component
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	switch c.Type {
	case "Sequence":
		var seq []preTokenizer
		for _, r := range c.Pretokenizers {
			p, err := parsePreTokenizer(r)
			if err != nil {
				return nil, err
			}
			if p != nil {
				seq = append(seq, p)
			}
		}
		return func(ps []piece) []piece {
			for _, p := range seq {
				ps = p(ps)
			}
			return ps
		}, nil
	case "Split":
		if c.Invert {
			return nil, fmt.Errorf("Split pre-tokenizer with invert is not supported")
		}
		if c.Pattern == nil {
			return nil, fmt.Errorf("Split pre-tokenizer without a pattern")
		}
		var sr *splitRegex
		var err error
		switch {
		case c.Pattern.Regex != nil:
			sr, err = compileOnig(*c.Pattern.Regex)
		case c.Pattern.String != nil:
			sr = &splitRegex{re: regexp.MustCompile(regexp.QuoteMeta(*c.Pattern.String)), wsGroup: -1}
		default:
			err = fmt.Errorf("Split pre-tokenizer without a pattern")
		}
		if err != nil {
			return nil, err
		}
		behavior := c.Behavior
		switch behavior {
		case "Isolated", "Removed", "MergedWithPrevious", "MergedWithNext", "Contiguous":
		default:
			return nil, fmt.Errorf("Split behavior %q is not supported", behavior)
		}
		return func(ps []piece) []piece { return splitAll(ps, sr, behavior) }, nil
	case "ByteLevel":
		prefix := c.AddPrefixSpace != nil && *c.AddPrefixSpace
		useRegex := c.UseRegex == nil || *c.UseRegex
		var sr *splitRegex
		if useRegex {
			var err error
			if sr, err = compileOnig(gpt2Pattern); err != nil {
				return nil, err
			}
		}
		return func(ps []piece) []piece {
			out := make([]piece, 0, len(ps))
			for _, p := range ps {
				if prefix && !strings.HasPrefix(p.text, " ") {
					p.text = " " + p.text
				}
				parts := []piece{p}
				if sr != nil {
					parts = splitAll(parts, sr, "Isolated")
				}
				for _, q := range parts {
					q.text = byteLevelEncode(q.text)
					out = append(out, q)
				}
			}
			return out
		}, nil
	case "Metaspace":
		repl := c.Replacement
		if repl == "" {
			repl = "▁"
		}
		scheme := c.PrependScheme
		if scheme == "" {
			scheme = "always"
			if c.AddPrefixSpace != nil && !*c.AddPrefixSpace {
				scheme = "never"
			}
		}
		split := c.Split == nil || *c.Split
		return func(ps []piece) []piece {
			out := make([]piece, 0, len(ps))
			for _, p := range ps {
				p.text = strings.ReplaceAll(p.text, " ", repl)
				if (scheme == "always" || (scheme == "first" && p.first)) && !strings.HasPrefix(p.text, repl) {
					p.text = repl + p.text
				}
				if !split {
					out = append(out, p)
					continue
				}
				out = append(out, splitAll([]piece{p}, &splitRegex{re: regexp.MustCompile(regexp.QuoteMeta(repl)), wsGroup: -1}, "MergedWithNext")...)
			}
			return out
		}, nil
	case "Digits":
		expr := `\p{Nd}+`
		if c.IndividualDigits {
			expr = `\p{Nd}`
		}
		sr := &splitRegex{re: regexp.MustCompile(expr), wsGroup: -1}
		return func(ps []piece) []piece { return splitAll(ps, sr, "Isolated") }, nil
	case "WhitespaceSplit":
		return func(ps []piece) []piece {
			var out []piece
			for _, p := range ps {
				for i, f := range strings.FieldsFunc(p.text, unicode.IsSpace) {
					out = append(out, piece{text: f, first: p.first && i == 0})
				}
			}
			return out
		}, nil
	case "Whitespace":
		sr := &splitRegex{re: regexp.MustCompile(`[\p{L}\p{M}\p{Nd}\p{Pc}]+|[^\p{L}\p{M}\p{Nd}\p{Pc}` + wsClass + `]+`), wsGroup: -1}
		return func(ps []piece) []piece { return splitAll(ps, sr, "Removed") }, nil
	}
	return nil, fmt.Errorf("unsupported pre-tokenizer %q", c.Type)
}

const gpt2Pattern = `'s|'t|'re|'ve|'m|'ll|'d| ?\p{L}+| ?\p{N}+| ?[^\s\p{L}\p{N}]+|\s+(?!\S)|\s+`

func splitAll(ps []piece, sr *splitRegex, behavior string) []piece {
	out := make([]piece, 0, len(ps))
	for _, p := range ps {
		out = splitOne(out, p, sr, behavior)
	}
	return out
}

func splitOne(out []piece, p piece, sr *splitRegex, behavior string) []piece {
	type span struct {
		start, end int
		match      bool
	}
	var spans []span
	prev := 0
	for _, m := range sr.matches(p.text) {
		if m[0] > prev {
			spans = append(spans, span{prev, m[0], false})
		}
		spans = append(spans, span{m[0], m[1], true})
		prev = m[1]
	}
	if prev < len(p.text) {
		spans = append(spans, span{prev, len(p.text), false})
	}

	var merged []span
	switch behavior {
	case "Isolated":
		merged = spans
	case "Removed":
		for _, s := range spans {
			if !s.match {
				merged = append(merged, s)
			}
		}
	case "MergedWithPrevious":
		for _, s := range spans {
			if s.match && len(merged) > 0 && !merged[len(merged)-1].match {
				merged[len(merged)-1].end = s.end
				merged[len(merged)-1].match = true
				continue
			}
			merged = append(merged, s)
		}
	case "MergedWithNext":
		pending := -1
		for _, s := range spans {
			if s.match {
				if pending >= 0 {
					merged = append(merged, span{pending, s.start, true})
				}
				pending = s.start
				continue
			}
			if pending >= 0 {
				merged = append(merged, span{pending, s.end, false})
				pending = -1
				continue
			}
			merged = append(merged, s)
		}
		if pending >= 0 {
			merged = append(merged, span{pending, len(p.text), true})
		}
	case "Contiguous":
		for _, s := range spans {
			if s.match && len(merged) > 0 && merged[len(merged)-1].match {
				merged[len(merged)-1].end = s.end
				continue
			}
			merged = append(merged, s)
		}
	}
	for _, s := range merged {
		if s.end > s.start {
			out = append(out, piece{text: p.text[s.start:s.end], first: p.first && s.start == 0})
		}
	}
	return out
}

// byteToRune is GPT-2's bytes_to_unicode: printable bytes map to themselves, the rest to U+0100 onwards.
var byteToRune = func() [256]rune {
	var t [256]rune
	n := 0
	for b := 0; b < 256; b++ {
		if (b >= '!' && b <= '~') || (b >= 0xA1 && b <= 0xAC) || (b >= 0xAE && b <= 0xFF) {
			t[b] = rune(b)
		} else {
			t[b] = rune(256 + n)
			n++
		}
	}
	return t
}()

func byteLevelEncode(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)
	for i := 0; i < len(s); i++ {
		b.WriteRune(byteToRune[s[i]])
	}
	return b.String()
}
