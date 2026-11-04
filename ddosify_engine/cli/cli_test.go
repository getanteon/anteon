package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/types"
)

func run(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Main(args, BuildInfo{Version: "v2.0.0-test", Commit: "abc", Date: "today"},
		Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

func writeConfig(t *testing.T, cfg string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

type counter struct {
	n       int64
	headers atomic.Value
}

func countingServer(t *testing.T, c *counter) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&c.n, 1)
		c.headers.Store(r.Header.Clone())
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"text-delta\",\"delta\":\"x\"}\n\ndata: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestNormalizeArgs(t *testing.T) {
	cases := map[string][]string{
		"-config x.json -debug":     {"run", "--config", "x.json", "--debug"},
		"-t http://a -h X-A:1 -n 5": {"run", "-t", "http://a", "-H", "X-A:1", "-n", "5"},
		"-cert_path=c.pem":          {"run", "--cert-path=c.pem"},
		"-version":                  {"version"},
		"run -t http://a":           {"run", "-t", "http://a"},
		"--help":                    {"--help"},
	}
	for in, want := range cases {
		if got := normalizeArgs(strings.Fields(in)); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: expected %v, got %v", in, want, got)
		}
	}
}

func TestVersion(t *testing.T) {
	code, out, _ := run(t, "version")
	if code != ExitOK || !strings.Contains(out, "v2.0.0-test") || !strings.Contains(out, "abc") {
		t.Errorf("version: %d %q", code, out)
	}
	code, out, _ = run(t, "version", "--json")
	var v map[string]string
	if code != ExitOK || json.Unmarshal([]byte(out), &v) != nil || v["version"] != "v2.0.0-test" || v["go"] == "" {
		t.Errorf("version --json: %d %q", code, out)
	}
	if code, out, _ = run(t, "-version"); code != ExitOK || !strings.Contains(out, "v2.0.0-test") {
		t.Errorf("-version keeps working: %d %q", code, out)
	}
}

func TestCompletionAndDocs(t *testing.T) {
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		if code, out, _ := run(t, "completion", sh); code != ExitOK || !strings.Contains(out, "ddosify") {
			t.Errorf("completion %s: %d", sh, code)
		}
	}
	dir := t.TempDir()
	if code, _, _ := run(t, "docs", "man", "--dir", dir); code != ExitOK {
		t.Fatalf("docs man: %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "ddosify-run.1")); err != nil {
		t.Error(err)
	}
	if code, _, _ := run(t, "docs", "markdown", "--dir", dir); code != ExitOK {
		t.Fatalf("docs markdown: %d", code)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "ddosify_run.md")); err != nil || !strings.Contains(string(b), "DDOSIFY_") {
		t.Errorf("markdown reference: %v", err)
	}
}

func TestUsageErrorsExit2(t *testing.T) {
	cfg := writeConfig(t, `{"steps":[{"id":1,"url":"http://127.0.0.1:1"}]}`)
	cases := [][]string{
		{"run"},                                    // no target, no config
		{"run", "not a url"},                       // invalid target
		{"run", "-t", "http://a", "http://b"},      // two targets
		{"run", "--no-such-flag"},                  // cobra's own error
		{"run", "--config", "/does/not/exist"},     // missing file
		{"run", "--config", cfg, "-t", "http://a"}, // a request flag with a config
		{"run", "http://a", "-H", "no colon"},
		{"run", "http://a", "--load-type", "spiral"},
		{"run", "http://a", "--ui", "fancy"},
		{"run", "http://a", "--html", "r.html"}, // no html observer in this build
		{"run", "http://a", "--json", "-o", "stdout"},
		{"nonsense"},
	}
	for _, args := range cases {
		code, out, errOut := run(t, args...)
		if code != ExitUsage || errOut == "" || out != "" {
			t.Errorf("%v: expected exit 2 with an error on stderr only, got %d stdout %q stderr %q", args, code, out, errOut)
		}
	}
}

func TestRunFromFlagsJSON(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	code, out, errOut := run(t, "run", srv.URL, "-n", "3", "-d", "1", "-H", "X-Test: yes", "--json")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var res map[string]interface{}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("stdout must be the JSON report only: %v %q", err, out)
	}
	if res["success_count"] != float64(3) || atomic.LoadInt64(&c.n) != 3 {
		t.Errorf("unexpected report %v, requests %d", res["success_count"], c.n)
	}
	if h, _ := c.headers.Load().(http.Header); h.Get("X-Test") != "yes" {
		t.Errorf("header not sent: %v", h)
	}
	if errOut != "" {
		t.Errorf("the JSON output writes nothing else: %q", errOut)
	}
}

func TestPrecedenceFlagsOverEnvOverConfig(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	cfg := writeConfig(t, fmt.Sprintf(`{"iteration_count": 2, "duration": 1, "output": "stdout",
		"steps":[{"id":1,"url":"%s","headers":{"X-From":"config"}}]}`, srv.URL))

	t.Setenv("DDOSIFY_ITERATIONS", "4")
	t.Setenv("DDOSIFY_OUTPUT", "stdout-json")
	code, out, errOut := run(t, "run", "--config", cfg)
	if code != ExitOK || atomic.LoadInt64(&c.n) != 4 || !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("env over config: exit %d, %d requests, %q %s", code, c.n, out, errOut)
	}

	atomic.StoreInt64(&c.n, 0)
	code, _, errOut = run(t, "run", "--config", cfg, "-n", "1", "-H", "X-From: flag")
	if code != ExitOK || atomic.LoadInt64(&c.n) != 1 {
		t.Fatalf("flag over env: exit %d, %d requests %s", code, c.n, errOut)
	}
	if h, _ := c.headers.Load().(http.Header); h.Get("X-From") != "flag" {
		t.Errorf("a header flag overrides the config's: %v", h)
	}

	t.Setenv("DDOSIFY_ITERATIONS", "many")
	if code, _, _ := run(t, "run", "--config", cfg); code != ExitUsage {
		t.Errorf("a bad environment variable is a usage error, got %d", code)
	}
}

func TestLegacyConfigFlag(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	cfg := writeConfig(t, fmt.Sprintf(`{"iteration_count": 1, "duration": 1, "steps":[{"id":1,"url":"%s"}]}`, srv.URL))
	code, out, errOut := run(t, "-config", cfg)
	if code != ExitOK || !strings.Contains(out, "RESULT") {
		t.Errorf("ddosify -config keeps working: %d %q %s", code, out, errOut)
	}
	if !strings.Contains(errOut, "Initializing") || strings.Contains(out, "Initializing") {
		t.Errorf("banners go to stderr, the report to stdout: %q %q", out, errOut)
	}
}

func TestTestFailureExit3(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	criteria := writeConfig(t, fmt.Sprintf(`{"iteration_count": 2, "duration": 1,
		"success_criterias": [{"rule": "fail_count_perc > 0.5", "abort": false}],
		"steps":[{"id":1,"url":"%s"}]}`, srv.URL))
	if code, _, errOut := run(t, "run", "--config", criteria); code != ExitTestFailed {
		t.Errorf("a failed success criterion exits 3, got %d %s", code, errOut)
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		time.Sleep(20 * time.Millisecond)
		fmt.Fprint(w, "data: {\"type\":\"text-delta\"}\n\ndata: [DONE]\n\n")
	}))
	defer slow.Close()
	slo := writeConfig(t, fmt.Sprintf(`{"iteration_count": 2, "duration": 1,
		"slo": {"first_token_ms": 1, "goodput_perc": 100},
		"steps":[{"id":1,"url":"%s","stream":{"type":"sse"}}]}`, slow.URL))
	code, out, errOut := run(t, "run", "--config", slo, "--json")
	if code != ExitTestFailed {
		t.Errorf("goodput under the SLO exits 3, got %d %s", code, errOut)
	}
	var res struct {
		Failed []struct{ Rule string } `json:"failed_criterias"`
	}
	if json.Unmarshal([]byte(out), &res) != nil || len(res.Failed) != 1 || res.Failed[0].Rule != "goodput_perc >= 100" {
		t.Errorf("the failed rule is in the report: %s", out)
	}
}

func TestInsecureWarns(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	code, _, errOut := run(t, "run", srv.URL, "-n", "1", "-d", "1", "--insecure")
	if code != ExitOK || !strings.Contains(errOut, "not verified") {
		t.Errorf("--insecure warns on stderr: %d %q", code, errOut)
	}
}

type fakeTUI struct {
	updates int
	final   *live.Final
	opts    live.Options
}

func (f *fakeTUI) Start(p live.Plan)      {}
func (f *fakeTUI) Update(s live.Snapshot) { f.updates++ }
func (f *fakeTUI) Finish(fi live.Final)   { f.final = &fi }
func (f *fakeTUI) OwnsTerminal() bool     { return true }

func TestObserversAndHeldReport(t *testing.T) {
	tui := &fakeTUI{}
	live.Register("test-tui", func(o live.Options) (live.Observer, error) { tui.opts = o; return tui, nil })
	c := &counter{}
	srv := countingServer(t, c)
	code, out, errOut := run(t, "run", srv.URL, "-n", "2", "-d", "1", "--observer", "test-tui=arg1", "--no-color")
	if code != ExitOK {
		t.Fatalf("exit %d %s", code, errOut)
	}
	if tui.final == nil || tui.updates == 0 || tui.opts.Arg != "arg1" || tui.opts.Color {
		t.Errorf("observer not driven: %+v", tui)
	}
	if !strings.Contains(out, "RESULT") {
		t.Errorf("the plain report is printed after the TUI closes: %q", out)
	}
}

func TestHammerFromFlags(t *testing.T) {
	o := &runOptions{target: "https://example.com", iterations: 7, duration: 3, loadType: "Waved", method: "post",
		body: "{}", auth: "u:p", headers: []string{"A: 1", "B:2"}, timeout: 9, proxy: "http://proxy:8080",
		output: "stdout", engineMode: types.EngineModeDistinctUser, insecure: true}
	h, err := hammerFromFlags(o)
	if err != nil {
		t.Fatal(err)
	}
	st := h.Scenario.Steps[0]
	if h.IterationCount != 7 || h.TestDuration != 3 || h.LoadType != "waved" || st.Method != "POST" || st.Payload != "{}" ||
		st.Auth.Username != "u" || st.Headers["A"] != "1" || st.Headers["B"] != "2" || st.Timeout != 9 ||
		h.Proxy.Addr.Host != "proxy:8080" || !h.Insecure || h.EngineMode != types.EngineModeDistinctUser {
		t.Errorf("unexpected hammer %+v", h)
	}
	if err := h.Validate(); err != nil {
		t.Error(err)
	}
	if _, err := hammerFromFlags(&runOptions{target: "https://a", auth: "nocolon"}); err == nil {
		t.Error("auth needs a colon")
	}
	if _, err := hammerFromFlags(&runOptions{target: "https://a", certPath: "c.pem"}); err == nil {
		t.Error("a cert without a key is an error")
	}
}

func TestSingleDashLongFlagHint(t *testing.T) {
	cases := map[string][]string{
		"--config":    {"run", "-config", "demo.json"},
		"--html":      {"run", "-html", "r.html", "https://example.com"},
		"--cert-path": {"run", "-cert_path", "c.pem", "https://example.com"},
	}
	for want, args := range cases {
		code, _, stderr := run(t, args...)
		if code != ExitUsage {
			t.Errorf("%v: exit %d", args, code)
		}
		if !strings.Contains(stderr, "Did you mean "+want+"?") {
			t.Errorf("%v: no hint for %s in %q", args, want, stderr)
		}
	}
	if _, _, stderr := run(t, "run", "-b", "-config", "-zz", "https://example.com"); strings.Contains(stderr, "Did you mean") {
		t.Errorf("a hint for a flag that does not exist: %q", stderr)
	}
}
