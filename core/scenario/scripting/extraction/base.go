package extraction

import (
	"fmt"
	"net/http"

	"go.ddosify.com/ddosify/core/types"
)

func ExtractAndPopulate(source interface{}, ce types.CapturedEnv, extractedVars map[string]interface{}) error {
	je := JsonExtractor{}
	f := func(source interface{}, jsonPath string) (interface{}, error) {
		switch s := source.(type) {
		case []byte: // from response body
			return je.ExtractFromByteSlice(s, jsonPath)
		case string: // from response header
			return je.ExtractFromString(s, jsonPath)
		default:
			return "", fmt.Errorf("Unsupported type for extraction source")
		}
	}

	// from header
	if ce.From == "header" {
		val, err := extractFromHttpHeader(source.(http.Header), ce.Key)
		if err != nil {
			return err
		}
		extractedVars[ce.Name] = val
	}

	// from body
	if ce.From == "body" {
		if ce.JsonPath != "" {
			val, err := f(source, ce.JsonPath)
			if err != nil {
				return err
			}
			extractedVars[ce.Name] = val
		}
		// TODOcorr: add xpath
	}

	return nil
}

func extractFromHttpHeader(header http.Header, key string) (string, error) {
	if val := header.Get(key); val != "" {
		return val, nil
	}
	return "", fmt.Errorf("Http Header %s not found", key)
}
