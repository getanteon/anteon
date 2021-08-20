package types

import (
	"fmt"
	"net/http"
	"net/url"

	"ddosify.com/hammer/core/util"
)

const (
	ProtocolHTTP  = "HTTP"
	ProtocolHTTPS = "HTTPS"
)

var supportedProtocols = [...]string{ProtocolHTTP, ProtocolHTTPS}
var supportedProtocolMethods = map[string][]string{
	ProtocolHTTP:  {http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
	ProtocolHTTPS: {http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}}

type Scenario struct {
	Scenario []ScenarioItem
}

func (s *Scenario) validate() error {
	for _, si := range s.Scenario {
		if err := si.validate(); err != nil {
			return err
		}
	}
	return nil
}

type ScenarioItem struct {
	// ID of the Item. Should be given by the client.
	ID int16

	// Protocol of the requests.
	Protocol string

	// Request Method Spesific To Protocol
	Method string

	// Request Headers
	Headers map[string]string

	// Request payload
	Payload string

	// Target URL
	URL url.URL

	// Connection timeout duration of the request in miliseconds
	Timeout int

	// Protocol spesific request parameters. For ex: DisableRedirects:true for Http requests
	Custom map[string]interface{}
}

func (si *ScenarioItem) validate() error {
	if !util.StringInSlice(si.Protocol, supportedProtocols[:]) {
		return fmt.Errorf("Unsupported Protocol: %s", si.Protocol)
	}
	if !util.StringInSlice(si.Method, supportedProtocolMethods[si.Protocol][:]) {
		return fmt.Errorf("Unsupported Request Method: %s", si.Method)
	}
	if si.ID == 0 {
		return fmt.Errorf("Each scenario item should have an unique ID")
	}
	return nil
}
