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
	"time"

	"go.ddosify.com/ddosify/core/scenario/requester"
	"go.ddosify.com/ddosify/core/types"
)

type MockRequester struct {
	InitCalled bool
	SendCalled bool
	DoneCalled bool

	FailInit    bool
	FailInitMsg string

	EnvsSet bool

	ReturnSend *types.ScenarioStepResult
}

func (m *MockRequester) Init(ctx context.Context, s types.ScenarioStep, proxyAddr *url.URL, debug bool) (err error) {
	m.InitCalled = true
	if m.FailInit {
		return fmt.Errorf(m.FailInitMsg)
	}
	return
}

func (m *MockRequester) Send(envs map[string]interface{}) (res *types.ScenarioStepResult) {
	m.SendCalled = true
	return m.ReturnSend
}

func (m *MockRequester) Done() {
	m.DoneCalled = true
}

type MockSleep struct {
	SleepCalled    bool
	SleepCallCount int
}

func (msl *MockSleep) sleep() {
	msl.SleepCalled = true
	msl.SleepCallCount++
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

			if reflect.TypeOf(expectedVal[i].sleeper) != reflect.TypeOf(val[i].sleeper) {
				return fmt.Errorf("[sleep] Expected %#v, Found %#v", expectedVal, val)
			}

			if !reflect.DeepEqual(expectedVal[i].sleeper, val[i].sleeper) {
				return fmt.Errorf("[sleep] Expected %#v, Found %#v", expectedVal, val)
			}
		}
	}
	return nil
}

func TestInitService(t *testing.T) {
	t.Parallel()

	// Arrange
	scenario := types.Scenario{
		Steps: []types.ScenarioStep{
			{
				ID:      1,
				Method:  types.DefaultMethod,
				URL:     "test.com",
				Timeout: types.DefaultDuration,
				Sleep:   "300-500",
			},
			{
				ID:      2,
				Method:  types.DefaultMethod,
				URL:     "test2.com",
				Timeout: types.DefaultDuration,
				Sleep:   "1000",
			},
			{
				ID:      3,
				Method:  types.DefaultMethod,
				URL:     "test3.com",
				Timeout: types.DefaultDuration,
			},
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	p2, _ := url.Parse("http://proxy_server2.com:8000")
	proxies := []*url.URL{p1, p2}
	ctx := context.TODO()
	expectedClients := map[*url.URL][]scenarioItemRequester{
		p1: {
			{
				scenarioItemID: 1,
				requester:      &requester.HttpRequester{},
				sleeper:        &RangeSleep{min: 300, max: 500},
			},
			{
				scenarioItemID: 2,
				requester:      &requester.HttpRequester{},
				sleeper:        &DurationSleep{duration: 1000},
			},
			{
				scenarioItemID: 3,
				requester:      &requester.HttpRequester{},
			},
		},
		p2: {
			{
				scenarioItemID: 1,
				requester:      &requester.HttpRequester{},
				sleeper:        &RangeSleep{min: 300, max: 500},
			},
			{
				scenarioItemID: 2,
				requester:      &requester.HttpRequester{},
				sleeper:        &DurationSleep{duration: 1000},
			},
			{
				scenarioItemID: 3,
				requester:      &requester.HttpRequester{},
			},
		},
	}

	// Act
	service := ScenarioService{}
	err := service.Init(ctx, scenario, proxies, false)

	// Assert
	if err != nil {
		t.Fatalf("TestInitFunc error occurred %v", err)
	}

	if err = compareScenarioServiceClients(expectedClients, service.clients); err != nil {
		t.Fatal(err)
	}
}

func TestDo(t *testing.T) {
	t.Parallel()

	// Arrange
	scenario := types.Scenario{
		Steps: []types.ScenarioStep{
			{
				ID:      1,
				Method:  types.DefaultMethod,
				URL:     "test.com",
				Timeout: types.DefaultDuration,
			},
			{
				ID:      2,
				Method:  types.DefaultMethod,
				URL:     "test.com",
				Timeout: types.DefaultDuration,
			},
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	ctx := context.TODO()
	mockSleep := &MockSleep{}

	requesters := []scenarioItemRequester{
		{
			scenarioItemID: 1,
			sleeper:        mockSleep,
			requester:      &MockRequester{ReturnSend: &types.ScenarioStepResult{StepID: 1}},
		},
		{
			scenarioItemID: 2,
			requester:      &MockRequester{ReturnSend: &types.ScenarioStepResult{StepID: 2}},
		},
	}
	service := ScenarioService{
		clients: map[*url.URL][]scenarioItemRequester{
			p1: requesters,
		},
		scenario: scenario,
		ctx:      ctx,
	}

	expectedResponse := types.ScenarioResult{
		ProxyAddr: p1,
		StepResults: []*types.ScenarioStepResult{
			{StepID: 1}, {StepID: 2},
		},
	}
	// Act
	response, err := service.Do(p1, time.Now())

	// Assert
	if err != nil {
		t.Fatalf("TestDo errored: %v", err)
	}
	if response.ProxyAddr != expectedResponse.ProxyAddr {
		t.Fatalf("[ProxyAddr] Expected %v, Found: %v", expectedResponse.ProxyAddr, response.ProxyAddr)
	}
	if !reflect.DeepEqual(expectedResponse.StepResults, response.StepResults) {
		t.Fatalf("[ResponseItem] Expected %#v, Found: %#v", expectedResponse.StepResults, response.StepResults)
	}
	if !mockSleep.SleepCalled {
		t.Fatalf("[Sleep] Sleep should be called")
	}
	if mockSleep.SleepCallCount != 1 {
		t.Fatalf("[Sleep] Sleep call count expected: %d, Found: %d", 1, mockSleep.SleepCallCount)
	}
}

func TestDoErrorOnSend(t *testing.T) {
	t.Parallel()

	// Arrange
	scenario := types.Scenario{
		Steps: []types.ScenarioStep{
			{
				ID:      1,
				Method:  types.DefaultMethod,
				URL:     "test.com",
				Timeout: types.DefaultDuration,
			},
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	ctx := context.TODO()

	requestersProxyError := []scenarioItemRequester{
		{
			scenarioItemID: 1,
			requester:      &MockRequester{ReturnSend: &types.ScenarioStepResult{Err: types.RequestError{Type: types.ErrorProxy}}},
		},
	}
	requestersIntentedError := []scenarioItemRequester{
		{
			scenarioItemID: 1,
			requester:      &MockRequester{ReturnSend: &types.ScenarioStepResult{Err: types.RequestError{Type: types.ErrorIntented}}},
		},
	}
	requestersConnError := []scenarioItemRequester{
		{
			scenarioItemID: 1,
			requester:      &MockRequester{ReturnSend: &types.ScenarioStepResult{Err: types.RequestError{Type: types.ErrorConn}}},
		},
	}

	tests := []struct {
		name                     string
		requesters               []scenarioItemRequester
		shouldErr                bool
		errorType                string
		responseItemsShouldEmpty bool
	}{
		{"ProxyError", requestersProxyError, true, types.ErrorProxy, false},
		{"IntentedError", requestersIntentedError, true, types.ErrorIntented, true},
		{"ConnError", requestersConnError, false, "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := ScenarioService{
				clients: map[*url.URL][]scenarioItemRequester{
					p1: test.requesters,
				},
				scenario: scenario,
				ctx:      ctx,
			}

			// Act
			res, err := service.Do(p1, time.Now())

			// Assert
			if test.shouldErr {
				if err == nil {
					t.Fatalf("Should be errored")
				}
				if err.Type != test.errorType {
					t.Fatalf("Expected: %v, Found: %v", test.errorType, err.Type)
				}
			} else {
				if err != nil {
					t.Fatalf("Errored: %v", err)
				}
			}
			if test.responseItemsShouldEmpty && len(res.StepResults) > 0 {
				t.Fatalf("ResponseItem should be empty: %v", res.StepResults)
			}
			if !test.responseItemsShouldEmpty && len(res.StepResults) == 0 {
				t.Fatal("ResponseItem shouldn't be empty")
			}

		})
	}
}

// func TestDoErrorOnNewRequester(t *testing.T) {
// 	t.Parallel()

// 	// Arrange
// 	scenario := types.Scenario{
// 		Steps: []types.ScenarioStep{
// 			{
// 				ID:      1,
// 				Method:  types.DefaultMethod,
// 				URL:     "test.com",
// 				Timeout: types.DefaultDuration,
// 			},
// 		},
// 	}
// 	p1, _ := url.Parse("http://proxy_server.com:80")
// 	ctx := context.TODO()

// 	service := ScenarioService{
// 		clients:  map[*url.URL][]scenarioItemRequester{},
// 		scenario: scenario,
// 		ctx:      ctx,
// 	}

// 	// Act
// 	_, err := service.Do(p1, time.Now())

// 	// Assert
// 	if err == nil {
// 		t.Fatalf("TestDoErrorOnNewRequester should be errored")
// 	}
// 	if err.Type != types.ErrorUnkown {
// 		t.Fatalf("Do should return types.ErrorUnkown error type")
// 	}
// }

func TestDone(t *testing.T) {
	t.Parallel()

	// Arrange
	scenario := types.Scenario{
		Steps: []types.ScenarioStep{
			{
				ID:      1,
				Method:  types.DefaultMethod,
				URL:     "test.com",
				Timeout: types.DefaultDuration,
			},
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	p2, _ := url.Parse("http://proxy_server.com:8080")
	ctx := context.TODO()

	requester1 := &MockRequester{ReturnSend: &types.ScenarioStepResult{StepID: 1}}
	requester2 := &MockRequester{ReturnSend: &types.ScenarioStepResult{StepID: 2}}
	requester3 := &MockRequester{ReturnSend: &types.ScenarioStepResult{StepID: 1}}
	requester4 := &MockRequester{ReturnSend: &types.ScenarioStepResult{StepID: 2}}
	service := ScenarioService{
		clients: map[*url.URL][]scenarioItemRequester{
			p1: {
				{
					scenarioItemID: 1,
					requester:      requester1,
				},
				{
					scenarioItemID: 2,
					requester:      requester2,
				},
			},
			p2: {
				{
					scenarioItemID: 1,
					requester:      requester3,
				},
				{
					scenarioItemID: 2,
					requester:      requester4,
				},
			},
		},
		scenario: scenario,
		ctx:      ctx,
	}

	// Act
	service.Done()

	// Assert
	if !requester1.DoneCalled {
		t.Fatalf("Requester1 Done should be called")
	}
	if !requester2.DoneCalled {
		t.Fatalf("Requester2 Done should be called")
	}
	if !requester3.DoneCalled {
		t.Fatalf("Requester3 Done should be called")
	}
	if !requester4.DoneCalled {
		t.Fatalf("Requester4 Done should be called")
	}
}

func TestGetOrCreateRequesters(t *testing.T) {
	t.Parallel()

	// Arrange
	scenario := types.Scenario{
		Steps: []types.ScenarioStep{
			{
				ID:      1,
				Method:  types.DefaultMethod,
				URL:     "test.com",
				Timeout: types.DefaultDuration,
			},
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	proxies := []*url.URL{p1}
	ctx := context.TODO()

	service := ScenarioService{}
	service.Init(ctx, scenario, proxies, false)

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
	t.Parallel()

	// Arrange
	scenario := types.Scenario{
		Steps: []types.ScenarioStep{
			{
				ID:      1,
				Method:  types.DefaultMethod,
				URL:     "test.com",
				Timeout: types.DefaultDuration,
			},
		},
	}
	p1, _ := url.Parse("http://proxy_server.com:80")
	proxies := []*url.URL{p1}
	ctx := context.TODO()

	service := ScenarioService{}
	service.Init(ctx, scenario, proxies, false)

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

func TestCreateRequestersErrorOnRequesterInit(t *testing.T) {
	t.Parallel()

	// Arrange
	scenario := types.Scenario{
		Steps: []types.ScenarioStep{
			{
				ID:      1,
				Method:  "?", // To fail HttpRequesters.Init method
				URL:     "test.com",
				Timeout: types.DefaultDuration,
			},
		},
	}
	p, _ := url.Parse("http://proxy_server.com:80")
	ctx := context.TODO()

	service := ScenarioService{
		clients:  map[*url.URL][]scenarioItemRequester{},
		scenario: scenario,
		ctx:      ctx,
	}

	// Act
	err := service.createRequesters(p)

	// Assert
	if err == nil {
		t.Fatal("TestCreateRequestersFailOnNewRequester should be errored")
	}
}

func TestnewSleeper(t *testing.T) {
	t.Parallel()

	sleepRange := "300-500"
	sleepRangeReverse := "500-300"
	sleepDuration := "1000"

	expectedSleepRange := &RangeSleep{
		min: 300,
		max: 500,
	}
	exptectedSleepDuration := &DurationSleep{
		duration: 1000,
	}

	// "range" sleep strategy test
	sleep := newSleeper(sleepRange)
	if !reflect.DeepEqual(sleep, expectedSleepRange) {
		t.Errorf("Expected %v, Found: %v", expectedSleepRange, sleep)
	}
	sleep = newSleeper(sleepRangeReverse)
	if !reflect.DeepEqual(sleep, expectedSleepRange) {
		t.Errorf("Expected %v, Found: %v", expectedSleepRange, sleep)
	}

	// "duration" sleep strategy test
	sleep = newSleeper(sleepDuration)
	if !reflect.DeepEqual(sleep, exptectedSleepDuration) {
		t.Errorf("Expected %v, Found: %v", exptectedSleepDuration, sleep)
	}
}

func TestSleep(t *testing.T) {
	t.Parallel()

	delta := time.Duration(100)
	min := 300
	max := 500
	dur := 1000

	if testing.Short() {
		// Arrange durations for poor machines
		delta = time.Duration(600)
		min = 750
		max = 1250
		dur = 1000
	}

	sleepDuration := &DurationSleep{
		duration: dur,
	}
	sleepRange := &RangeSleep{
		min: min,
		max: max,
	}

	// Test range
	start := time.Now()
	sleepRange.sleep()
	elapsed := time.Duration(time.Since(start) / time.Millisecond)
	if elapsed > time.Duration(max)+delta || elapsed < time.Duration(min)-delta {
		t.Errorf("Expected: [%d-%d], Found: %d", min, max, elapsed)
	}

	// Test exact duration
	start = time.Now()
	sleepDuration.sleep()
	elapsed = time.Duration(time.Since(start) / time.Millisecond)
	if elapsed > time.Duration(dur)+delta {
		t.Errorf("Expected: %d, Found: %d", dur, elapsed)
	}

}

func TestInjectDynamicVars(t *testing.T) {
	invalidDynamicKey := "{{_randomDdppdd}}"
	envs := map[string]interface{}{
		"country":            "{{_randomCountry}}",
		"X":                  "Y",
		"{{xx}}":             "xx",
		"notFoundDynamicKey": invalidDynamicKey,
	}

	beforeLen := len(envs)

	injectDynamicVars(envs)

	afterLen := len(envs)

	if beforeLen != afterLen {
		t.Errorf("number of envs changed during dynamic var injection")
	}

	if val, ok := envs["country"]; !ok || val == "{{_randomCountry}}" {
		t.Errorf("injection failure")
	}

	if val, ok := envs["notFoundDynamicKey"]; !ok || val != invalidDynamicKey {
		t.Errorf("not found key should stay same")
	}
}
