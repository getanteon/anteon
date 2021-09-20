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

package main

import (
	"os"
	"os/exec"
	"reflect"
	"testing"

	"ddosify.com/hammer/core/types"
)

func TestMain(m *testing.M) {
	// Mock run function to prevent engine starting
	run = func(h types.Hammer) {
		return
	}
	os.Exit(m.Run())
}

func TestDefaultFlagValues(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "-t=example.com"}

	parseFlags()

	if *reqCount != types.DefaultReqCount {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultReqCount, *reqCount)
	}
	if *loadType != types.DefaultLoadType {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultLoadType, *loadType)
	}
	if *duration != types.DefaultDuration {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultDuration, *duration)
	}
	if *protocol != types.DefaultProtocol {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultProtocol, *protocol)
	}
	if *method != types.DefaultMethod {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultMethod, *method)
	}
	if *payload != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *payload)
	}
	if *auth != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *auth)
	}
	if reflect.DeepEqual(headers, []string{}) {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", []string{}, headers)
	}
	if *timeout != types.DefaultTimeout {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultTimeout, *timeout)
	}
	if *proxy != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *proxy)
	}
	if *output != types.DefaultOutputType {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultOutputType, *output)
	}
	if *configPath != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *configPath)
	}
}

func TestTargetEmpty(t *testing.T) {
	// Below cmd codes triggers this block
	if os.Getenv("TARGET_EMPTY") == "1" {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"cmd", ""}
		main()
		return
	}

	// Since we reexecute the test here, this test case doesnt' increment the coverage.
	cmd := exec.Command(os.Args[0], "-test.run=TestTargetEmpty")
	cmd.Env = append(os.Environ(), "TARGET_EMPTY=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}

	t.Errorf("TestTargetEmpty should be failed with exit code 1 found: %v", err)
}

func TestCreateHammerFromConfigFile(t *testing.T) {
	expectedHammer := types.Hammer{
		TotalReqCount:     20,
		LoadType:          types.LoadTypeLinear,
		TestDuration:      5,
		ReportDestination: types.OutputTypeTimescale,
		Proxy:             types.Proxy{Strategy: "single"},
		Scenario: types.Scenario{
			Scenario: []types.ScenarioItem{
				{
					ID:       1,
					Protocol: "HTTPS",
					Method:   "GET",
					Timeout:  5,
					URL:      "https://test.com",
				},
			},
		},
	}

	var createdHammer types.Hammer
	run = func(h types.Hammer) {
		createdHammer = h
		return
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "-config=main_testdata/config.json"}
	main()

	if !reflect.DeepEqual(expectedHammer, createdHammer) {
		t.Errorf("TestCreateHammerFromConfigFile failed, expected %v, found %v", expectedHammer, createdHammer)
	}
}
