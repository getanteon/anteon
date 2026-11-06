package report

import (
	"strings"
	"testing"

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
