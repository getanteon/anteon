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
	"time"

	"go.ddosify.com/ddosify/v2/core/proxy"
	"go.ddosify.com/ddosify/v2/core/util"
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

	// Seeds the poisson arrivals and the random data rows when the config gives no seed
	DefaultSeed int64 = 1
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
	Name     string
	Warmup   bool
	// Concurrency makes the stage closed-loop: Count iterations, at most Concurrency in flight, a new
	// one starting as one ends. Duration is then an optional cap rather than the stage's length.
	Concurrency int
}

// Totals is the iteration count and the duration in seconds the stages add up to. A closed-loop
// stage adds its cap, 0 without one: how long it takes is known only when it ends.
func (t TimeRunCount) Totals() (iterations, duration int) {
	for _, s := range t {
		iterations += s.Count
		duration += s.Duration
	}
	return
}

// ClosedLoop tells whether the stages are closed-loop, by concurrency, rather than arrival rates.
func (t TimeRunCount) ClosedLoop() bool {
	for _, s := range t {
		if s.Concurrency > 0 {
			return true
		}
	}
	return false
}

// MaxConcurrency is the largest concurrency of the stages.
func (t TimeRunCount) MaxConcurrency() int {
	m := 0
	for _, s := range t {
		m = max(m, s.Concurrency)
	}
	return m
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

	// How the HTML report reads the run as capacity: GPUs and SLO tiers
	Capacity *Capacity

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

	// How iterations are spread inside a second or a manual_load stage: LoadSpreadFront (default), LoadSpreadEven or LoadSpreadPoisson
	LoadSpread string

	// Seeds the poisson arrivals and the random data rows, so a run can be repeated
	Seed int64

	// Once a step's outcome is not ok, the iteration sends none of its remaining steps
	StopOnFailure bool

	// One line per sent request in the stdout-json output
	RequestLog bool

	// Thresholds an iteration's streams have to meet to count as goodput
	SLO *SLO

	// Skip TLS certificate verification on every step
	Insecure bool

	// After an interrupt, how long iterations in flight may finish before they are cut
	Drain time.Duration
}

// Validate validates attack metadata and executes the validation methods of the services.
func (h *Hammer) Validate() error {
	if len(h.Scenario.Steps) == 0 {
		return fmt.Errorf("scenario or target is empty")
	}

	h.Scenario.CsvVars = getCsvEnvs(h.TestDataConf)

	if h.Insecure {
		for i := range h.Scenario.Steps {
			h.Scenario.Steps[i].Insecure = true
		}
	}

	if err := h.Scenario.validate(); err != nil {
		return err
	}

	if h.LoadType != "" && !util.StringInSlice(h.LoadType, loadTypes[:]) {
		return fmt.Errorf("unsupported LoadType: %s", h.LoadType)
	}
	if h.EngineMode != "" && !util.StringInSlice(h.EngineMode, engineModes[:]) {
		return fmt.Errorf("unsupported EngineMode: %s", h.EngineMode)
	}

	if h.TimeRunCountMap.ClosedLoop() {
		for i, t := range h.TimeRunCountMap {
			switch {
			case t.Concurrency < 1:
				return fmt.Errorf("manual_load stage %d has no concurrency: stages with a concurrency and stages with an arrival rate can not be mixed", i+1)
			case t.Count < 1:
				return fmt.Errorf("manual_load stage %d: a stage with a concurrency needs a count of 1 or more", i+1)
			case t.Duration < 0:
				return fmt.Errorf("manual_load stage %d: duration can not be negative", i+1)
			}
		}
		if h.LoadSpread == LoadSpreadPoisson {
			return fmt.Errorf("load_spread %s needs arrival rates; stages with a concurrency start a new iteration as one ends", LoadSpreadPoisson)
		}
	} else if len(h.TimeRunCountMap) > 0 {
		for _, t := range h.TimeRunCountMap {
			if t.Duration < 1 {
				return fmt.Errorf("duration in manual_load should be greater than 0")
			}
		}
	}

	if h.LoadSpread != "" && h.LoadSpread != LoadSpreadFront && h.LoadSpread != LoadSpreadEven && h.LoadSpread != LoadSpreadPoisson {
		return fmt.Errorf("unsupported load_spread: %s, supported: %s, %s, %s", h.LoadSpread, LoadSpreadFront, LoadSpreadEven, LoadSpreadPoisson)
	}
	if h.LoadSpread == LoadSpreadPoisson && len(h.TimeRunCountMap) == 0 && h.LoadType != "" && h.LoadType != LoadTypeLinear {
		return fmt.Errorf("load_spread %s needs manual_load or load_type %s", LoadSpreadPoisson, LoadTypeLinear)
	}

	if h.SLO != nil {
		if err := h.SLO.validate(); err != nil {
			return err
		}
	}
	if err := h.Capacity.validate(); err != nil {
		return err
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
