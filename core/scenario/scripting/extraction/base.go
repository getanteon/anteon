package extraction

import (
	"fmt"
	"net/http"
	"strings"

	"go.ddosify.com/ddosify/core/types"
)

func ExtractAndPopulate(source interface{}, ce types.EnvCaptureConf, extractedVars map[string]interface{}) error {
	var val interface{}
	var err error
	switch ce.From {
	case types.Header:
		header := source.(http.Header)
		if ce.Key != nil { // key specified
			if val = header.Get(*ce.Key); val != "" {
				err = fmt.Errorf("Http Header %s not found", *ce.Key)
			}
		} else if ce.RegExp != nil { // run regex for each key and value
			val, err = extractFromHttpHeader(header, ce.RegExp)
		}
	case types.Body:
		if ce.JsonPath != nil {
			val, err = extractFromJson(source, *ce.JsonPath)
		} else if ce.RegExp != nil {
			// TODOcorr
		}

		// TODOcorr: add xpath
	}

	if err != nil {
		return EnvironmentCaptureError{
			msg:        fmt.Sprintf("env capture failed for %s, %v", ce.Name, err),
			wrappedErr: err,
		}
	}

	extractedVars[ce.Name] = val
	return nil
}

func extractFromJson(source interface{}, jsonPath string) (interface{}, error) {
	je := JsonExtractor{}
	switch s := source.(type) {
	case []byte: // from response body
		return je.ExtractFromByteSlice(s, jsonPath)
	case string: // from response header
		return je.ExtractFromString(s, jsonPath)
	default:
		return "", fmt.Errorf("Unsupported type for extraction source")
	}
}

func extractFromHttpHeader(header http.Header, regexConf *types.RegexCaptureConf) (interface{}, error) {
	var match interface{}
	var err error
	re := CreateRegexExtractor(*regexConf.Exp)
	for k, v := range header {
		if match, err = re.ExtractFromString(k, regexConf.No); err == nil { // key match
			break
		} else if match, err = re.ExtractFromString(strings.Join(v, " "), regexConf.No); err == nil { // value match
			break
		}
	}
	return match, err
}

type EnvironmentCaptureError struct { // UnWrappable
	msg        string
	wrappedErr error
}

func (sc EnvironmentCaptureError) Error() string {
	return sc.msg
}

func (sc EnvironmentCaptureError) Unwrap() error {
	return sc.wrappedErr
}
