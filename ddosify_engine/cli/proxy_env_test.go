package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// forwardProxy is a plain HTTP forward proxy that answers itself: a client that uses it sends
// the absolute URI of the target, which is recorded.
type forwardProxy struct {
	mu   sync.Mutex
	uris []string
}

func (p *forwardProxy) seen() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.uris...)
}

func newForwardProxy(t *testing.T) (*forwardProxy, *httptest.Server) {
	p := &forwardProxy{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !r.URL.IsAbs() {
			http.Error(w, "not a proxy request", http.StatusBadRequest)
			return
		}
		p.mu.Lock()
		p.uris = append(p.uris, r.URL.String())
		p.mu.Unlock()
		fmt.Fprint(w, "via proxy")
	}))
	t.Cleanup(srv.Close)
	return p, srv
}

// The target's host never resolves (.invalid), so a successful request can only have gone
// through the proxy.
const proxiedTarget = "http://target.invalid/hello"

func checkProxied(t *testing.T, p *forwardProxy, code int, out, errOut string, n int) {
	t.Helper()
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var res map[string]interface{}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("stdout is not the JSON report: %v %q", err, out)
	}
	if res["success_count"] != float64(n) {
		t.Errorf("success_count %v, want %d: %s", res["success_count"], n, out)
	}
	uris := p.seen()
	if len(uris) != n {
		t.Fatalf("the proxy saw %d requests, want %d", len(uris), n)
	}
	for _, u := range uris {
		if u != proxiedTarget {
			t.Errorf("the proxy saw %q, want %q", u, proxiedTarget)
		}
	}
}

func TestRunProxyFromOsEnvInConfig(t *testing.T) {
	p, srv := newForwardProxy(t)
	t.Setenv("DDOSIFY_TEST_PROXY_URL", srv.URL)
	cfg := writeConfig(t, `{"iteration_count": 3, "duration": 1, "proxy": "{{$DDOSIFY_TEST_PROXY_URL}}",
		"steps": [{"id": 1, "url": "`+proxiedTarget+`"}]}`)
	code, out, errOut := run(t, "run", "--config", cfg, "--json")
	checkProxied(t, p, code, out, errOut, 3)
}

func TestRunProxyFromOsEnvInFlag(t *testing.T) {
	p, srv := newForwardProxy(t)
	t.Setenv("DDOSIFY_TEST_PROXY_URL", srv.URL)
	code, out, errOut := run(t, "run", proxiedTarget, "-n", "2", "-d", "1", "-P", "{{$DDOSIFY_TEST_PROXY_URL}}", "--json")
	checkProxied(t, p, code, out, errOut, 2)
}

func TestRunProxyFromOsEnvInDdosifyProxy(t *testing.T) {
	p, srv := newForwardProxy(t)
	t.Setenv("DDOSIFY_TEST_PROXY_URL", srv.URL)
	t.Setenv("DDOSIFY_PROXY", "{{$DDOSIFY_TEST_PROXY_URL}}")
	cfg := writeConfig(t, `{"iteration_count": 2, "duration": 1, "steps": [{"id": 1, "url": "`+proxiedTarget+`"}]}`)
	code, out, errOut := run(t, "run", "--config", cfg, "--json")
	checkProxied(t, p, code, out, errOut, 2)
}

func TestRunProxyMissingOsEnvStopsBeforeSending(t *testing.T) {
	p, srv := newForwardProxy(t)
	t.Setenv("DDOSIFY_TEST_PROXY_URL", srv.URL)
	cfg := writeConfig(t, `{"iteration_count": 2, "duration": 1, "proxy": "{{$DDOSIFY_TEST_PROXY_MISSING}}",
		"steps": [{"id": 1, "url": "`+proxiedTarget+`"}]}`)
	for name, args := range map[string][]string{
		"config": {"run", "--config", cfg, "--json"},
		"flag":   {"run", proxiedTarget, "-n", "2", "-d", "1", "-P", "{{$DDOSIFY_TEST_PROXY_MISSING}}", "--json"},
	} {
		code, out, errOut := run(t, args...)
		if code != ExitUsage || out != "" {
			t.Errorf("%s: exit %d, stdout %q", name, code, out)
		}
		if !strings.Contains(errOut, "DDOSIFY_TEST_PROXY_MISSING is not set") {
			t.Errorf("%s: stderr does not name the missing variable: %s", name, errOut)
		}
	}
	if n := len(p.seen()); n != 0 {
		t.Errorf("%d requests were sent", n)
	}
}
