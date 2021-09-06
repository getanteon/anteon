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

package types

import (
	"fmt"
	"net/http"

	"ddosify.com/hammer/core/util"
)

const (
	LoadTypeLinear      = "linear"
	LoadTypeIncremental = "incremental"
	LoadTypeWaved       = "waved"

	OutputTypeStdout    = "stdout"
	OutputTypeTimescale = "timescale"

	DdosifyUserAgent = "DdosifyHammer"

	// Default Values
	DefaultReqCount   = 100
	DefaultLoadType   = LoadTypeLinear
	DefaultDuration   = 10
	DefaultTimeout    = 5
	DefaultProtocol   = ProtocolHTTPS
	DefaultMethod     = http.MethodGet
	DefaultOutputType = OutputTypeStdout
)

var loadTypes = [...]string{LoadTypeLinear, LoadTypeIncremental, LoadTypeWaved}
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
