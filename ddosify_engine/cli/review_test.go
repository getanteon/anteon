package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/live"
)

// stopAfterSchedule calls Stop once the schedule of a 1 s test has ended.
type stopAfterSchedule struct {
	o       live.Options
	stopped int32
}

func (s *stopAfterSchedule) Start(live.Plan) {}
func (s *stopAfterSchedule) Update(sn live.Snapshot) {
	if sn.Elapsed > 1300*time.Millisecond && atomic.CompareAndSwapInt32(&s.stopped, 0, 1) {
		s.o.Stop()
	}
}
func (s *stopAfterSchedule) Finish(live.Final) {}

func slowServer(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		select {
		case <-time.After(10 * time.Second):
		case <-r.Context().Done():
			return
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestInterruptAfterTheScheduleExits130(t *testing.T) {
	live.Register("test-late-stop", func(o live.Options) (live.Observer, error) { return &stopAfterSchedule{o: o}, nil })
	defer live.Register("test-late-stop", nil)
	srv := slowServer(t)
	begin := time.Now()
	code, out, errOut := run(t, "run", srv.URL, "-n", "1", "-d", "1", "-T", "20", "--drain", "200ms",
		"--observer", "test-late-stop", "--ui", "plain")
	if code != ExitInterrupted {
		t.Errorf("an interrupt after the schedule exits 130, got %d: %s", code, errOut)
	}
	if !strings.Contains(out, "Interrupted:") || strings.Contains(out, "Test Status : Success") {
		t.Errorf("the in-flight iteration is reported as interrupted: %s", out)
	}
	if time.Since(begin) > 4*time.Second {
		t.Errorf("--drain cuts the 10 s stream, the run took %v", time.Since(begin))
	}
}

func TestAllRefusedExits3(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(429) }))
	defer srv.Close()
	code, out, _ := run(t, "run", srv.URL, "-n", "3", "-d", "1")
	if code != ExitTestFailed || strings.Contains(out, "Test Status : Success") {
		t.Errorf("a run where every request is refused fails with 3, got %d: %s", code, out)
	}
}

func TestUnwritableHTMLFailsBeforeTheRun(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	live.Register("html", func(o live.Options) (live.Observer, error) {
		return nil, errors.New("--html: open " + o.Arg + ": no such file or directory")
	})
	defer live.Register("html", nil)
	code, _, errOut := run(t, "run", srv.URL, "-n", "1", "-d", "1", "--html", filepath.Join(t.TempDir(), "x", "r.html"))
	if code != ExitUsage || atomic.LoadInt64(&c.n) != 0 || !strings.Contains(errOut, "--html") {
		t.Errorf("a bad output path fails before any request: %d %d %q", code, c.n, errOut)
	}
}

type failingWriter struct{ stopper }

func (f *failingWriter) Err() error { return errors.New("html report r.html: disk full") }

func TestOutputWriteFailureExits1(t *testing.T) {
	live.Register("test-failing", func(o live.Options) (live.Observer, error) { return &failingWriter{}, nil })
	defer live.Register("test-failing", nil)
	c := &counter{}
	srv := countingServer(t, c)
	code, out, errOut := run(t, "run", srv.URL, "-n", "1", "-d", "1", "--observer", "test-failing", "--ui", "plain")
	if code != ExitError || !strings.Contains(out, "RESULT") || !strings.Contains(errOut, "disk full") {
		t.Errorf("a failed write exits 1 and the report still prints: %d %q %q", code, out, errOut)
	}
}

func TestRemovedV1Flags(t *testing.T) {
	code, _, errOut := run(t, "-t", "http://a", "-p", "HTTPS")
	if code != ExitUsage || !strings.Contains(errOut, "scheme goes in the URL") || !strings.Contains(errOut, "ddosify run --help") {
		t.Errorf("-p explains what replaced it: %d %q", code, errOut)
	}
	code, _, errOut = run(t, "run", "http://a", "--no-such")
	if code != ExitUsage || !strings.Contains(errOut, "ddosify run --help") {
		t.Errorf("the hint names the failing command: %q", errOut)
	}
}

func TestPresetErrorLabel(t *testing.T) {
	_, _, errOut := run(t, "run", "--preset", "openai", "--model", "m")
	if strings.Contains(errOut, "Error: : ") || !strings.Contains(errOut, "--preset") {
		t.Errorf("an error without a config file names the preset: %q", errOut)
	}
}
