package evaluator

import "testing"

func TestEmptyArraysOnMinMaxAvgFuncs(t *testing.T) {
	empty := []int64{}
	_, err := min(empty)
	if err == nil {
		t.Errorf("expected error on empty array on min func")
	}

	_, err = max(empty)
	if err == nil {
		t.Errorf("expected error on empty array on max func")
	}

	_, err = avg(empty)
	if err == nil {
		t.Errorf("expected error on empty array on avg func")
	}

	_, err = percentile(empty, 99)
	if err == nil {
		t.Errorf("expected error on empty array on percentile func")
	}
}
