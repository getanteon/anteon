package types

import (
	"fmt"

	"ddosify.com/hammer/core/util"
)

var supportedProtocols = [...]string{"HTTP", "HTTPS"}
var supportedProtocolMethods = map[string][]string{
	"HTTP":  {"GET", "POST", "PUT", "DELETE", "UPDATE", "PATCH"},
	"HTTPS": {"GET", "POST", "PUT", "DELETE", "UPDATE", "PATCH"}}

// Network Packet Context
type Packet struct {
	// Protocol of the requests.
	Protocol string

	// Request Method Spesific To Protocol
	Method string

	// Request Headers
	Headers map[string]string

	// Request payload
	Payload string
}

func (p *Packet) validate() error {
	if !util.StringInSlice(p.Protocol, supportedProtocols[:]) {
		return fmt.Errorf("Unsupported Protocol: %s", p.Protocol)
	}
	if !util.StringInSlice(p.Method, supportedProtocolMethods[p.Protocol][:]) {
		return fmt.Errorf("Unsupported Request Method: %s", p.Method)
	}

	return nil
}
