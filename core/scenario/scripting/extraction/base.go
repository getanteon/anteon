package extraction

import (
	"errors"
	"fmt"
	"net/http"

	"go.ddosify.com/ddosify/core/types"
)

func Extract(source interface{}, ce types.EnvCaptureConf) (val interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic")
			}
			val = nil
		}
	}()

	if source == nil {
		return "", ExtractionError{
			msg: "source is nil",
		}
	}

	switch ce.From {
	case types.Header:
		header := source.(http.Header)
		if ce.Key != nil { // key specified
			val = header.Get(*ce.Key)
			if val == "" {
				err = fmt.Errorf("http header %s not found", *ce.Key)
			} else if ce.RegExp != nil { // run regex for found value
				val, err = ExtractWithRegex(val, *ce.RegExp)
			}
		} else {
			err = fmt.Errorf("http header key not specified")
		}
	case types.Body:
		if ce.JsonPath != nil {
			val, err = ExtractFromJson(source, *ce.JsonPath)
		} else if ce.RegExp != nil {
			val, err = ExtractWithRegex(source, *ce.RegExp)
		} else if ce.Xpath != nil {
			val, err = ExtractFromXml(source, *ce.Xpath)
		}
	case types.Cookie:
		cookies := source.(map[string]*http.Cookie)
		if ce.CookieName != nil { // cookie name specified
			c, ok := cookies[*ce.CookieName]
			if !ok {
				err = fmt.Errorf("cookie %s not found", *ce.CookieName)
			} else {
				val = c.Value
			}
		} else {
			err = fmt.Errorf("cookie name not specified")
		}
	}

	if err != nil {
		return "", ExtractionError{
			msg:        fmt.Sprintf("%v", err),
			wrappedErr: err,
		}
	}
	return val, nil

}

func ExtractWithRegex(source interface{}, regexConf types.RegexCaptureConf) (val interface{}, err error) {
	re := regexExtractor{}
	re.Init(*regexConf.Exp)
	switch s := source.(type) {
	case []byte: // from response body
		return re.extractFromByteSlice(s, regexConf.No)
	case string: // from response header
		return re.extractFromString(s, regexConf.No)
	default:
		return "", fmt.Errorf("Unsupported type for extraction source")
	}
}

func ExtractFromJson(source interface{}, jsonPath string) (interface{}, error) {
	je := jsonExtractor{}
	switch s := source.(type) {
	case []byte: // from response body
		return je.extractFromByteSlice(s, jsonPath)
	case string: // from response header
		return je.extractFromString(s, jsonPath)
	default:
		return "", fmt.Errorf("Unsupported type for extraction source")
	}
}

func ExtractFromXml(source interface{}, xPath string) (interface{}, error) {
	xe := xmlExtractor{}
	switch s := source.(type) {
	case []byte: // from response body
		return xe.extractFromByteSlice(s, xPath)
	case string: // from response header
		return xe.extractFromString(s, xPath)
	default:
		return "", fmt.Errorf("Unsupported type for extraction source")
	}
}

type ExtractionError struct { // UnWrappable
	msg        string
	wrappedErr error
}

func (sc ExtractionError) Error() string {
	return sc.msg
}

func (sc ExtractionError) Unwrap() error {
	return sc.wrappedErr
}
