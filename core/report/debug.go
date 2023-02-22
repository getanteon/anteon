package report

import (
	"encoding/json"
	"html"
	"net/http"
	"strings"

	"go.ddosify.com/ddosify/core/types"
)

type verboseRequest struct {
	Url     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
}

type verboseResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
}

type verboseHttpRequestInfo struct {
	StepId           uint16                  `json:"stepId"`
	StepName         string                  `json:"stepName"`
	Request          verboseRequest          `json:"request"`
	Response         verboseResponse         `json:"response"`
	Envs             map[string]interface{}  `json:"envs"`
	TestData         map[string]interface{}  `json:"testData"`
	FailedCaptures   map[string]string       `json:"failedCaptures"`
	FailedAssertions []types.FailedAssertion `json:"failedAssertions"`
	Error            string                  `json:"error"`
}

func ScenarioStepResultToVerboseHttpRequestInfo(sr *types.ScenarioStepResult) verboseHttpRequestInfo {
	var verboseInfo verboseHttpRequestInfo

	verboseInfo.StepId = sr.StepID
	verboseInfo.StepName = sr.StepName

	if sr.Err.Type == types.ErrorInvalidRequest {
		// could not prepare request at all
		verboseInfo.Error = sr.Err.Error()
		return verboseInfo
	}

	requestHeaders, requestBody, _ := decode(sr.ReqHeaders,
		sr.ReqBody)
	verboseInfo.Request = struct {
		Url     string            "json:\"url\""
		Method  string            "json:\"method\""
		Headers map[string]string "json:\"headers\""
		Body    interface{}       "json:\"body\""
	}{
		Url:     sr.Url,
		Method:  sr.Method,
		Headers: requestHeaders,
		Body:    requestBody,
	}

	if sr.Err.Type != "" {
		verboseInfo.Error = sr.Err.Error()
	} else {
		responseHeaders, responseBody, _ := decode(sr.RespHeaders,
			sr.RespBody)
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

	envs := make(map[string]interface{})
	testData := make(map[string]interface{})
	for key, val := range sr.UsableEnvs {
		if strings.HasPrefix(key, "data.") {
			testData[key] = val
		} else {
			envs[key] = val
		}
	}

	verboseInfo.Envs = envs
	verboseInfo.TestData = testData
	verboseInfo.FailedCaptures = sr.FailedCaptures
	verboseInfo.FailedAssertions = sr.FailedAssertions

	return verboseInfo
}

func decode(headers http.Header, byteBody []byte) (map[string]string, interface{}, error) {
	contentType := headers.Get("Content-Type")
	var reqBody interface{}

	hs := make(map[string]string, 0)
	for k, v := range headers {
		values := strings.Join(v, ";")
		hs[k] = values
	}

	if strings.Contains(contentType, "text/html") {
		unescapedHmtl := html.UnescapeString(string(byteBody))
		reqBody = unescapedHmtl
	} else if strings.Contains(contentType, "application/json") {
		err := json.Unmarshal(byteBody, &reqBody)
		if err != nil {
			reqBody = string(byteBody)
		}
	} else { // for remaining content-types return plain string
		// xml.Unmarshal() needs xml tags to decode encoded xml, we have no knowledge about the xml structure
		reqBody = string(byteBody)
	}

	return hs, reqBody, nil
}

func isVerboseInfoRequestEmpty(req verboseRequest) bool {
	if req.Url == "" && req.Method == "" && req.Headers == nil && req.Body == nil {
		return true
	}
	return false
}
