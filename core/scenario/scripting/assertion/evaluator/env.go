package evaluator

import "net/http"

type AssertEnv struct {
	StatusCode   int64
	ResponseSize int64
	ResponseTime int64
	Body         string
	Headers      http.Header
	Variables    map[string]interface{}
}
