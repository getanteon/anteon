package injection

import (
	"encoding/json"
	"fmt"
	"reflect"

	"go.ddosify.com/ddosify/core/types/regex"
)

func (ei *EnvironmentInjector) InjectDynamic(text string) (string, error) {
	errors := []error{}

	injectStrFunc := getInjectStrFunc(regex.DynamicVariableRegex, ei, nil, &errors)
	injectToJsonByteFunc := getInjectJsonFunc(regex.JsonDynamicVariableRegex, ei, nil, &errors)

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
