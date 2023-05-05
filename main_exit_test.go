//go:build linux || darwin
// +build linux darwin

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
	"fmt"
	"os"
	"syscall"
	"testing"

	"go.ddosify.com/ddosify/core/types"
)

func TestExitStatusOnTestFail(t *testing.T) {
	index := os.Getenv("index")
	if index == "" { // parent
		// start a test in child proc, look for its exit status
		env := fmt.Sprintf("index=%d", 1)
		cPid, err := syscall.ForkExec(os.Args[0], os.Args, &syscall.ProcAttr{Files: []uintptr{0, 1, 2}, Env: []string{env}})
		if err != nil {
			panic(err.Error())
		}

		proc, err := os.FindProcess(cPid)
		if err != nil {
			panic(err.Error())
		}

		// expected child to fail with exit code 1
		pState, err := proc.Wait()
		if err != nil {
			panic(err.Error())
		}
		if pState.Success() {
			t.Fail()
		}
	} else {
		// run a failed engine
		*configPath = "config/config_testdata/config_test_assertion_fail.json"
		run = tempRun
		start()
		run = func(h types.Hammer) {}
	}
}
