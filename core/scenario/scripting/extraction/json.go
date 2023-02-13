package extraction

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

type jsonExtractor struct {
}

var unmarshalJsonCapture = func(result gjson.Result) (interface{}, error) {
	bRaw := []byte(result.Raw)
	if result.IsObject() {
		jObject := map[string]interface{}{}
		err := json.Unmarshal(bRaw, &jObject)
		if err == nil {
			return jObject, err
		}
	}

	if result.IsArray() {
		jInterfaceSlice := []interface{}{}
		err := json.Unmarshal(bRaw, &jInterfaceSlice)
		if err == nil {
			return jInterfaceSlice, err
		}
	}

	if result.IsBool() {
		jBool := false
		err := json.Unmarshal(bRaw, &jBool)
		if err == nil {
			return jBool, err
		}
	}

	return nil, fmt.Errorf("json could not be unmarshaled")
}

func (je jsonExtractor) extractFromString(source string, jsonPath string) (interface{}, error) {
	result := gjson.Get(source, jsonPath)

	// path not found
	if result.Raw == "" && result.Type == gjson.Null {
		return "", fmt.Errorf("no match for the json path: %s", jsonPath)
	}

	switch result.Type {
	case gjson.String:
		return result.String(), nil
	case gjson.Null:
		return nil, nil
	case gjson.False:
		return false, nil
	case gjson.Number:
		number := result.String()
		if strings.Contains(number, ".") { // float
			return result.Float(), nil
		}
		return result.Int(), nil
	case gjson.True:
		return true, nil
	case gjson.JSON:
		return unmarshalJsonCapture(result)
	default:
		return "", nil
	}
}

func (je jsonExtractor) extractFromByteSlice(source []byte, jsonPath string) (interface{}, error) {
	result := gjson.GetBytes(source, jsonPath)

	// path not found
	if result.Raw == "" && result.Type == gjson.Null {
		return "", fmt.Errorf("no match for the json path: %s", jsonPath)
	}

	switch result.Type {
	case gjson.String:
		return result.String(), nil
	case gjson.Null:
		return nil, nil
	case gjson.False:
		return false, nil
	case gjson.Number:
		number := result.String()
		if strings.Contains(number, ".") { // float
			return result.Float(), nil
		}
		return result.Int(), nil
	case gjson.True:
		return true, nil
	case gjson.JSON:
		return unmarshalJsonCapture(result)
	default:
		return "", nil
	}
}
