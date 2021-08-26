package core

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"ddosify.com/hammer/core/types"
)

var e = CreateEngine(context.TODO(), newDummyHammer())

func newDummyHammer() types.Hammer {
	return types.Hammer{
		Proxy:             types.Proxy{Strategy: "single"},
		ReportDestination: "stdout",
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

	e2 := CreateEngine(context.TODO(), newDummyHammer())
	if e != e2 {
		t.Errorf("CreateEngine should be singleton")
	}
}

// TODO: Add other load types as you implement
func TestReqCountArr(t *testing.T) {

	tests := []struct {
		name           string
		loadType       string
		duration       int
		reqCount       int
		expectedReqArr []int
	}{
		{"Linear1", types.LoadTypeLinear, 1, 10, []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
		{"Linear2", types.LoadTypeLinear, 1, 5, []int{1, 1, 1, 1, 1, 0, 0, 0, 0, 0}},
		{"Linear3", types.LoadTypeLinear, 2, 23,
			[]int{2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			e.hammer.LoadType = test.loadType
			e.hammer.TestDuration = test.duration
			e.hammer.TotalReqCount = test.reqCount

			e.Init()
			if !reflect.DeepEqual(e.reqCountArr, test.expectedReqArr) {
				t.Errorf("Expected: %v, Found: %v", test.expectedReqArr, e.reqCountArr)
			}
		}
		t.Run(test.name, tf)
	}
}

func TestStart(t *testing.T) {
	var uri, header1, header2, body, protocol string

	// Test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		protocol = r.Proto
		uri = r.RequestURI
		header1 = r.Header.Get("Test1")
		header2 = r.Header.Get("Test2")

		bodyByte, _ := ioutil.ReadAll(r.Body)
		body = string(bodyByte)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// Prepare
	e.hammer.Scenario.Scenario[0] = types.ScenarioItem{
		ID:       1,
		Protocol: "HTTP",
		Method:   "GET",
		URL:      server.URL + "/get_test_data",
		Headers:  map[string]string{"Test1": "Test1Value", "Test2": "Test2Value"},
		Payload:  "Body content",
	}
	e.hammer.LoadType = types.LoadTypeLinear
	e.hammer.TestDuration = 1
	e.hammer.TotalReqCount = 1

	// Act
	e.Init()
	e.Start()

	// Assert
	if uri != "/get_test_data" {
		t.Errorf("invalid uri recieved: %s", uri)
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

	if protocol != "HTTP/1.1" {
		t.Errorf("invalid protocol recieved: %v", protocol)
	}
}
