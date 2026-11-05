package report

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"go.ddosify.com/ddosify/v2/core/types"
)

type verboseRequest struct {
	Url     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
}

type verboseResponse struct {
	StatusCode   int               `json:"status_code"`
	Headers      map[string]string `json:"headers"`
	Body         interface{}       `json:"body"`
	ResponseTime int64             `json:"response_time"` // in milliseconds
	Stream       *verboseStream    `json:"stream,omitempty"`
}

// verboseStream is a streamed response's metrics, in milliseconds.
type verboseStream struct {
	TTFB          float64 `json:"ttfb_ms"`
	FirstEvent    float64 `json:"first_event_ms"`
	FirstToken    float64 `json:"first_token_ms"`
	HasFirstToken bool    `json:"has_first_token"`
	Total         float64 `json:"total_ms"`
	Events        int     `json:"events"`
	Comments      int     `json:"comments"`
	ErrorEvents   int     `json:"error_events"`
	GapMax        float64 `json:"gap_max_ms"`
	GapP50        float64 `json:"gap_p50_ms"`
	GapP95        float64 `json:"gap_p95_ms"`
	End           string  `json:"end"`
	EndReason     string  `json:"end_reason,omitempty"`
	Bytes         int64   `json:"bytes"`
	BodyTruncated bool    `json:"body_truncated"`

	LLM *types.LLMResult `json:"llm,omitempty"`
}

func toVerboseStream(r *types.StreamResult) *verboseStream {
	if r == nil {
		return nil
	}
	return &verboseStream{
		TTFB: round1(ms(r.TTFB)), FirstEvent: round1(ms(r.FirstEvent)), FirstToken: round1(ms(r.FirstToken)),
		HasFirstToken: r.HasFirstToken, Total: round1(ms(r.Total)), Events: r.Events, Comments: r.Comments,
		ErrorEvents: r.ErrorEvents, GapMax: round1(ms(r.GapMax)), GapP50: round1(ms(r.GapP50)),
		GapP95: round1(ms(r.GapP95)), End: r.End, EndReason: r.EndReason, Bytes: r.Bytes, BodyTruncated: r.BodyTruncated,
		LLM: r.LLM,
	}
}

type verboseHttpRequestInfo struct {
	StepId           uint16                  `json:"step_id"`
	StepName         string                  `json:"step_name"`
	Request          verboseRequest          `json:"request"`
	Response         verboseResponse         `json:"response"`
	Envs             map[string]interface{}  `json:"envs"`
	TestData         map[string]interface{}  `json:"test_data"`
	FailedCaptures   map[string]string       `json:"failed_captures"`
	FailedAssertions []types.FailedAssertion `json:"failed_assertions"`
	Error            string                  `json:"error"`
}

func ScenarioStepResultToVerboseHttpRequestInfo(sr *types.ScenarioStepResult) verboseHttpRequestInfo {
	var verboseInfo verboseHttpRequestInfo

	verboseInfo.StepId = sr.StepID
	verboseInfo.StepName = sr.StepName

	if sr.Err.Type == types.ErrorInvalidRequest {
		// could not prepare request at all
		verboseInfo.Error = redactText(sr.Err.Error())
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
		Url:     redactURL(sr.Url),
		Method:  sr.Method,
		Headers: requestHeaders,
		Body:    requestBody,
	}

	if sr.Err.Type != "" {
		verboseInfo.Error = redactText(sr.Err.Error())
	} else {
		responseHeaders, responseBody, _ := decode(sr.RespHeaders,
			sr.RespBody)
		// TODO what to do with error
		verboseInfo.Response = verboseResponse{
			StatusCode:   sr.StatusCode,
			Headers:      responseHeaders,
			Body:         responseBody,
			ResponseTime: sr.Duration.Milliseconds(),
			Stream:       toVerboseStream(sr.Stream),
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

const redacted = "[REDACTED]"

// Headers and query parameters that carry credentials; debug output shows them as [REDACTED].
var secretHeaders = map[string]bool{"Authorization": true, "Proxy-Authorization": true, "Cookie": true, "Set-Cookie": true,
	"X-Api-Key": true, "Api-Key": true}
var secretParams = map[string]bool{"api_key": true, "apikey": true, "key": true, "token": true, "access_token": true}

var secretParamRe = regexp.MustCompile(`(?i)\b(api_key|apikey|key|token|access_token)=[^&\s"']+`)

// redactText hides credential query values in a text such as an error that quotes a URL.
func redactText(s string) string {
	return secretParamRe.ReplaceAllString(s, "${1}="+redacted)
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.User != nil {
		u.User = url.User(u.User.Username())
	}
	if u.RawQuery == "" {
		return u.String()
	}
	q := u.Query()
	changed := false
	for k := range q {
		if secretParams[strings.ToLower(k)] {
			q[k] = []string{redacted}
			changed = true
		}
	}
	if changed {
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func decode(headers http.Header, byteBody []byte) (map[string]string, interface{}, error) {
	contentType := headers.Get("Content-Type")
	var reqBody interface{}

	hs := make(map[string]string, 0)
	for k, v := range headers {
		values := strings.Join(v, ",")
		if secretHeaders[http.CanonicalHeaderKey(k)] {
			values = redacted
		}
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
