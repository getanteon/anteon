package cli

import (
	"strings"
	"testing"
)

func TestMissingEnvNamesEveryOne(t *testing.T) {
	t.Setenv("DDOSIFY_TEST_SET", "x")
	cfg := writeConfig(t, `{"iteration_count": 1, "duration": 1, "steps": [{"id": 1, "url": "{{$DDOSIFY_TEST_URL}}/{{$DDOSIFY_TEST_SET}}",
		"headers": {"Authorization": "Bearer {{ $DDOSIFY_TEST_KEY }}", "X-Again": "{{$DDOSIFY_TEST_URL}}"}}]}`)
	code, out, errOut := run(t, "run", "--config", cfg, "--json")
	if code != ExitUsage || out != "" {
		t.Fatalf("exit %d, stdout %q", code, out)
	}
	for _, want := range []string{"DDOSIFY_TEST_URL, DDOSIFY_TEST_KEY are not set", "-e NAME=value"} {
		if !strings.Contains(errOut, want) {
			t.Errorf("stderr lacks %q: %s", want, errOut)
		}
	}
	if strings.Contains(errOut, "DDOSIFY_TEST_SET") {
		t.Errorf("a set variable is named: %s", errOut)
	}
}
