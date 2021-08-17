package types

import "fmt"

const (
	// Type
	ErrorProxy = "ProxyError"

	// Reasons
	ReasonProxyFailed  = "proxy failed"
	ReasonProxyTimeout = "proxy timeout"
)

type Error struct {
	Type   string
	Reason string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s - %s", e.Type, e.Reason)
}
