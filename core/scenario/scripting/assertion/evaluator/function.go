package evaluator

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"go.ddosify.com/ddosify/core/scenario/scripting/extraction"
	"go.ddosify.com/ddosify/core/types"
)

var less_than = func(variable int64, limit int64) bool {
	return variable < limit
}
var greater_than = func(variable int64, limit int64) bool {
	return variable > limit
}

var not = func(b bool) bool {
	return !b
}

var equals = func(a, b interface{}) (bool, error) {
	b, err := evalInfixExpression("==", a, b)
	if err != nil {
		return false, err
	}
	return b.(bool), nil
}

var in = func(a interface{}, b []interface{}) (bool, error) {
	for _, elem := range b {
		if eq, err := equals(a, elem); eq {
			return true, nil
		} else if err != nil {
			return false, err
		}
	}
	return false, nil
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

var jsonExtract = func(source interface{}, jsonPath string) (interface{}, error) {
	val, err := extraction.ExtractFromJson(source, jsonPath)
	return val, err
}

var xmlExtract = func(source interface{}, xPath string) (interface{}, error) {
	val, err := extraction.ExtractFromXml(source, xPath)
	return val, err
}

var regexExtract = func(source interface{}, xPath string, matchNo int64) (interface{}, error) {
	val, err := extraction.ExtractWithRegex(source, types.RegexCaptureConf{
		Exp: &xPath,
		No:  int(matchNo),
	})
	return val, err
}

var equalsOnFile = func(source interface{}, filepath string) (bool, error) {
	fileBytes, err := os.ReadFile(filepath)
	if err != nil {
		return false, err
	}

	if strings.HasSuffix(filepath, ".json") {
		sourceType := reflect.ValueOf(source).Kind() // json extracted types may be map or slice etc

		if sourceType == reflect.String {
			// in case of direct body comparison, source param will be string
			var src interface{}
			err := json.Unmarshal([]byte(source.(string)), &src)
			if err != nil {
				return false, err
			}

			var fileB interface{}
			err = json.Unmarshal(fileBytes, &fileB)
			if err != nil {
				return false, err
			}

			if reflect.DeepEqual(src, fileB) {
				return true, nil
			}
		}

		var fs interface{}
		json.Unmarshal(fileBytes, &fs)
		if reflect.DeepEqual(source, fs) {
			return true, nil
		}

		return false, nil
	}

	if fmt.Sprint(source) == string(fileBytes) {
		return true, nil
	}

	return false, nil
}

var assertionFuncMap = map[string]struct{}{
	NOT:          {},
	LESSTHAN:     {},
	GREATERTHAN:  {},
	EQUALS:       {},
	EQUALSONFILE: {},
	IN:           {},
	JSONPATH:     {},
	XMLPATH:      {},
	REGEXP:       {},
	EXISTS:       {},
	CONTAINS:     {},
	RANGE:        {},
}

const (
	NOT          = "not"
	LESSTHAN     = "less_than"
	GREATERTHAN  = "greater_than"
	EQUALS       = "equals"
	IN           = "in"
	JSONPATH     = "json_path"
	XMLPATH      = "xml_path"
	REGEXP       = "regexp"
	EXISTS       = "exists"
	CONTAINS     = "contains"
	RANGE        = "range"
	EQUALSONFILE = "equals_on_file"
)
