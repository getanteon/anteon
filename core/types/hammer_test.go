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
	"testing"

	"go.ddosify.com/ddosify/core/proxy"
)

func newDummyHammer() Hammer {
	return Hammer{
		Proxy:             proxy.Proxy{Strategy: proxy.ProxyTypeSingle},
		ReportDestination: DefaultOutputType,
		Scenario: Scenario{
			Scenario: []ScenarioItem{
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

func TestHammerValidAuth(t *testing.T) {
	for _, v := range supportedAuthentications {
		for _, a := range v {
			h := newDummyHammer()
			h.Scenario.Scenario[0].Auth = Auth{
				Type:     a,
				Username: "test",
				Password: "123",
			}

			if err := h.Validate(); err != nil {
				t.Errorf("TestHammerValidAuth errored: %v", err)
			}
		}
	}
}

func TestHammerInValidAuth(t *testing.T) {
	h := newDummyHammer()
	h.Scenario.Scenario[0].Auth = Auth{
		Type:     "invalidAuthType",
		Username: "test",
		Password: "123",
	}

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInValidReportDestination errored")
	}
}

func TestHammerValidScenario(t *testing.T) {
	// Single Scenario
	for _, p := range SupportedProtocols {
		for _, m := range supportedProtocolMethods[p] {
			h := newDummyHammer()
			h.Scenario = Scenario{
				Scenario: []ScenarioItem{
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
	for _, p := range SupportedProtocols {
		for _, m := range supportedProtocolMethods[p] {
			h := newDummyHammer()
			h.Scenario = Scenario{
				Scenario: []ScenarioItem{
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

func TestHammerEmptyScenario(t *testing.T) {
	h := newDummyHammer()
	h.Scenario = Scenario{}

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerEmptyScenario errored")
	}
}
func TestHammerInvalidScenarioProtocol(t *testing.T) {
	// Single Scenario
	h := newDummyHammer()
	h.Scenario = Scenario{
		Scenario: []ScenarioItem{
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
	h.Scenario = Scenario{
		Scenario: []ScenarioItem{
			{
				ID:       1,
				Protocol: SupportedProtocols[0],
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
	h.Scenario = Scenario{
		Scenario: []ScenarioItem{
			{
				ID:       1,
				Protocol: SupportedProtocols[0],
				Method:   "GETT",
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenarioMethod errored")
	}

	// Multi Scenario
	h = newDummyHammer()
	h.Scenario = Scenario{
		Scenario: []ScenarioItem{
			{
				ID:       1,
				Protocol: SupportedProtocols[0],
				Method:   supportedProtocolMethods["HTTP"][1],
			},
			{
				ID:       1,
				Protocol: SupportedProtocols[0],
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
	h.Scenario = Scenario{
		Scenario: []ScenarioItem{
			{
				Protocol: SupportedProtocols[0],
				Method:   supportedProtocolMethods["HTTP"][1],
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenarioItemID errored")
	}

	// Multi Scenario
	h = newDummyHammer()
	h.Scenario = Scenario{
		Scenario: []ScenarioItem{
			{
				ID:       1,
				Protocol: SupportedProtocols[0],
				Method:   supportedProtocolMethods["HTTP"][1],
			},
			{
				Protocol: SupportedProtocols[0],
				Method:   supportedProtocolMethods["HTTP"][1],
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenarioItemID errored")
	}
}

func TestHammerInvalidManualLoadDuration(t *testing.T) {
	// Duration = 0
	h := newDummyHammer()
	h.TimeRunCountMap = TimeRunCount{
		{Duration: 10, Count: 10},
		{Duration: 0, Count: 10},
	}

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidManualLoadDuration errored")
	}

	// Duration is negatie
	h = newDummyHammer()
	h.TimeRunCountMap = TimeRunCount{
		{Duration: 10, Count: 10},
		{Duration: -1, Count: 10},
	}

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidManualLoadDuration errored")
	}
}
