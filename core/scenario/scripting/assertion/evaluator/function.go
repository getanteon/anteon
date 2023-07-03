package evaluator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"time"

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

// assumed given array is sorted
var percentile = func(arr []int64, num int) (int64, error) {
	if len(arr) == 0 {
		return 0, fmt.Errorf("empty input array on percentile func")
	}

	index := int(math.Ceil(float64(len(arr)*num)/100)) - 1

	if index < 0 {
		index = 0
	}

	return arr[index], nil
}

var min = func(arr []int64) (int64, error) {
	if len(arr) == 0 {
		return 0, fmt.Errorf("empty input array on min func")
	}
	min := arr[0]

	for _, i := range arr {
		if min > i {
			min = i
		}
	}

	return min, nil
}

var max = func(arr []int64) (int64, error) {
	if len(arr) == 0 {
		return 0, fmt.Errorf("empty input array on max func")
	}
	max := arr[0]

	for _, i := range arr {
		if max < i {
			max = i
		}
	}

	return max, nil
}

var avg = func(arr []int64) (float64, error) {
	if len(arr) == 0 {
		return 0, fmt.Errorf("empty input array on avg func")
	}
	var total int64

	for _, i := range arr {
		total += i
	}

	return float64(total) / float64(len(arr)), nil
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

var timeF = func(t string) (time.Time, error) {
	res, err := time.Parse(time.RFC1123, t)
	if err != nil {
		return time.Time{}, err
	}
	return res, nil
}

var rangeF = func(x float64, low float64, hi float64) bool {
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
	MIN:          {},
	MAX:          {},
	AVG:          {},
	P99:          {},
	P98:          {},
	P95:          {},
	P90:          {},
	P80:          {},
	TIME:         {},
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
	TIME         = "time"

	MIN = "min"
	MAX = "max"
	AVG = "avg"
	P99 = "p99"
	P98 = "p98"
	P95 = "p95"
	P90 = "p90"
	P80 = "p80"
)
