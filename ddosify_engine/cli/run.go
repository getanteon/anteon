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

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"go.ddosify.com/ddosify/v2/config"
	"go.ddosify.com/ddosify/v2/core"
	"go.ddosify.com/ddosify/v2/core/hooks"
	"go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/proxy"
	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/types"
)

type runOptions struct {
	config      string
	target      string
	iterations  int
	duration    int
	stages      []string
	loadType    string
	loadSpread  string
	seed        int64
	engineMode  string
	method      string
	body        string
	auth        string
	headers     []string
	timeout     int
	proxy       string
	output      string
	json        bool
	debug       bool
	certPath    string
	certKeyPath string
	insecure    bool
	presets     []string
	html        string
	observers   []string
	ui          string
	noColor     bool

	drain     time.Duration
	tokenizer string
	llmURL    string
	model     string
	dataset   string
	apiKey    string
	maxTokens int

	inputTokens int
	ignoreEOS   bool

	concurrency      []int
	perLevel         int
	warmupIterations int
	gpus             int
}

// Flags of the top level "llm" config, the OpenAI compatible preset
var llmFlags = []string{"url", "model", "dataset", "api-key", "max-tokens", "input-tokens", "ignore-eos"}

// Flags that only describe a single request; with --config the scenario comes from the file.
var requestOnlyFlags = []string{"target", "method", "body", "auth", "cert-path", "cert-key-path"}

func newRunCmd(s Streams) *cobra.Command {
	return newRunCmdWith(&runOptions{}, s)
}

func newRunCmdWith(o *runOptions, s Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [TARGET_URL]",
		Short: "Run a load test from a config file or from flags",
		Long: `Run a load test. Give a config file with --config for a scenario of several steps,
streams, stages and an SLO, or a target URL for one request.

Every flag can also be set as DDOSIFY_<FLAG>, such as DDOSIFY_ITERATIONS=100 or
DDOSIFY_INSECURE=true. A flag wins over its environment variable, which wins over
the config file.

Exit codes: 0 the test passed, 1 it could not run, 2 a flag or the config is wrong,
3 a success criterion or the SLO's goodput failed, 130 interrupted.`,
		Example: `  ddosify run https://example.com -n 100 -d 10
  ddosify run --config config.json --json > result.json
  DDOSIFY_ITERATIONS=50 ddosify run --config config.json --duration 30
  ddosify run --config config.json --stage 60s:4:warm-up:warmup --stage 4m:15:6/min
  ddosify run --preset openai --url http://localhost:8000/v1/chat/completions --model qwen3 \
    --dataset prompts.jsonl --tokenizer tokenizer.json -n 120 -d 60`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := applyEnv(cmd.Flags()); err != nil {
				return usageErr(err)
			}
			if len(args) == 1 {
				if cmd.Flags().Changed("target") {
					return usageErr(fmt.Errorf("give the target once, as an argument or with --target"))
				}
				o.target = args[0]
				cmd.Flags().Lookup("target").Changed = true
			}
			return runTest(cmd, o, s)
		},
	}

	f := cmd.Flags()
	f.SortFlags = false
	f.StringVarP(&o.config, "config", "f", "", "JSON config file; flags and DDOSIFY_* variables override it")
	f.StringVarP(&o.target, "target", "t", "", "target URL, when there is no config file")
	f.IntVarP(&o.iterations, "iterations", "n", types.DefaultIterCount, "total iterations")
	f.IntVarP(&o.duration, "duration", "d", types.DefaultDuration, "test duration in seconds")
	f.StringArrayVar(&o.stages, "stage", nil, "a manual_load stage, "+stageFormat+"; repeat it, and the stages replace the config's manual_load")
	f.IntSliceVar(&o.concurrency, "concurrency", nil, "closed-loop stages, one per level, such as 1,2,4,8: at most that many iterations in flight; they replace the config's manual_load")
	f.IntVar(&o.perLevel, "per-level", 0, "with --concurrency: iterations of each level; 0 is max(10 x level, "+fmt.Sprint(minPerLevel)+")")
	f.IntVar(&o.warmupIterations, "warmup-iterations", 0, "with --concurrency: a warm-up stage of this many iterations at the first level, left out of the results")
	f.StringVarP(&o.loadType, "load-type", "l", types.DefaultLoadType, "linear, incremental or waved")
	f.StringVar(&o.loadSpread, "load-spread", "", "front, even or poisson: where in a second or stage iterations start")
	f.Int64Var(&o.seed, "seed", types.DefaultSeed, "seeds the poisson arrivals and the random data rows; the same seed repeats a run")
	f.StringVar(&o.engineMode, "engine-mode", types.EngineModeDdosify, "ddosify, distinct-user or repeated-user")
	f.StringVarP(&o.method, "method", "m", types.DefaultMethod, "request method")
	f.StringVarP(&o.body, "body", "b", "", "request body")
	f.StringVarP(&o.auth, "auth", "a", "", "basic authentication, username:password")
	f.StringArrayVarP(&o.headers, "header", "H", nil, `request header, "Name: value"; repeat it, and with a config it is added to every step`)
	f.IntVarP(&o.timeout, "timeout", "T", types.DefaultTimeout, "request timeout in seconds; for a stream it covers the whole stream")
	f.StringVarP(&o.proxy, "proxy", "P", "", "proxy, protocol://username:password@host:port")
	f.StringVarP(&o.output, "output", "o", types.DefaultOutputType, "report format: stdout or stdout-json")
	f.BoolVar(&o.json, "json", false, "print the report as JSON on stdout, the same as -o stdout-json")
	f.BoolVar(&o.debug, "debug", false, "run one iteration and print every request and response")
	f.StringVar(&o.certPath, "cert-path", "", "client certificate file (cert.pem)")
	f.StringVar(&o.certKeyPath, "cert-key-path", "", "client certificate key file (key.pem)")
	f.BoolVar(&o.insecure, "insecure", false, "do not verify the target's TLS certificate, for a self-signed one")
	f.StringArrayVar(&o.presets, "preset", nil, fmt.Sprintf("apply a config preset; repeat it (available: %s)", strings.Join(presetNames(), ", ")))
	f.StringVar(&o.html, "html", "", "also write an HTML report to this file")
	f.StringArrayVar(&o.observers, "observer", nil, "add a live observer, name or name=arg")
	f.StringVar(&o.ui, "ui", "auto", "auto (the TUI on a terminal, ndjson snapshots in a pipe or with --json, when built in), tui, ndjson or plain")
	f.BoolVar(&o.noColor, "no-color", false, "no colors; NO_COLOR and TERM=dumb do the same")
	f.DurationVar(&o.drain, "drain", 30*time.Second, "after an interrupt, how long iterations in flight may finish; a second interrupt cuts them")
	f.StringVar(&o.tokenizer, "tokenizer", "", "Hugging Face tokenizer.json to count the streamed answers' tokens")
	f.StringVar(&o.llmURL, "url", "", "with --preset openai: the chat completions URL")
	f.StringVar(&o.model, "model", "", "with --preset openai: the model")
	f.StringVar(&o.dataset, "dataset", "", "with --preset openai: prompts, a .jsonl, .csv or .txt file")
	f.StringVar(&o.apiKey, "api-key", "", "with --preset openai: sent as Authorization: Bearer; DDOSIFY_API_KEY keeps it off the command line")
	f.IntVar(&o.maxTokens, "max-tokens", 0, "with --preset openai: max_tokens of every request")
	f.IntVar(&o.inputTokens, "input-tokens", 0, "with --preset openai: every prompt is exactly this many tokens of the dataset's text (needs --tokenizer)")
	f.BoolVar(&o.ignoreEOS, "ignore-eos", false, "with --preset openai: send ignore_eos, so every answer runs to --max-tokens")
	f.IntVar(&o.gpus, "gpus", 0, "GPUs serving the model, to give the HTML report's throughput per GPU (the config's capacity.gpus)")
	return cmd
}

func presetNames() []string {
	if n := hooks.PresetNames(); len(n) > 0 {
		return n
	}
	return []string{"none in this build"}
}

// applyEnv sets every flag the command line left unset from DDOSIFY_<FLAG>.
func applyEnv(fs *pflag.FlagSet) error {
	var err error
	fs.VisitAll(func(f *pflag.Flag) {
		if err != nil || f.Changed {
			return
		}
		name := EnvPrefix + strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		v, ok := os.LookupEnv(name)
		if !ok {
			return
		}
		if e := fs.Set(f.Name, v); e != nil {
			err = fmt.Errorf("%s: %w", name, e)
		}
	})
	return err
}

func runTest(cmd *cobra.Command, o *runOptions, s Streams) error {
	flags := cmd.Flags()
	color := useColor(o.noColor, s.Out)
	setColor(color)

	if o.json {
		if flags.Changed("output") && o.output != report.OutputTypeStdoutJson {
			return usageErr(fmt.Errorf("--json and --output %s disagree", o.output))
		}
		o.output = report.OutputTypeStdoutJson
		flags.Lookup("output").Changed = true
	}

	h, err := buildHammer(flags, o)
	if err != nil {
		return usageErr(err)
	}
	if err := h.Validate(); err != nil {
		return usageErr(err)
	}
	h.Drain = o.drain
	// "output": "stdout-json" in the config or -o stdout-json is --json, streams and all.
	if h.ReportDestination == report.OutputTypeStdoutJson && !h.Debug {
		o.json = true
	}
	// While a TUI owns the terminal nothing else may write to it: stderr is held and printed after.
	errw := &heldWriter{w: s.Err}
	s.Err = errw
	defer errw.release()

	// The first interrupt (CTRL+C, SIGTERM or an observer's Stop) ends the schedule and lets the
	// iterations in flight finish for --drain; the second cuts them.
	ctx, interrupt := context.WithCancel(context.Background())
	defer interrupt()
	var cut atomic.Value
	var interrupts int32
	stop := func() {
		switch atomic.AddInt32(&interrupts, 1) {
		case 1:
			fmt.Fprintf(s.Err, "\nStopping: iterations in flight get %v to finish (--drain); interrupt again to cut them now.\n", h.Drain)
			interrupt()
		case 2:
			if f, ok := cut.Load().(func()); ok {
				fmt.Fprintln(s.Err, "Cutting the iterations in flight.")
				f()
			}
		}
	}
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		for {
			select {
			case <-sigs:
				stop()
			case <-finished:
				return
			}
		}
	}()

	observers, ownsTerminal, err := buildObservers(o, s, color, stop)
	if err != nil {
		return usageErr(err)
	}

	// While a TUI owns the terminal, the plain report waits until it closes.
	stdout := s.Out
	var held bytes.Buffer
	if ownsTerminal {
		stdout = &held
	}
	if ownsTerminal {
		errw.hold()
	}
	if h.Insecure {
		fmt.Fprintln(s.Err, "Warning: TLS certificates are not verified (--insecure).")
	}
	report.SetOutputs(stdout, s.Err, !ownsTerminal && isTerminal(errw.w))
	report.SetBanners(!ownsTerminal)
	report.SetWidth(terminalWidth(s.Out))

	es, err := core.InitEngineServices(h)
	if err != nil {
		return usageErr(err)
	}
	es.Observers = observers

	engine, err := core.NewEngine(ctx, h, es)
	if err != nil {
		return runtimeErr(err)
	}
	if err := engine.Init(); err != nil {
		return usageErr(err)
	}
	cut.Store(engine.Cut)
	status := engine.Start()

	if ownsTerminal {
		errw.release()
		io.Copy(s.Out, &held)
	}
	for _, obs := range observers {
		if f, ok := obs.(live.Failer); ok && f.Err() != nil {
			return runtimeErr(f.Err())
		}
	}
	switch {
	case status == "stopped":
		return &exitError{code: ExitInterrupted}
	case engine.IsTestFailed():
		return &exitError{code: ExitTestFailed, err: fmt.Errorf("the test failed: a success criterion, the SLO's goodput, or no iteration succeeded")}
	}
	return nil
}

// buildObservers picks the live view: the TUI on a full terminal, ndjson snapshots when stdout
// is not a terminal or with --json, then --html and every --observer.
func buildObservers(o *runOptions, s Streams, color bool, stop func()) ([]live.Observer, bool, error) {
	interactive := isTerminal(s.Out) && isTerminal(s.In)
	dumb := os.Getenv("TERM") == "dumb"
	opts := live.Options{Stdout: s.Out, Stderr: s.Err, Color: color, Interactive: interactive && !dumb, Stop: stop, JSON: o.json}
	registered := func(name string) bool { _, ok := live.Lookup(name); return ok }

	var names, argsOf []string
	add := func(name, arg string) { names, argsOf = append(names, name), append(argsOf, arg) }
	switch o.ui {
	case "plain":
	case "ndjson":
		add("ndjson", "")
	case "tui":
		if dumb {
			fmt.Fprintln(s.Err, "Warning: TERM=dumb has no full-screen TUI; printing plain output.")
			break
		}
		add("tui", "")
	case "auto":
		switch {
		case o.debug:
		case interactive && !dumb && !o.json && registered("tui"):
			add("tui", "")
		case (!isTerminal(s.Out) || o.json) && registered("ndjson"):
			add("ndjson", "")
			// Under --json the data is on stdout; stderr says, for a person, what the run is doing.
			if o.json && registered("status") {
				add("status", "")
			}
		}
	default:
		return nil, false, fmt.Errorf("--ui must be auto, tui, ndjson or plain, got %q", o.ui)
	}
	if o.html != "" {
		names, argsOf = append(names, "html"), append(argsOf, o.html)
	}
	for _, spec := range o.observers {
		name, arg, _ := strings.Cut(spec, "=")
		names, argsOf = append(names, name), append(argsOf, arg)
	}

	var out []live.Observer
	owns := false
	for i, name := range names {
		opts.Arg = argsOf[i]
		obs, err := live.Build(name, opts)
		if err != nil {
			return nil, false, err
		}
		if t, ok := obs.(live.TerminalOwner); ok && t.OwnsTerminal() {
			owns = true
		}
		out = append(out, obs)
	}
	return out, owns, nil
}

func anyChanged(flags *pflag.FlagSet, names []string) bool {
	for _, n := range names {
		if flags.Changed(n) {
			return true
		}
	}
	return false
}

func buildHammer(flags *pflag.FlagSet, o *runOptions) (types.Hammer, error) {
	fromPreset := o.config == "" && (len(o.presets) > 0 || anyChanged(flags, llmFlags))
	if o.config == "" && !fromPreset {
		h, err := hammerFromFlags(o)
		if err == nil {
			err = applyStages(&h, flags, o)
		}
		if err == nil {
			err = applyTokenizer(&h, flags, o)
		}
		return h, err
	}
	for _, name := range requestOnlyFlags {
		if flags.Changed(name) {
			return types.Hammer{}, fmt.Errorf("--%s describes a single request and can not be used with --config or --preset", name)
		}
	}
	raw := []byte("{}")
	if o.config != "" {
		var err error
		if raw, err = os.ReadFile(o.config); err != nil {
			return types.Hammer{}, err
		}
	}
	raw, err := applyLLMFlags(raw, flags, o)
	if err != nil {
		return types.Hammer{}, err
	}
	raw, err = hooks.ApplyPresets(raw, o.presets)
	if err != nil {
		return types.Hammer{}, err
	}
	if err := missingEnv(raw); err != nil {
		return types.Hammer{}, fmt.Errorf("%s: %w", configLabel(o), err)
	}
	reader, err := config.NewConfigReader(raw, config.ConfigTypeJson)
	if err != nil {
		return types.Hammer{}, fmt.Errorf("%s: %w", configLabel(o), err)
	}
	h, err := reader.CreateHammer()
	if err != nil {
		return h, fmt.Errorf("%s: %w", configLabel(o), err)
	}
	if err := applyStages(&h, flags, o); err != nil {
		return h, err
	}
	if err := overrideHammer(&h, flags, o); err != nil {
		return h, err
	}
	return h, applyTokenizer(&h, flags, o)
}

func configLabel(o *runOptions) string {
	if o.config != "" {
		return o.config
	}
	return "--preset"
}

// applyLLMFlags puts --url, --model, --dataset, --api-key and --max-tokens into the top level "llm" config.
func applyLLMFlags(raw []byte, flags *pflag.FlagSet, o *runOptions) ([]byte, error) {
	if !anyChanged(flags, llmFlags) {
		return raw, nil
	}
	var c map[string]interface{}
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	m, ok := c["llm"].(map[string]interface{})
	if !ok {
		m = map[string]interface{}{}
		c["llm"] = m
	}
	if flags.Changed("url") {
		m["url"] = o.llmURL
	}
	if flags.Changed("model") {
		m["model"] = o.model
	}
	if flags.Changed("dataset") {
		m["dataset"] = map[string]interface{}{"path": o.dataset}
	}
	if flags.Changed("api-key") {
		m["api_key"] = o.apiKey
	}
	if flags.Changed("max-tokens") {
		m["max_tokens"] = o.maxTokens
	}
	if flags.Changed("input-tokens") {
		m["input_tokens"] = o.inputTokens
	}
	if flags.Changed("ignore-eos") {
		m["ignore_eos"] = o.ignoreEOS
	}
	return json.Marshal(c)
}

// applyTokenizer gives every streamed step --tokenizer; a step without an llm config gets one.
func applyTokenizer(h *types.Hammer, flags *pflag.FlagSet, o *runOptions) error {
	if !flags.Changed("tokenizer") {
		return nil
	}
	if _, err := os.Stat(o.tokenizer); err != nil {
		return fmt.Errorf("--tokenizer: %w", err)
	}
	for i := range h.Scenario.Steps {
		st := &h.Scenario.Steps[i]
		if st.LLM == nil {
			if st.Stream == nil {
				continue
			}
			st.LLM = &types.LLMConf{}
		}
		st.LLM.Tokenizer = o.tokenizer
	}
	return nil
}

// overrideHammer applies the flags and environment variables that were given over the config.
func overrideHammer(h *types.Hammer, flags *pflag.FlagSet, o *runOptions) error {
	changed := flags.Changed
	if (changed("iterations") || changed("duration")) && len(h.TimeRunCountMap) > 0 {
		return fmt.Errorf("--iterations and --duration can not change a config with manual_load")
	}
	if changed("iterations") {
		h.IterationCount = o.iterations
	}
	if changed("duration") {
		h.TestDuration = o.duration
	}
	if changed("load-type") {
		h.LoadType = strings.ToLower(o.loadType)
	}
	if changed("load-spread") {
		h.LoadSpread = strings.ToLower(o.loadSpread)
	}
	if changed("seed") {
		h.Seed = o.seed
	}
	if changed("engine-mode") {
		h.EngineMode = o.engineMode
	}
	if changed("output") {
		h.ReportDestination = o.output
	}
	if changed("debug") {
		h.Debug = o.debug
	}
	if changed("insecure") {
		h.Insecure = o.insecure
	}
	if changed("proxy") {
		p, err := parseProxy(o.proxy)
		if err != nil {
			return err
		}
		h.Proxy = p
	}
	headers, err := parseHeaders(o.headers)
	if err != nil {
		return err
	}
	for i := range h.Scenario.Steps {
		st := &h.Scenario.Steps[i]
		if changed("timeout") {
			st.Timeout = o.timeout
		}
		for k, v := range headers {
			if st.Headers == nil {
				st.Headers = map[string]string{}
			}
			st.Headers[k] = v
		}
	}
	return nil
}

func hammerFromFlags(o *runOptions) (types.Hammer, error) {
	if o.target == "" {
		return types.Hammer{}, fmt.Errorf("give a target URL, or a config file with --config")
	}
	if err := types.IsTargetValid(o.target); err != nil {
		return types.Hammer{}, err
	}
	var a types.Auth
	if o.auth != "" {
		user, pass, ok := strings.Cut(o.auth, ":")
		if !ok {
			return types.Hammer{}, fmt.Errorf("--auth must be username:password")
		}
		a = types.Auth{Type: types.AuthHttpBasic, Username: user, Password: pass}
	}
	headers, err := parseHeaders(o.headers)
	if err != nil {
		return types.Hammer{}, err
	}
	step := types.ScenarioStep{ID: 1, Method: strings.ToUpper(o.method), Auth: a, Headers: headers,
		Payload: o.body, URL: o.target, Timeout: o.timeout}
	if (o.certPath == "") != (o.certKeyPath == "") {
		return types.Hammer{}, fmt.Errorf("--cert-path and --cert-key-path go together")
	}
	if o.certPath != "" {
		cert, pool, err := types.ParseTLS(o.certPath, o.certKeyPath)
		if err != nil {
			return types.Hammer{}, err
		}
		step.Cert, step.CertPool = cert, pool
	}
	p, err := parseProxy(o.proxy)
	if err != nil {
		return types.Hammer{}, err
	}
	return types.Hammer{
		IterationCount:    o.iterations,
		LoadType:          strings.ToLower(o.loadType),
		LoadSpread:        strings.ToLower(o.loadSpread),
		Seed:              o.seed,
		EngineMode:        o.engineMode,
		TestDuration:      o.duration,
		Scenario:          types.Scenario{Steps: []types.ScenarioStep{step}},
		Proxy:             p,
		ReportDestination: o.output,
		Debug:             o.debug,
		Insecure:          o.insecure,
		SamplingRate:      types.DefaultSamplingCount,
		SingleMode:        types.DefaultSingleMode,
	}, nil
}

func parseProxy(s string) (proxy.Proxy, error) {
	p := proxy.Proxy{Strategy: proxy.ProxyTypeSingle}
	if s == "" {
		return p, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return p, fmt.Errorf("--proxy: %w", err)
	}
	p.Addr = u
	return p, nil
}

var headerRe = regexp.MustCompile(`^\s*([^:\s]+)\s*:\s*(.*)$`)

func parseHeaders(in []string) (map[string]string, error) {
	out := make(map[string]string, len(in))
	for _, h := range in {
		m := headerRe.FindStringSubmatch(h)
		if m == nil {
			return nil, fmt.Errorf("header must be \"Name: value\", got %q", h)
		}
		out[m[1]] = m[2]
	}
	return out, nil
}

var envRef = regexp.MustCompile(`\{\{\s*\$([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// missingEnv names every {{$NAME}} the config uses that is not set, at once and before anything
// is sent: a run inside a container sees only the variables passed with -e.
func missingEnv(raw []byte) error {
	var missing []string
	seen := map[string]bool{}
	for _, m := range envRef.FindAllSubmatch(raw, -1) {
		name := string(m[1])
		if seen[name] {
			continue
		}
		seen[name] = true
		if _, ok := os.LookupEnv(name); !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	what := "the environment variables " + strings.Join(missing, ", ") + " are not set"
	if len(missing) == 1 {
		what = "the environment variable " + missing[0] + " is not set"
	}
	return fmt.Errorf("%s; the config uses {{$NAME}}. Set it, or with docker run pass each one with -e NAME=value "+
		"(-e NAME alone passes it only when it is exported in this shell)", what)
}
