package requester

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

// step is one write of the test server: wait, then write and flush.
type sseStep struct {
	wait time.Duration
	data string
}

func sseServer(t *testing.T, steps []sseStep) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Error("response writer can not flush")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		f.Flush()
		for _, s := range steps {
			select {
			case <-time.After(s.wait):
			case <-r.Context().Done():
				return
			}
			fmt.Fprint(w, s.data)
			f.Flush()
		}
	}))
}

func sendStep(t *testing.T, step types.ScenarioStep) *types.ScenarioStepResult {
	t.Helper()
	h := &HttpRequester{}
	if err := h.Init(context.Background(), step, nil, false, nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer h.Done()
	return h.Send(nil, map[string]interface{}{})
}

// within checks got is not earlier than want, which a timer can not be, and at most slack later.
func within(t *testing.T, name string, got, want, slack time.Duration) {
	t.Helper()
	if got < want-5*time.Millisecond || got > want+slack {
		t.Errorf("%s: expected %v (+%v), measured %v", name, want, slack, got)
	}
}

func TestStreamAgainstRealServer(t *testing.T) {
	ms := time.Millisecond
	srv := sseServer(t, []sseStep{
		{150 * ms, "data: {\"type\":\"start\",\"messageId\":\"m\"}\n\n"},
		{250 * ms, ": keepalive\n\n"},
		{50 * ms, "data: {\"type\":\"text-delta\",\"delta\":\"Mer\"}\n\n"},
		{100 * ms, "data: {\"type\":\"text-delta\",\"delta\":\"haba\"}\r\n\r\n"},
		{100 * ms, "data: {\"type\":\"finish\"}\n\ndata: [DONE]\n\n"},
	})
	defer srv.Close()

	res := sendStep(t, types.ScenarioStep{
		ID: 1, Method: http.MethodPost, URL: srv.URL, Timeout: 10,
		Stream: types.DefaultStreamConf(),
	})
	if res.Err.Type != "" {
		t.Fatalf("unexpected error %v", res.Err)
	}
	st := res.Stream
	if st == nil {
		t.Fatal("stream metrics missing")
	}
	t.Logf("ttfb %v first event %v first token %v total %v gaps max %v p50 %v p95 %v",
		st.TTFB, st.FirstEvent, st.FirstToken, st.Total, st.GapMax, st.GapP50, st.GapP95)
	slack := 150 * ms
	if st.TTFB > slack {
		t.Errorf("headers are flushed at once, TTFB measured %v", st.TTFB)
	}
	within(t, "first event", st.FirstEvent, 150*ms, slack)
	within(t, "first token", st.FirstToken, 450*ms, slack)
	within(t, "total", st.Total, 650*ms, slack)
	within(t, "max gap", st.GapMax, 300*ms, slack)
	within(t, "gap p50", st.GapP50, 100*ms, slack/2)
	within(t, "gap p95", st.GapP95, 300*ms, slack)
	if st.Events != 4 || st.Comments != 1 || st.End != types.StreamEndDone || !st.HasFirstToken {
		t.Errorf("unexpected stream %#v", st)
	}
	if res.RespBody != nil {
		t.Error("no assertion, capture or debug: the body should not be kept")
	}
	// the step lasts as long as its stream, not until its headers
	if res.Duration != st.Total {
		t.Errorf("a streamed step's duration is the stream's total %v, got %v", st.Total, res.Duration)
	}
	if read, _ := res.Custom["resDuration"].(time.Duration); read < 500*ms {
		t.Errorf("response read runs from the headers to the last event, got %v", read)
	}
}

func TestStreamDetectedWithoutConfig(t *testing.T) {
	srv := sseServer(t, []sseStep{{0, "data: {\"type\":\"text-delta\",\"delta\":\"çğış\"}\n\n"}})
	defer srv.Close()

	res := sendStep(t, types.ScenarioStep{ID: 1, Method: http.MethodGet, URL: srv.URL, Timeout: 5,
		Assertions: []string{`contains(body,"çğış")`}})
	if res.Stream == nil || res.Stream.End != types.StreamEndClosed || res.Stream.Events != 1 {
		t.Fatalf("a text/event-stream response should be read as a stream: %#v", res.Stream)
	}
	if len(res.FailedAssertions) > 0 {
		t.Errorf("body assertions should see the whole stream: %#v", res.FailedAssertions)
	}
}

func TestStreamErrorStatusIsReadWhole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.Header().Set("Retry-After", "41")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"code":"rate_limit_end_user"}`)
	}))
	defer srv.Close()

	res := sendStep(t, types.ScenarioStep{ID: 1, Method: http.MethodPost, URL: srv.URL, Timeout: 5,
		Stream: types.DefaultStreamConf(), Assertions: []string{"equals(status_code,429)"}})
	if res.Stream != nil || res.StatusCode != 429 || !strings.Contains(string(res.RespBody), "rate_limit") {
		t.Errorf("a 429 problem document is not a stream: %#v %d %s", res.Stream, res.StatusCode, res.RespBody)
	}
}

func TestStreamCutByServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		chunk := "data: {\"type\":\"start\"}\n\ndata: {\"type\":\"text-delta\",\"delta\":\"ha"
		fmt.Fprintf(buf, "HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nTransfer-Encoding: chunked\r\n\r\n%x\r\n%s\r\n", len(chunk), chunk)
		buf.Flush()
		time.Sleep(50 * time.Millisecond)
		conn.Close() // no terminating chunk
	}))
	defer srv.Close()

	res := sendStep(t, types.ScenarioStep{ID: 1, Method: http.MethodPost, URL: srv.URL, Timeout: 5,
		Stream: types.DefaultStreamConf()})
	if res.Stream == nil || res.Stream.End != types.StreamEndCut || res.Stream.Events != 1 || res.Stream.HasFirstToken {
		t.Fatalf("expected a cut stream with one event: %#v", res.Stream)
	}
	if res.Err.Type == "" {
		t.Error("a cut connection is a request error")
	}
}

func TestStreamTimeout(t *testing.T) {
	srv := sseServer(t, []sseStep{{0, "data: {\"type\":\"start\"}\n\n"}, {3 * time.Second, "data: [DONE]\n\n"}})
	defer srv.Close()

	res := sendStep(t, types.ScenarioStep{ID: 1, Method: http.MethodPost, URL: srv.URL, Timeout: 1,
		Stream: types.DefaultStreamConf()})
	if res.Stream == nil || res.Stream.End != types.StreamEndTimeout {
		t.Fatalf("expected a timed out stream: %#v", res.Stream)
	}
	if res.Err.Reason != types.ReasonReadTimeout {
		t.Errorf("expected %q, got %q", types.ReasonReadTimeout, res.Err.Reason)
	}
	within(t, "total", res.Stream.Total, time.Second, 500*time.Millisecond)
}
