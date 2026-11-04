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

package requester

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"mime"
	"net"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"go.ddosify.com/ddosify/core/types"
)

const (
	// An event's data, and a line, are cut beyond this so one huge event can not grow memory.
	sseMaxEventBytes = 1 << 20

	// Gaps kept per stream for its percentiles; a reservoir sample beyond this.
	sseMaxGapSamples = 4096

	sseReadBufSize = 32 * 1024
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

var sseReadBufPool = sync.Pool{New: func() interface{} {
	b := make([]byte, sseReadBufSize)
	return &b
}}

type sseEvent struct {
	Event     string // "message" when the stream gave none
	Data      string
	ID        string
	Truncated bool
}

// sseParser follows the WHATWG event stream interpretation: LF, CR or CRLF end a line,
// a blank line dispatches, a line starting with ':' is a comment, data lines are joined with LF.
// Chunks can split a line, or a CRLF, anywhere.
type sseParser struct {
	onEvent   func(ev *sseEvent)
	onComment func()

	line     []byte
	lineCut  bool
	afterCR  bool
	firstLn  bool
	data     bytes.Buffer
	hasData  bool
	dataCut  bool
	event    string
	lastID   string
	maxBytes int
}

func newSSEParser(onEvent func(ev *sseEvent), onComment func()) *sseParser {
	return &sseParser{onEvent: onEvent, onComment: onComment, firstLn: true, maxBytes: sseMaxEventBytes}
}

func (p *sseParser) feed(chunk []byte) {
	for len(chunk) > 0 {
		if p.afterCR {
			p.afterCR = false
			if chunk[0] == '\n' {
				chunk = chunk[1:]
				continue
			}
		}
		i := bytes.IndexAny(chunk, "\r\n")
		if i < 0 {
			p.appendLine(chunk)
			return
		}
		p.appendLine(chunk[:i])
		p.processLine()
		if chunk[i] == '\r' {
			p.afterCR = true
		}
		chunk = chunk[i+1:]
	}
}

func (p *sseParser) appendLine(b []byte) {
	room := p.maxBytes - len(p.line)
	if len(b) > room {
		b = b[:room]
		p.lineCut = true
	}
	p.line = append(p.line, b...)
}

// pending tells whether the stream stopped in the middle of an event.
func (p *sseParser) pending() bool {
	return p.hasData || len(p.line) > 0
}

func (p *sseParser) processLine() {
	line := p.line
	cut := p.lineCut
	p.line = p.line[:0]
	p.lineCut = false

	if p.firstLn {
		p.firstLn = false
		line = bytes.TrimPrefix(line, utf8BOM)
	}

	if len(line) == 0 {
		p.dispatch()
		return
	}
	if line[0] == ':' {
		if p.onComment != nil {
			p.onComment()
		}
		return
	}

	var field, value []byte
	if i := bytes.IndexByte(line, ':'); i >= 0 {
		field, value = line[:i], line[i+1:]
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}
	} else {
		field = line
	}

	switch string(field) {
	case "data":
		if p.hasData {
			p.appendData([]byte{'\n'})
		}
		p.appendData(value)
		p.hasData = true
		if cut {
			p.dataCut = true
		}
	case "event":
		p.event = string(value)
	case "id":
		if bytes.IndexByte(value, 0) < 0 {
			p.lastID = string(value)
		}
	}
}

func (p *sseParser) appendData(b []byte) {
	room := p.maxBytes - p.data.Len()
	if len(b) > room {
		b = b[:room]
		p.dataCut = true
	}
	p.data.Write(b)
}

func (p *sseParser) dispatch() {
	if !p.hasData {
		p.event = ""
		return
	}
	ev := &sseEvent{Event: p.event, Data: p.data.String(), ID: p.lastID, Truncated: p.dataCut}
	if ev.Event == "" {
		ev.Event = "message"
	}
	p.data.Reset()
	p.hasData = false
	p.dataCut = false
	p.event = ""
	if p.onEvent != nil {
		p.onEvent(ev)
	}
}

// matchEvent: an empty match matches every event. json_path alone needs a value that is not empty.
func matchEvent(m *types.StreamMatch, ev *sseEvent) bool {
	if m.Event != "" && ev.Event != m.Event {
		return false
	}
	value := ev.Data
	if m.JsonPath != "" {
		r := gjson.Get(ev.Data, m.JsonPath)
		if !r.Exists() {
			return false
		}
		value = r.String()
		if m.Equals == nil && m.Regexp == "" && value == "" {
			return false
		}
	}
	if m.Equals != nil && value != *m.Equals {
		return false
	}
	if m.Regexp != "" {
		re := m.Re()
		if re == nil { // not compiled by Init
			ok, err := regexp.MatchString(m.Regexp, value)
			return err == nil && ok
		}
		if !re.MatchString(value) {
			return false
		}
	}
	return true
}

func isEventStream(contentType string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "text/event-stream")
	}
	return mt == "text/event-stream"
}

type gapRecorder struct {
	max     time.Duration
	seen    int64
	samples []time.Duration
}

func (g *gapRecorder) add(d time.Duration) {
	if d > g.max {
		g.max = d
	}
	g.seen++
	if len(g.samples) < sseMaxGapSamples {
		g.samples = append(g.samples, d)
		return
	}
	if j := rand.Int63n(g.seen); j < sseMaxGapSamples {
		g.samples[j] = d
	}
}

func (g *gapRecorder) percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p*float64(len(sorted))+0.999999) - 1 // nearest rank
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func (g *gapRecorder) fill(r *types.StreamResult) {
	r.GapMax = g.max
	if len(g.samples) == 0 {
		return
	}
	sort.Slice(g.samples, func(i, j int) bool { return g.samples[i] < g.samples[j] })
	r.GapP50 = g.percentile(g.samples, 0.50)
	r.GapP95 = g.percentile(g.samples, 0.95)
}

// cappedBuffer keeps at most limit bytes; limit 0 keeps everything.
type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (c *cappedBuffer) Write(b []byte) {
	if c.limit > 0 {
		room := c.limit - c.buf.Len()
		if room <= 0 {
			if len(b) > 0 {
				c.truncated = true
			}
			return
		}
		if len(b) > room {
			b = b[:room]
			c.truncated = true
		}
	}
	c.buf.Write(b)
}

// streamRead holds what reading one streamed body gave.
type streamRead struct {
	result *types.StreamResult
	body   []byte
	err    error // a read error that is not the end of the stream
}

// eventObserver sees each event's data at its time from the send; the llm package counts tokens with it.
type eventObserver interface {
	Event(data string, at time.Duration)
}

// readSSE reads body as server-sent events until the done marker, EOF or an error.
// sentAt is when the request was sent and firstByteAt when the first response byte came.
func readSSE(ctx context.Context, body io.Reader, conf *types.StreamConf, sentAt, firstByteAt time.Time, keepBody bool) streamRead {
	return readSSEObserved(ctx, body, conf, sentAt, firstByteAt, keepBody, nil)
}

func readSSEObserved(ctx context.Context, body io.Reader, conf *types.StreamConf, sentAt, firstByteAt time.Time, keepBody bool, obs eventObserver) streamRead {
	res := &types.StreamResult{}
	if !firstByteAt.IsZero() {
		res.TTFB = firstByteAt.Sub(sentAt)
	}

	var kept *cappedBuffer
	if keepBody {
		kept = &cappedBuffer{limit: conf.MaxBodyBytes}
	}

	var now, lastEventAt time.Time
	var gaps gapRecorder
	done := false

	parser := newSSEParser(func(ev *sseEvent) {
		if done {
			return
		}
		if conf.Done != "" && ev.Data == conf.Done {
			done = true
			return
		}
		res.Events++
		if !res.HasFirstEvent {
			res.HasFirstEvent = true
			res.FirstEvent = now.Sub(sentAt)
		} else {
			gaps.add(now.Sub(lastEventAt))
		}
		lastEventAt = now
		if !res.HasFirstToken && matchEvent(&conf.FirstToken, ev) {
			res.HasFirstToken = true
			res.FirstToken = now.Sub(sentAt)
		}
		if !conf.Error.IsZero() && matchEvent(&conf.Error, ev) {
			res.ErrorEvents++
		}
		if obs != nil {
			obs.Event(ev.Data, now.Sub(sentAt))
		}
	}, func() {
		res.Comments++
	})

	bufp := sseReadBufPool.Get().(*[]byte)
	defer sseReadBufPool.Put(bufp)
	buf := *bufp

	var readErr error
	for !done {
		n, err := body.Read(buf)
		now = time.Now()
		if n > 0 {
			res.Bytes += int64(n)
			if kept != nil {
				kept.Write(buf[:n])
			}
			parser.feed(buf[:n])
		}
		if err != nil {
			readErr = err
			break
		}
	}
	res.Total = now.Sub(sentAt)
	gaps.fill(res)

	out := streamRead{result: res}
	if kept != nil {
		out.body = kept.buf.Bytes()
		res.BodyTruncated = kept.truncated
	}

	switch {
	case done:
		res.End = types.StreamEndDone
	case readErr == io.EOF:
		if parser.pending() {
			res.End = types.StreamEndCut
			res.EndReason = "stream ended inside an event"
		} else {
			res.End = types.StreamEndClosed
		}
	case isTimeout(readErr):
		res.End = types.StreamEndTimeout
		res.EndReason = readErr.Error()
		out.err = readErr
	default:
		res.End = types.StreamEndCut
		res.EndReason = readErr.Error()
		out.err = readErr
	}
	if ctx != nil && ctx.Err() == context.Canceled && out.err != nil {
		out.err = context.Canceled
	}
	return out
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}
