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
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Hugging Face tokenizers run their split patterns on Oniguruma. Go's RE2 differs in two ways that
// matter for the GPT-2, Llama 3 and Qwen patterns: \s, \d and \w are ASCII only, and there is no
// lookahead. translatePattern rewrites the classes, and the one lookahead those patterns use,
// \s+(?!\S), becomes a named group whose match is shortened by one rune when a non-space follows.

const wsClass = `\t\n\v\f\r \x{85}\x{A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}`

const wordClass = `\p{L}\p{M}\p{Nd}\p{Pc}`

const wsLookahead = `\s+(?!\S)`

const wsGroup = "ddosifyws"

type splitRegex struct {
	re      *regexp.Regexp
	wsGroup int // submatch index of the rewritten \s+(?!\S), -1 when the pattern has none
}

func compileOnig(pattern string) (*splitRegex, error) {
	hasLookahead := strings.Contains(pattern, wsLookahead)
	p := strings.ReplaceAll(pattern, wsLookahead, `(?P<`+wsGroup+`>\s+)`)
	for _, unsupported := range []string{"(?=", "(?!", "(?<=", "(?<!"} {
		if strings.Contains(p, unsupported) {
			return nil, fmt.Errorf("tokenizer split pattern uses a lookaround Go can not run: %q", pattern)
		}
	}
	translated, err := translateClasses(p)
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(translated)
	if err != nil {
		return nil, fmt.Errorf("tokenizer split pattern %q: %v", pattern, err)
	}
	sr := &splitRegex{re: re, wsGroup: -1}
	if hasLookahead {
		sr.wsGroup = re.SubexpIndex(wsGroup)
	}
	return sr, nil
}

func translateClasses(p string) (string, error) {
	var b strings.Builder
	inClass := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c == '\\' && i+1 < len(p) {
			n := p[i+1]
			i++
			switch n {
			case 's':
				if inClass {
					b.WriteString(wsClass)
				} else {
					b.WriteString("[" + wsClass + "]")
				}
			case 'S':
				if inClass {
					return "", fmt.Errorf("tokenizer split pattern: \\S inside a class is not supported")
				}
				b.WriteString("[^" + wsClass + "]")
			case 'd':
				b.WriteString(`\p{Nd}`)
			case 'D':
				b.WriteString(`\P{Nd}`)
			case 'w':
				if inClass {
					b.WriteString(wordClass)
				} else {
					b.WriteString("[" + wordClass + "]")
				}
			case 'W':
				if inClass {
					return "", fmt.Errorf("tokenizer split pattern: \\W inside a class is not supported")
				}
				b.WriteString("[^" + wordClass + "]")
			case 'u':
				if i+4 < len(p) {
					b.WriteString(`\x{` + p[i+1:i+5] + `}`)
					i += 4
				} else {
					return "", fmt.Errorf("tokenizer split pattern: bad \\u escape")
				}
			default:
				b.WriteByte('\\')
				b.WriteByte(n)
			}
			continue
		}
		switch {
		case c == '[' && !inClass:
			inClass = true
			b.WriteByte(c)
			// a ']' or '^]' right after '[' is a literal
			if i+1 < len(p) && p[i+1] == '^' {
				b.WriteByte('^')
				i++
			}
			if i+1 < len(p) && p[i+1] == ']' {
				b.WriteString(`\]`)
				i++
			}
		case c == ']' && inClass:
			inClass = false
			b.WriteByte(c)
		case c == '[' && inClass:
			return "", fmt.Errorf("tokenizer split pattern: nested classes are not supported")
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), nil
}

func isOnigSpace(r rune) bool {
	return unicode.IsSpace(r)
}

// matches returns the [start, end) byte spans the pattern matches in s, left to right.
func (sr *splitRegex) matches(s string) [][2]int {
	var out [][2]int
	pos := 0
	for pos < len(s) {
		loc := sr.re.FindStringSubmatchIndex(s[pos:])
		if loc == nil {
			break
		}
		start, end := pos+loc[0], pos+loc[1]
		if end == start {
			// an empty match: step one rune to make progress
			_, size := utf8.DecodeRuneInString(s[end:])
			pos = end + size
			continue
		}
		if sr.wsGroup >= 0 && loc[2*sr.wsGroup] >= 0 && end < len(s) {
			next, _ := utf8.DecodeRuneInString(s[end:])
			if !isOnigSpace(next) {
				last, size := utf8.DecodeLastRuneInString(s[start:end])
				if last != utf8.RuneError && end-size > start {
					end -= size
				}
			}
		}
		out = append(out, [2]int{start, end})
		pos = end
	}
	return out
}
