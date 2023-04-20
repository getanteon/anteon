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
	"unsafe"

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

	injectStrFunc := getInjectStrFunc(regex.EnvironmentVariableRegex, ei, envs, errors)
	injectToJsonByteFunc := getInjectJsonFunc(regex.JsonEnvironmentVarRegex, ei, envs, errors)

	// json injection
	bText := StringToBytes(text)
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

// expects an empty buffer and writes the result to it
func (ei *EnvironmentInjector) InjectEnvIntoBuffer(text string, envs map[string]interface{}, buffer *bytes.Buffer) (*bytes.Buffer, error) {
	// TODO: if did not inject anything, write text to buffer
	errors := []error{}
	if buffer == nil {
		buffer = &bytes.Buffer{}
	}
	injectStrFunc := getInjectStrFunc(regex.EnvironmentVariableRegex, ei, envs, errors)
	injectToJsonByteFunc := getInjectJsonFunc(regex.JsonEnvironmentVarRegex, ei, envs, errors)

	// json injection
	bText := StringToBytes(text)
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

func StringToBytes(s string) (b []byte) {
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sliceHeader.Data = stringHeader.Data
	sliceHeader.Len = len(s)
	sliceHeader.Cap = len(s)
	return b
}

func getInjectStrFunc(rx string,
	ei *EnvironmentInjector,
	envs map[string]interface{},
	errors []error,
) func(string) string {
	return func(s string) string {
		var truncated string
		var env interface{}
		var err error

		truncated = truncateTag(string(s), rx)

		if rx == regex.EnvironmentVariableRegex {
			env, err = ei.getEnv(envs, truncated)
		} else if rx == regex.DynamicVariableRegex {
			env, err = ei.getFakeData(truncated)
		} else {
			// this should never happen
			panic("invalid regex")
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
}

func getInjectJsonFunc(rx string,
	ei *EnvironmentInjector,
	envs map[string]interface{},
	errors []error,
) func(s []byte) []byte {
	return func(s []byte) []byte {
		var truncated string
		var env interface{}
		var err error

		truncated = truncateTag(string(s), rx)
		if rx == regex.JsonDynamicVariableRegex {
			env, err = ei.getFakeData(truncated)
		} else if rx == regex.JsonEnvironmentVarRegex {
			env, err = ei.getEnv(envs, truncated)
		} else {
			// this should never happen
			panic("invalid regex")
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
}
