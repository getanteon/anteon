package injection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"go.ddosify.com/ddosify/core/types/regex"
)

func (ei *EnvironmentInjector) InjectDynamicIntoBuffer(text string, buffer *bytes.Buffer) (*bytes.Buffer, error) {
	errors := []error{}
	if buffer == nil {
		buffer = &bytes.Buffer{}
	}

	injectStrFunc := getInjectStrFunc(regex.DynamicVariableRegex, ei, nil, errors)
	injectToJsonByteFunc := getInjectJsonFunc(regex.JsonDynamicVariableRegex, ei, nil, errors)

	// json injection
	bText := StringToBytes(text)
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

func (ei *EnvironmentInjector) InjectDynamic(text string) (string, error) {
	errors := []error{}

	injectStrFunc := getInjectStrFunc(regex.DynamicVariableRegex, ei, nil, errors)
	injectToJsonByteFunc := getInjectJsonFunc(regex.JsonDynamicVariableRegex, ei, nil, errors)

	// json injection
	bText := StringToBytes(text)
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
