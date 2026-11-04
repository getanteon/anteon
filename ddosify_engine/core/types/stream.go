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

package types

import (
	"fmt"
	"regexp"
	"time"
)

const (
	// StreamTypeSSE reads a response as server-sent events.
	StreamTypeSSE = "sse"

	// DefaultStreamDone is the data payload that ends an OpenAI or AI SDK stream.
	DefaultStreamDone = "[DONE]"

	// DefaultStreamMaxBodyBytes caps the response body kept for assertions, captures and debug.
	DefaultStreamMaxBodyBytes = 1 << 20

	// Ends of a stream
	StreamEndDone    = "done"
	StreamEndClosed  = "closed"
	StreamEndCut     = "cut"
	StreamEndTimeout = "timeout"

	// Load spreads
	LoadSpreadFront = "front"
	LoadSpreadEven  = "even"
)

// StreamMatch selects SSE events. Every field that is set has to match.
type StreamMatch struct {
	// SSE event name, the "event:" field. Empty matches any.
	Event string

	// gjson path into the event data parsed as JSON.
	JsonPath string

	// The value at JsonPath must equal this.
	Equals *string

	// The data, or the value at JsonPath, must match this expression.
	Regexp string

	re *regexp.Regexp
}

func (m *StreamMatch) IsZero() bool {
	return m.Event == "" && m.JsonPath == "" && m.Equals == nil && m.Regexp == ""
}

// Compile prepares the expression. Safe to call more than once.
func (m *StreamMatch) Compile() error {
	if m.Regexp == "" || m.re != nil {
		return nil
	}
	re, err := regexp.Compile(m.Regexp)
	if err != nil {
		return fmt.Errorf("stream match regexp is not valid: %v", err)
	}
	m.re = re
	return nil
}

func (m *StreamMatch) Re() *regexp.Regexp {
	return m.re
}

// StreamConf is the step's "stream" config.
type StreamConf struct {
	Type string

	// Marks the first token: by default the first event whose JSON "type" is "text-delta".
	FirstToken StreamMatch

	// Marks an error inside a 200 stream: by default an event whose JSON "type" is "error".
	Error StreamMatch

	// Data payload that ends the stream.
	Done string

	// Cap on the response body kept when an assertion, a capture or debug needs it. 0 is no cap.
	MaxBodyBytes int
}

func stringPtr(s string) *string { return &s }

// DefaultStreamConf fits the Vercel AI SDK UI message stream.
func DefaultStreamConf() *StreamConf {
	return &StreamConf{
		Type:         StreamTypeSSE,
		FirstToken:   StreamMatch{JsonPath: "type", Equals: stringPtr("text-delta")},
		Error:        StreamMatch{JsonPath: "type", Equals: stringPtr("error")},
		Done:         DefaultStreamDone,
		MaxBodyBytes: DefaultStreamMaxBodyBytes,
	}
}

func (c *StreamConf) validate() error {
	if c.Type != StreamTypeSSE {
		return fmt.Errorf("unsupported stream type: %q, supported: %q", c.Type, StreamTypeSSE)
	}
	if c.MaxBodyBytes < 0 {
		return fmt.Errorf("stream max_body_bytes can not be negative")
	}
	if err := c.FirstToken.Compile(); err != nil {
		return err
	}
	return c.Error.Compile()
}

// StreamResult is what one streamed response measured. Durations are from the moment the request was sent.
type StreamResult struct {
	// Time to the first byte of the response.
	TTFB time.Duration

	// Time to the first event. Zero with HasFirstEvent false when no event came.
	FirstEvent    time.Duration
	HasFirstEvent bool

	// Time to the first event matching StreamConf.FirstToken.
	FirstToken    time.Duration
	HasFirstToken bool

	// Time to the end of the body.
	Total time.Duration

	// Events dispatched, the done marker excluded.
	Events int

	// Comment lines, the keepalives.
	Comments int

	// Events matching StreamConf.Error.
	ErrorEvents int

	// Gaps between consecutive events.
	GapMax time.Duration
	GapP50 time.Duration
	GapP95 time.Duration

	// One of StreamEndDone, StreamEndClosed, StreamEndCut, StreamEndTimeout.
	End string

	// Why a stream was cut.
	EndReason string

	Bytes         int64
	BodyTruncated bool

	// Metrics hooks.StreamObserver implementations returned, such as "output_tokens"
	Extra map[string]float64
}

// Clean tells whether the stream ended with the done marker or a close between events.
func (r *StreamResult) Clean() bool {
	return r.End == StreamEndDone || r.End == StreamEndClosed
}

// SLO holds thresholds in milliseconds. Zero means the threshold is not checked.
type SLO struct {
	TTFBMs       int64 `json:"ttfb_ms,omitempty"`
	FirstEventMs int64 `json:"first_event_ms,omitempty"`
	FirstTokenMs int64 `json:"first_token_ms,omitempty"`
	TotalMs      int64 `json:"total_ms,omitempty"`
	MaxGapMs     int64 `json:"max_gap_ms,omitempty"`

	// The test fails, exit code 3, when the goodput is below this percentage
	GoodputPerc float64 `json:"goodput_perc,omitempty"`
}

func (s *SLO) validate() error {
	if s.TTFBMs < 0 || s.FirstEventMs < 0 || s.FirstTokenMs < 0 || s.TotalMs < 0 || s.MaxGapMs < 0 {
		return fmt.Errorf("slo thresholds can not be negative")
	}
	if s.GoodputPerc < 0 || s.GoodputPerc > 100 {
		return fmt.Errorf("slo goodput_perc must be between 0 and 100")
	}
	return nil
}

// Met tells whether a stream meets the thresholds.
func (s *SLO) Met(r *StreamResult) bool {
	if !r.Clean() || r.ErrorEvents > 0 {
		return false
	}
	within := func(limit int64, d time.Duration, present bool) bool {
		return limit == 0 || (present && d <= time.Duration(limit)*time.Millisecond)
	}
	return within(s.TTFBMs, r.TTFB, true) &&
		within(s.FirstEventMs, r.FirstEvent, r.HasFirstEvent) &&
		within(s.FirstTokenMs, r.FirstToken, r.HasFirstToken) &&
		within(s.TotalMs, r.Total, true) &&
		within(s.MaxGapMs, r.GapMax, true)
}

// LoadStage tags an iteration with the manual_load stage it was started in.
type LoadStage struct {
	Index  int
	Name   string
	Warmup bool
}
