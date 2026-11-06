package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	elive "go.ddosify.com/ddosify/v2/core/live"
)

func TestReportCommand(t *testing.T) {
	var gotIn, gotTitle, gotPath string
	old := elive.LogReport
	t.Cleanup(func() { elive.LogReport = old })
	elive.LogReport = func(in io.Reader, title, path string) error {
		b, _ := io.ReadAll(in)
		gotIn, gotTitle, gotPath = string(b), title, path
		if gotIn == "bad" {
			return errors.New(`no "report" line`)
		}
		return nil
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "run.ndjson")
	if err := os.WriteFile(log, []byte("lines"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "r.html")
	code, _, stderr := run(t, "report", log, "-o", out, "--title", "Night run")
	if code != ExitOK || gotIn != "lines" || gotTitle != "Night run" || gotPath != out || !strings.Contains(stderr, "HTML report written to "+out) {
		t.Errorf("exit %d, in %q, title %q, path %q, stderr %q", code, gotIn, gotTitle, gotPath, stderr)
	}
	if code, _, stderr = run(t, "report", filepath.Join(dir, "missing.ndjson")); code != ExitUsage || !strings.Contains(stderr, "missing.ndjson") {
		t.Errorf("a missing log: exit %d, stderr %q", code, stderr)
	}
	if err := os.WriteFile(log, []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr = run(t, "report", log, "-o", out); code != ExitUsage || !strings.Contains(stderr, `no "report" line`) {
		t.Errorf("a log with no report: exit %d, stderr %q", code, stderr)
	}
	elive.LogReport = nil
	if code, _, stderr = run(t, "report", log); code != ExitError || !strings.Contains(stderr, "no HTML report") {
		t.Errorf("a build without the HTML report: exit %d, stderr %q", code, stderr)
	}
}
