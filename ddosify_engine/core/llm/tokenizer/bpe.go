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
	"container/heap"
	"encoding/json"
	"fmt"
	"strings"
)

type pair struct{ a, b int }

type merge struct {
	rank int
	id   int
}

type bpe struct {
	vocab        map[string]int
	merges       map[pair]merge
	byteFallback bool
	ignoreMerges bool
	fuseUnk      bool
	unk          int // -1 without an unk token
	byteIDs      [256]int
}

type bpeModel struct {
	Type                    string            `json:"type"`
	Vocab                   map[string]int    `json:"vocab"`
	Merges                  []json.RawMessage `json:"merges"`
	UnkToken                *string           `json:"unk_token"`
	ByteFallback            bool              `json:"byte_fallback"`
	IgnoreMerges            bool              `json:"ignore_merges"`
	FuseUnk                 bool              `json:"fuse_unk"`
	ContinuingSubwordPrefix *string           `json:"continuing_subword_prefix"`
	EndOfWordSuffix         *string           `json:"end_of_word_suffix"`
	Dropout                 *float64          `json:"dropout"`
}

func newBPE(m *bpeModel) (*bpe, error) {
	if m.Type != "BPE" {
		return nil, fmt.Errorf("unsupported tokenizer model %q, only BPE is supported", m.Type)
	}
	if (m.ContinuingSubwordPrefix != nil && *m.ContinuingSubwordPrefix != "") || (m.EndOfWordSuffix != nil && *m.EndOfWordSuffix != "") {
		return nil, fmt.Errorf("BPE with a continuing_subword_prefix or end_of_word_suffix is not supported")
	}
	if len(m.Vocab) == 0 {
		return nil, fmt.Errorf("BPE model has no vocab")
	}
	b := &bpe{
		vocab:        m.Vocab,
		merges:       make(map[pair]merge, len(m.Merges)),
		byteFallback: m.ByteFallback,
		ignoreMerges: m.IgnoreMerges,
		fuseUnk:      m.FuseUnk,
		unk:          -1,
	}
	if m.UnkToken != nil {
		if id, ok := m.Vocab[*m.UnkToken]; ok {
			b.unk = id
		}
	}
	for i := 0; i < 256; i++ {
		b.byteIDs[i] = -1
		if id, ok := m.Vocab[fmt.Sprintf("<0x%02X>", i)]; ok {
			b.byteIDs[i] = id
		}
	}
	for rank, raw := range m.Merges {
		left, right, err := parseMerge(raw)
		if err != nil {
			return nil, fmt.Errorf("merge %d: %v", rank, err)
		}
		a, okA := m.Vocab[left]
		c, okB := m.Vocab[right]
		id, okN := m.Vocab[left+right]
		if !okA || !okB || !okN {
			return nil, fmt.Errorf("merge %d %q %q names a token that is not in the vocab", rank, left, right)
		}
		p := pair{a, c}
		if _, exists := b.merges[p]; !exists {
			b.merges[p] = merge{rank: rank, id: id}
		}
	}
	return b, nil
}

// parseMerge reads either form tokenizer.json uses: "a b" or ["a", "b"].
func parseMerge(raw json.RawMessage) (string, string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		parts := strings.SplitN(s, " ", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("bad merge %q", s)
		}
		return parts[0], parts[1], nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) != 2 {
		return "", "", fmt.Errorf("bad merge %s", string(raw))
	}
	return arr[0], arr[1], nil
}

type symbol struct {
	id         int
	prev, next int
	alive      bool
}

type candidate struct {
	rank, pos, left, right, id int
}

type candidates []candidate

func (c candidates) Len() int { return len(c) }
func (c candidates) Less(i, j int) bool {
	if c[i].rank != c[j].rank {
		return c[i].rank < c[j].rank
	}
	return c[i].pos < c[j].pos
}
func (c candidates) Swap(i, j int)       { c[i], c[j] = c[j], c[i] }
func (c *candidates) Push(x interface{}) { *c = append(*c, x.(candidate)) }
func (c *candidates) Pop() interface{} {
	old := *c
	x := old[len(old)-1]
	*c = old[:len(old)-1]
	return x
}

// encodeWord appends the ids of one pre-token, merging the lowest ranked pair first as Hugging Face does.
func (b *bpe) encodeWord(word string, out []int) []int {
	if b.ignoreMerges {
		if id, ok := b.vocab[word]; ok {
			return append(out, id)
		}
	}
	syms := make([]symbol, 0, len(word))
	push := func(id int) {
		if id == b.unk && b.fuseUnk && len(syms) > 0 && syms[len(syms)-1].id == b.unk {
			return
		}
		syms = append(syms, symbol{id: id, alive: true})
	}
	for _, r := range word {
		s := string(r)
		if id, ok := b.vocab[s]; ok {
			push(id)
			continue
		}
		if b.byteFallback {
			all := true
			for i := 0; i < len(s); i++ {
				if b.byteIDs[s[i]] < 0 {
					all = false
					break
				}
			}
			if all {
				for i := 0; i < len(s); i++ {
					push(b.byteIDs[s[i]])
				}
				continue
			}
		}
		push(b.unk)
	}
	for i := range syms {
		syms[i].prev = i - 1
		syms[i].next = i + 1
	}
	if len(syms) > 0 {
		syms[len(syms)-1].next = -1
	}

	h := make(candidates, 0, len(syms))
	addPair := func(i int) {
		if i < 0 || syms[i].next < 0 {
			return
		}
		j := syms[i].next
		if m, ok := b.merges[pair{syms[i].id, syms[j].id}]; ok {
			heap.Push(&h, candidate{rank: m.rank, pos: i, left: syms[i].id, right: syms[j].id, id: m.id})
		}
	}
	for i := 0; i+1 < len(syms); i++ {
		addPair(i)
	}
	for h.Len() > 0 {
		c := heap.Pop(&h).(candidate)
		s := &syms[c.pos]
		if !s.alive || s.id != c.left || s.next < 0 {
			continue
		}
		n := &syms[s.next]
		if n.id != c.right {
			continue
		}
		s.id = c.id
		n.alive = false
		s.next = n.next
		if n.next >= 0 {
			syms[n.next].prev = c.pos
		}
		addPair(s.prev)
		addPair(c.pos)
	}
	for i := 0; i >= 0 && i < len(syms); i = syms[i].next {
		if syms[i].id >= 0 || b.unk < 0 {
			out = append(out, syms[i].id)
		}
	}
	return out
}
