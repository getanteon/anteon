package report

import (
	"encoding/json"
	"html"
	"net/http"
	"strings"

	"go.ddosify.com/ddosify/core/types"
)

type verboseHttpRequestInfo struct {
	StepId   uint16 `json:"stepId"`
	StepName string `json:"stepName"`
	Request  struct {
		Url     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
		Body    interface{}       `json:"body"`
	} `json:"request"`
	Response struct {
		StatusCode int               `json:"statusCode"`
		Headers    map[string]string `json:"headers"`
		Body       interface{}       `json:"body"`
	} `json:"response"`
	Error string `json:"error"`
}

func ScenarioStepResultToVerboseHttpRequestInfo(sr *types.ScenarioStepResult) verboseHttpRequestInfo {
	var verboseInfo verboseHttpRequestInfo

	verboseInfo.StepId = sr.StepID
	verboseInfo.StepName = sr.StepName
	requestHeaders, requestBody, _ := decodeRequest(sr)
	verboseInfo.Request = struct {
		Url     string            "json:\"url\""
		Method  string            "json:\"method\""
		Headers map[string]string "json:\"headers\""
		Body    interface{}       "json:\"body\""
	}{
		Url:     sr.DebugInfo["url"].(string),
		Method:  sr.DebugInfo["method"].(string),
		Headers: requestHeaders,
		Body:    requestBody,
	}

	if sr.Err.Type != "" {
		verboseInfo.Error = sr.Err.Error()
	} else {
		responseHeaders, responseBody, _ := decodeResponse(sr)
		// TODO what to do with error
		verboseInfo.Response = struct {
			StatusCode int               "json:\"statusCode\""
			Headers    map[string]string "json:\"headers\""
			Body       interface{}       `json:"body"`
		}{
			StatusCode: sr.StatusCode,
			Headers:    responseHeaders,
			Body:       responseBody,
		}
	}

	return verboseInfo
}

func decodeRequest(sr *types.ScenarioStepResult) (map[string]string, interface{}, error) {
	requestHeaders := make(map[string]string, 0)
	for k, v := range sr.DebugInfo["requestHeaders"].(http.Header) {
		values := strings.Join(v, ",")
		requestHeaders[k] = values
	}

	contentType := sr.DebugInfo["requestHeaders"].(http.Header).Get("content-type")
	byteBody := sr.DebugInfo["requestBody"].([]byte)

	var respBody interface{}
	if strings.Contains(contentType, "text/html") {
		unescapedHmtl := html.UnescapeString(string(byteBody))
		respBody = unescapedHmtl
	} else if strings.Contains(contentType, "application/json") {
		err := json.Unmarshal(byteBody, &respBody)
		if err != nil {
			return requestHeaders, respBody, err
		}
	} else if strings.Contains(contentType, "application/xml") {
		// xml.Unmarshal() needs xml tags to decode encoded xml, we have no knowledge about the xml structure
		respBody = string(byteBody)
	} else { // for remaining content-types return plain string
		respBody = string(byteBody)
	}

	return requestHeaders, respBody, nil

}

func decodeResponse(sr *types.ScenarioStepResult) (map[string]string, interface{}, error) {
	responseHeaders := make(map[string]string, 0)
	for k, v := range sr.DebugInfo["responseHeaders"].(http.Header) {
		values := strings.Join(v, ",")
		responseHeaders[k] = values
	}

	contentType := sr.DebugInfo["responseHeaders"].(http.Header).Get("content-type")
	byteBody := sr.DebugInfo["responseBody"].([]byte)

	var respBody interface{}
	if strings.Contains(contentType, "text/html") {
		unescapedHmtl := html.UnescapeString(string(byteBody))
		respBody = unescapedHmtl
	} else if strings.Contains(contentType, "application/json") {
		err := json.Unmarshal(byteBody, &respBody)
		if err != nil {
			return responseHeaders, respBody, err
		}
	} else if strings.Contains(contentType, "application/xml") {
		// xml.Unmarshal() needs xml tags to decode encoded xml, we have no knowledge about the xml structure
		respBody = string(byteBody)
	} else { // for remaining content-types return plain string
		respBody = string(byteBody)
	}

	return responseHeaders, respBody, nil
}
