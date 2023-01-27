package evaluator

import (
	"strings"

	"go.ddosify.com/ddosify/core/scenario/scripting/extraction"
	"go.ddosify.com/ddosify/core/types"
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

var jsonExtract = func(source string, jsonPath string) interface{} {
	val, _ := extraction.ExtractFromJson(source, jsonPath) // TODO handle error
	return val
}

var xmlExtract = func(source string, xPath string) interface{} {
	val, _ := extraction.ExtractFromXml(source, xPath) // TODO handle error
	return val
}

var regexExtract = func(source string, xPath string, matchNo int64) interface{} {
	val, _ := extraction.ExtractWithRegex(source, types.RegexCaptureConf{
		Exp: &xPath,
		No:  int(matchNo),
	}) // TODO handle error
	return val
}

var assertionFuncMap = map[string]struct{}{
	NOT:      {},
	LESSTHAN: {},
	EQUALS:   {},
	IN:       {},
	JSONPATH: {},
	XMLPATH:  {},
	REGEXP:   {},
	HAS:      {},
	CONTAINS: {},
	RANGE:    {},
}

const (
	NOT      = "not"
	LESSTHAN = "less_than"
	EQUALS   = "equals"
	IN       = "in"
	JSONPATH = "json_path"
	XMLPATH  = "xml_path"
	REGEXP   = "regexp"
	HAS      = "has"
	CONTAINS = "contains"
	RANGE    = "range"
)
