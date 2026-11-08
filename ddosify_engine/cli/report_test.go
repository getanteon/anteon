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
	var gotIn, gotTitle string
	old := elive.LogReport
	t.Cleanup(func() { elive.LogReport = old })
	elive.LogReport = func(in io.Reader, o elive.ReportOptions, open func() (io.Writer, error)) error {
		b, _ := io.ReadAll(in)
		gotIn, gotTitle = string(b), o.Title
		if gotIn == "bad" {
			return errors.New(`no "report" line`)
		}
		w, err := open()
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, "<html>")
		return err
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "run.ndjson")
	if err := os.WriteFile(log, []byte("lines"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "r.html")
	code, _, stderr := run(t, "report", log, "-o", out, "--title", "Night run")
	if b, _ := os.ReadFile(out); code != ExitOK || gotIn != "lines" || gotTitle != "Night run" || string(b) != "<html>" || !strings.Contains(stderr, "HTML report written to "+out) {
		t.Errorf("exit %d, in %q, title %q, file %q, stderr %q", code, gotIn, gotTitle, b, stderr)
	}
	code, stdout, stderr := run(t, "report", log, "-o", "-")
	if code != ExitOK || stdout != "<html>" || stderr != "" {
		t.Errorf("-o -: exit %d, stdout %q, stderr %q", code, stdout, stderr)
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
	if b, _ := os.ReadFile(out); string(b) != "<html>" {
		t.Errorf("a log with no report replaced the earlier report: %q", b)
	}
	elive.LogReport = nil
	if code, _, stderr = run(t, "report", log); code != ExitError || !strings.Contains(stderr, "no HTML report") {
		t.Errorf("a build without the HTML report: exit %d, stderr %q", code, stderr)
	}
}

func TestReportCompareAndOptions(t *testing.T) {
	old, oldC := elive.LogReport, elive.CompareLogs
	t.Cleanup(func() { elive.LogReport, elive.CompareLogs = old, oldC })
	var got elive.ReportOptions
	var labels []string
	elive.CompareLogs = func(logs []elive.LabeledLog, o elive.ReportOptions, open func() (io.Writer, error)) error {
		got, labels = o, nil
		for _, l := range logs {
			labels = append(labels, l.Label)
		}
		w, err := open()
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, "<cmp>")
		return err
	}
	elive.LogReport = func(in io.Reader, o elive.ReportOptions, open func() (io.Writer, error)) error {
		got = o
		return nil
	}
	dir := t.TempDir()
	a, b := filepath.Join(dir, "qwen-a.ndjson"), filepath.Join(dir, "qwen-b.ndjson")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	code, stdout, stderr := run(t, "report", a, b, "-o", "-", "--gpus", "2", "--tier", "fast:500:30")
	if code != ExitOK || stdout != "<cmp>" || strings.Join(labels, ",") != "qwen-a,qwen-b" || got.GPUs != 2 ||
		len(got.Tiers) != 1 || got.Tiers[0].Name != "fast" || got.Tiers[0].FirstTokenMs != 500 || got.Tiers[0].TPOTMs != 30 {
		t.Errorf("exit %d, stdout %q, stderr %q, labels %v, options %+v", code, stdout, stderr, labels, got)
	}
	if run(t, "report", a, b, "-o", "-", "--label", "A", "--label", "B"); strings.Join(labels, ",") != "A,B" {
		t.Errorf("labels %v", labels)
	}
	for _, bad := range [][]string{
		{"report", a, b, "--label", "only one"},
		{"report", a, "--tier", "x:0:0"},
		{"report", a, "--tier", "x:1"},
		{"report", a, "--gpus", "0"},
	} {
		if code, _, _ := run(t, bad...); code != ExitUsage {
			t.Errorf("%v: exit %d, want a usage error", bad, code)
		}
	}
}

func TestReportResources(t *testing.T) {
	old, oldC := elive.LogReport, elive.CompareLogs
	t.Cleanup(func() { elive.LogReport, elive.CompareLogs = old, oldC })
	var got elive.ReportOptions
	var read []string
	keep := func(o elive.ReportOptions) {
		got, read = o, nil
		for _, r := range o.Resources {
			b, _ := io.ReadAll(r.In)
			read = append(read, r.Run+":"+string(b))
		}
	}
	elive.LogReport = func(in io.Reader, o elive.ReportOptions, open func() (io.Writer, error)) error {
		keep(o)
		return nil
	}
	elive.CompareLogs = func(logs []elive.LabeledLog, o elive.ReportOptions, open func() (io.Writer, error)) error {
		keep(o)
		return nil
	}
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	a, b := write("a.ndjson", "x"), write("b.ndjson", "x")
	pods, gpus := write("pods.csv", "P"), write("gpus.csv", "G")

	if code, _, stderr := run(t, "report", a, "-o", "-", "--resources", pods, "--resources", gpus, "--resources-per", "masking"); code != ExitOK ||
		strings.Join(read, ",") != ":P,:G" || strings.Join(got.ResourcesPer, ",") != "masking" || got.Stderr == nil {
		t.Errorf("one run: exit %d, stderr %q, read %v, options %+v", code, stderr, read, got)
	}
	// in a comparison each file names its run; a label may hold "="
	if code, _, stderr := run(t, "report", a, b, "-o", "-", "--label", "cpu", "--label", "gpu=x",
		"--resources", "gpu=x="+gpus, "--resources", "cpu="+pods); code != ExitOK || strings.Join(read, ",") != "gpu=x:G,cpu:P" {
		t.Errorf("comparison: exit %d, stderr %q, read %v", code, stderr, read)
	}
	// the default labels are the file names
	if code, _, _ := run(t, "report", a, b, "-o", "-", "--resources", "b="+gpus); code != ExitOK || strings.Join(read, ",") != "b:G" {
		t.Errorf("default labels: exit %d, read %v", code, read)
	}
	for _, bad := range [][]string{
		{"report", a, "--resources", filepath.Join(dir, "missing.csv")},
		{"report", a, b, "--resources", pods},              // no label in a comparison
		{"report", a, b, "--resources", "nope=" + pods},    // not a run's label
		{"report", a, "--resources-per", "masking"},        // nothing to filter
		{"report", a, b, "--resources", "a=" + dir + "/x"}, // the file is missing
	} {
		if code, _, _ := run(t, bad...); code != ExitUsage {
			t.Errorf("%v: exit %d, want a usage error", bad, code)
		}
	}
}
