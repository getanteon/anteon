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

package core

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/report"
	"go.ddosify.com/ddosify/core/types"
)

//TODO: Engine stop channel close order test

func newDummyHammer() types.Hammer {
	return types.Hammer{
		Proxy:             proxy.Proxy{Strategy: proxy.ProxyTypeSingle},
		ReportDestination: report.OutputTypeStdout,
		LoadType:          types.LoadTypeLinear,
		TestDuration:      1,
		TotalReqCount:     1,
		Scenario: types.Scenario{
			Scenario: []types.ScenarioItem{
				{
					ID:       1,
					Protocol: "HTTP",
					Method:   "GET",
					URL:      "http://127.0.0.1",
				},
			},
		},
	}
}

func TestCreateEngine(t *testing.T) {
	t.Parallel()

	hInvalidProxy := newDummyHammer()
	hInvalidProxy.Proxy = proxy.Proxy{Strategy: "invalidProxy"}

	hInvalidReport := newDummyHammer()
	hInvalidReport.ReportDestination = "invalidReport"

	tests := []struct {
		name      string
		hammer    types.Hammer
		shouldErr bool
	}{
		{"Normal", newDummyHammer(), false},
		{"InvalidProxy", hInvalidProxy, true},
		{"InvalidReport", hInvalidReport, true},
	}

	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			e, err := NewEngine(context.TODO(), test.hammer)

			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Error occurred %v", err)
				}

				if e.proxyService == nil {
					t.Errorf("Proxy Service should be created")
				}
				if e.scenarioService == nil {
					t.Errorf("Scenario Service should be created")
				}
				if e.reportService == nil {
					t.Errorf("Report Service should be created")
				}
			}
		})
	}
}

// TODO: Add other load types as you implement
func TestRequestCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		loadType       string
		duration       int
		reqCount       int
		expectedReqArr []int
		delta          int
	}{
		{"Linear1", types.LoadTypeLinear, 1, 10, []int{10}, 0},
		{"Linear2", types.LoadTypeLinear, 3, 2, []int{1, 1, 0}, 0},
		{"Linear3", types.LoadTypeLinear, 3, 30,
			[]int{10, 10, 10}, 1},
		{"Linear4", types.LoadTypeLinear, 3, 32,
			[]int{11, 11, 10}, 1},
		{"Incremental1", types.LoadTypeIncremental, 1, 10,
			[]int{10}, 0},
		{"Incremental2", types.LoadTypeIncremental, 3, 2,
			[]int{0, 1, 1}, 0},
		{"Incremental3", types.LoadTypeIncremental, 3, 60,
			[]int{10, 20, 30}, 1},
		{"Incremental4", types.LoadTypeIncremental, 3, 555,
			[]int{93, 185, 277}, 2},
		{"Waved1", types.LoadTypeWaved, 1, 3,
			[]int{3}, 0},
		{"Waved2", types.LoadTypeWaved, 3, 10,
			[]int{2, 3, 5}, 0},
		{"Waved3", types.LoadTypeWaved, 4, 32,
			[]int{5, 11, 11, 5}, 0},
		{"Waved4", types.LoadTypeWaved, 5, 32,
			[]int{5, 11, 11, 5, 0}, 0},
		{"Waved5", types.LoadTypeWaved, 9, 100,
			[]int{6, 11, 16, 16, 11, 6, 6, 11, 17}, 1},
	}

	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var timeReqMap map[int]int
			var now time.Time
			var m sync.Mutex

			// Test server
			handler := func(w http.ResponseWriter, r *http.Request) {
				m.Lock()
				i := time.Since(now).Milliseconds()/tickerInterval - 1
				timeReqMap[int(i)]++
				m.Unlock()
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			// Prepare
			h := newDummyHammer()
			h.LoadType = test.loadType
			h.TestDuration = test.duration
			h.TotalReqCount = test.reqCount
			h.Scenario.Scenario[0].URL = server.URL

			now = time.Now()
			timeReqMap = make(map[int]int, 0)

			e, err := NewEngine(context.TODO(), h)
			if err != nil {
				t.Errorf("TestRequestCount error occurred %v", err)
			}

			// Act
			err = e.Init()
			if err != nil {
				t.Errorf("TestRequestCount error occurred %v", err)
			}

			e.Start()

			m.Lock()
			// Assert create reqCountArr
			if !reflect.DeepEqual(e.reqCountArr, test.expectedReqArr) {
				t.Errorf("Expected: %v, Found: %v", test.expectedReqArr, e.reqCountArr)
			}

			// Assert sent request count
			if testing.Short() {
				// Poor machine's test case assertions are special since they can't run the test fast.
				totalRecieved := 0
				for _, v := range timeReqMap {
					totalRecieved += v
				}
				expected := arraySum(test.expectedReqArr)
				if totalRecieved != expected {
					t.Errorf("Poor Machine Expected: %v, Received: %v", totalRecieved, expected)
				}
			} else {
				for i, v := range test.expectedReqArr {
					if timeReqMap[i] > v+test.delta || timeReqMap[i] < v-test.delta {
						t.Errorf("Expected: %v, Received: %v, Tick: %v", v, timeReqMap[i], i)
					}
				}
			}

			m.Unlock()
		})
	}
}

func TestRequestData(t *testing.T) {
	t.Parallel()

	var uri, header1, header2, body, protocol, method string

	// Test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		protocol = r.Proto
		method = r.Method
		uri = r.RequestURI
		header1 = r.Header.Get("Test1")
		header2 = r.Header.Get("Test2")

		bodyByte, _ := ioutil.ReadAll(r.Body)
		body = string(bodyByte)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// Prepare
	h := newDummyHammer()
	h.Scenario.Scenario[0] = types.ScenarioItem{
		ID:       1,
		Protocol: "HTTP",
		Method:   "GET",
		URL:      server.URL + "/get_test_data",
		Headers:  map[string]string{"Test1": "Test1Value", "Test2": "Test2Value"},
		Payload:  "Body content",
	}

	// Act
	e, err := NewEngine(context.TODO(), h)
	if err != nil {
		t.Errorf("TestRequestData error occurred %v", err)
	}

	err = e.Init()
	if err != nil {
		t.Errorf("TestRequestData error occurred %v", err)
	}

	e.Start()

	// Assert
	if uri != "/get_test_data" {
		t.Errorf("invalid uri received: %s", uri)
	}

	if protocol != "HTTP/1.1" {
		t.Errorf("invalid protocol received: %v", protocol)
	}

	if method != "GET" {
		t.Errorf("invalid method received: %v", method)
	}

	if header1 != "Test1Value" {
		t.Errorf("invalid header1 receieved: %s", header1)
	}

	if header2 != "Test2Value" {
		t.Errorf("invalid header2 receieved: %s", header2)
	}

	if body != "Body content" {
		t.Errorf("invalid body received: %v", body)
	}
}

func TestRequestDataForMultiScenarioStep(t *testing.T) {
	t.Parallel()

	var uri, header, body, protocol, method []string

	var m sync.Mutex

	// Test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		m.Lock()
		protocol = append(protocol, r.Proto)
		method = append(method, r.Method)
		uri = append(uri, r.RequestURI)
		header = append(header, r.Header.Get("Test"))

		bodyByte, _ := ioutil.ReadAll(r.Body)
		body = append(body, string(bodyByte))
		m.Unlock()
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// Prepare
	h := newDummyHammer()
	h.Scenario = types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: "HTTP",
				Method:   "GET",
				URL:      server.URL + "/api_get",
				Headers:  map[string]string{"Test": "h1"},
				Payload:  "Body 1",
			},
			{
				ID:       2,
				Protocol: "HTTP",
				Method:   "POST",
				URL:      server.URL + "/api_post",
				Headers:  map[string]string{"Test": "h2"},
				Payload:  "Body 2",
			},
		}}

	// Act
	e, err := NewEngine(context.TODO(), h)
	if err != nil {
		t.Errorf("TestRequestDataForMultiScenarioStep error occurred %v", err)
	}

	err = e.Init()
	if err != nil {
		t.Errorf("TestRequestDataForMultiScenarioStep error occurred %v", err)
	}

	e.Start()

	// Assert
	expected := []string{"/api_get", "/api_post"}
	if !reflect.DeepEqual(uri, expected) {
		t.Logf("%#v - %#v", uri, expected)
		t.Errorf("invalid uri receieved: %#v expected %#v", uri, expected)
	}

	expected = []string{"HTTP/1.1", "HTTP/1.1"}
	if !reflect.DeepEqual(protocol, expected) {
		t.Errorf("invalid protocol receieved: %#v expected %#v", protocol, expected)
	}

	expected = []string{"GET", "POST"}
	if !reflect.DeepEqual(method, expected) {
		t.Errorf("invalid method receieved: %#v expected %#v", method, expected)
	}

	expected = []string{"h1", "h2"}
	if !reflect.DeepEqual(header, expected) {
		t.Errorf("invalid header receieved: %#v expected %#v", header, expected)
	}

	expected = []string{"Body 1", "Body 2"}
	if !reflect.DeepEqual(body, expected) {
		t.Errorf("invalid body receieved: %#v expected %#v", body, expected)
	}
}

func TestRequestTimeout(t *testing.T) {
	t.Parallel()

	// Prepare
	tests := []struct {
		name     string
		timeout  int
		expected bool
	}{
		{"Timeout", 1, false},
		{"NotTimeout", 3, true},
	}

	// Act
	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := false
			var m sync.Mutex

			// Test server
			handler := func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Duration(2) * time.Second)

				m.Lock()
				result = true
				m.Unlock()
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			h := newDummyHammer()
			h.Scenario.Scenario[0].Timeout = test.timeout
			h.Scenario.Scenario[0].URL = server.URL

			e, err := NewEngine(context.TODO(), h)
			if err != nil {
				t.Errorf("TestRequestTimeout error occurred %v", err)
			}

			err = e.Init()
			if err != nil {
				t.Errorf("TestRequestTimeout error occurred %v", err)
			}

			e.Start()

			// Assert
			m.Lock()
			if result != test.expected {
				t.Errorf("Expected %v, Found :%v", test.expected, result)
			}
			m.Unlock()
		})
	}
}

func TestEngineResult(t *testing.T) {
	t.Parallel()

	// Prepare
	tests := []struct {
		name           string
		cancelCtx      bool
		expectedStatus string
	}{
		{"CtxCancel", true, "stopped"},
		{"Normal", false, "done"},
	}

	// Act
	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var m sync.Mutex

			// Test server
			handler := func(w http.ResponseWriter, r *http.Request) {
				return
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			h := newDummyHammer()
			h.TestDuration = 2
			h.Scenario.Scenario[0].URL = server.URL

			ctx, cancel := context.WithCancel(context.Background())
			e, err := NewEngine(ctx, h)
			if err != nil {
				t.Errorf("TestRequestTimeout error occurred %v", err)
			}

			err = e.Init()
			if err != nil {
				t.Errorf("TestRequestTimeout error occurred %v", err)
			}

			if test.cancelCtx {
				time.AfterFunc(time.Duration(500)*time.Millisecond, func() {
					cancel()
				})
			}

			res := e.Start()
			cancel()

			// Assert
			m.Lock()
			if res != test.expectedStatus {
				t.Errorf("Expected %v, Found %v", test.expectedStatus, res)
			}
			m.Unlock()
		})
	}
}
