package live

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	elive "go.ddosify.com/ddosify/v2/core/live"
)

func TestStatusLines(t *testing.T) {
	var b bytes.Buffer
	o, _ := newStatusObserver(elive.Options{Stderr: &b, Arg: "1ns"})
	plan := elive.Plan{Iterations: 3, Duration: 60 * time.Second,
		Stages: []elive.Stage{{Index: 0, Name: "smoke", Iterations: 3, End: 60 * time.Second}},
		Steps:  []elive.StepInfo{{ID: 1, Name: "Create chat"}, {ID: 2, Name: "Turn 1"}}}
	o.Start(plan)
	snap := elive.Snapshot{Elapsed: 10 * time.Second, Started: 2, InFlight: 1,
		Total: elive.Metrics{Outcomes: map[string]int64{"ok": 1, "network_error": 2}, TTFT: elive.Percentiles{Count: 1, P50: 2400}}}
	time.Sleep(time.Millisecond)
	o.Update(snap)
	o.Finish(elive.Final{Status: "done", Elapsed: 61 * time.Second, Passed: false, Last: snap})
	got := b.String()
	for _, want := range []string{
		`Started: 3 iterations over 1:00; stages: smoke 3 in 1:00; steps: 1 "Create chat", 2 "Turn 1"`,
		"0:10/1:00 · started 2 · in flight 1 · ok 1 · failed 2 (network_error 2) · first token p50 2.4 s",
		"Finished in 1:01: 2 iterations started, requests ok 1, failed 2 (network_error 2); the test failed",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lacks %q in:\n%s", want, got)
		}
	}
}

func TestNDJSONGoesToStdoutUnderJSON(t *testing.T) {
	var out, errOut bytes.Buffer
	for _, asJSON := range []bool{true, false} {
		out.Reset()
		errOut.Reset()
		o, _ := newNDJSONObserver(elive.Options{Stdout: &out, Stderr: &errOut, JSON: asJSON})
		o.Start(elive.Plan{})
		o.Finish(elive.Final{})
		w, other := &out, &errOut
		if !asJSON {
			w, other = &errOut, &out
		}
		var line map[string]any
		if err := json.Unmarshal(w.Bytes(), &line); err != nil || line["type"] != "progress" || other.Len() != 0 {
			t.Errorf("json %v: data %q, other %q", asJSON, w.String(), other.String())
		}
	}
}
