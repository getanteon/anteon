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

package requester

import (
	"reflect"
	"testing"

	"go.ddosify.com/ddosify/core/types"
)

var protocolStrategiesStructMap = map[string]reflect.Type{
	types.ProtocolHTTP:  reflect.TypeOf(&HttpRequester{}),
	types.ProtocolHTTPS: reflect.TypeOf(&HttpRequester{}),
}

func TestNewRequester(t *testing.T) {

	// Valid output types
	for _, o := range types.SupportedProtocols {
		service, err := NewRequester(types.ScenarioStep{Protocol: o})

		if err != nil {
			t.Errorf("TestNewRequester %v", err)
		}

		if reflect.TypeOf(service) != protocolStrategiesStructMap[o] {
			t.Errorf("Expected %v, Found %v", protocolStrategiesStructMap[o], reflect.TypeOf(service))
		}
	}

	// Invalid output type
	_, err := NewRequester(types.ScenarioStep{Protocol: "invalid_protocol"})
	if err == nil {
		t.Errorf("TestNewRequester invalid protocol should errored")
	}
}
