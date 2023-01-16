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
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	gopsProc "github.com/shirou/gopsutil/v3/process"
	"golang.org/x/exp/constraints"
)

var table = []struct {
	input string
	// in percents
	cpuTimeThreshold float64
	maxMemThreshold  float32
	avgMemThreshold  float32
}{
	{
		input:            "config/config_testdata/benchmark/config_json.json",
		cpuTimeThreshold: 0.05,
		maxMemThreshold:  1,
		avgMemThreshold:  1,
	},
	{
		input:            "config/config_testdata/benchmark/config_correlation_load_1.json",
		cpuTimeThreshold: 0.350,
		maxMemThreshold:  1,
		avgMemThreshold:  1,
	},
	{
		input:            "config/config_testdata/benchmark/config_correlation_load_2.json",
		cpuTimeThreshold: 2.5,
		maxMemThreshold:  2,
		avgMemThreshold:  2,
	},
	{
		input:            "config/config_testdata/benchmark/config_correlation_load_3.json",
		cpuTimeThreshold: 15.5,
		maxMemThreshold:  13,
		avgMemThreshold:  8,
	},
	{
		input:            "config/config_testdata/benchmark/config_correlation_load_4.json",
		cpuTimeThreshold: 25,
		maxMemThreshold:  25,
		avgMemThreshold:  16,
	},
	{
		input:            "config/config_testdata/benchmark/config_correlation_load_5.json",
		cpuTimeThreshold: 70,
		maxMemThreshold:  70,
		avgMemThreshold:  45,
	},
}

func BenchmarkEngines(t *testing.B) {
	index := os.Getenv("index")
	if index == "" {
		// parent
		for i, _ := range table { // open a new process for each test config
			// start a child
			env := fmt.Sprintf("index=%d", i)
			cPid, err := syscall.ForkExec(os.Args[0], os.Args, &syscall.ProcAttr{Files: []uintptr{0, 1, 2}, Env: []string{env}})
			if err != nil {
				panic(err.Error())
			}

			proc, err := os.FindProcess(cPid)
			if err != nil {
				panic(err.Error())
			}

			proc.Wait()
			if err != nil {
				panic(err.Error())
			}
		}
	} else {
		// child proc
		i, _ := strconv.Atoi(index)
		v := table[i]
		t.Run(fmt.Sprintf("config_%s", v.input), func(t *testing.B) {
			var memPercents []float32
			var cpuStats []*cpu.TimesStat

			*configPath = v.input
			run = tempRun
			doneChan := make(chan struct{}, 1)
			go func() {
				ticker := time.NewTicker(time.Duration(100 * time.Millisecond))
				pid := os.Getpid()
				proc, _ := gopsProc.NewProcess(int32(pid))
				for {
					select {
					case <-ticker.C:
						cpuStat, _ := proc.Times()
						cpuStats = append(cpuStats, cpuStat)
						proc.CPUPercent()

						memPerc, _ := proc.MemoryPercent()
						memPercents = append(memPercents, memPerc)
					case <-doneChan:
						return
					}
				}
			}()
			start()
			doneChan <- struct{}{}

			lastCpuStat := cpuStats[len(cpuStats)-1]
			cpuTime := lastCpuStat.User + lastCpuStat.System
			fmt.Printf("cpuTime: %f / %f \n", cpuTime, v.cpuTimeThreshold)

			avgMem := sum(memPercents) / float32(len(memPercents))
			maxMem := max(memPercents)
			fmt.Printf("Avg mem: %f / %f \n", avgMem, v.avgMemThreshold)
			fmt.Printf("Max mem: %f / %f \n\n", maxMem, v.maxMemThreshold)

			if cpuTime > v.cpuTimeThreshold {
				t.Errorf("Cpu time %f, higher than cpuTimeThreshold %f", cpuTime, v.cpuTimeThreshold)
			}
			if avgMem > v.avgMemThreshold {
				t.Errorf("Avg mem %f, higher than avgMemThreshold %f", avgMem, v.avgMemThreshold)
			}
			if maxMem > v.maxMemThreshold {
				t.Errorf("Max mem %f, higher than maxMemThreshold %f", maxMem, v.maxMemThreshold)
			}

		})
	}
}

func max[T constraints.Ordered](s []T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	m := s[0]
	for _, v := range s {
		if m < v {
			m = v
		}
	}
	return m
}

func sum[T constraints.Ordered](s []T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	var m T
	for _, v := range s {
		m += v
	}
	return m
}
