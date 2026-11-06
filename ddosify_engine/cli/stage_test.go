package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"go.ddosify.com/ddosify/v2/core/types"
)

func TestParseStages(t *testing.T) {
	got, err := parseStages([]string{"60s:4:warm-up:warmup", "4m:15:6/min", "2000ms:0", "1h:3::warmup", "1s:2:"})
	if err != nil {
		t.Fatal(err)
	}
	want := types.TimeRunCount{
		{Duration: 60, Count: 4, Name: "warm-up", Warmup: true},
		{Duration: 240, Count: 15, Name: "6/min"},
		{Duration: 2, Count: 0},
		{Duration: 3600, Count: 3, Warmup: true},
		{Duration: 1, Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %+v, got %+v", want, got)
	}
	if it, d := got.Totals(); it != 24 || d != 3903 {
		t.Errorf("totals: %d iterations, %d seconds", it, d)
	}
}

func TestParseStagesInvalid(t *testing.T) {
	cases := map[string]string{
		"60s":                   "no COUNT",
		"60s:":                  "COUNT",
		"60s:x":                 "COUNT",
		"60s:-1":                "COUNT",
		"60s:1.5":               "COUNT",
		"60:4":                  "not a duration",
		"soon:4":                "not a duration",
		"1.5s:4":                "whole number of seconds",
		"1500ms:4":              "whole number of seconds",
		"0s:4":                  "greater than 0",
		"-5s:4":                 "greater than 0",
		"60s:4:name:Warmup":     "fourth field",
		"60s:4:name:cold":       "fourth field",
		"60s:4:a:b:warmup":      "too many fields",
		"60s:4:12:00:warmup":    "too many fields",
		"":                      "no COUNT",
		"60s:4:warm-up:warmup:": "too many fields",
	}
	for v, why := range cases {
		_, err := parseStages([]string{"1s:1", v})
		if err == nil {
			t.Errorf("%q: expected an error", v)
			continue
		}
		if !strings.Contains(err.Error(), strconv.Quote(v)) || !strings.Contains(err.Error(), why) {
			t.Errorf("%q: the error must name the value and say %q: %v", v, why, err)
		}
	}
}

// hammerFor builds the Hammer the run command would build from args, environment included.
func hammerFor(t *testing.T, args ...string) (types.Hammer, error) {
	t.Helper()
	o := &runOptions{}
	cmd := newRunCmdWith(o, Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}})
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatal(err)
	}
	if err := applyEnv(cmd.Flags()); err != nil {
		return types.Hammer{}, err
	}
	if rest := cmd.Flags().Args(); len(rest) == 1 {
		o.target = rest[0]
	}
	return buildHammer(cmd.Flags(), o)
}

const stageConfigRest = `"load_type": "waved", "load_spread": "poisson", "seed": 9007199254740993,
	"stop_on_failure": true, "request_log": true, "output": "stdout-json",
	"slo": {"first_token_ms": 500, "goodput_perc": 90},
	"data": {"rows": {"path": "rows.csv", "vars": {"0": {"tag": "q"}}}},
	"steps": [{"id": 1, "url": "http://127.0.0.1:1/{{data.rows.q}}"}]`

func TestStageFlagBuildsTheManualLoadHammer(t *testing.T) {
	fromFile := writeConfig(t, `{"manual_load": [
		{"duration": 60, "count": 4, "name": "warm-up", "warmup": true},
		{"duration": 240, "count": 15, "name": "6/min"},
		{"duration": 30, "count": 0}],
		`+stageConfigRest+`}`)
	want, err := hammerFor(t, "--config", fromFile)
	if err != nil {
		t.Fatal(err)
	}
	if want.IterationCount != 19 || want.TestDuration != 330 || want.Seed != 9007199254740993 {
		t.Fatalf("the config file's hammer: %+v", want)
	}

	stages := []string{"--stage", "60s:4:warm-up:warmup", "--stage", "4m:15:6/min", "--stage", "30s:0"}
	others := map[string]string{
		"another manual_load":    `{"manual_load": [{"duration": 5, "count": 500, "name": "old"}], ` + stageConfigRest + `}`,
		"duration and iteration": `{"duration": 7, "iteration_count": 70, ` + stageConfigRest + `}`,
		"no load at all":         `{` + stageConfigRest + `}`,
	}
	for name, cfg := range others {
		got, err := hammerFor(t, append([]string{"--config", writeConfig(t, cfg)}, stages...)...)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: --stage must build the same hammer as manual_load\nwant %+v\n got %+v", name, want, got)
		}
	}

	t.Setenv("DDOSIFY_STAGE", "2s:6:only")
	got, err := hammerFor(t, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if got.IterationCount != 6 || got.TestDuration != 2 || len(got.TimeRunCountMap) != 1 || got.TimeRunCountMap[0].Name != "only" {
		t.Errorf("DDOSIFY_STAGE with a target: %+v", got)
	}
}

func TestStageFlagRefused(t *testing.T) {
	cfg := writeConfig(t, `{"steps":[{"id":1,"url":"http://127.0.0.1:1"}]}`)
	cases := map[string][]string{
		`"60s:-1"`:     {"run", "--config", cfg, "--stage", "60s:-1"},
		`"1.5s:3"`:     {"run", "--config", cfg, "--stage", "1s:1", "--stage", "1.5s:3"},
		"--iterations": {"run", "--config", cfg, "--stage", "1s:1", "-n", "5"},
		"--duration":   {"run", "http://127.0.0.1:1", "--stage", "1s:1", "-d", "5"},
	}
	for want, args := range cases {
		code, out, errOut := run(t, args...)
		if code != ExitUsage || out != "" || !strings.Contains(errOut, want) {
			t.Errorf("%v: expected exit 2 naming %s, got %d %q %q", args, want, code, out, errOut)
		}
	}
}

func TestStageFlagRunsTheStages(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	cfg := writeConfig(t, fmt.Sprintf(`{"manual_load": [{"duration": 30, "count": 1000, "name": "from the file"}],
		"load_spread": "even", "steps":[{"id":1,"url":"%s"}]}`, srv.URL))
	code, out, errOut := run(t, "run", "--config", cfg, "--stage", "1s:2:warm-up:warmup", "--stage", "1s:3:6/min", "--json")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if errOut != "" {
		t.Errorf("--stage prints nothing on stderr: %q", errOut)
	}
	if n := atomic.LoadInt64(&c.n); n != 5 {
		t.Errorf("the stages start 5 iterations, the server saw %d", n)
	}
	var res struct {
		Stages []struct {
			Index      int    `json:"index"`
			Name       string `json:"name"`
			Warmup     bool   `json:"warmup"`
			Iterations int64  `json:"iterations"`
		} `json:"stages"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("stdout must be the JSON report: %v %q", err, out)
	}
	want := []struct {
		name       string
		warmup     bool
		iterations int64
	}{{"warm-up", true, 2}, {"6/min", false, 3}}
	if len(res.Stages) != len(want) {
		t.Fatalf("expected %d stages in the report, got %+v", len(want), res.Stages)
	}
	for i, w := range want {
		s := res.Stages[i]
		if s.Index != i || s.Name != w.name || s.Warmup != w.warmup || s.Iterations != w.iterations {
			t.Errorf("stage %d: expected %+v, got %+v", i, w, s)
		}
	}
}

func TestConcurrencyFlagBuildsClosedLoopStages(t *testing.T) {
	cfg := writeConfig(t, `{`+stageConfigRest+`}`)
	got, err := hammerFor(t, "--config", cfg, "--concurrency", "1,4,16", "--warmup-iterations", "3")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		name     string
		count, c int
		warmup   bool
	}{{"warm-up", 3, 1, true}, {"c=1", 20, 1, false}, {"c=4", 40, 4, false}, {"c=16", 160, 16, false}}
	if len(got.TimeRunCountMap) != len(want) || got.IterationCount != 223 || got.TestDuration != 0 {
		t.Fatalf("stages %+v, iterations %d, duration %d", got.TimeRunCountMap, got.IterationCount, got.TestDuration)
	}
	for i, w := range want {
		s := got.TimeRunCountMap[i]
		if s.Name != w.name || s.Count != w.count || s.Concurrency != w.c || s.Warmup != w.warmup {
			t.Errorf("stage %d: %+v, want %+v", i, s, w)
		}
	}
	got, err = hammerFor(t, "--config", cfg, "--concurrency", "2", "--per-level", "7", "--gpus", "2")
	if err != nil {
		t.Fatal(err)
	}
	if got.TimeRunCountMap[0].Count != 7 || got.Capacity == nil || got.Capacity.GPUs != 2 {
		t.Errorf("--per-level and --gpus: %+v %+v", got.TimeRunCountMap, got.Capacity)
	}
	st, err := parseStages([]string{"c=8:80:eight", "c=2:10:warm:warmup"})
	if err != nil || st[0].Concurrency != 8 || st[0].Count != 80 || st[0].Name != "eight" || !st[1].Warmup || st[1].Concurrency != 2 {
		t.Errorf("--stage c=N: %+v %v", st, err)
	}
}

func TestConcurrencyFlagRefused(t *testing.T) {
	cfg := writeConfig(t, `{"steps":[{"id":1,"url":"http://127.0.0.1:1"}]}`)
	cases := map[string][]string{
		"use one":               {"run", "--config", cfg, "--concurrency", "2", "--stage", "1s:1"},
		"--iterations":          {"run", "--config", cfg, "--concurrency", "2", "-n", "5"},
		"go with --concurrency": {"run", "--config", cfg, "--per-level", "5"},
		"1 or more":             {"run", "--config", cfg, "--concurrency", "0"},
		`CONCURRENCY "0"`:       {"run", "--config", cfg, "--stage", "c=0:5"},
	}
	for want, args := range cases {
		code, _, errOut := run(t, args...)
		if code != ExitUsage || !strings.Contains(errOut, want) {
			t.Errorf("%v: expected exit 2 naming %s, got %d %q", args, want, code, errOut)
		}
	}
}

func TestConcurrencyFlagRunsTheStages(t *testing.T) {
	c := &counter{}
	srv := countingServer(t, c)
	cfg := writeConfig(t, fmt.Sprintf(`{"steps":[{"id":1,"url":"%s"}]}`, srv.URL))
	code, out, errOut := run(t, "run", "--config", cfg, "--concurrency", "1,3", "--per-level", "4", "--warmup-iterations", "2", "--json")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if n := atomic.LoadInt64(&c.n); n != 10 {
		t.Errorf("the stages run 10 iterations, the server saw %d", n)
	}
	var res struct {
		Stages []struct {
			Name        string `json:"name"`
			Iterations  int64  `json:"iterations"`
			Concurrency int    `json:"concurrency"`
		} `json:"stages"`
	}
	if err := json.Unmarshal([]byte(lastJSONLine(out)), &res); err != nil {
		t.Fatalf("the report line: %v %q", err, out)
	}
	if len(res.Stages) != 3 || res.Stages[1].Name != "c=1" || res.Stages[2].Concurrency != 3 || res.Stages[2].Iterations != 4 {
		t.Errorf("stages %+v", res.Stages)
	}
}

// lastJSONLine is the last line of a --json output, the report.
func lastJSONLine(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return lines[len(lines)-1]
}
