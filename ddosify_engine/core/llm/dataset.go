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
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// Dataset is a file of prompts: JSONL, CSV or plain text, one session a line.
type Dataset struct {
	Path string `json:"path"`

	// jsonl, csv or txt; by default from the extension
	Format string `json:"format"`

	// JSONL key, or CSV column (a header name or a 0 based index), of the prompt
	Field string `json:"field"`

	// CSV columns of a session's turns, in order
	Fields []string `json:"fields"`

	// The CSV's first row is a header. Implied when a column is named.
	Header bool `json:"header"`

	// Read at most this many sessions; 0 reads all
	Limit int `json:"limit"`
}

const maxLineBytes = 16 << 20

// Load reads the sessions, each a list of user turns.
func (d *Dataset) Load() ([][]string, error) {
	f, err := os.Open(d.Path)
	if err != nil {
		return nil, fmt.Errorf("llm dataset: %v", err)
	}
	defer f.Close()
	format := strings.ToLower(d.Format)
	if format == "" {
		switch strings.ToLower(filepath.Ext(d.Path)) {
		case ".jsonl", ".ndjson", ".json":
			format = "jsonl"
		case ".csv":
			format = "csv"
		default:
			format = "txt"
		}
	}
	var sessions [][]string
	switch format {
	case "jsonl":
		sessions, err = d.readJSONL(f)
	case "csv":
		sessions, err = d.readCSV(f)
	case "txt":
		sessions, err = d.readText(f)
	default:
		return nil, fmt.Errorf("llm dataset format %q is not supported: jsonl, csv or txt", d.Format)
	}
	if err != nil {
		return nil, fmt.Errorf("llm dataset %s: %v", d.Path, err)
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("llm dataset %s has no prompts", d.Path)
	}
	return sessions, nil
}

func (d *Dataset) full(n int) bool { return d.Limit > 0 && n >= d.Limit }

func newScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), maxLineBytes)
	return sc
}

func (d *Dataset) readText(r io.Reader) ([][]string, error) {
	var out [][]string
	sc := newScanner(r)
	for sc.Scan() && !d.full(len(out)) {
		if line := strings.TrimRight(sc.Text(), "\r"); strings.TrimSpace(line) != "" {
			out = append(out, []string{line})
		}
	}
	return out, sc.Err()
}

func (d *Dataset) readJSONL(r io.Reader) ([][]string, error) {
	var out [][]string
	sc := newScanner(r)
	line := 0
	for sc.Scan() && !d.full(len(out)) {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		if !gjson.Valid(text) {
			return nil, fmt.Errorf("line %d is not JSON", line)
		}
		turns := jsonTurns(gjson.Parse(text), d.Field)
		if len(turns) == 0 {
			return nil, fmt.Errorf("line %d has no prompt", line)
		}
		out = append(out, turns)
	}
	return out, sc.Err()
}

func stringsOf(v gjson.Result) []string {
	if v.Type == gjson.String {
		if v.Str == "" {
			return nil
		}
		return []string{v.Str}
	}
	var out []string
	if v.IsArray() {
		for _, e := range v.Array() {
			if e.Type == gjson.String && e.Str != "" {
				out = append(out, e.Str)
			}
		}
	}
	return out
}

// jsonTurns reads a row: "turns", OpenAI "messages", ShareGPT "conversations", or a prompt field.
func jsonTurns(row gjson.Result, field string) []string {
	if field != "" {
		return stringsOf(row.Get(field))
	}
	if t := stringsOf(row.Get("turns")); len(t) > 0 {
		return t
	}
	var out []string
	for _, m := range row.Get("messages").Array() {
		if m.Get("role").Str == "user" {
			if c := m.Get("content"); c.Type == gjson.String && c.Str != "" {
				out = append(out, c.Str)
			}
		}
	}
	if len(out) > 0 {
		return out
	}
	for _, m := range row.Get("conversations").Array() {
		if from := m.Get("from").Str; from == "human" || from == "user" {
			if v := m.Get("value").Str; v != "" {
				out = append(out, v)
			}
		}
	}
	if len(out) > 0 {
		return out
	}
	for _, k := range []string{"prompt", "text", "question", "input"} {
		if t := stringsOf(row.Get(k)); len(t) > 0 {
			return t
		}
	}
	return nil
}

func (d *Dataset) readCSV(r io.Reader) ([][]string, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	columns := d.Fields
	if len(columns) == 0 {
		columns = []string{d.Field}
		if d.Field == "" {
			columns = []string{"0"}
		}
	}
	header := d.Header
	for _, c := range columns {
		if _, err := strconv.Atoi(c); err != nil {
			header = true
		}
	}
	idx := make([]int, len(columns))
	if header {
		names, err := cr.Read()
		if err != nil {
			return nil, err
		}
		for i, c := range columns {
			idx[i] = -1
			if n, err := strconv.Atoi(c); err == nil {
				idx[i] = n
				continue
			}
			for j, name := range names {
				if strings.TrimSpace(strings.TrimPrefix(name, "\ufeff")) == c {
					idx[i] = j
				}
			}
			if idx[i] < 0 {
				return nil, fmt.Errorf("column %q is not in the header", c)
			}
		}
	} else {
		for i, c := range columns {
			idx[i], _ = strconv.Atoi(c)
		}
	}
	var out [][]string
	for !d.full(len(out)) {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		var turns []string
		for _, i := range idx {
			if i < len(rec) && strings.TrimSpace(rec[i]) != "" {
				turns = append(turns, rec[i])
			}
		}
		if len(turns) > 0 {
			out = append(out, turns)
		}
	}
	return out, nil
}
