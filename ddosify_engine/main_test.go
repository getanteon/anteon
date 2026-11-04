package main

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

// TestExitCode runs the binary's main in a child process and reads its real exit code.
func TestExitCode(t *testing.T) {
	if args := os.Getenv("DDOSIFY_TEST_MAIN_ARGS"); args != "" {
		os.Args = append([]string{"ddosify"}, args)
		main()
		return
	}
	cases := map[string]int{"version": 0, "nonsense-command": 2}
	for arg, want := range cases {
		cmd := exec.Command(os.Args[0], "-test.run=^TestExitCode$")
		cmd.Env = append(os.Environ(), "DDOSIFY_TEST_MAIN_ARGS="+arg)
		err := cmd.Run()
		got := 0
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			got = ee.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("ddosify %s: expected exit %d, got %d", arg, want, got)
		}
	}
}
