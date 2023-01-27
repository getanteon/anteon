package evaluator

import (
	"strings"

	"go.ddosify.com/ddosify/core/scenario/scripting/extraction"
)

var less_than = func(variable int64, limit int64) bool {
	return variable < limit
}

var not = func(b bool) bool {
	return !b
}

var equals = func(a, b interface{}) bool {
	b, err := evalInfixExpression("==", a, b)
	if err != nil { // TODO propagate error
		return false
	}
	return b.(bool)
}

var in = func(a interface{}, b []interface{}) bool {
	for _, elem := range b {
		if equals(a, elem) {
			return true
		}
	}
	return false
}

var jsonExtract = func(source string, jsonPath string) interface{} {
	val, _ := extraction.ExtractFromJson(source, jsonPath) // TODO handle error
	return val
}

var contains = func(source string, substr string) bool {
	if strings.Contains(source, substr) {
		return true
	}
	return false
}

var rangeF = func(x int64, low int64, hi int64) bool {
	if x >= low && x < hi {
		return true
	}
	return false
}

var assertionFuncMap = map[string]struct{}{
	"not":       {},
	"less_than": {},
	"equals":    {},
	"in":        {},
	"json_path": {},
	"has":       {},
	"contains":  {},
	"range":     {},
}
