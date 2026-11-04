package requester

import (
	"context"
	"errors"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

type collected struct {
	events   []sseEvent
	comments int
}

func parseChunks(chunks ...string) (*collected, *sseParser) {
	c := &collected{}
	p := newSSEParser(func(ev *sseEvent) { c.events = append(c.events, *ev) }, func() { c.comments++ })
	for _, ch := range chunks {
		p.feed([]byte(ch))
	}
	return c, p
}

func splitEvery(s string, n int) []string {
	out := []string{}
	for len(s) > n {
		out = append(out, s[:n])
		s = s[n:]
	}
	return append(out, s)
}

const aiSDKStream = "data: {\"type\":\"start\",\"messageId\":\"m1\"}\n\n" +
	": keepalive\n\n" +
	"id: 1\n" +
	"data: {\"type\":\"text-start\",\"id\":\"t\"}\n\n" +
	"data: {\"type\":\"text-delta\",\"id\":\"t\",\"delta\":\"Merhaba, \\u00e7\\u0131kt\\u0131 \"}\n\n" +
	"data: {\"type\":\"text-delta\",\"id\":\"t\",\"delta\":\"şöyle\"}\n\n" +
	"data: {\"type\":\"finish\"}\n\n" +
	"data: [DONE]\n\n"

func TestSSEParser_WholeAndByteByByte(t *testing.T) {
	whole, _ := parseChunks(aiSDKStream)
	if len(whole.events) != 6 {
		t.Fatalf("expected 6 events, got %d: %#v", len(whole.events), whole.events)
	}
	if whole.comments != 1 {
		t.Errorf("expected 1 comment, got %d", whole.comments)
	}
	if whole.events[1].ID != "1" || whole.events[0].Event != "message" {
		t.Errorf("unexpected id or event name: %#v", whole.events[:2])
	}
	if whole.events[5].Data != "[DONE]" {
		t.Errorf("last event should be the done marker, got %q", whole.events[5].Data)
	}

	for _, n := range []int{1, 2, 3, 7, 13} {
		got, _ := parseChunks(splitEvery(aiSDKStream, n)...)
		if !reflect.DeepEqual(got, whole) {
			t.Errorf("chunks of %d bytes parsed differently:\n%#v\n%#v", n, got.events, whole.events)
		}
	}
}

func TestSSEParser_LineEndings(t *testing.T) {
	lf := "event: a\ndata: x\n\ndata: y\n\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")
	cr := strings.ReplaceAll(lf, "\n", "\r")
	want, _ := parseChunks(lf)
	if len(want.events) != 2 || want.events[0].Event != "a" || want.events[1].Event != "message" {
		t.Fatalf("unexpected LF parse: %#v", want.events)
	}
	for name, in := range map[string]string{"crlf": crlf, "cr": cr} {
		got, _ := parseChunks(in)
		if !reflect.DeepEqual(got.events, want.events) {
			t.Errorf("%s: expected %#v, got %#v", name, want.events, got.events)
		}
		for n := 1; n < 6; n++ {
			got, _ := parseChunks(splitEvery(in, n)...)
			if !reflect.DeepEqual(got.events, want.events) {
				t.Errorf("%s in %d byte chunks: expected %#v, got %#v", name, n, want.events, got.events)
			}
		}
	}

	// a CR at the end of one chunk and its LF at the start of the next is one line end
	got, _ := parseChunks("data: x\r", "\n\r", "\n")
	if len(got.events) != 1 || got.events[0].Data != "x" {
		t.Errorf("split CRLF: got %#v", got.events)
	}
}

func TestSSEParser_MultiLineDataAndFields(t *testing.T) {
	got, _ := parseChunks("data: first\ndata:second\ndata\ndata:  two spaces\n\n")
	if len(got.events) != 1 {
		t.Fatalf("expected one event, got %#v", got.events)
	}
	if want := "first\nsecond\n\n two spaces"; got.events[0].Data != want {
		t.Errorf("expected %q, got %q", want, got.events[0].Data)
	}

	// a blank line without data dispatches nothing and resets the event name
	got, _ = parseChunks("event: lonely\n\n\n\ndata: z\n\nretry: 10\nunknown: v\n\n")
	if len(got.events) != 1 || got.events[0].Event != "message" || got.events[0].Data != "z" {
		t.Errorf("unexpected events %#v", got.events)
	}

	// a byte order mark before the first line is ignored
	got, _ = parseChunks("\xEF\xBB", "\xBFdata: bom\n\n")
	if len(got.events) != 1 || got.events[0].Data != "bom" {
		t.Errorf("bom: unexpected events %#v", got.events)
	}
}

func TestSSEParser_CommentsAndKeepalives(t *testing.T) {
	got, _ := parseChunks(":\n", ": ping\n\n", ":keepalive\r\n", "data: a\n", ": inside an event\n", "\n")
	if got.comments != 4 {
		t.Errorf("expected 4 comments, got %d", got.comments)
	}
	if len(got.events) != 1 || got.events[0].Data != "a" {
		t.Errorf("comments should not dispatch or alter events: %#v", got.events)
	}
}

func TestSSEParser_LongLineIsCut(t *testing.T) {
	c := &collected{}
	p := newSSEParser(func(ev *sseEvent) { c.events = append(c.events, *ev) }, nil)
	p.maxBytes = 16
	p.feed([]byte("data: " + strings.Repeat("x", 100) + "\n\ndata: ok\n\n"))
	if len(c.events) != 2 || !c.events[0].Truncated || len(c.events[0].Data) > 16 || c.events[1].Truncated {
		t.Errorf("unexpected events %#v", c.events)
	}
}

func TestSSEParser_Pending(t *testing.T) {
	_, p := parseChunks("data: a\n\n")
	if p.pending() {
		t.Error("a complete event is not pending")
	}
	_, p = parseChunks("data: a\n")
	if !p.pending() {
		t.Error("data without a blank line is pending")
	}
	_, p = parseChunks("data: a\n\ndata: half")
	if !p.pending() {
		t.Error("a half line is pending")
	}
}

func TestMatchEvent(t *testing.T) {
	td := "text-delta"
	ev := &sseEvent{Event: "message", Data: `{"type":"text-delta","delta":"x","choices":[{"delta":{"content":""}}]}`}
	cases := []struct {
		name string
		m    types.StreamMatch
		want bool
	}{
		{"empty matches all", types.StreamMatch{}, true},
		{"equals", types.StreamMatch{JsonPath: "type", Equals: &td}, true},
		{"equals other", types.StreamMatch{JsonPath: "type", Equals: strPtr("start")}, false},
		{"missing path", types.StreamMatch{JsonPath: "nope"}, false},
		{"path only needs a value", types.StreamMatch{JsonPath: "choices.0.delta.content"}, false},
		{"regexp on path", types.StreamMatch{JsonPath: "type", Regexp: "^text-"}, true},
		{"regexp on data", types.StreamMatch{Regexp: `"delta":"x"`}, true},
		{"event name", types.StreamMatch{Event: "token"}, false},
		{"event name default", types.StreamMatch{Event: "message"}, true},
	}
	for _, c := range cases {
		m := c.m
		if err := m.Compile(); err != nil {
			t.Fatal(err)
		}
		if got := matchEvent(&m, ev); got != c.want {
			t.Errorf("%s: expected %v, got %v", c.name, c.want, got)
		}
	}
	if matchEvent(&types.StreamMatch{JsonPath: "type"}, &sseEvent{Data: "not json"}) {
		t.Error("json_path on data that is not JSON should not match")
	}
}

func strPtr(s string) *string { return &s }

func TestIsEventStream(t *testing.T) {
	for in, want := range map[string]bool{
		"text/event-stream":                true,
		"text/event-stream; charset=utf-8": true,
		"Text/Event-Stream":                true,
		"application/json":                 false,
		"":                                 false,
	} {
		if got := isEventStream(in); got != want {
			t.Errorf("%q: expected %v, got %v", in, want, got)
		}
	}
}

// scriptedReader returns its chunks, then err. Reading after the done marker fails the test.
type scriptedReader struct {
	t      *testing.T
	chunks []string
	err    error
}

func (r *scriptedReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, r.err
	}
	n := copy(p, r.chunks[0])
	if n < len(r.chunks[0]) {
		r.chunks[0] = r.chunks[0][n:]
	} else {
		r.chunks = r.chunks[1:]
	}
	return n, nil
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "read timed out" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

var _ net.Error = timeoutErr{}

func TestReadSSE_Ends(t *testing.T) {
	conf := types.DefaultStreamConf()
	start := time.Now()
	cases := []struct {
		name      string
		chunks    []string
		err       error
		end       string
		events    int
		hasErr    bool
		errEvents int
	}{
		{"done", splitEvery(aiSDKStream, 5), errors.New("must not read after done"), types.StreamEndDone, 5, false, 0},
		{"closed between events", []string{"data: {\"type\":\"text-delta\"}\n\n"}, io.EOF, types.StreamEndClosed, 1, false, 0},
		{"eof inside an event", []string{"data: {\"type\":\"text-delta\"}\n\ndata: {\"ty"}, io.EOF, types.StreamEndCut, 1, false, 0},
		{"connection reset", []string{"data: {\"type\":\"start\"}\n\n"}, io.ErrUnexpectedEOF, types.StreamEndCut, 1, true, 0},
		{"timeout", []string{"data: {\"type\":\"start\"}\n\n: keepalive\n\n"}, timeoutErr{}, types.StreamEndTimeout, 1, true, 0},
		{"error chunk", []string{"data: {\"type\":\"start\"}\n\ndata: {\"type\":\"error\",\"errorText\":\"x\"}\n\ndata: [DONE]\n\n"}, io.EOF, types.StreamEndDone, 2, false, 1},
	}
	for _, c := range cases {
		r := &scriptedReader{t: t, chunks: append([]string{}, c.chunks...), err: c.err}
		out := readSSE(context.Background(), r, conf, start, start, false)
		res := out.result
		if res.End != c.end || res.Events != c.events || (out.err != nil) != c.hasErr || res.ErrorEvents != c.errEvents {
			t.Errorf("%s: got end %q events %d err %v error events %d", c.name, res.End, res.Events, out.err, res.ErrorEvents)
		}
		if out.body != nil {
			t.Errorf("%s: body kept although not asked", c.name)
		}
	}

	out := readSSE(context.Background(), &scriptedReader{chunks: []string{aiSDKStream}, err: io.EOF}, conf, start, start, false)
	if !out.result.HasFirstEvent || !out.result.HasFirstToken || out.result.FirstToken < out.result.FirstEvent {
		t.Errorf("first event and first token should be set in order: %#v", out.result)
	}
}

func TestReadSSE_BodyCap(t *testing.T) {
	conf := types.DefaultStreamConf()
	conf.MaxBodyBytes = 10
	now := time.Now()
	out := readSSE(context.Background(), &scriptedReader{chunks: []string{aiSDKStream}, err: io.EOF}, conf, now, now, true)
	if len(out.body) != 10 || !out.result.BodyTruncated {
		t.Errorf("expected a 10 byte truncated body, got %d %v", len(out.body), out.result.BodyTruncated)
	}
	if out.result.Events != 5 {
		t.Errorf("the cap must not change parsing, got %d events", out.result.Events)
	}

	conf.MaxBodyBytes = 0
	out = readSSE(context.Background(), &scriptedReader{chunks: []string{aiSDKStream}, err: io.EOF}, conf, now, now, true)
	if string(out.body) != aiSDKStream || out.result.BodyTruncated {
		t.Errorf("0 keeps the whole body, got %d bytes", len(out.body))
	}
}

func TestGapRecorder(t *testing.T) {
	var g gapRecorder
	for i := 1; i <= 100; i++ {
		g.add(time.Duration(i) * time.Millisecond)
	}
	var r types.StreamResult
	g.fill(&r)
	if r.GapMax != 100*time.Millisecond || r.GapP50 != 50*time.Millisecond || r.GapP95 != 95*time.Millisecond {
		t.Errorf("unexpected gaps max %v p50 %v p95 %v", r.GapMax, r.GapP50, r.GapP95)
	}

	var big gapRecorder
	for i := 0; i < sseMaxGapSamples*3; i++ {
		big.add(time.Millisecond)
	}
	if len(big.samples) != sseMaxGapSamples {
		t.Errorf("samples should stay bounded, got %d", len(big.samples))
	}
}
