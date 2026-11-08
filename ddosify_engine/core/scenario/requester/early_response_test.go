package requester

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

// earlyResponder is a server that answers "503 busy" before it has read the request body, as a
// busy file service does to a large upload. It reads the request head and then readFirst bytes of
// the body (none when readFirst is 0), writes the answer, and then either closes the connection or
// keeps it open without reading, so the client's body write stalls on full socket buffers.
func earlyResponder(t *testing.T, readFirst int, keepOpen bool) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var wg sync.WaitGroup
	t.Cleanup(func() {
		close(done)
		ln.Close()
		wg.Wait()
	})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer conn.Close()
				r := bufio.NewReader(conn)
				if readFirst > 0 {
					for { // the request head
						line, err := r.ReadString('\n')
						if err != nil {
							return
						}
						if line == "\r\n" {
							break
						}
					}
					if _, err := io.ReadFull(r, make([]byte, readFirst)); err != nil {
						return
					}
				}
				io.WriteString(conn, "HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain\r\n"+
					"Content-Length: 4\r\nConnection: close\r\n\r\nbusy")
				if keepOpen {
					select {
					case <-done:
					case <-time.After(30 * time.Second):
					}
				}
			}()
		}
	}()
	return "http://" + ln.Addr().String() + "/upload"
}

// A server that answers before the client finished writing a large body must not leave Send
// waiting on a timing phase that never comes: the result arrives within the step timeout and
// carries the 503, whether or not the step has assertions (which read the response time before
// the result is built).
func TestSendReturnsWhenServerAnswersBeforeTheBodyIsWritten(t *testing.T) {
	payload := strings.Repeat("x", 8<<20) // 8 MB, far past the socket buffers
	cases := []struct {
		name       string
		readFirst  int
		keepOpen   bool
		assertions []string
	}{
		{"reads a few bytes, closes, asserts", 16, false, []string{"equals(status_code,503)"}},
		{"reads a few bytes, keeps the connection, asserts", 16, true, []string{"equals(status_code,503)"}},
		{"reads nothing, keeps the connection, asserts", 0, true, []string{"equals(status_code,503)"}},
		{"reads a few bytes, keeps the connection, no assertions", 16, true, nil},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			url := earlyResponder(t, c.readFirst, c.keepOpen)
			s := types.ScenarioStep{
				ID:         1,
				Method:     http.MethodPost,
				URL:        url,
				Payload:    payload,
				Headers:    map[string]string{"Content-Type": "application/octet-stream"},
				Timeout:    5,
				Assertions: c.assertions,
			}
			h := &HttpRequester{}
			if err := h.Init(context.Background(), s, nil, false, nil); err != nil {
				t.Fatal(err)
			}
			got := make(chan *types.ScenarioStepResult, 1)
			go func() { got <- h.Send(nil, map[string]interface{}{}) }()

			var res *types.ScenarioStepResult
			select {
			case res = <-got:
			case <-time.After(15 * time.Second): // three times the step timeout
				t.Fatal("Send hung after an early response, past the step timeout")
			}
			if c.readFirst == 0 && !c.keepOpen && res.Err.Type != "" {
				return // a server that closes before reading may reset the connection first
			}
			if res.StatusCode != http.StatusServiceUnavailable || res.Err.Type != "" {
				t.Fatalf("expected the 503 counted as a response, got status %d err %+v", res.StatusCode, res.Err)
			}
			if len(res.FailedAssertions) != 0 {
				t.Errorf("assertions saw a wrong response: %+v", res.FailedAssertions)
			}
			for _, k := range []string{"dnsDuration", "connDuration", "reqDuration", "serverProcessDuration", "resDuration"} {
				if d, ok := res.Custom[k].(time.Duration); !ok || d < 0 {
					t.Errorf("%s = %v, want a non-negative duration", k, res.Custom[k])
				}
			}
			if res.Duration <= 0 || res.Duration > 15*time.Second {
				t.Errorf("total duration %v", res.Duration)
			}
		})
	}
}

// returnsWithin runs f and fails the test, instead of hanging, when f does not return in time.
func returnsWithin(t *testing.T, limit time.Duration, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); f() }()
	select {
	case <-done:
	case <-time.After(limit):
		t.Fatal("a duration call blocked waiting for a hook that never fired")
	}
}

// A request whose body write fails (the WroteRequest hook reports an error, so no write time is
// known) still yields its phases at once: no call waits for an event that will not come.
func TestDurationsWithoutWroteRequestDoNotBlock(t *testing.T) {
	d := &duration{}
	trace := newTrace(d, nil, nil)
	var req, srv, total time.Duration
	returnsWithin(t, 5*time.Second, func() {
		trace.GotConn(httptrace.GotConnInfo{})
		time.Sleep(5 * time.Millisecond)
		trace.WroteRequest(httptrace.WroteRequestInfo{Err: io.ErrClosedPipe})
		trace.GotFirstResponseByte()
		d.setResDur()
		srv, req, total = d.getServerProcessDur(), d.getReqDur(), d.totalDuration()
		d.getResDur()
	})
	if srv != 0 {
		t.Errorf("server processing without a written request should be 0, got %v", srv)
	}
	// the request was being written until the answer came, so the time is not lost
	if req < 5*time.Millisecond || total < 5*time.Millisecond {
		t.Errorf("request write %v, total %v: the time to the answer should count as writing", req, total)
	}
}

// A request that got no response at all (a timeout or a refused connection) yields zero phases
// past the connection, at once.
func TestDurationsWithoutResponseDoNotBlock(t *testing.T) {
	d := &duration{}
	trace := newTrace(d, nil, nil)
	var srv, res time.Duration
	returnsWithin(t, 5*time.Second, func() {
		trace.GotConn(httptrace.GotConnInfo{})
		trace.WroteRequest(httptrace.WroteRequestInfo{})
		srv, res = d.getServerProcessDur(), d.getResDur()
		d.totalDuration()
	})
	if srv != 0 || res != 0 {
		t.Errorf("server %v, res %v, want 0 without a response", srv, res)
	}
}

// In the normal order the phases add up: write, then server processing until the first byte,
// then reading.
func TestDurationsNormalOrder(t *testing.T) {
	d := &duration{}
	trace := newTrace(d, nil, nil)
	var req, srv, res, total time.Duration
	returnsWithin(t, 5*time.Second, func() {
		trace.GotConn(httptrace.GotConnInfo{})
		time.Sleep(5 * time.Millisecond)
		trace.WroteRequest(httptrace.WroteRequestInfo{})
		time.Sleep(20 * time.Millisecond)
		trace.GotFirstResponseByte()
		time.Sleep(5 * time.Millisecond)
		d.setResDur()
		req, srv, res, total = d.getReqDur(), d.getServerProcessDur(), d.getResDur(), d.totalDuration()
	})
	if req < 5*time.Millisecond || srv < 20*time.Millisecond || res < 5*time.Millisecond {
		t.Errorf("req %v, server %v, res %v", req, srv, res)
	}
	if total != req+srv+res {
		t.Errorf("total %v is not the sum of its phases %v", total, req+srv+res)
	}
}
