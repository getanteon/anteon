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

	"ddosify.com/hammer/core/types"
)

//TODO: Engine stop channel close order test

func newDummyHammer() types.Hammer {
	return types.Hammer{
		Proxy:             types.Proxy{Strategy: "single"},
		ReportDestination: "stdout",
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

func TestServices(t *testing.T) {
	h := newDummyHammer()
	e := NewEngine(context.TODO(), h)
	e.Init()

	if e.proxyService == nil {
		t.Errorf("Proxy Service should be initialized")
	}
	if e.scenarioService == nil {
		t.Errorf("Scenario Service should be initialized")
	}
	if e.reportService == nil {
		t.Errorf("Report Service should be initialized")
	}
}

// TODO: Add other load types as you implement
func TestRequestCount(t *testing.T) {
	var timeReqMap map[int]int
	var now time.Time
	var mutex = &sync.Mutex{}

	// Test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		i := time.Since(now).Milliseconds()/tickerInterval - 1
		timeReqMap[int(i)]++
		mutex.Unlock()
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	tests := []struct {
		name           string
		loadType       string
		duration       int
		reqCount       int
		expectedReqArr []int
		delta          int
	}{
		{"Linear1", types.LoadTypeLinear, 1, 100, []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10}, 1},
		{"Linear2", types.LoadTypeLinear, 1, 5, []int{1, 1, 1, 1, 1, 0, 0, 0, 0, 0}, 0},
		{"Linear3", types.LoadTypeLinear, 2, 23,
			[]int{2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, 0},
		{"Incremental1", types.LoadTypeIncremental, 3, 1022,
			[]int{17, 17, 17, 17, 17, 17, 17, 17, 17, 17, 35, 34, 34, 34,
				34, 34, 34, 34, 34, 34, 52, 51, 51, 51, 51, 51, 51, 51, 51, 51}, 2},
		{"Incremental2", types.LoadTypeIncremental, 5, 10,
			[]int{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1,
				0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0}, 0},
		{"Incremental3", types.LoadTypeIncremental, 4, 10,
			[]int{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1,
				0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0}, 0},
		{"Waved1", types.LoadTypeWaved, 4, 32,
			[]int{1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0}, 0},
		{"Waved2", types.LoadTypeWaved, 5, 10,
			[]int{1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1,
				0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 0},
		{"Waved3", types.LoadTypeWaved, 9, 1000,
			[]int{6, 6, 6, 6, 6, 6, 5, 5, 5, 5, 12, 11, 11, 11, 11, 11, 11, 11, 11, 11, 17, 17, 17, 17,
				17, 17, 16, 16, 16, 16, 17, 17, 17, 17, 17, 17, 16, 16, 16, 16, 12, 11, 11, 11, 11, 11,
				11, 11, 11, 11, 6, 6, 6, 6, 6, 6, 5, 5, 5, 5, 6, 6, 6, 6, 6, 6, 5, 5, 5, 5, 12, 11, 11,
				11, 11, 11, 11, 11, 11, 11, 17, 17, 17, 17, 17, 17, 17, 16, 16, 16}, 1},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Prepare
			h := newDummyHammer()
			h.LoadType = test.loadType
			h.TestDuration = test.duration
			h.TotalReqCount = test.reqCount
			h.Scenario.Scenario[0].URL = server.URL

			now = time.Now()
			timeReqMap = make(map[int]int, 0)

			e := NewEngine(context.TODO(), h)

			// Act
			e.Init()
			e.Start()

			// Assert create reqCountArr
			if !reflect.DeepEqual(e.reqCountArr, test.expectedReqArr) {
				t.Errorf("Expected: %v, Found: %v", test.expectedReqArr, e.reqCountArr)
			}

			// Assert sent request count
			for i, v := range test.expectedReqArr {
				if timeReqMap[i] > v+test.delta || timeReqMap[i] < v-test.delta {
					t.Errorf("Expected: %v, Recieved: %v, Tick: %v", v, timeReqMap[i], i)
				}
			}
		}
		t.Run(test.name, tf)
	}
}

func TestRequestData(t *testing.T) {
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
	e := NewEngine(context.TODO(), h)
	e.Init()
	e.Start()

	// Assert
	if uri != "/get_test_data" {
		t.Errorf("invalid uri recieved: %s", uri)
	}

	if protocol != "HTTP/1.1" {
		t.Errorf("invalid protocol recieved: %v", protocol)
	}

	if method != "GET" {
		t.Errorf("invalid method recieved: %v", method)
	}

	if header1 != "Test1Value" {
		t.Errorf("invalid header1 receieved: %s", header1)
	}

	if header2 != "Test2Value" {
		t.Errorf("invalid header2 receieved: %s", header2)
	}

	if body != "Body content" {
		t.Errorf("invalid body recieved: %v", body)
	}
}

func TestRequestDataForMultiScenarioStep(t *testing.T) {
	var uri, header, body, protocol, method = make([]string, 2), make([]string, 2), make([]string, 2),
		make([]string, 2), make([]string, 2)

	// Test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		protocol = append(protocol, r.Proto)
		method = append(method, r.Method)
		uri = append(uri, r.RequestURI)
		header = append(header, r.Header.Get("Test"))

		bodyByte, _ := ioutil.ReadAll(r.Body)
		body = append(body, string(bodyByte))
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
				Protocol: "HTTPS",
				Method:   "POST",
				URL:      server.URL + "/api_post",
				Headers:  map[string]string{"Test": "h2"},
				Payload:  "Body 2",
			},
		}}

	// Act
	e := NewEngine(context.TODO(), h)
	e.Init()
	e.Start()

	// Assert
	if reflect.DeepEqual(uri, []string{"/api_get", "/api_post"}) {
		t.Errorf("invalid uri recieved: %s", uri)
	}

	if reflect.DeepEqual(protocol, []string{"HTTP/1.1", "HTTPS/1.1"}) {
		t.Errorf("invalid protocol receieved: %s", protocol)
	}

	if reflect.DeepEqual(method, []string{"GET", "POST"}) {
		t.Errorf("invalid method receieved: %s", method)
	}

	if reflect.DeepEqual(header, []string{"h1", "h2"}) {
		t.Errorf("invalid header recieved: %v", header)
	}

	if reflect.DeepEqual(body, []string{"Body 1", "Body 2"}) {
		t.Errorf("invalid body recieved: %v", body)
	}
}

func TestRequestTimeout(t *testing.T) {
	var result bool

	// Test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(1100) * time.Millisecond)
		result = true
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// Prepare
	tests := []struct {
		name     string
		timeout  int
		expected bool
	}{
		{"Timeout", 1, false},
		{"NotTimeout", 2, true},
	}

	// Act
	for _, test := range tests {
		tf := func(t *testing.T) {
			result = false
			h := newDummyHammer()
			h.Scenario.Scenario[0].Timeout = test.timeout
			h.Scenario.Scenario[0].URL = server.URL

			e := NewEngine(context.TODO(), h)
			e.Init()
			e.Start()

			// Assert
			if result != test.expected {
				t.Errorf("Expected %v, Found :%v", test.expected, result)
			}
		}
		t.Run(test.name, tf)
	}
}
