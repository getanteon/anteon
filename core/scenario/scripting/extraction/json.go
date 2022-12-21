package extraction

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
)

type JsonExtractor struct {
}

var unmarshalJsonCapture = func(result gjson.Result) (interface{}, error) {
	if result.IsObject() {
		jObject := map[string]interface{}{}
		err := json.Unmarshal([]byte(result.Raw), &jObject)
		if err == nil {
			return jObject, err
		}
	}

	if result.IsArray() {
		jStrSlice := []string{}
		err := json.Unmarshal([]byte(result.Raw), &jStrSlice)
		if err == nil {
			return jStrSlice, err
		}

		jIntSlice := []int{}
		err = json.Unmarshal([]byte(result.Raw), &jIntSlice)
		if err == nil {
			return jIntSlice, err
		}
	}

	if result.IsBool() {
		jBool := false
		err := json.Unmarshal([]byte(result.Raw), &jBool)
		if err == nil {
			return jBool, err
		}
	}

	return nil, fmt.Errorf("json could not be unmarshaled")
}

func (je JsonExtractor) ExtractFromString(source string, jsonPath string) (interface{}, error) {
	result := gjson.Get(source, jsonPath)
	// TODOcorr: write test for below

	switch result.Type {
	case gjson.String:
		return result.String(), nil
	case gjson.Null:
		return nil, nil
	case gjson.False:
		return false, nil
	case gjson.Number:
		return result.Float(), nil // TODO: check for int
	case gjson.True:
		return true, nil
	case gjson.JSON:
		return unmarshalJsonCapture(result)
	default:
		return "", nil
	}
}

func (je JsonExtractor) ExtractFromByteSlice(source []byte, jsonPath string) (interface{}, error) {
	result := gjson.GetBytes(source, jsonPath)

	switch result.Type {
	case gjson.String:
		return result.String(), nil
	case gjson.Null:
		return nil, nil
	case gjson.False:
		return false, nil
	case gjson.Number:
		return result.Float(), nil // TODO: check for int
	case gjson.True:
		return true, nil
	case gjson.JSON:
		return unmarshalJsonCapture(result)
	default:
		return "", nil
	}
}
