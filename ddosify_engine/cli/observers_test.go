package cli

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/live"
)

type stopper struct {
	mu      sync.Mutex
	opts    live.Options
	plan    live.Plan
	updates int
	stopAt  int
	final   *live.Final
}

func (s *stopper) Start(p live.Plan) { s.mu.Lock(); s.plan = p; s.mu.Unlock() }
func (s *stopper) Update(live.Snapshot) {
	s.mu.Lock()
	s.updates++
	stop := s.stopAt > 0 && s.updates == s.stopAt
	s.mu.Unlock()
	if stop {
		s.opts.Stop()
		s.opts.Stop() // twice is fine
	}
}
func (s *stopper) Finish(f live.Final) { s.mu.Lock(); s.final = &f; s.mu.Unlock() }

func TestObserverStopsTheTest(t *testing.T) {
	obs := &stopper{stopAt: 2}
	live.Register("test-stopper", func(o live.Options) (live.Observer, error) { obs.opts = o; return obs, nil })
	c := &counter{}
	srv := countingServer(t, c)
	cfg := writeConfig(t, fmt.Sprintf(`{"iteration_count": 100, "duration": 10,
		"slo": {"first_token_ms": 5000, "goodput_perc": 90},
		"steps":[{"id":1,"url":"%s","stream":{"type":"sse"}}]}`, srv.URL))

	begin := time.Now()
	code, _, errOut := run(t, "run", "--config", cfg, "--observer", "test-stopper", "--ui", "plain")
	if code != ExitInterrupted || time.Since(begin) > 5*time.Second {
		t.Errorf("Stop ends the test early and exits 130: %d after %v %s", code, time.Since(begin), errOut)
	}
	obs.mu.Lock()
	defer obs.mu.Unlock()
	if obs.final == nil || obs.final.Status != "stopped" {
		t.Errorf("Finish comes after a stop: %+v", obs.final)
	}
	if obs.plan.SLO == nil || obs.plan.SLO.FirstTokenMs != 5000 || obs.plan.SLO.GoodputPerc != 90 {
		t.Errorf("the plan carries the SLO: %+v", obs.plan.SLO)
	}
}

type named struct {
	name  string
	mu    sync.Mutex
	built int
}

func (n *named) factory(o live.Options) (live.Observer, error) {
	n.mu.Lock()
	n.built++
	n.mu.Unlock()
	return &stopper{opts: o}, nil
}

func TestNdjsonInAPipeAndNoTUIOnDumbTerminals(t *testing.T) {
	nd := &named{name: "ndjson"}
	tui := &named{name: "tui"}
	live.Register("ndjson", nd.factory)
	live.Register("tui", tui.factory)
	defer func() { live.Register("ndjson", nil); live.Register("tui", nil) }()

	c := &counter{}
	srv := countingServer(t, c)
	if code, _, errOut := run(t, "run", srv.URL, "-n", "1", "-d", "1"); code != ExitOK || nd.built != 1 || tui.built != 0 {
		t.Errorf("stdout is not a terminal: ndjson, not the TUI (%d %d %d %s)", code, nd.built, tui.built, errOut)
	}
	if code, _, _ := run(t, "run", srv.URL, "-n", "1", "-d", "1", "--ui", "plain"); code != ExitOK || nd.built != 1 {
		t.Errorf("--ui plain builds no observer")
	}

	t.Setenv("TERM", "dumb")
	code, _, errOut := run(t, "run", srv.URL, "-n", "1", "-d", "1", "--ui", "tui")
	if code != ExitOK || tui.built != 0 || !strings.Contains(errOut, "TERM=dumb") {
		t.Errorf("TERM=dumb never gets the full-screen TUI: %d %d %q", code, tui.built, errOut)
	}
}

// owner stands in for the TUI: it owns the terminal from Start to Finish.
type owner struct {
	stopper
	out, err          *syncBuf
	atStart, atFinish [2]int
}

func (o *owner) OwnsTerminal() bool { return true }
func (o *owner) Start(p live.Plan) {
	o.atStart = [2]int{o.out.Len(), o.err.Len()}
	o.stopper.Start(p)
}
func (o *owner) Finish(f live.Final) {
	o.atFinish = [2]int{o.out.Len(), o.err.Len()}
	o.stopper.Finish(f)
}

type syncBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) Len() int                    { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Len() }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func TestNothingElseWritesWhileATUIOwnsTheTerminal(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	cfg := writeConfig(t, fmt.Sprintf(`{"iteration_count": 100, "duration": 10,
		"steps":[{"id":1,"url":"%s","stream":{"type":"sse"}}]}`, srv.URL))

	out, errb := &syncBuf{}, &syncBuf{}
	obs := &owner{stopper: stopper{stopAt: 2}, out: out, err: errb}
	live.Register("test-owner", func(o live.Options) (live.Observer, error) { obs.opts = o; return obs, nil })
	code := Main([]string{"run", "--config", cfg, "--observer", "test-owner", "--ui", "plain", "--insecure"},
		BuildInfo{Version: "test"}, Streams{In: strings.NewReader(""), Out: out, Err: errb})
	if code != ExitInterrupted {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if obs.atStart != obs.atFinish {
		t.Errorf("written while the TUI owned the terminal: stdout and stderr went from %v to %v", obs.atStart, obs.atFinish)
	}
	e := errb.String()
	if strings.Contains(e, "Initializing") {
		t.Errorf("the v1 banner printed in TUI mode: %q", e)
	}
	if !strings.Contains(e, "Stopping:") || !strings.Contains(e, "--insecure") {
		t.Errorf("held stderr was not printed after the TUI: %q", e)
	}
	if !strings.Contains(out.String(), "Test Status") {
		t.Errorf("the plain report did not follow the TUI: %q", out.String())
	}

	// Plain mode keeps its banner.
	code, _, plainErr := run(t, "run", srv.URL, "-n", "1", "-d", "1", "--ui", "plain")
	if code != ExitOK || !strings.Contains(plainErr, "Initializing") {
		t.Errorf("plain mode lost its banner: %d %q", code, plainErr)
	}
}
