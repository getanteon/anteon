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

package config

import (
	"reflect"
	"testing"

	"ddosify.com/hammer/core/types"
)

func TestCreateHammerDefaultValues(t *testing.T) {
	jsonReader, _ := NewConfigReader("config_testdata/empty.json", ConfigTypeJson)
	expectedHammer := types.Hammer{
		TotalReqCount:     types.DefaultReqCount,
		LoadType:          types.DefaultLoadType,
		TestDuration:      types.DefaultDuration,
		ReportDestination: types.DefaultOutputType,
		Scenario: types.Scenario{
			Scenario: []types.ScenarioItem{{
				ID:       1,
				URL:      types.DefaultProtocol + "://test.com",
				Protocol: types.DefaultProtocol,
				Method:   types.DefaultMethod,
				Timeout:  types.DefaultTimeout,
			}},
		},
		Proxy: types.Proxy{
			Strategy: "single",
		},
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammerDefaultValues error occured: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("Expected: %v, Found: %v", expectedHammer, h)
	}
}
