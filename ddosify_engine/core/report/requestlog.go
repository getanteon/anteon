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

package report

import (
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	"go.ddosify.com/ddosify/v2/core/types"
)

// RequestLogSetter is implemented by the report services that can write a line per request.
type RequestLogSetter interface {
	SetRequestLog(on bool)
}

// requestLog writes one "request" line per sent step. Only the report's goroutine calls it, the one
// that also writes the report line, so the lines on its writer never interleave. The buffer is reused.
type requestLog struct {
	buf []byte
}

func (l *requestLog) write(w io.Writer, scr *types.ScenarioResult) {
	stage, staged := stageOf(scr)
	intended, _ := scr.Others[types.OthersIntendedStart].(time.Time)
	for _, sr := range scr.StepResults {
		b := l.buf[:0]
		b = append(b, `{"type":"request"`...)
		sent := !sr.RequestTime.IsZero()
		if sent {
			b = appendTime(b, "time", sr.RequestTime.Add(sr.Duration))
		}
		b = append(b, `,"stage":`...)
		b = appendJSONString(b, stage.Name)
		if staged {
			b = append(b, `,"stage_index":`...)
			b = strconv.AppendInt(b, int64(stage.Index), 10)
		}
		b = append(b, `,"warmup":`...)
		b = strconv.AppendBool(b, staged && stage.Warmup)
		b = append(b, `,"iteration":`...)
		b = strconv.AppendInt(b, scr.Iteration, 10)
		b = append(b, `,"step_id":`...)
		b = strconv.AppendUint(b, uint64(sr.StepID), 10)
		b = append(b, `,"step_name":`...)
		b = appendJSONString(b, sr.StepName)
		if !intended.IsZero() {
			b = appendTime(b, "intended_start", intended)
		}
		if sent {
			b = appendTime(b, "sent", sr.RequestTime)
		}
		if sr.StatusCode != 0 {
			b = append(b, `,"status_code":`...)
			b = strconv.AppendInt(b, int64(sr.StatusCode), 10)
		}
		b = append(b, `,"outcome":"`...)
		b = append(b, OutcomeOf(sr)...)
		b = append(b, '"')
		if st := sr.Stream; st != nil && st.HasFirstToken {
			b = appendMs(b, "first_token_ms", st.FirstToken)
		}
		if sent {
			b = appendMs(b, "total_ms", sr.Duration)
		}
		if st := sr.Stream; st != nil && st.LLM != nil {
			if st.LLM.Tokens() {
				b = append(b, `,"output_tokens":`...)
			} else {
				b = append(b, `,"output_chunks":`...)
			}
			b = strconv.AppendInt(b, int64(st.LLM.Output), 10)
			if st.LLM.Input > 0 {
				b = append(b, `,"input_tokens":`...)
				b = strconv.AppendInt(b, int64(st.LLM.Input), 10)
			}
		}
		b = append(b, "}\n"...)
		_, _ = w.Write(b)
		l.buf = b
	}
}

func appendTime(b []byte, key string, t time.Time) []byte {
	b = append(b, ',', '"')
	b = append(b, key...)
	b = append(b, `":"`...)
	b = t.UTC().AppendFormat(b, time.RFC3339Nano)
	return append(b, '"')
}

func appendMs(b []byte, key string, d time.Duration) []byte {
	b = append(b, ',', '"')
	b = append(b, key...)
	b = append(b, `":`...)
	return strconv.AppendFloat(b, ms(d), 'f', 3, 64)
}

const hexDigits = "0123456789abcdef"

// appendJSONString quotes s as a JSON string; invalid UTF-8 becomes U+FFFD, as encoding/json does.
func appendJSONString(b []byte, s string) []byte {
	b = append(b, '"')
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			switch {
			case c == '"' || c == '\\':
				b = append(b, '\\', c)
			case c == '\n':
				b = append(b, '\\', 'n')
			case c == '\r':
				b = append(b, '\\', 'r')
			case c == '\t':
				b = append(b, '\\', 't')
			case c < 0x20:
				b = append(b, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			default:
				b = append(b, c)
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b = append(b, `�`...)
		} else {
			b = append(b, s[i:i+size]...)
		}
		i += size
	}
	return append(b, '"')
}
