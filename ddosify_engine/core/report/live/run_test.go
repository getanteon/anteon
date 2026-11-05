package live

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// syncBuffer is a bytes.Buffer safe for the renderer and the test to share.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// engine sends n snapshots from its own goroutine, stopping early when stop is called.
func engine(n int, every time.Duration) (<-chan Snapshot, func(), *atomic.Int32) {
	ch := make(chan Snapshot)
	stopped := make(chan struct{})
	var stops atomic.Int32
	var once sync.Once
	stop := func() {
		stops.Add(1)
		once.Do(func() { close(stopped) })
	}
	go func() {
		defer close(ch)
		for i := 0; i < n; i++ {
			select {
			case <-stopped:
				return
			case ch <- synthetic(i):
			}
			if every > 0 {
				time.Sleep(every)
			}
		}
	}()
	return ch, stop, &stops
}

func TestRunTUIUntilClose(t *testing.T) {
	ch, _, _ := engine(200, 0)
	out := &syncBuffer{}
	res, err := RunTUI(context.Background(), ch, Options{Out: out, In: strings.NewReader(""), Env: []string{"TERM=xterm-256color"}, FPS: 10, Width: 100, Height: 30})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != ModeTUI || !res.Seen || res.Forced || res.Last.Iterations != synthetic(199).Iterations {
		t.Fatalf("result %+v", res)
	}
	if !strings.Contains(out.String(), "ddosify") {
		t.Fatalf("nothing rendered: %q", out.String())
	}
}

// The race test: snapshots arrive from the engine goroutine while a stop comes from another.
func TestRunTUIStopRace(t *testing.T) {
	ch, stop, stops := engine(100000, time.Millisecond)
	go func() {
		time.Sleep(30 * time.Millisecond)
		stop()
	}()
	res, err := RunTUI(context.Background(), ch, Options{
		Out: &syncBuffer{}, In: strings.NewReader(""), Env: []string{"NO_COLOR=1"}, Stop: stop,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Seen || res.Last.Iterations >= synthetic(100000-1).Iterations {
		t.Fatalf("result %+v", res)
	}
	if stops.Load() != 1 {
		t.Fatalf("stop called %d times", stops.Load())
	}
}

func TestRunTUIContextCancelDrains(t *testing.T) {
	ch, _, _ := engine(1000, 2*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := RunTUI(ctx, ch, Options{Out: &syncBuffer{}, In: strings.NewReader("")})
	if err == nil {
		t.Fatal("no error after the context ended")
	}
	// The engine must be able to finish its sends: drain reads them.
	deadline := time.After(20 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("channel not drained")
		}
	}
}

func TestRunNDJSON(t *testing.T) {
	ch := make(chan Snapshot)
	var errOut syncBuffer
	var stdout bytes.Buffer
	done := make(chan struct{})
	var res Result
	go func() {
		defer close(done)
		var err error
		res, err = Run(context.Background(), ch, Options{Out: &stdout, Err: &errOut, JSONEvery: -1})
		if err != nil {
			t.Error(err)
		}
	}()
	for i := 0; i < 5; i++ {
		ch <- synthetic(i)
	}
	close(ch)
	<-done
	if res.Mode != ModeNDJSON {
		t.Fatalf("mode %v without a TTY", res.Mode)
	}
	if stdout.Len() != 0 {
		t.Fatal("progress went to stdout, which belongs to the final report")
	}
	sc := bufio.NewScanner(strings.NewReader(errOut.String()))
	n := 0
	for sc.Scan() {
		var line map[string]any
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			t.Fatalf("line %d is not JSON: %v", n, err)
		}
		for _, k := range []string{"type", "time", "elapsed_s", "remaining_s", "stage", "target_rps", "achieved_rps", "in_flight", "ttft_ms", "itl_ms", "e2e_ms", "goodput", "outcomes"} {
			if _, ok := line[k]; !ok {
				t.Errorf("line %d lacks %q", n, k)
			}
		}
		if got := line["outcomes"].(map[string]any)["rate_limited_429"]; got != float64(31) {
			t.Errorf("429 count %v", got)
		}
		n++
	}
	if n != 5 {
		t.Fatalf("%d lines, want 5", n)
	}
}

func TestRunNDJSONThrottlesAndKeepsLast(t *testing.T) {
	ch := make(chan Snapshot, 100)
	for i := 0; i < 100; i++ {
		ch <- synthetic(i)
	}
	close(ch)
	var errOut bytes.Buffer
	if _, err := RunNDJSON(context.Background(), ch, Options{Err: &errOut, JSONEvery: time.Hour}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(errOut.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("%d lines, want the first and the last", len(lines))
	}
	var last jsonLine
	if err := json.Unmarshal([]byte(lines[1]), &last); err != nil {
		t.Fatal(err)
	}
	if last.Iterations != synthetic(99).Iterations {
		t.Fatalf("last line is iteration %d", last.Iterations)
	}
}

func TestJSONFlagOnTerminalWritesNDJSON(t *testing.T) {
	ch := make(chan Snapshot)
	close(ch)
	res, err := Run(context.Background(), ch, Options{JSON: true, Err: &bytes.Buffer{}})
	if err != nil || res.Mode != ModeNDJSON {
		t.Fatalf("mode %v, err %v", res.Mode, err)
	}
}

func TestTeeRace(t *testing.T) {
	ch, _, _ := engine(500, 0)
	var mu sync.Mutex
	seen := 0
	out := Tee(ch, func(Snapshot) { mu.Lock(); seen++; mu.Unlock() })
	got := 0
	for range out {
		got++
	}
	mu.Lock()
	defer mu.Unlock()
	if got != 500 || seen != 500 {
		t.Fatalf("forwarded %d, observed %d", got, seen)
	}
}

func TestAdapt(t *testing.T) {
	type engineSnap struct{ inflight int64 }
	in := make(chan engineSnap)
	go func() {
		defer close(in)
		for i := 0; i < 50; i++ {
			in <- engineSnap{inflight: int64(i)}
		}
	}()
	var last Snapshot
	n := 0
	for s := range Adapt(in, func(e engineSnap) Snapshot { return Snapshot{InFlight: e.inflight} }) {
		last = s
		n++
	}
	if n != 50 || last.InFlight != 49 {
		t.Fatalf("%d snapshots, last %d", n, last.InFlight)
	}
}
