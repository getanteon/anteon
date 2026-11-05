/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

// Package cli is the ddosify command line, following https://clig.dev: the report on
// stdout, banners, progress and errors on stderr, exit codes that tell a failed test from
// a broken one, and flags over DDOSIFY_* environment variables over the config file.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Exit codes
const (
	ExitOK          = 0
	ExitError       = 1 // the test could not run: a network, file or engine error
	ExitUsage       = 2 // a flag, an argument or the config is wrong
	ExitTestFailed  = 3 // the test ran and a success criterion or the SLO's goodput failed
	ExitInterrupted = 130
)

// EnvPrefix prefixes the environment variable of every run flag: --iterations is DDOSIFY_ITERATIONS.
const EnvPrefix = "DDOSIFY_"

// BuildInfo is set by the release build through -ldflags.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// Streams are the process's standard streams; tests replace them.
type Streams struct {
	In       io.Reader
	Out, Err io.Writer
}

// exitError carries an exit code to Main.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *exitError) Unwrap() error { return e.err }

func usageErr(err error) error   { return &exitError{code: ExitUsage, err: err} }
func runtimeErr(err error) error { return &exitError{code: ExitError, err: err} }

// Main runs the command line and returns the exit code.
func Main(args []string, info BuildInfo, s Streams) int {
	root := newRoot(info, s)
	args = normalizeArgs(args)
	if len(args) > 0 && args[0] == "run" {
		if msg := removedFlag(args[1:]); msg != "" {
			fmt.Fprintln(s.Err, "Error:", msg)
			fmt.Fprintln(s.Err, "Run 'ddosify run --help' for the flags.")
			return ExitUsage
		}
	}
	if len(args) > 0 {
		if sub, _, err := root.Find(args[:1]); err == nil && sub != root {
			if msg := singleDashLongFlag(sub, args[1:]); msg != "" {
				fmt.Fprintln(s.Err, "Error:", msg)
				fmt.Fprintf(s.Err, "Run '%s --help' for the flags.\n", sub.CommandPath())
				return ExitUsage
			}
		}
	}
	root.SetArgs(args)
	cmd, err := root.ExecuteC()
	if err == nil {
		return ExitOK
	}
	var ee *exitError
	if errors.As(err, &ee) {
		if ee.err != nil && ee.code != ExitTestFailed && ee.code != ExitInterrupted {
			fmt.Fprintln(s.Err, "Error:", ee.err)
		}
		return ee.code
	}
	// cobra's own errors: an unknown command or flag, a missing argument
	fmt.Fprintln(s.Err, "Error:", err)
	path := root.CommandPath()
	if cmd != nil {
		path = cmd.CommandPath()
		if hint := flagHint(cmd, err); hint != "" {
			fmt.Fprintln(s.Err, hint)
		}
	}
	fmt.Fprintf(s.Err, "Run '%s --help' for usage.\n", path)
	return ExitUsage
}

// removedV1Flags are the v1 flags v2 has no place for, and what replaced them.
var removedV1Flags = map[string]string{
	"-p":         "-p (protocol) is gone: the scheme goes in the URL, such as https://example.com",
	"--protocol": "--protocol is gone: the scheme goes in the URL, such as https://example.com",
	"-h":         "-h is help now: give a header with -H 'Name: value'",
}

// removedFlag tells what replaced a v1 flag in args, or "" when there is none.
func removedFlag(args []string) string {
	for i, a := range args {
		if a == "--" {
			return ""
		}
		name, _, _ := strings.Cut(a, "=")
		msg, ok := removedV1Flags[name]
		if !ok {
			continue
		}
		// -h alone asks for help; -h followed by a value was v1's header
		if name == "-h" && (i+1 >= len(args) || strings.HasPrefix(args[i+1], "-")) {
			continue
		}
		return msg
	}
	return ""
}

func newRoot(info BuildInfo, s Streams) *cobra.Command {
	root := &cobra.Command{
		Use:   "ddosify",
		Short: "Load test HTTP and streaming (SSE) APIs, LLM endpoints included",
		Long: `Ddosify sends a scenario of HTTP requests at a planned arrival rate and reports latency,
time to first token and goodput. Run a test with "ddosify run".`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       info.Version,
	}
	root.SetIn(s.In)
	root.SetOut(s.Out)
	root.SetErr(s.Err)
	root.SetVersionTemplate("{{.Version}}\n")
	root.CompletionOptions.HiddenDefaultCmd = false
	root.AddCommand(newRunCmd(s), newVersionCmd(info, s), newDocsCmd(s))
	return root
}

// normalizeArgs keeps the v1 command line working: "ddosify -t URL" and "ddosify -config FILE"
// run a test, single-dash long flags become double-dash, and v1's -h HEADER becomes -H.
func normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	switch args[0] {
	case "-version", "--version":
		return []string{"version"}
	case "-h", "--help", "help":
		return args
	}
	if !strings.HasPrefix(args[0], "-") {
		return args
	}
	legacy := map[string]string{"-config": "--config", "-debug": "--debug", "-cert_path": "--cert-path",
		"-cert_key_path": "--cert-key-path", "-version": "--version"}
	out := []string{"run"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, value, hasValue := strings.Cut(a, "=")
		if long, ok := legacy[name]; ok {
			if hasValue {
				a = long + "=" + value
			} else {
				a = long
			}
		} else if a == "-h" && i+1 < len(args) && strings.Contains(args[i+1], ":") {
			a = "-H"
		}
		out = append(out, a)
	}
	return out
}

func isTerminal(w interface{}) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// useColor follows NO_COLOR, TERM=dumb and --no-color; fatih/color already checks stdout is a terminal.
func useColor(noColorFlag bool, w io.Writer) bool {
	if noColorFlag || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(w)
}

func setColor(enabled bool) {
	color.NoColor = !enabled
}

var singleDashLong = regexp.MustCompile(`unknown shorthand flag: '.' in -([A-Za-z][A-Za-z0-9_-]+)`)

// flagHint suggests the double-dash spelling when a long flag was given with one dash, as v1 took it.
func flagHint(cmd *cobra.Command, err error) string {
	m := singleDashLong.FindStringSubmatch(err.Error())
	if m == nil {
		return ""
	}
	for _, name := range []string{m[1], strings.ReplaceAll(m[1], "_", "-")} {
		if cmd.Flags().Lookup(name) != nil || cmd.InheritedFlags().Lookup(name) != nil {
			return fmt.Sprintf("Did you mean --%s? Long flags take two dashes.", name)
		}
	}
	return ""
}

// singleDashLongFlag finds a long flag written with one dash, such as -config, which pflag would
// otherwise read as shorthands with a value (-c onfig, or -H tml for -html), and says how to write it.
func singleDashLongFlag(cmd *cobra.Command, args []string) string {
	takesValue := func(a string) bool {
		if strings.Contains(a, "=") || !strings.HasPrefix(a, "-") {
			return false
		}
		var f *pflag.Flag
		if strings.HasPrefix(a, "--") {
			f = cmd.Flags().Lookup(a[2:])
		} else if len(a) == 2 {
			f = cmd.Flags().ShorthandLookup(a[1:])
		}
		return f != nil && f.Value.Type() != "bool"
	}
	for i, a := range args {
		if a == "--" {
			return ""
		}
		if len(a) < 3 || a[0] != '-' || a[1] == '-' || (i > 0 && takesValue(args[i-1])) {
			continue
		}
		name, _, _ := strings.Cut(a[1:], "=")
		for _, n := range []string{name, strings.ReplaceAll(name, "_", "-")} {
			if len(n) > 1 && (cmd.Flags().Lookup(n) != nil || cmd.InheritedFlags().Lookup(n) != nil) {
				return fmt.Sprintf("%s has one dash; long flags take two. Did you mean --%s?", a, n)
			}
		}
	}
	return ""
}

// terminalWidth is w's width in columns, or 0 when w is not a terminal.
func terminalWidth(w interface{}) int {
	f, ok := w.(interface{ Fd() uintptr })
	if !ok || !isTerminal(w) {
		return 0
	}
	width, _, err := term.GetSize(f.Fd())
	if err != nil {
		return 0
	}
	return width
}
