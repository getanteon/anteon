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

package scenario

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	"go.ddosify.com/ddosify/core/scenario/requester"
	"go.ddosify.com/ddosify/core/types"
)

func TestInitService(t *testing.T) {
	// Arrange
	scenario := types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: types.DefaultProtocol,
				Method:   types.DefaultMethod,
				URL:      "test.com",
				Timeout:  types.DefaultDuration,
			},
			{
				ID:       2,
				Protocol: types.DefaultProtocol,
				Method:   types.DefaultMethod,
				URL:      "test2.com",
				Timeout:  types.DefaultDuration,
			},
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	p2, _ := url.Parse("http://proxy_server2.com:8000")
	proxies := []*url.URL{p1, p2}
	ctx := context.TODO()
	expecedClients := map[*url.URL][]scenarioItemRequester{
		p1: {
			{scenarioItemID: 1, requester: &requester.HttpRequester{}},
			{scenarioItemID: 2, requester: &requester.HttpRequester{}},
		},
		p2: {
			{scenarioItemID: 1, requester: &requester.HttpRequester{}},
			{scenarioItemID: 2, requester: &requester.HttpRequester{}},
		},
	}

	// Act
	service := ScenarioService{}
	err := service.Init(ctx, scenario, proxies)

	// Assert
	if err != nil {
		t.Fatalf("TestInitFunc error occurred %v", err)
	}

	if len(expecedClients) != len(service.clients) {
		t.Fatalf("[length] Expected %v, Found %v", expecedClients, service.clients)
	}

	for k, expectedVal := range expecedClients {
		val, ok := service.clients[k]

		if !ok {
			t.Fatalf("[key] Expected %#v, Found %#v", expecedClients, service.clients)
		}

		if len(expectedVal) != len(val) {
			t.Fatalf("[valLength] Expected %v, Found %v", expectedVal, val)
		}

		if (expectedVal[0].scenarioItemID != val[0].scenarioItemID) ||
			(expectedVal[1].scenarioItemID != val[1].scenarioItemID) {
			t.Fatalf("[scenarioItemID] Expected %#v, Found %#v", expectedVal, val)
		}

		if (expectedVal[0].scenarioItemID != val[0].scenarioItemID) ||
			(expectedVal[1].scenarioItemID != val[1].scenarioItemID) {
			t.Fatalf("[scenarioItemID] Expected %#v, Found %#v", expectedVal, val)
		}

		if reflect.TypeOf(expectedVal[0].requester) != reflect.TypeOf(val[0].requester) ||
			reflect.TypeOf(expectedVal[1].requester) != reflect.TypeOf(val[1].requester) {
			t.Fatalf("[requester] Expected %#v, Found %#v", expectedVal, val)
		}
	}
}

func TestInitServiceFail(t *testing.T) {
	// Arrange
	scenario := types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: "invalid_protocol",
				Method:   types.DefaultMethod,
				URL:      "test.com",
				Timeout:  types.DefaultDuration,
			},
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	p2, _ := url.Parse("http://proxy_server2.com:8000")
	proxies := []*url.URL{p1, p2}
	ctx := context.TODO()

	// Act
	service := ScenarioService{}
	err := service.Init(ctx, scenario, proxies)

	// Assert
	if err == nil {
		t.Fatalf("TestInitFunc should be errored")
	}

}
