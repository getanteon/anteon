package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// busyFileService answers every second upload with "503 busy" after reading only a few bytes of
// the body, and keeps that connection open without reading, as a busy file service does; the
// other uploads are read whole and answered 200.
func busyFileService(t *testing.T, answered *int64) string {
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
	var n int64
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
				for {
					length := 0
					for { // the request head
						line, err := r.ReadString('\n')
						if err != nil {
							return
						}
						if line == "\r\n" {
							break
						}
						if k, v, ok := strings.Cut(line, ":"); ok && strings.EqualFold(k, "Content-Length") {
							length, _ = strconv.Atoi(strings.TrimSpace(v))
						}
					}
					if atomic.AddInt64(&n, 1)%2 == 0 {
						io.ReadFull(r, make([]byte, 16))
						io.WriteString(conn, "HTTP/1.1 503 Service Unavailable\r\nContent-Length: 4\r\nConnection: close\r\n\r\nbusy")
						atomic.AddInt64(answered, 1)
						select {
						case <-done:
						case <-time.After(30 * time.Second):
						}
						return
					}
					if _, err := io.CopyN(io.Discard, r, int64(length)); err != nil {
						return
					}
					io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok")
					atomic.AddInt64(answered, 1)
				}
			}()
		}
	}()
	return "http://" + ln.Addr().String() + "/upload"
}

// A closed-loop run uploading a large payload_file to a server that sometimes answers 503 before
// reading the body ends its stages, and counts every answer: no iteration stays in flight.
func TestClosedLoopEndsWhenServerAnswersBeforeTheUploadIsWritten(t *testing.T) {
	var answered int64
	url := busyFileService(t, &answered)
	payload := filepath.Join(t.TempDir(), "upload.bin")
	if err := os.WriteFile(payload, make([]byte, 8<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := writeConfig(t, fmt.Sprintf(`{"steps":[{"id":1,"url":"%s","method":"POST","payload_file":"%s",
		"headers":{"Content-Type":"application/octet-stream"},"timeout":5,
		"assertion":["equals(status_code,200)"]}]}`, url, payload))

	type outcome struct {
		code        int
		out, errOut string
	}
	got := make(chan outcome, 1)
	go func() {
		code, out, errOut := run(t, "run", "--config", cfg, "--concurrency", "1,2", "--per-level", "4",
			"--warmup-iterations", "0", "--json")
		got <- outcome{code, out, errOut}
	}()
	var o outcome
	select {
	case o = <-got:
	case <-time.After(60 * time.Second):
		t.Fatalf("the run hung with an iteration in flight; the server answered %d of 8", atomic.LoadInt64(&answered))
	}

	var res struct {
		Success       int64 `json:"success_count"`
		ServerFail    int64 `json:"server_fail_count"`
		AssertionFail int64 `json:"assertion_fail_count"`
		Steps         map[string]struct {
			StatusCodeDist map[string]int `json:"status_code_dist"`
		} `json:"steps"`
		Stages []struct {
			Name       string `json:"name"`
			Iterations int64  `json:"iterations"`
		} `json:"stages"`
	}
	if err := json.Unmarshal([]byte(lastJSONLine(o.out)), &res); err != nil {
		t.Fatalf("exit %d, the report line: %v %q %s", o.code, err, o.out, o.errOut)
	}
	if total := res.Success + res.ServerFail + res.AssertionFail; total != 8 {
		t.Errorf("8 iterations should be counted, got %d (%+v)", total, res)
	}
	dist := res.Steps["1"].StatusCodeDist
	if dist["200"] == 0 || dist["503"] == 0 || dist["200"]+dist["503"] != 8 {
		t.Errorf("status codes %v, want the 200s and the early 503s, 8 in all", dist)
	}
	if res.AssertionFail != int64(dist["503"]) {
		t.Errorf("each early 503 should fail its assertion: %d assertion fails, %d 503s", res.AssertionFail, dist["503"])
	}
	if len(res.Stages) != 2 || res.Stages[0].Iterations != 4 || res.Stages[1].Iterations != 4 {
		t.Errorf("stages %+v", res.Stages)
	}
}
