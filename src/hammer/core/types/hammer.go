package types

import (
	"fmt"

	"ddosify.com/hammer/core/util"
)

const (
	LoadTypeLinear   = "linear"
	LoadTypeCapacity = "capacity"
	LoadTypeStress   = "stress"
	LoadTypeSoak     = "soak"

	OutputTypeStdout    = "stdout"
	OutputTypeTimescale = "timescale"
)

var loadTypes = [...]string{LoadTypeLinear, LoadTypeCapacity, LoadTypeStress, LoadTypeSoak}
var supportedOutputs = [...]string{OutputTypeStdout, OutputTypeTimescale}

type Hammer struct {
	// TODO: Do we need this?
	// Number of concurrency
	Concurrency int

	// TODO: Do we need this?
	// Total CPU count to be used by Hammer.
	CPUCount int

	// Total request count
	TotalReqCount int

	// Type of the load.
	LoadType string

	// Total Duration of the test in seconds.
	TestDuration int

	// Duration (in second) - Request count map. Example: {10: 1500, 50: 400, ...}
	TimeReqCountMap map[int]int

	// Test Scenario
	Scenario Scenario

	// Proxy/Proxies to use
	Proxy Proxy

	// Network Packet parameters
	Packet Packet

	// Destination of the results data.
	ReportDestination string

}

func (h *Hammer) Validate() error {
	if err := h.Proxy.validate(); err != nil {
		return err
	}

	if err := h.Packet.validate(); err != nil {
		return err
	}

	if !util.StringInSlice(h.ReportDestination, supportedOutputs[:]) {
		return fmt.Errorf("Unsupported Report Destination: %s", h.ReportDestination)
	}

	if h.LoadType != "" && !util.StringInSlice(h.LoadType, loadTypes[:]) {
		return fmt.Errorf("Unsupported LoadType: %s", h.LoadType)
	}

	return nil
}
