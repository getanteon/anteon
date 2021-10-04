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
	"flag"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/types"
)

func TestMain(m *testing.M) {
	// Mock run function to prevent engine starting
	run = func(h types.Hammer) {
		return
	}
	os.Exit(m.Run())
}

func resetFlags() {
	*reqCount = types.DefaultReqCount
	*loadType = types.DefaultLoadType
	*duration = types.DefaultDuration

	*protocol = types.DefaultProtocol
	*method = types.DefaultMethod
	*payload = ""
	*auth = ""
	headers = header{}

	*target = ""
	*timeout = types.DefaultTimeout

	*proxyFlag = ""
	*output = types.DefaultOutputType

	*configPath = ""
}

func TestDefaultFlagValues(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "-t=example.com"}

	flag.Parse()

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
	if *proxyFlag != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *proxyFlag)
	}
	if *output != types.DefaultOutputType {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultOutputType, *output)
	}
	if *configPath != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *configPath)
	}
}

func TestCreateHammer(t *testing.T) {
	tests := []struct {
		name      string
		args      string
		fromFlags bool
		fromFile  bool
	}{
		{"Flag", "-t=dummy.com -config=", true, false},
		{"File", "-config=dummy.json -t=", false, true},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Arrange
			resetFlags()
			oldArgs := os.Args
			oldFileFunc := createHammerFromConfigFile
			oldFlagFunc := createHammerFromFlags
			defer func() {
				os.Args = oldArgs
				createHammerFromConfigFile = oldFileFunc
				createHammerFromFlags = oldFlagFunc
			}()

			fromFileCalled := false
			fromFlagsCalled := false
			createHammerFromConfigFile = func() (h types.Hammer, err error) {
				fromFileCalled = true
				return
			}
			createHammerFromFlags = func() (h types.Hammer, err error) {
				fromFlagsCalled = true
				return
			}

			// Act
			os.Args = []string{"cmd", test.args}
			flag.Parse()
			createHammer()

			// Assert
			if fromFileCalled != test.fromFile {
				t.Errorf("createHammerFromConfigFileCalled expected %v found %v", test.fromFile, fromFileCalled)
			}
			if fromFlagsCalled != test.fromFlags {
				t.Errorf("createHammerFromFlagsCalled expected %v found %v", test.fromFlags, fromFlagsCalled)
			}
		}

		t.Run(test.name, tf)
	}
}

func TestCreateScenario(t *testing.T) {
	url := "https://test.com"
	valid := types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: types.DefaultProtocol,
				Method:   types.DefaultMethod,
				URL:      url,
				Timeout:  types.DefaultDuration,
			},
		},
	}
	validWithAuth := types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: types.DefaultProtocol,
				Method:   types.DefaultMethod,
				URL:      url,
				Timeout:  types.DefaultDuration,
				Auth: types.Auth{
					Type:     types.AuthHttpBasic,
					Username: "testuser",
					Password: "pass",
				},
			},
		},
	}

	tests := []struct {
		name      string
		args      []string
		shouldErr bool
		expected  types.Scenario
	}{
		{"InvalidAuth", []string{"-t=https://test.com", "-a=no_pass_included"}, true, types.Scenario{}},
		{"InvalidTarget", []string{"-t=asds.x.x.x"}, true, types.Scenario{}},
		{"Valid", []string{"-t=https://test.com"}, false, valid},
		{"ValidWithAuth", []string{"-t=https://test.com", "-a=testuser:pass"}, false, validWithAuth},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Arrange
			resetFlags()
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
			}()

			os.Args = []string{"cmd"}
			for _, a := range test.args {
				os.Args = append(os.Args, a)
			}

			// Act
			flag.Parse()
			s, err := createScenario()

			// Assert
			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Errored: %v", err)
				}
				if reflect.DeepEqual(test.expected, s) {
					t.Errorf("Expected %v, Found %v", test.expected, s)
				}
			}

		}

		t.Run(test.name, tf)
	}
}

func TestCreateProxy(t *testing.T) {
	addr, _ := url.Parse("http://127.0.0.1:80")
	withAddr := proxy.Proxy{
		Strategy: proxy.ProxyTypeSingle,
		Addr:     addr,
	}
	withoutAddr := proxy.Proxy{
		Strategy: proxy.ProxyTypeSingle,
		Addr:     nil,
	}

	tests := []struct {
		name      string
		args      []string
		shouldErr bool
		expected  proxy.Proxy
	}{
		{"InvalidProxy", []string{"-t=https://test.com", "-P=127.0.0.1:09"}, true, proxy.Proxy{}},
		{"ValidWithAddr", []string{"-t=https://test.com", "-P=http://127.0.0.1:80"}, false, withAddr},
		{"ValidWithoutAddr", []string{"-t=https://test.com"}, false, withoutAddr},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Arrange
			resetFlags()
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
			}()

			os.Args = []string{"cmd"}
			for _, a := range test.args {
				os.Args = append(os.Args, a)
			}

			// Act
			flag.Parse()
			p, err := createProxy()

			// Assert
			t.Log(test.args)
			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Errored: %v", err)
				}
				if test.expected.Strategy != p.Strategy {
					t.Errorf("Expected Strategy %v, Found %v", test.expected.Strategy, p.Strategy)
				}
				if (test.expected.Addr != nil && *test.expected.Addr != *p.Addr) || (test.expected.Addr == nil && p.Addr != nil) {
					t.Errorf("Expected Addr %v, Found %v", test.expected.Addr, p.Addr)
				}
			}

		}

		t.Run(test.name, tf)
	}
}

func TestParseHeaders(t *testing.T) {
	validSingleHeader := map[string]string{"header": "value"}
	validMultiHeader := map[string]string{"header-1": "value-1", "header-2": "value-2"}

	invalidHeader := header{}
	invalidHeader.Set("invalid|header?: value-1")

	tests := []struct {
		name      string
		args      header
		shouldErr bool
		expected  map[string]string
	}{
		{"ValidSingleHeder", []string{"header: value"}, false, validSingleHeader},
		{"ValidMultiHeader", []string{"header-1: value-1", "header-2: value-2"}, false, validMultiHeader},
		{"InvalidHeader", []string{"-t=https://test.com"}, true, map[string]string{}},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			headers := header{}
			for _, h := range test.args {
				headers.Set(h)
			}

			// Arrange
			h, err := parseHeaders(headers)

			// Assert
			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Errored: %v", err)
				}
				if !reflect.DeepEqual(test.expected, h) {
					t.Errorf("Expected  %#v, Found %#v", test.expected, h)
				}
			}

		}

		t.Run(test.name, tf)
	}
}

func TestRun(t *testing.T) {
	// Arrange
	resetFlags()
	runCalled := false
	run = func(h types.Hammer) {
		runCalled = true
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Act
	os.Args = []string{"cmd", "-t=test.com"}
	main()

	// Assert
	if !runCalled {
		t.Errorf("Run should be called")
	}
}

func TestTargetEmpty(t *testing.T) {
	// Below cmd code triggers this block
	if os.Getenv("TARGET_EMPTY") == "1" {
		resetFlags()
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"cmd", "asd"}
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

func TestTargetInvalidHammer(t *testing.T) {
	// Below cmd code triggers this block
	if os.Getenv("TARGET_EMPTY") == "1" {
		resetFlags()
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"cmd", "-t=dummy.com -l invalidLoadType"}
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

func TestVersion(t *testing.T) {
	// Below cmd code triggers this block
	if os.Getenv("VERSION") == "1" {
		resetFlags()
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"cmd", "-version"}
		main()
		return
	}

	// Since we reexecute the test here, this test case doesnt' increment the coverage.
	cmd := exec.Command(os.Args[0], "-test.run=TestVersion")
	cmd.Env = append(os.Environ(), "VERSION=1")
	err := cmd.Run()

	if err == nil {
		return
	}

	t.Errorf("TestVersion should not be failed")
}

func Test_versionTemplate(t *testing.T) {
	GitVersion = "v0.0.2"
	GitCommit = "akjsghsajghas"
	BuildDate = "2021-10-03T15:16:52Z"
	tests := []struct {
		name string
		want string
	}{
		{name: "version", want: "Version:        v0.0.2\nGit commit:     akjsghsajghas\nBuilt           2021-10-03T15:16:52Z\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionTemplate(); !strings.Contains(got, tt.want) {
				t.Errorf("versionTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}
