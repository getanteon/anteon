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

	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/util"
)

// Constants for Hammer field values
const (
	// Constants of the Load Types
	LoadTypeLinear      = "linear"
	LoadTypeIncremental = "incremental"
	LoadTypeWaved       = "waved"

	// Default Values
	DefaultReqCount   = 100
	DefaultLoadType   = LoadTypeLinear
	DefaultDuration   = 10
	DefaultTimeout    = 5
	DefaultProtocol   = ProtocolHTTPS
	DefaultMethod     = http.MethodGet
	DefaultOutputType = "stdout" // TODO: get this value from report.OutputTypeStdout when import cycle resolved.
)

var loadTypes = [...]string{LoadTypeLinear, LoadTypeIncremental, LoadTypeWaved}

// TimeRunCount is the data structure to store manual load type data.
type TimeRunCount []struct {
	Duration int
	Count    int
}

// Hammer is like a lighter for the engine.
// It includes attack metadata and all necessary data to initialize the internal services in the engine.
type Hammer struct {
	// Total request count
	TotalReqCount int

	// Type of the load.
	LoadType string

	// Total Duration of the test in seconds.
	TestDuration int

	// Duration (in second) - Request count map. Example: {10: 1500, 50: 400, ...}
	TimeRunCountMap TimeRunCount

	// Test Scenario
	Scenario Scenario

	// Proxy/Proxies to use
	Proxy proxy.Proxy

	// Destination of the results data.
	ReportDestination string

	// Report percentiles
	ReportPercentiles bool

	// Dynamic field for extra parameters.
	Others map[string]interface{}
}

// Validate validates attack metadata and executes the validation methods of the services.
func (h *Hammer) Validate() error {
	if len(h.Scenario.Scenario) == 0 {
		return fmt.Errorf("scenario or target is empty")
	} else if err := h.Scenario.validate(); err != nil {
		return err
	}

	if h.LoadType != "" && !util.StringInSlice(h.LoadType, loadTypes[:]) {
		return fmt.Errorf("unsupported LoadType: %s", h.LoadType)
	}

	if len(h.TimeRunCountMap) > 0 {
		for _, t := range h.TimeRunCountMap {
			if t.Duration < 1 {
				return fmt.Errorf("duration in manual_load should be greater than 0")
			}
		}
	}

	return nil
}
