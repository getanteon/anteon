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

package report

import (
	"reflect"
	"testing"

	"ddosify.com/hammer/core/types"
)

var outputStrategiesStructMap = map[string]reflect.Type{
	types.OutputTypeStdout:    reflect.TypeOf(&stdout{}),
	types.OutputTypeTimescale: reflect.TypeOf(&timescale{}),
}

func TestNewReportService(t *testing.T) {

	// Valid output types
	for _, o := range types.SupportedOutputs {
		service, err := NewReportService(o)

		if err != nil {
			t.Errorf("TestNewReportService %v", err)
		}

		if reflect.TypeOf(service) != outputStrategiesStructMap[o] {
			t.Errorf("Expected %v, Found %v", outputStrategiesStructMap[o], reflect.TypeOf(service))
		}
	}

	// Invalid output type
	_, err := NewReportService("invalid_output_type")
	if err == nil {
		t.Errorf("TestNewReportService invalid output should errored")
	}
}
