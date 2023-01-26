package evaluator

import (
	"fmt"

	"go.ddosify.com/ddosify/core/scenario/scripting/extraction"
)

var less_than = func(variable int64, limit int64) bool {
	fmt.Println("less_than called")
	return variable < limit
}
var not = func(b bool) bool {
	fmt.Println("not called")
	return !b
}

var equals = func(a, b interface{}) bool {
	fmt.Println("equals called")
	b, err := evalInfixExpression("==", a, b)
	if err != nil { // TODO propagate error
		return false
	}
	return b.(bool)
}

var in = func(a interface{}, b []interface{}) bool {
	fmt.Println("in called")

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

var assertionFuncMap = map[string]struct{}{
	"not":       {},
	"less_than": {},
	"equals":    {},
	"in":        {},
	"json_path": {},
}
