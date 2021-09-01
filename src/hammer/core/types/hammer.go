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

	// Destination of the results data.
	ReportDestination string

}

func (h *Hammer) Validate() error {
	if err := h.Proxy.validate(); err != nil {
		return err
	}

	if len(h.Scenario.Scenario) == 0 {
		return fmt.Errorf("scenario or target is empty")
	} else if err := h.Scenario.validate(); err != nil {
		return err
	}

	if !util.StringInSlice(h.ReportDestination, supportedOutputs[:]) {
		return fmt.Errorf("unsupported Report Destination: %s", h.ReportDestination)
	}

	if h.LoadType != "" && !util.StringInSlice(h.LoadType, loadTypes[:]) {
		return fmt.Errorf("unsupported LoadType: %s", h.LoadType)
	}

	return nil
}
