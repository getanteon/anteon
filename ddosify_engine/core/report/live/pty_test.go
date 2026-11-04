//go:build darwin || linux

package live

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// On a real terminal that allows hard tabs, the TUI must still write no tab: a screen capture
// (tmux capture-pane) keeps them and the columns then fall apart.
func TestTUIOnPTYWritesNoTabs(t *testing.T) {
	master, slave, err := openPTY()
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	defer master.Close()
	defer slave.Close()
	if err := unix.IoctlSetWinsize(int(slave.Fd()), ioctlSetWinsize, &unix.Winsize{Row: 34, Col: 110}); err != nil {
		t.Fatal(err)
	}
	before, err := unix.IoctlGetTermios(int(slave.Fd()), ioctlGetTermios)
	if err != nil {
		t.Fatal(err)
	}
	before.Oflag &^= unix.TABDLY // TAB0: the terminal allows hard tabs
	if err := unix.IoctlSetTermios(int(slave.Fd()), ioctlSetTermios, before); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := master.Read(buf)
			mu.Lock()
			out.Write(buf[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	ch := make(chan Snapshot)
	go func() {
		defer close(ch)
		for i := 0; i < 30; i++ {
			s := synthetic(i)
			s.All = &Scope{TTFT: s.TTFT, ITL: s.ITL, E2E: s.E2E, Goodput: s.Goodput, Outcomes: s.Outcomes}
			ch <- s
			time.Sleep(20 * time.Millisecond)
		}
	}()
	if _, err := RunTUI(context.Background(), ch, Options{Out: slave, In: slave, Env: []string{"TERM=xterm-256color"}}); err != nil {
		t.Fatal(err)
	}
	after, err := unix.IoctlGetTermios(int(slave.Fd()), ioctlGetTermios)
	if err != nil {
		t.Fatal(err)
	}
	if after.Oflag&unix.TABDLY != before.Oflag&unix.TABDLY {
		t.Errorf("tab setting not restored: %#x, was %#x", after.Oflag&unix.TABDLY, before.Oflag&unix.TABDLY)
	}
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	got := out.Bytes()
	mu.Unlock()
	if !bytes.Contains(got, []byte("TTFT")) {
		t.Fatalf("nothing rendered: %q", got)
	}
	if i := bytes.IndexByte(got, '\t'); i >= 0 {
		t.Errorf("tab written at %d: %q", i, got[max(0, i-60):min(len(got), i+20)])
	}
}
