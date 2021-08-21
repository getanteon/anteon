package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"ddosify.com/hammer/core"
	"ddosify.com/hammer/core/types"
)

//TODO: what about -preview flag? Users can see how many requests will be sent per second with the given parameters.

const headerRegexp = `^([\w-]+):\s*(.+)`

// We might consider to use Viper: https://github.com/spf13/viper
var (
	reqCount = flag.Int("n", 1000, "Total request count")
	loadType = flag.String("l", "linear", "Type of the load test [linear, capacity, stress, soak")
	duration = flag.Int("d", 10, "Test duration in seconds")

	protocol = flag.String("p", "HTTP", "[HTTP, HTTPS]")
	method   = flag.String("m", "GET", "Request Method Type. For Http(s):[GET, POST, PUT, DELETE, UPDATE, PATCH]")
	payload  = flag.String("b", "", "Payload of the network packet")
	headers  header

	target  = flag.String("t", "", "Target URL")
	timeout = flag.Int("T", 10, "Request timeout in seconds")

	//TODO: read from json file with whole parameters. config.json
	// scenario = flag.String("s", "", "Test scenario content in json format. Ex: [{url: 'sample.com', timeout: 10}, {url: 'sample.com/t', timeout: 12}]")

	proxy  = flag.String("P", "", "Proxy address as host:port")
	output = flag.String("o", "stdout", "Output destination")
)

func main() {
	flag.Var(&headers, "h", "Request Headers. Ex: -H 'Accept: text/html' -H 'Content-Type: application/xml'")
	flag.Parse()

	if *target == "" {
		exitWithMsg("Please provide the target url")
	}

	s := createScenario()
	p := createProxy()

	h := createHammer(s, p)
	if err := h.Validate(); err != nil {
		exitWithMsg(err.Error())
	}

	run(h)
}

func run(h types.Hammer) {
	ctx, cancel := context.WithCancel(context.Background())

	engine := core.CreateEngine(ctx, h)
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
	var s types.Scenario
	if target != nil {
		s = types.Scenario{
			Scenario: []types.ScenarioItem{
				{
					ID:       1,
					Protocol: strings.ToUpper(*protocol),
					Method:   strings.ToUpper(*method),
					Headers:  parseHeaders(headers),
					Payload:  *payload,
					URL:      *target,
					Timeout:  *timeout,
				},
			},
		}
	} else {
		exitWithMsg("Target is not provided.")
	}
	return s
}

func exitWithMsg(msg string) {
	if msg != "" {
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
