package report

import (
	"strings"
	"testing"
	"unicode/utf8"

	"go.ddosify.com/ddosify/v2/core/types"
)

func failed(id uint16, name, reason string, code int) *types.ScenarioStepResult {
	sr := &types.ScenarioStepResult{StepID: id, StepName: name, StatusCode: code}
	if reason != "" {
		sr.Err = types.RequestError{Type: types.ErrorConn, Reason: reason}
	}
	return sr
}

func TestFirstErrorsOncePerKind(t *testing.T) {
	var b strings.Builder
	var f firstErrors
	tls := `Post "https://api.example.internal/api/gurus/%s/chats": tls: failed to verify certificate: x509: certificate signed by unknown authority`
	for _, id := range []string{"a", "b", "c"} {
		f.note(&b, &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{failed(1, "Create chat", strings.Replace(tls, "%s", id, 1), 0)}})
	}
	f.note(&b, &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{failed(2, "", "", 503), failed(2, "", "", 503), {StepID: 3, StatusCode: 200}}})
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want one line per kind, got %q", b.String())
	}
	if !strings.HasPrefix(lines[0], `Error: step 1 "Create chat": network_error: Post "https://api.example.internal/api/gurus/a/chats": tls:`) || !strings.Contains(lines[0], "x509") {
		t.Errorf("first line %q", lines[0])
	}
	if lines[1] != "Error: step 2: unavailable_503: HTTP 503" {
		t.Errorf("second line %q", lines[1])
	}
}

func TestFirstErrorsAreBounded(t *testing.T) {
	var b strings.Builder
	var f firstErrors
	for i := range 100 {
		f.note(&b, &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{failed(uint16(i), "s", "boom", 0)}})
	}
	if n := strings.Count(b.String(), "\n"); n != maxFirstErrors+1 {
		t.Errorf("%d lines, want %d errors and one line saying the rest are in the report", n, maxFirstErrors)
	}
}

func TestFirstErrorsSayWhy(t *testing.T) {
	var b strings.Builder
	var f firstErrors
	refused := &types.ScenarioStepResult{StepID: 1, Err: types.RequestError{Type: types.ErrorConn, Reason: types.ReasonConnRefused,
		Detail: `Post "http://api.example.internal:9/chats": dial tcp 10.0.0.1:9: connect: connection refused`}}
	denied := &types.ScenarioStepResult{StepID: 2, StatusCode: 401, RespBody: []byte("\n  {\"error\":\"Invalid API key\"}\nsecond line")}
	long := &types.ScenarioStepResult{StepID: 3, StatusCode: 500, RespBody: []byte(strings.Repeat("ş", 150))}
	f.note(&b, &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{refused, denied, long}})
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %q", b.String())
	}
	if lines[0] != `Error: step 1: network_error: Post "http://api.example.internal:9/chats": dial tcp 10.0.0.1:9: connect: connection refused` {
		t.Errorf("the detail, with the address: %q", lines[0])
	}
	if lines[1] != `Error: step 2: client_error_4xx: HTTP 401: {"error":"Invalid API key"}` {
		t.Errorf("the body's first line: %q", lines[1])
	}
	if !strings.HasSuffix(lines[2], "...") || !utf8.ValidString(lines[2]) || len(lines[2]) > 260 {
		t.Errorf("a long body is cut on a rune: %q", lines[2])
	}
}
