package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"ddosify.com/hammer/core"
	"ddosify.com/hammer/core/types"
)

// We might consider to use Viper: https://github.com/spf13/viper
var (
	concurrency = flag.Int("c", 50, "Concurrency count")
	cpuCount    = flag.Int("cpus", runtime.GOMAXPROCS(-1), "Number of CPU to be used")
	reqCount    = flag.Int("n", 1000, "Total request count")
	loadType    = flag.String("l", "linear", "Type of the load test [linear, capacity, stress, soak")
	duration    = flag.Int("d", 10, "Test duration in seconds")

	protocol = flag.String("p", "HTTP", "[HTTP, HTTPS]")
	method   = flag.String("m", "GET", "Request Method Type. For Http(s):[GET, POST, PUT, DELETE, UPDATE, PATCH]")
	payload  = flag.String("b", "", "Payload of the network packet")
	headers  header

	target   = flag.String("t", "", "Target URL")
	timeout  = flag.Int("T", 20, "Request timeout in seconds")
	scenario = flag.String("s", "", "Test scenario content in json format. Ex: [{url: 'sample.com', timeout: 10}, {url: 'sample.com/t', timeout: 12}]")

	proxy  = flag.String("P", "", "Proxy address as host:port")
	output = flag.String("o", "stdout", "Output destination")
)

func main() {
	flag.Var(&headers, "h", "Request Headers. Ex: -H 'Accept: text/html' -H 'Content-Type: application/xml'")
	flag.Parse()

	proxyURL, err := url.Parse(*proxy)
	if err != nil {
		exitWithMsg(err.Error())
	}
	pc := types.Proxy{
		Strategy: "single",
		Addr:     proxyURL,
	}

	//TODO: change scenario input flow. Suan bu kod calismiyor...
	if *target == "" && *scenario == "" {
		exitWithMsg("Provide target url (-t) or scenario (-s) option")
	}
	if target != nil {
		*scenario = "{'scenario': [{'url':'" + *target + "', 'timeout':'" + strconv.Itoa(*timeout) + "'}]}"
	}
	var sc types.Scenario
	json.Unmarshal([]byte(*scenario), &sc)

	packet := types.Packet{
		Protocol: strings.ToUpper(*protocol),
		Method:   strings.ToUpper(*method),
		Payload:  *payload,
	}

	h := types.Hammer{
		Concurrency:       *concurrency,
		CPUCount:          *cpuCount,
		TotalReqCount:     *reqCount,
		LoadType:          *loadType,
		TestDuration:      *duration,
		Scenario:          sc,
		Proxy:             pc,
		Packet:            packet,
		ReportDestination: *output,
	}
	if err = h.Validate(); err != nil {
		exitWithMsg(err.Error())
	}

	hammer, err := core.CreateEngine(h)
	if err != nil {
		exitWithMsg(err.Error())
	}
	hammer.Start()
	time.Sleep(time.Second * 3)
	hammer.Stop()
}

func exitWithMsg(msg string) {
	if msg != "" {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}

type header []string

func (h *header) String() string {
	return fmt.Sprintf("%s - %d", *h, len(*h))
}

func (h *header) Set(value string) error {
	*h = append(*h, value)
	return nil
}
