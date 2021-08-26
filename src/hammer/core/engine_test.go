package core

import (
	"context"
	"reflect"
	"testing"

	"ddosify.com/hammer/core/types"
)

var e = CreateEngine(context.TODO(), newDummyHammer())

func newDummyHammer() types.Hammer {
	return types.Hammer{
		Proxy:             types.Proxy{Strategy: "single"},
		ReportDestination: "stdout",
	}
}

func TestCreateEngineSingleton(t *testing.T) {

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
		{"Linear3", types.LoadTypeLinear, 2, 23, []int{2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
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
