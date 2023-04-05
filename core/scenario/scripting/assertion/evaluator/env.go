package evaluator

import "net/http"

type AssertEnv struct {
	StatusCode   int64
	ResponseSize int64
	ResponseTime int64 // in ms
	Body         string
	Headers      http.Header
	Variables    map[string]interface{}
	Cookies      map[string]*http.Cookie // cookies sent by the server, name -> cookie
}
