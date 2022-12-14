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
	requestHeaders, requestBody, _ := decode(sr.DebugInfo["requestHeaders"].(http.Header),
		sr.DebugInfo["requestBody"].([]byte))
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
		responseHeaders, responseBody, _ := decode(sr.DebugInfo["responseHeaders"].(http.Header),
			sr.DebugInfo["responseBody"].([]byte))
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

func decode(headers http.Header, byteBody []byte) (map[string]string, interface{}, error) {
	contentType := headers.Get("Content-Type")
	var reqBody interface{}

	hs := make(map[string]string, 0)
	for k, v := range headers {
		values := strings.Join(v, ",")
		hs[k] = values
	}

	if strings.Contains(contentType, "text/html") {
		unescapedHmtl := html.UnescapeString(string(byteBody))
		reqBody = unescapedHmtl
	} else if strings.Contains(contentType, "application/json") {
		err := json.Unmarshal(byteBody, &reqBody)
		if err != nil {
			return hs, reqBody, err
		}
	} else { // for remaining content-types return plain string
		// xml.Unmarshal() needs xml tags to decode encoded xml, we have no knowledge about the xml structure
		reqBody = string(byteBody)
	}

	return hs, reqBody, nil
}
