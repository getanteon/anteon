package types_test

import (
	"net/http"
	"testing"

	"ddosify.com/hammer/core/types"
)

// TODO: Auth struct tests

var supportedProtocols = [...]string{"HTTP", "HTTPS"}
var supportedProtocolMethods = map[string][]string{
	"HTTP": {
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions,
	},
	"HTTPS": {
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions,
	},
}
var availableProxyStrategies = [...]string{"single"}
var supportedDestinations = [...]string{"stdout", "timescale"}

func newDummyHammer() types.Hammer {
	return types.Hammer{
		Proxy:             types.Proxy{Strategy: "single"},
		ReportDestination: "stdout",
		Scenario: types.Scenario{
			Scenario: []types.ScenarioItem{
				{
					ID:       1,
					Protocol: "HTTP",
					Method:   "GET",
					URL:      "http://127.0.0.1",
				},
			},
		},
	}
}

func TestHammerValidAttackType(t *testing.T) {
	var loadTypes = [...]string{"linear", "incremental", "waved"}

	for _, l := range loadTypes {
		h := newDummyHammer()
		h.LoadType = l

		if err := h.Validate(); err != nil {
			t.Errorf("TestHammerValidAttackType errored: %v", err)
		}
	}
}

func TestHammerInValidAttackType(t *testing.T) {
	h := newDummyHammer()
	h.LoadType = "strees"

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInValidAttackType errored")
	}
}

func TestHammerValidReportDestination(t *testing.T) {

	for _, rd := range supportedDestinations {
		h := newDummyHammer()
		h.ReportDestination = rd

		if err := h.Validate(); err != nil {
			t.Errorf("TestHammerValidReportDestination errored: %v", err)
		}
	}
}

func TestHammerInValidReportDestination(t *testing.T) {
	h := newDummyHammer()
	h.ReportDestination = "output_dummy"

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInValidReportDestination errored")
	}
}

func TestHammerValidProxy(t *testing.T) {
	var availableProxyStrategies = [...]string{"single"}

	for _, ps := range availableProxyStrategies {
		h := newDummyHammer()
		h.Proxy = types.Proxy{Strategy: ps}

		if err := h.Validate(); err != nil {
			t.Errorf("TestHammerValidProxy errored: %v", err)
		}
	}
}

func TestHammerInValidProxy(t *testing.T) {
	h := newDummyHammer()
	h.Proxy = types.Proxy{Strategy: "dummy_strategy"}

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInValidProxy errored")
	}
}

func TestHammerValidScenario(t *testing.T) {
	// Single Scenario
	for _, p := range supportedProtocols {
		for _, m := range supportedProtocolMethods[p] {
			h := newDummyHammer()
			h.Scenario = types.Scenario{
				Scenario: []types.ScenarioItem{
					{
						ID:       1,
						Protocol: p,
						Method:   m,
					},
				},
			}

			if err := h.Validate(); err != nil {
				t.Errorf("TestHammerValidScenario single scenario errored: %v", err)
			}
		}
	}

	// Multiple Scenario
	for _, p := range supportedProtocols {
		for _, m := range supportedProtocolMethods[p] {
			h := newDummyHammer()
			h.Scenario = types.Scenario{
				Scenario: []types.ScenarioItem{
					{
						ID:       1,
						Protocol: p,
						Method:   m,
					}, {
						ID:       2,
						Protocol: p,
						Method:   m,
					},
				},
			}

			if err := h.Validate(); err != nil {
				t.Errorf("TestHammerValidScenario multi scenario errored: %v", err)
			}
		}
	}
}

func TestHammerInvalidScenarioProtocol(t *testing.T) {
	// Single Scenario
	h := newDummyHammer()
	h.Scenario = types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: "HTTPP",
				Method:   supportedProtocolMethods["HTTP"][1],
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenario errored")
	}

	// Multi Scenario
	h = newDummyHammer()
	h.Scenario = types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: supportedProtocols[0],
				Method:   supportedProtocolMethods["HTTP"][1],
			},
			{
				ID:       1,
				Protocol: "HTTPP",
				Method:   supportedProtocolMethods["HTTP"][1],
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenario errored")
	}
}

func TestHammerInvalidScenarioMethod(t *testing.T) {
	// Single Scenario
	h := newDummyHammer()
	h.Scenario = types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: supportedProtocols[0],
				Method:   "GETT",
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenarioMethod errored")
	}

	// Multi Scenario
	h = newDummyHammer()
	h.Scenario = types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: supportedProtocols[0],
				Method:   supportedProtocolMethods["HTTP"][1],
			},
			{
				ID:       1,
				Protocol: supportedProtocols[0],
				Method:   "GETT",
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenarioMethod errored")
	}
}

func TestHammerEmptyScenarioItemID(t *testing.T) {
	// Single Scenario
	h := newDummyHammer()
	h.Scenario = types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				Protocol: supportedProtocols[0],
				Method:   supportedProtocolMethods["HTTP"][1],
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenarioItemID errored")
	}

	// Multi Scenario
	h = newDummyHammer()
	h.Scenario = types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: supportedProtocols[0],
				Method:   supportedProtocolMethods["HTTP"][1],
			},
			{
				Protocol: supportedProtocols[0],
				Method:   supportedProtocolMethods["HTTP"][1],
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenarioItemID errored")
	}
}
