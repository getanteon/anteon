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

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"go.ddosify.com/ddosify/config"
	"go.ddosify.com/ddosify/core"
	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/types"
)

//TODO: what about -preview flag? Users can see how many requests will be sent per second with the given parameters.

const headerRegexp = `^*(.+):\s*(.+)`

// We might consider to use Viper: https://github.com/spf13/viper
var (
	iterCount = flag.Int("n", types.DefaultIterCount, "Total iteration count")
	duration  = flag.Int("d", types.DefaultDuration, "Test duration in seconds")
	loadType  = flag.String("l", types.DefaultLoadType, "Type of the load test [linear, incremental, waved]")

	method = flag.String("m", types.DefaultMethod,
		"Request Method Type. For Http(s):[GET, POST, PUT, DELETE, UPDATE, PATCH]")
	payload = flag.String("b", "", "Payload of the network packet (body)")
	auth    = flag.String("a", "", "Basic authentication, username:password")
	headers header

	target  = flag.String("t", "", "Target URL")
	timeout = flag.Int("T", types.DefaultTimeout, "Request timeout in seconds")

	proxyFlag = flag.String("P", "",
		"Proxy address as protocol://username:password@host:port. Supported proxies [http(s), socks]")
	output = flag.String("o", types.DefaultOutputType, "Output destination")

	configPath = flag.String("config", "",
		"Json config file path. If a config file is provided, other flag values will be ignored")

	certPath    = flag.String("cert_path", "", "A path to a certificate file (usually called 'cert.pem')")
	certKeyPath = flag.String("cert_key_path", "", "A path to a certificate key file (usually called 'key.pem')")

	version = flag.Bool("version", false, "Prints version, git commit, built date (utc), go information and quit")
	debug   = flag.Bool("debug", false, "Iterates the scenario once and prints curl-like verbose result")
)

var (
	GitVersion = "development"
	GitCommit  = "unknown"
	BuildDate  = time.Now().UTC().Format(time.RFC3339)
)

func main() {
	flag.Var(&headers, "h", "Request Headers. Ex: -h 'Accept: text/html' -h 'Content-Type: application/xml'")
	flag.Parse()

	if *version {
		printVersionAndExit()
	}

	start()
}

func start() {
	h, err := createHammer()

	if err != nil {
		exitWithMsg(err.Error())
	}

	if err := h.Validate(); err != nil {
		exitWithMsg(err.Error())
	}

	run(h)
}

func createHammer() (h types.Hammer, err error) {
	if *configPath != "" {
		// running with config and debug mode set from cli
		return createHammerFromConfigFile(*debug)
	}
	return createHammerFromFlags()
}

var createHammerFromConfigFile = func(debug bool) (h types.Hammer, err error) {
	f, err := os.Open(*configPath)
	if err != nil {
		return
	}

	byteValue, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	c, err := config.NewConfigReader(byteValue, config.ConfigTypeJson)
	if err != nil {
		return
	}

	h, err = c.CreateHammer()
	if err != nil {
		return
	}

	if isFlagPassed("debug") {
		h.Debug = debug // debug flag from cli overrides debug in config file
	}

	return
}

var run = func(h types.Hammer) {
	ctx, cancel := context.WithCancel(context.Background())

	engine, err := core.NewEngine(ctx, h)
	if err != nil {
		exitWithMsg(err.Error())
	}

	err = engine.Init()
	if err != nil {
		exitWithMsg(err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()

	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	engine.Start()
}

var createHammerFromFlags = func() (h types.Hammer, err error) {
	if *target == "" {
		err = fmt.Errorf("Please provide the target url with -t flag")
		return
	}

	s, err := createScenario()
	if err != nil {
		return
	}

	p, err := createProxy()
	if err != nil {
		return
	}

	h = types.Hammer{
		IterationCount:    *iterCount,
		LoadType:          strings.ToLower(*loadType),
		TestDuration:      *duration,
		Scenario:          s,
		Proxy:             p,
		ReportDestination: *output,
		Debug:             *debug,
	}
	return
}

func createProxy() (p proxy.Proxy, err error) {
	var proxyURL *url.URL
	if *proxyFlag != "" {
		proxyURL, err = url.Parse(*proxyFlag)
		if err != nil {
			return
		}
	}

	p = proxy.Proxy{
		Strategy: proxy.ProxyTypeSingle,
		Addr:     proxyURL,
	}
	return
}

func createScenario() (s types.Scenario, err error) {
	// Auth
	var a types.Auth
	if *auth != "" {
		creds := strings.Split(*auth, ":")
		if len(creds) != 2 {
			err = fmt.Errorf("auth credentials couldn't be parsed")
			return
		}

		a = types.Auth{
			Type:     types.AuthHttpBasic,
			Username: creds[0],
			Password: creds[1],
		}
	}

	err = types.IsTargetValid(*target)
	if err != nil {
		return
	}

	h, err := parseHeaders(headers)
	if err != nil {
		return
	}

	step := types.ScenarioStep{
		ID:      1,
		Method:  strings.ToUpper(*method),
		Auth:    a,
		Headers: h,
		Payload: *payload,
		URL:     *target,
		Timeout: *timeout,
	}

	// TODO : if whether certPath or certKeyPath doesn't exist and another one exists, we should return an error to user.
	if *certPath != "" && *certKeyPath != "" {
		cert, pool, e := types.ParseTLS(*certPath, *certKeyPath)
		if e != nil {
			err = e
			return
		}

		step.Cert = cert
		step.CertPool = pool
	}
	s = types.Scenario{Steps: []types.ScenarioStep{step}}

	return
}

func versionTemplate() string {
	b := strings.Builder{}
	w := tabwriter.NewWriter(&b, 0, 0, 5, ' ', 0)
	fmt.Fprintf(w, "Version:\t%s\n", GitVersion)
	fmt.Fprintf(w, "Git commit:\t%s\n", GitCommit)
	fmt.Fprintf(w, "Built\t%s\n", BuildDate)
	fmt.Fprintf(w, "Go version:\t%s\n", runtime.Version())
	fmt.Fprintf(w, "OS/Arch:\t%s\n", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
	w.Flush()

	return b.String()
}

func printVersionAndExit() {
	fmt.Println(versionTemplate())
	os.Exit(0)
}

func exitWithMsg(msg string) {
	if msg != "" {
		msg = "err: " + msg
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}

func parseHeaders(headersArr []string) (headersMap map[string][]string, err error) {
	re := regexp.MustCompile(headerRegexp)
	headersMap = make(map[string][]string)
	for _, h := range headersArr {
		matches := re.FindStringSubmatch(h)
		if len(matches) < 1 {
			err = fmt.Errorf("invalid header:  %v", h)
			return
		}
		if _, found := headersMap[matches[1]]; found {
			headersMap[matches[1]] = append(headersMap[matches[1]], matches[2])
		} else {
			headersMap[matches[1]] = []string{matches[2]}
		}
	}
	return
}

type header []string

func (h *header) String() string {
	return fmt.Sprintf("%s - %d", *h, len(*h))
}

func (h *header) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
