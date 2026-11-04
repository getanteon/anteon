package injection

import (
	"encoding/json"
	"fmt"
	"go.ddosify.com/ddosify/v2/core/hooks"
	"reflect"

	"go.ddosify.com/ddosify/v2/core/types/regex"
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
		if fn, ok := hooks.Variable(key); ok {
			return fn(), nil
		}
		return nil, fmt.Errorf("%s is not a valid dynamic variable", key)
	}

	// one faker serves every injector, and its rand.Rand is not safe for concurrent use
	preventRaceOnRandomFunc := func(fakeFunc interface{}) interface{} {
		fakerMu.Lock()
		defer fakerMu.Unlock()
		return reflect.ValueOf(fakeFunc).Call(nil)[0].Interface()
	}

	return preventRaceOnRandomFunc(fakeFunc), nil
}
