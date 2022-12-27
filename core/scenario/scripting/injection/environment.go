package injection

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"go.ddosify.com/ddosify/core/types/regex"
)

type EnvironmentInjector struct {
	r                    *regexp.Regexp
	jr                   *regexp.Regexp
	dr                   *regexp.Regexp
	jdr                  *regexp.Regexp
	getInjectable        func(string) (interface{}, error)
	getDynamicInjectable func(string) (interface{}, error)
}

func (ei *EnvironmentInjector) Init() {
	ei.r = regexp.MustCompile(regex.EnvironmentVariableRegex)
	ei.jr = regexp.MustCompile(regex.JsonEnvironmentVarRegex)
	ei.dr = regexp.MustCompile(regex.DynamicVariableRegex)
	ei.jdr = regexp.MustCompile(regex.JsonDynamicVariableRegex)
	ei.getDynamicInjectable = ei.getFakeData
}

func (ei *EnvironmentInjector) SetInjectableFunc(getInjectable func(string) (interface{}, error)) {
	ei.getInjectable = getInjectable
}

func (ei *EnvironmentInjector) getFakeData(key string) (interface{}, error) {
	var fakeFunc interface{}
	var keyExists bool
	if fakeFunc, keyExists = dynamicFakeDataMap[key]; !keyExists {
		return nil, fmt.Errorf("%s is not a valid dynamic variable", key)
	}

	res := reflect.ValueOf(fakeFunc).Call(nil)[0].Interface()
	return res, nil
}

func (ei *EnvironmentInjector) Inject(text string, dynamic bool) (string, error) {
	errors := []error{}

	truncateTag := func(tag string, rx string) string {
		if strings.EqualFold(rx, regex.EnvironmentVariableRegex) {
			return tag[2 : len(tag)-2] // {{...}}
		} else if strings.EqualFold(rx, regex.JsonEnvironmentVarRegex) {
			return tag[3 : len(tag)-3] // "{{...}}"
		} else if strings.EqualFold(rx, regex.DynamicVariableRegex) {
			return tag[3 : len(tag)-2] // {{_...}}
		} else if strings.EqualFold(rx, regex.JsonDynamicVariableRegex) {
			return tag[4 : len(tag)-3] //"{{_...}}"
		}
		return ""
	}
	injectStrFunc := func(s string) string {
		var truncated string
		var env interface{}
		var err error
		if dynamic {
			truncated = truncateTag(string(s), regex.DynamicVariableRegex)
			env, err = ei.getDynamicInjectable(truncated)
		} else {
			truncated = truncateTag(string(s), regex.EnvironmentVariableRegex)
			env, err = ei.getInjectable(truncated)
		}

		if err == nil {
			switch env.(type) {
			case string:
				return env.(string)
			case []byte:
				return string(env.([]byte))
			case int64:
				return fmt.Sprintf("%d", env)
			case int:
				return fmt.Sprintf("%d", env)
			case float64:
				return fmt.Sprintf("%g", env) // %g it is the smallest number of digits necessary to identify the value uniquely
			case bool:
				return fmt.Sprintf("%t", env)
			default:
				return fmt.Sprint(env)
			}
		}
		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}
	injectToJsonByteFunc := func(s []byte) []byte {
		var truncated string
		var env interface{}
		var err error
		if dynamic {
			truncated = truncateTag(string(s), regex.JsonDynamicVariableRegex)
			env, err = ei.getDynamicInjectable(truncated)
		} else {
			truncated = truncateTag(string(s), regex.JsonEnvironmentVarRegex)
			env, err = ei.getInjectable(truncated)
		}
		mEnv, err := json.Marshal(env)
		if err == nil {
			return mEnv
		}

		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}

	var jsonRegexp *regexp.Regexp
	var strRexexp *regexp.Regexp
	if dynamic {
		jsonRegexp = ei.jdr
		strRexexp = ei.dr
	} else {
		jsonRegexp = ei.jr
		strRexexp = ei.r
	}

	// json injection
	bText := []byte(text)
	if json.Valid(bText) {
		if ei.jr.Match(bText) {
			replacedBytes := jsonRegexp.ReplaceAllFunc(bText, injectToJsonByteFunc)
			return string(replacedBytes), nil
		}
	}

	// string injection
	replaced := strRexexp.ReplaceAllStringFunc(text, injectStrFunc)
	if len(errors) == 0 {
		return replaced, nil
	}

	return replaced, unifyErrors(errors)

}

func unifyErrors(errors []error) error {
	sb := strings.Builder{}

	for _, err := range errors {
		sb.WriteString(err.Error())
	}

	return fmt.Errorf("%s", sb.String())
}
