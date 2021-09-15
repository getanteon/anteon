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
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"ddosify.com/hammer/configReader"
	"ddosify.com/hammer/core"
	"ddosify.com/hammer/core/types"
)

//TODO: what about -preview flag? Users can see how many requests will be sent per second with the given parameters.

const headerRegexp = `^([\w-]+):\s*(.+)`

//TODO: Add ddosify-hammer User-Agent to header config
// We might consider to use Viper: https://github.com/spf13/viper
var (
	reqCount = flag.Int("n", 1000, "Total request count")
	loadType = flag.String("l", types.LoadTypeLinear, "Type of the load test [linear, capacity, stress, soak")
	duration = flag.Int("d", 10, "Test duration in seconds")

	protocol = flag.String("p", types.ProtocolHTTP, "[HTTP, HTTPS]")
	method   = flag.String("m", http.MethodGet, "Request Method Type. For Http(s):[GET, POST, PUT, DELETE, UPDATE, PATCH]")
	payload  = flag.String("b", "", "Payload of the network packet")
	auth     = flag.String("a", "", "Basic authentication, username:password")
	headers  header

	target  = flag.String("t", "", "Target URL")
	timeout = flag.Int("T", 10, "Request timeout in seconds")

	proxy  = flag.String("P", "", "Proxy address as host:port")
	output = flag.String("o", types.OutputTypeStdout, "Output destination")

	configPath = flag.String("config", "",
		"Json config file path. If a config file is provided, other flag values will be ignored.")
)

func main() {
	flag.Var(&headers, "h", "Request Headers. Ex: -H 'Accept: text/html' -H 'Content-Type: application/xml'")
	flag.Parse()

	var h types.Hammer

	if *configPath != "" {
		c, err := configReader.NewConfigFileReader(*configPath, "jsonReader")
		if err != nil {
			exitWithMsg(err.Error())
		}

		h, err = c.CreateHammer()
		if err != nil {
			exitWithMsg(err.Error())
		}
	} else {
		if *target == "" {
			exitWithMsg("Please provide the target url")
		}

		s := createScenario()
		p := createProxy()

		h = createHammer(s, p)
	}

	if err := h.Validate(); err != nil {
		exitWithMsg(err.Error())
	}

	run(h)
}

func run(h types.Hammer) {
	ctx, cancel := context.WithCancel(context.Background())

	engine := core.NewEngine(ctx, h)
	err := engine.Init()
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

func createHammer(s types.Scenario, p types.Proxy) types.Hammer {
	h := types.Hammer{
		TotalReqCount:     *reqCount,
		LoadType:          strings.ToLower(*loadType),
		TestDuration:      *duration,
		Scenario:          s,
		Proxy:             p,
		ReportDestination: *output,
	}
	return h
}

func createProxy() types.Proxy {
	var proxyURL *url.URL
	if *proxy != "" {
		var err error
		proxyURL, err = url.Parse(*proxy)
		if err != nil {
			exitWithMsg(err.Error())
		}
	}

	p := types.Proxy{
		Strategy: "single",
		Addr:     proxyURL,
	}
	return p
}

func createScenario() types.Scenario {
	// Auth
	var a types.Auth
	if *auth != "" {
		creds := strings.Split(*auth, ":")
		if len(creds) != 2 {
			exitWithMsg("auth credentials couldn't be parsed")
		}

		a = types.Auth{
			Type:     types.AuthHttpBasic,
			Username: creds[0],
			Password: creds[1],
		}
	}

	// Protocol & URL
	url, err := url.Parse(*target)
	if err != nil {
		exitWithMsg("invalid target url")
	}
	if url.Scheme == "" {
		url.Scheme = *protocol
	}

	return types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: strings.ToUpper(url.Scheme),
				Method:   strings.ToUpper(*method),
				Auth:     a,
				Headers:  parseHeaders(headers),
				Payload:  *payload,
				URL:      url.String(),
				Timeout:  *timeout,
			},
		},
	}
}

func exitWithMsg(msg string) {
	if msg != "" {
		msg = "err: " + msg
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}

func parseHeaders(headersArr []string) map[string]string {
	re := regexp.MustCompile(headerRegexp)
	headersMap := make(map[string]string)
	for _, h := range headersArr {
		matches := re.FindStringSubmatch(h)
		if len(matches) < 1 {
			exitWithMsg(fmt.Sprintf("invalid header:  %v", h))
		}
		headersMap[matches[1]] = matches[2]
	}
	return headersMap
}

type header []string

func (h *header) String() string {
	return fmt.Sprintf("%s - %d", *h, len(*h))
}

func (h *header) Set(value string) error {
	*h = append(*h, value)
	return nil
}
