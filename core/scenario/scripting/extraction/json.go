package extraction

import (
	"encoding/json"

	"github.com/tidwall/gjson"
)

type JsonExtractor struct {
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
		return result.Raw, nil
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
		j := map[string]interface{}{}
		json.Unmarshal([]byte(result.Raw), &j)
		return j, nil
	default:
		return "", nil
	}
}
