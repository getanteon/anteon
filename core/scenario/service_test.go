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
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"go.ddosify.com/ddosify/core/scenario/requester"
	"go.ddosify.com/ddosify/core/types"
)

type MockRequester struct {
	InitCalled bool
	SendCalled bool

	FailInit    bool
	FailInitMsg string

	SendCallCount int
	ReturnSend    *types.ResponseItem
}

func (m *MockRequester) Init(ctx context.Context, s types.ScenarioItem, proxyAddr *url.URL) (err error) {
	m.InitCalled = true
	if m.FailInit {
		return fmt.Errorf(m.FailInitMsg)
	}
	return
}

func (m *MockRequester) Send() (res *types.ResponseItem) {
	m.SendCalled = true
	m.SendCallCount++
	return m.ReturnSend
}

func compareScenarioServiceClients(
	expectedClients map[*url.URL][]scenarioItemRequester,
	clients map[*url.URL][]scenarioItemRequester) error {

	if len(expectedClients) != len(clients) {
		return fmt.Errorf("[length] Expected %v, Found %v", expectedClients, clients)
	}

	for k, expectedVal := range expectedClients {
		val, ok := clients[k]

		if !ok {
			return fmt.Errorf("[key] Expected %#v, Found %#v", expectedClients, clients)
		}

		if len(expectedVal) != len(val) {
			return fmt.Errorf("[valLength] Expected %v, Found %v", expectedVal, val)
		}

		for i := 0; i < len(expectedVal); i++ {
			if expectedVal[i].scenarioItemID != val[i].scenarioItemID {
				return fmt.Errorf("[scenarioItemID] Expected %#v, Found %#v", expectedVal, val)
			}

			if expectedVal[i].scenarioItemID != val[i].scenarioItemID {
				return fmt.Errorf("[scenarioItemID] Expected %#v, Found %#v", expectedVal, val)
			}

			if reflect.TypeOf(expectedVal[i].requester) != reflect.TypeOf(val[i].requester) {
				return fmt.Errorf("[requester] Expected %#v, Found %#v", expectedVal, val)
			}
		}
	}
	return nil
}

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
	expectedClients := map[*url.URL][]scenarioItemRequester{
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

	if err = compareScenarioServiceClients(expectedClients, service.clients); err != nil {
		t.Fatal(err)
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

func TestGetOrCreateRequesters(t *testing.T) {
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
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	proxies := []*url.URL{p1}
	ctx := context.TODO()

	service := ScenarioService{}
	service.Init(ctx, scenario, proxies)

	expectedRequesters := []scenarioItemRequester{{scenarioItemID: 1, requester: &requester.HttpRequester{}}}
	expectedClients := map[*url.URL][]scenarioItemRequester{
		p1: expectedRequesters,
	}

	// Act
	requesters, err := service.getOrCreateRequesters(p1)

	// Assert
	if err != nil {
		t.Fatalf("TestGetOrCreateRequesters errored: %v", err)
	}

	if len(expectedRequesters) != len(requesters) ||
		expectedRequesters[0].scenarioItemID != requesters[0].scenarioItemID ||
		reflect.TypeOf(expectedRequesters[0].requester) != reflect.TypeOf(requesters[0].requester) {
		t.Fatalf("Expected: %v, Found: %v", expectedRequesters, requesters)
	}

	if err = compareScenarioServiceClients(expectedClients, service.clients); err != nil {
		t.Fatal(err)
	}
}

func TestGetOrCreateRequestersNewProxy(t *testing.T) {
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
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	proxies := []*url.URL{p1}
	ctx := context.TODO()

	service := ScenarioService{}
	service.Init(ctx, scenario, proxies)

	expectedRequesters := []scenarioItemRequester{{scenarioItemID: 1, requester: &requester.HttpRequester{}}}

	p2, _ := url.Parse("http://proxy_server2.com:8080")
	expectedClients := map[*url.URL][]scenarioItemRequester{
		p1: {{scenarioItemID: 1, requester: &requester.HttpRequester{}}},
		p2: {{scenarioItemID: 1, requester: &requester.HttpRequester{}}},
	}

	// Act
	requesters, err := service.getOrCreateRequesters(p2)

	// Assert
	if err != nil {
		t.Fatalf("TestGetOrCreateRequestersNewProxy errored: %v", err)
	}

	if len(expectedRequesters) != len(requesters) ||
		expectedRequesters[0].scenarioItemID != requesters[0].scenarioItemID ||
		reflect.TypeOf(expectedRequesters[0].requester) != reflect.TypeOf(requesters[0].requester) {
		t.Fatalf("Expected: %v, Found: %v", expectedRequesters, requesters)
	}

	if err = compareScenarioServiceClients(expectedClients, service.clients); err != nil {
		t.Fatal(err)
	}
}

func TestGetOrCreateRequestersFailed(t *testing.T) {
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
	// Left empty proxies to bypass Init method. So we can errored createRequesters method
	proxies := []*url.URL{}
	ctx := context.TODO()

	service := ScenarioService{}
	service.Init(ctx, scenario, proxies)

	p, _ := url.Parse("http://proxy_server2.com:8080")

	// Act
	_, err := service.getOrCreateRequesters(p)

	// Assert
	if err == nil {
		t.Fatalf("TestGetOrCreateRequestersFailed should be errored")
	}
}
