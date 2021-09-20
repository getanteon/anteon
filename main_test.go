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

func resetFlags() {
	*reqCount = types.DefaultReqCount
	*loadType = types.DefaultLoadType
	*duration = types.DefaultDuration

	*protocol = types.DefaultProtocol
	*method = types.DefaultMethod
	*payload = ""
	*auth = ""
	headers = []string{}

	*target = ""
	*timeout = types.DefaultTimeout

	*proxy = ""
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

	for _, tc := range tests {
		test := tc
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

func TestTargetEmpty(t *testing.T) {
	// Below cmd code triggers this block
	if os.Getenv("TARGET_EMPTY") == "1" {
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
