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
	"strings"

	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/util"
)

// Constants for Hammer field values
const (
	// Constants of the Load Types
	LoadTypeLinear      = "linear"
	LoadTypeIncremental = "incremental"
	LoadTypeWaved       = "waved"

	// EngineModes
	EngineModeDistinctUser = "distinct-user"
	EngineModeRepeatedUser = "repeated-user"
	EngineModeDdosify      = "ddosify"

	// Default Values
	DefaultIterCount     = 100
	DefaultLoadType      = LoadTypeLinear
	DefaultDuration      = 10
	DefaultTimeout       = 5
	DefaultMethod        = http.MethodGet
	DefaultOutputType    = "stdout" // TODO: get this value from report.OutputTypeStdout when import cycle resolved.
	DefaultSamplingCount = 3
	DefaultSingleMode    = true
)

var loadTypes = [...]string{LoadTypeLinear, LoadTypeIncremental, LoadTypeWaved}
var engineModes = [...]string{EngineModeDdosify, EngineModeDistinctUser, EngineModeRepeatedUser}

type TestAssertionOpt struct {
	Abort bool
	Delay int
}

// TimeRunCount is the data structure to store manual load type data.
type TimeRunCount []struct {
	Duration int
	Count    int
}

type Tag struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

type CsvConf struct {
	Path          string         `json:"path"`
	Delimiter     string         `json:"delimiter"`
	SkipFirstLine bool           `json:"skip_first_line"`
	Vars          map[string]Tag `json:"vars"` // "0":"name", "1":"city","2":"team"
	SkipEmptyLine bool           `json:"skip_empty_line"`
	AllowQuota    bool           `json:"allow_quota"`
	Order         string         `json:"order"`
}

// TimeRunCount is the data structure to store manual load type data.
type CustomCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  string `json:"expires"`
	MaxAge   int    `json:"max_age"`
	HttpOnly bool   `json:"http_only"`
	Secure   bool   `json:"secure"`
	Raw      string `json:"raw"`
}

// Hammer is like a lighter for the engine.
// It includes attack metadata and all necessary data to initialize the internal services in the engine.
type Hammer struct {
	// Total iteration count
	IterationCount int

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

	// Dynamic field for extra parameters.
	Others map[string]interface{}

	// Debug mode on/off
	Debug bool

	// Sampling rate
	SamplingRate int

	// Connection reuse
	EngineMode string

	// Test Data Config
	TestDataConf map[string]CsvConf

	// Custom Cookies
	Cookies []CustomCookie

	// Custom Cookies Enabled
	CookiesEnabled bool

	// Test-wide assertions
	Assertions map[string]TestAssertionOpt

	// Engine runs single
	SingleMode bool
}

// Validate validates attack metadata and executes the validation methods of the services.
func (h *Hammer) Validate() error {
	if len(h.Scenario.Steps) == 0 {
		return fmt.Errorf("scenario or target is empty")
	}

	h.Scenario.CsvVars = getCsvEnvs(h.TestDataConf)

	if err := h.Scenario.validate(); err != nil {
		return err
	}

	if h.LoadType != "" && !util.StringInSlice(h.LoadType, loadTypes[:]) {
		return fmt.Errorf("unsupported LoadType: %s", h.LoadType)
	}
	if h.EngineMode != "" && !util.StringInSlice(h.EngineMode, engineModes[:]) {
		return fmt.Errorf("unsupported EngineMode: %s", h.EngineMode)
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

func getCsvEnvs(testDataConf map[string]CsvConf) []string {
	csvVars := make([]string, 0)

	sb := strings.Builder{}
	for key, conf := range testDataConf {
		for _, tag := range conf.Vars {
			sb.WriteString("data.")
			sb.WriteString(key)
			sb.WriteString(".")
			sb.WriteString(tag.Tag)
			// data.info.name
			csvVars = append(csvVars, sb.String())
			sb.Reset()
		}
	}

	return csvVars
}
