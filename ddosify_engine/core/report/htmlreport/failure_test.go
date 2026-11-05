package htmlreport

import (
	"path/filepath"
	"strings"
	"testing"

	elive "go.ddosify.com/ddosify/v2/core/live"
)

func TestBadPathFailsBeforeTheRun(t *testing.T) {
	if _, err := newObserver(elive.Options{Arg: filepath.Join(t.TempDir(), "no", "such", "dir", "r.html")}); err == nil {
		t.Error("a path that can not be written fails when the observer is built")
	}
}

func TestWriteFailureIsReported(t *testing.T) {
	var errOut strings.Builder
	obs, err := newObserver(elive.Options{Arg: filepath.Join(t.TempDir(), "r.html"), Stderr: &errOut})
	if err != nil {
		t.Fatal(err)
	}
	o := obs.(*observer)
	o.f.Close() // the disk went away
	o.Finish(elive.Final{Status: "done"})
	if o.Err() == nil || !strings.Contains(errOut.String(), "html report") {
		t.Errorf("a failed write is kept for the CLI: %v %q", o.Err(), errOut.String())
	}
}
