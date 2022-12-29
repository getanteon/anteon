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
	r   *regexp.Regexp
	jr  *regexp.Regexp
	dr  *regexp.Regexp
	jdr *regexp.Regexp
}

func (ei *EnvironmentInjector) Init() {
	ei.r = regexp.MustCompile(regex.EnvironmentVariableRegex)
	ei.jr = regexp.MustCompile(regex.JsonEnvironmentVarRegex)
	ei.dr = regexp.MustCompile(regex.DynamicVariableRegex)
	ei.jdr = regexp.MustCompile(regex.JsonDynamicVariableRegex)
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

func truncateTag(tag string, rx string) string {
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

func (ei *EnvironmentInjector) InjectEnv(text string, envs map[string]interface{}) (string, error) {
	errors := []error{}

	injectStrFunc := func(s string) string {
		var truncated string
		var env interface{}
		var err error

		truncated = truncateTag(string(s), regex.EnvironmentVariableRegex)

		env, ok := envs[truncated]
		if !ok {
			err = fmt.Errorf("env not found")
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

		truncated = truncateTag(string(s), regex.JsonEnvironmentVarRegex)

		env, ok := envs[truncated]
		if !ok {
			err = fmt.Errorf("env not found")
		}

		if err == nil {
			mEnv, err := json.Marshal(env)
			if err == nil {
				return mEnv
			}
		}

		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}

	// json injection
	bText := []byte(text)
	if json.Valid(bText) {
		if ei.jr.Match(bText) {
			replacedBytes := ei.jr.ReplaceAllFunc(bText, injectToJsonByteFunc)
			return string(replacedBytes), nil
		}
	}

	// string injection
	replaced := ei.r.ReplaceAllStringFunc(text, injectStrFunc)
	if len(errors) == 0 {
		return replaced, nil
	}

	return replaced, unifyErrors(errors)

}

func (ei *EnvironmentInjector) InjectDynamic(text string) (string, error) {
	errors := []error{}

	injectStrFunc := func(s string) string {
		var truncated string
		var env interface{}
		var err error

		truncated = truncateTag(string(s), regex.DynamicVariableRegex)
		env, err = ei.getFakeData(truncated)

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

		truncated = truncateTag(string(s), regex.JsonDynamicVariableRegex)
		env, err = ei.getFakeData(truncated)

		if err == nil {
			mEnv, err := json.Marshal(env)
			if err == nil {
				return mEnv
			}
		}
		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}

	// json injection
	bText := []byte(text)
	if json.Valid(bText) {
		if ei.jr.Match(bText) {
			replacedBytes := ei.jdr.ReplaceAllFunc(bText, injectToJsonByteFunc)
			return string(replacedBytes), nil
		}
	}

	// string injection
	replaced := ei.dr.ReplaceAllStringFunc(text, injectStrFunc)
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
