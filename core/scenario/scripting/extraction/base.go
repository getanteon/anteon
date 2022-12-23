package extraction

import (
	"fmt"
	"net/http"

	"go.ddosify.com/ddosify/core/types"
)

func Extract(source interface{}, ce types.EnvCaptureConf) (interface{}, error) {
	var val interface{}
	var err error
	switch ce.From {
	case types.Header:
		header := source.(http.Header)
		if ce.Key != nil { // key specified
			val = header.Get(*ce.Key)
			if val == "" {
				err = fmt.Errorf("Http Header %s not found", *ce.Key)
			} else {
				if ce.RegExp != nil { // run regex for found value
					re := CreateRegexExtractor(*ce.RegExp.Exp)
					val, err = re.extractFromString(val.(string), ce.RegExp.No)
				}
			}
		}
	case types.Body:
		if ce.JsonPath != nil {
			val, err = extractFromJson(source, *ce.JsonPath)
		} else if ce.RegExp != nil {
			re := CreateRegexExtractor(*ce.RegExp.Exp)
			switch source.(type) {
			case string:
				val, err = re.extractFromString(source.(string), ce.RegExp.No)
			case []byte:
				val, err = re.extractFromByteSlice(source.([]byte), ce.RegExp.No)
			}
		} else if ce.Xpath != nil {
			switch source.(type) {
			case []byte:
				val, err = extractFromXml(source, *ce.Xpath)
			}

		}
	}

	if err != nil {
		return "", EnvironmentCaptureError{
			msg:        fmt.Sprintf("env capture failed for %s, %v", ce.Name, err),
			wrappedErr: err,
		}
	}
	return val, nil

}

func extractFromJson(source interface{}, jsonPath string) (interface{}, error) {
	je := JsonExtractor{}
	switch s := source.(type) {
	case []byte: // from response body
		return je.extractFromByteSlice(s, jsonPath)
	case string: // from response header
		return je.extractFromString(s, jsonPath)
	default:
		return "", fmt.Errorf("Unsupported type for extraction source")
	}
}

func extractFromXml(source interface{}, xPath string) (interface{}, error) {
	xe := XmlExtractor{}
	switch s := source.(type) {
	case []byte: // from response body
		return xe.extractFromByteSlice(s, xPath)
	default:
		return "", fmt.Errorf("Unsupported type for extraction source")
	}
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
