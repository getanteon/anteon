package types

import "fmt"

const (
	// Type
	ErrorProxy  = "proxyError"
	ErrorConn   = "connectionError"
	ErrorUnkown = "unkownError"

	// Reasons
	ReasonProxyFailed  = "proxy conn refused"
	ReasonProxyTimeout = "proxy timeout"
	ReasonConnTimeout  = "conn timeout"
)

type RequestError struct {
	Type   string
	Reason string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Reason)
}
