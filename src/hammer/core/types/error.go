package types

import "fmt"

const (
	// Type
	ErrorProxy  = "proxyError"
	ErrorConn   = "connectionError"
	ErrorUnkown = "unkownError"

	// Reasons
	ReasonProxyFailed  = "proxy connection refused"
	ReasonProxyTimeout = "proxy timeout"
	ReasonConnTimeout  = "connection timeout"
	ReasonReadTimeout  = "read timeout"
	ReasonConnRefused  = "connection refused"

	// In gracefully stop, engine cancels the ongoing requests.
	// We can detect the canceled requests with the help of this.
	ReasonCtxCanceled = "context canceled"
)

type RequestError struct {
	Type   string
	Reason string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Reason)
}
