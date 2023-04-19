package injection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.ddosify.com/ddosify/core/types/regex"
)

type EnvironmentInjector struct {
	r   *regexp.Regexp
	jr  *regexp.Regexp
	dr  *regexp.Regexp
	jdr *regexp.Regexp
	mu  sync.Mutex
}

func (ei *EnvironmentInjector) Init() {
	ei.r = regexp.MustCompile(regex.EnvironmentVariableRegex)
	ei.jr = regexp.MustCompile(regex.JsonEnvironmentVarRegex)
	ei.dr = regexp.MustCompile(regex.DynamicVariableRegex)
	ei.jdr = regexp.MustCompile(regex.JsonDynamicVariableRegex)
	rand.Seed(time.Now().UnixNano())
}

func (ei *EnvironmentInjector) getFakeData(key string) (interface{}, error) {
	var fakeFunc interface{}
	var keyExists bool
	if fakeFunc, keyExists = dynamicFakeDataMap[key]; !keyExists {
		return nil, fmt.Errorf("%s is not a valid dynamic variable", key)
	}

	preventRaceOnRandomFunc := func(fakeFunc interface{}) interface{} {
		ei.mu.Lock()
		defer ei.mu.Unlock()
		return reflect.ValueOf(fakeFunc).Call(nil)[0].Interface()
	}

	return preventRaceOnRandomFunc(fakeFunc), nil
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
		env, err = ei.getEnv(envs, truncated)

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
		env, err = ei.getEnv(envs, truncated)

		if err == nil {
			mEnv, err := json.Marshal(env)
			if err == nil {
				return mEnv
			}
		}

		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps: %v", truncated, err))
		return s
	}

	// json injection
	bText := []byte(text)
	if json.Valid(bText) {
		replacedBytes := ei.jr.ReplaceAllFunc(bText, injectToJsonByteFunc)
		if len(errors) == 0 {
			text = string(replacedBytes)
		} else {
			return "", unifyErrors(errors)
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

// expects an empty buffer and writes the result to it
func (ei *EnvironmentInjector) InjectEnvIntoBuffer(text string, envs map[string]interface{}, buffer *bytes.Buffer) (*bytes.Buffer, error) {
	// TODO: if did not inject anything, write text to buffer
	errors := []error{}
	if buffer == nil {
		buffer = &bytes.Buffer{}
	}
	injectStrFunc := func(s string) string {
		var truncated string
		var env interface{}
		var err error

		truncated = truncateTag(string(s), regex.EnvironmentVariableRegex)
		env, err = ei.getEnv(envs, truncated)

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
		env, err = ei.getEnv(envs, truncated)

		if err == nil {
			mEnv, err := json.Marshal(env)
			if err == nil {
				return mEnv
			}
		}

		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps: %v", truncated, err))
		return []byte(s)
	}

	// json injection
	bText := []byte(text)
	if json.Valid(bText) {
		foundMatches := ei.jr.FindAll(bText, -1)
		args := make([]string, 0)
		for _, match := range foundMatches {
			args = append(args, string(match))
			args = append(args, string(injectToJsonByteFunc(match)))
		}

		replacer := strings.NewReplacer(args...)
		_, err := replacer.WriteString(buffer, text)
		if err != nil {
			return nil, err
		}
		if len(errors) == 0 {
			text = buffer.String()
		} else {
			return nil, unifyErrors(errors)
		}
	}

	// continue with string injection
	// string injection
	foundMatches := ei.r.FindAllString(text, -1)
	if len(foundMatches) == 0 {
		return buffer, nil
	} else {
		buffer.Reset()

		args := make([]string, 0)
		for _, match := range foundMatches {
			args = append(args, match)
			args = append(args, injectStrFunc(match))
		}
		replacer := strings.NewReplacer(args...)
		_, err := replacer.WriteString(buffer, text)
		if err != nil {
			return nil, err
		}
	}

	if len(errors) == 0 {
		return buffer, nil
	}

	return nil, unifyErrors(errors)
}

func (ei *EnvironmentInjector) InjectDynamicIntoBuffer(text string, buffer *bytes.Buffer) (*bytes.Buffer, error) {
	errors := []error{}
	if buffer == nil {
		buffer = &bytes.Buffer{}
	}

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
			foundMatches := ei.jdr.FindAll(bText, -1)
			args := make([]string, 0)
			for _, match := range foundMatches {
				args = append(args, string(match))
				args = append(args, string(injectToJsonByteFunc(match)))
			}

			replacer := strings.NewReplacer(args...)
			_, err := replacer.WriteString(buffer, text)
			if err != nil {
				return nil, err
			}
			return buffer, nil
		}
	}

	// string injection
	foundMatches := ei.dr.FindAllString(text, -1)
	if len(foundMatches) == 0 {
		return buffer, nil
	} else {
		buffer.Reset()
		args := make([]string, 0)
		for _, match := range foundMatches {
			args = append(args, match)
			args = append(args, injectStrFunc(match))
		}
		replacer := strings.NewReplacer(args...)

		_, err := replacer.WriteString(buffer, text)
		if err != nil {
			return nil, err
		}
	}

	if len(errors) == 0 {
		return buffer, nil
	}

	return nil, unifyErrors(errors)

}

func (ei *EnvironmentInjector) getEnv(envs map[string]interface{}, key string) (interface{}, error) {
	var err error
	var val interface{}

	pickRand := strings.HasPrefix(key, "rand(") && strings.HasSuffix(key, ")")
	if pickRand {
		key = key[5 : len(key)-1]
	}

	var exists bool
	val, exists = envs[key]

	isOsEnv := strings.HasPrefix(key, "$")

	if isOsEnv {
		varName := key[1:]
		val, exists = os.LookupEnv(varName)
	}

	if !exists {
		err = fmt.Errorf("env not found")
	}

	if pickRand {
		switch v := val.(type) {
		case []interface{}:
			val = v[rand.Intn(len(v))]
		case []string:
			val = v[rand.Intn(len(v))]
		case []bool:
			val = v[rand.Intn(len(v))]
		case []int:
			val = v[rand.Intn(len(v))]
		case []float64:
			val = v[rand.Intn(len(v))]
		default:
			err = fmt.Errorf("can not perform rand() operation on non-array value")
		}
	}

	return val, err
}

func unifyErrors(errors []error) error {
	sb := strings.Builder{}

	for _, err := range errors {
		sb.WriteString(err.Error())
	}

	return fmt.Errorf("%s", sb.String())
}
