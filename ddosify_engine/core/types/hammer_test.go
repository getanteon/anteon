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
	"errors"
	"testing"

	"go.ddosify.com/ddosify/core/proxy"
)

func newDummyHammer() Hammer {
	return Hammer{
		Proxy:             proxy.Proxy{Strategy: proxy.ProxyTypeSingle},
		ReportDestination: DefaultOutputType,
		Scenario: Scenario{
			Steps: []ScenarioStep{
				{
					ID:     1,
					Method: "GET",
					URL:    "http://127.0.0.1",
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
		h := newDummyHammer()
		h.Scenario.Steps[0].Auth = Auth{
			Type:     v,
			Username: "test",
			Password: "123",
		}

		if err := h.Validate(); err != nil {
			t.Errorf("TestHammerValidAuth errored: %v", err)
		}
	}

}

func TestHammerInValidAuth(t *testing.T) {
	h := newDummyHammer()
	h.Scenario.Steps[0].Auth = Auth{
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
	for _, m := range supportedProtocolMethods {
		h := newDummyHammer()
		h.Scenario = Scenario{
			Steps: []ScenarioStep{
				{
					ID:     1,
					Method: m,
					URL:    "https://127.0.0.1",
				},
			},
		}

		if err := h.Validate(); err != nil {
			t.Errorf("TestHammerValidScenario single scenario errored: %v", err)
		}
	}

	// Multiple Scenario

	for _, m := range supportedProtocolMethods {
		h := newDummyHammer()
		h.Scenario = Scenario{
			Steps: []ScenarioStep{
				{
					ID:     1,
					Method: m,
					URL:    "https://127.0.0.1",
				}, {
					ID:     2,
					URL:    "https://127.0.0.1",
					Method: m,
				},
			},
		}

		if err := h.Validate(); err != nil {
			t.Errorf("TestHammerValidScenario multi scenario errored: %v", err)
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

func TestHammerInvalidScenarioMethod(t *testing.T) {
	// Single Scenario
	h := newDummyHammer()
	h.Scenario = Scenario{
		Steps: []ScenarioStep{
			{
				ID:     1,
				Method: "GETT",
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenarioMethod errored")
	}

	// Multi Scenario
	h = newDummyHammer()
	h.Scenario = Scenario{
		Steps: []ScenarioStep{
			{
				ID:     1,
				Method: supportedProtocolMethods[1],
			},
			{
				ID:     1,
				Method: "GETT",
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInvalidScenarioMethod errored")
	}
}

func TestHammerEmptyScenarioStepID(t *testing.T) {
	// Single Scenario
	h := newDummyHammer()
	h.Scenario = Scenario{
		Steps: []ScenarioStep{
			{
				Method: supportedProtocolMethods[1],
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("1- TestHammerEmptyScenarioStepID should be errored")
	}

	// Multi Scenario
	h = newDummyHammer()
	h.Scenario = Scenario{
		Steps: []ScenarioStep{
			{
				ID:     1,
				Method: supportedProtocolMethods[1],
			},
			{
				Method: supportedProtocolMethods[1],
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("2- TestHammerEmptyScenarioStepID should be errored")
	}
}

func TestHammerDuplicateScenarioStepID(t *testing.T) {
	// Single Scenario
	h := newDummyHammer()
	h.Scenario = Scenario{
		Steps: []ScenarioStep{
			{
				ID:     1,
				Method: supportedProtocolMethods[1],
			},
			{
				ID:     2,
				Method: supportedProtocolMethods[1],
			},
			{
				ID:     2,
				Method: supportedProtocolMethods[1],
			},
		},
	}
	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerDuplicateScenarioStepID should be errored")
	}
}

func TestHammerStepSleep(t *testing.T) {
	t.Parallel()

	invalidSleeps := []string{
		"-300",
		"-300-500",
		"300s",
		"as",
		"100000", // More than maxSleep
	}
	validSleeps := []string{
		"300-500",
		"1000",
	}

	tests := []struct {
		name      string
		sleep     string
		shouldErr bool
	}{
		{"Invalid 1", invalidSleeps[0], true},
		{"Invalid 2", invalidSleeps[1], true},
		{"Invalid 3", invalidSleeps[2], true},
		{"Invalid 4", invalidSleeps[3], true},
		{"Invalid 5", invalidSleeps[4], true},
		{"ValidRange", validSleeps[0], false},
		{"ValidDuration", validSleeps[1], false},
	}

	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			h := newDummyHammer()
			h.Scenario = Scenario{
				Steps: []ScenarioStep{
					{
						ID:     1,
						URL:    "target.com",
						Method: supportedProtocolMethods[1],
						Sleep:  test.sleep,
					},
				},
			}

			err := h.Validate()

			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Error occurred %v", err)
				}
			}

		})
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

func TestHammerAccessingNotDefinedCsvEnvs(t *testing.T) {
	h := newDummyHammer()
	h.TestDataConf = make(map[string]CsvConf)

	h.TestDataConf["info"] = CsvConf{
		Path:          "",
		Delimiter:     "",
		SkipFirstLine: false,
		Vars: map[string]Tag{
			"0": {
				Tag:  "a",
				Type: "string",
			},
		},
		SkipEmptyLine: false,
		AllowQuote:    false,
		Order:         "",
	}

	h.Scenario.Steps = []ScenarioStep{
		{
			ID:   1,
			Name: "x",

			Method: "GET",
			Headers: map[string]string{
				"{{data.info.x}}": "X",
			},
			Payload: "",
			URL:     "https://ddosify.com",
		},
	}
	err := h.Validate()

	var environmentNotDefined EnvironmentNotDefinedError

	if !errors.As(err, &environmentNotDefined) {
		t.Errorf("Should be EnvironmentNotDefinedError")
	}
}
