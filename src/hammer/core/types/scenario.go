package types

import (
	"fmt"
	"net/http"

	"ddosify.com/hammer/core/util"
)

const (
	ProtocolHTTP  = "HTTP"
	ProtocolHTTPS = "HTTPS"

	AuthHttpBasic = "basic"
)

var supportedProtocols = [...]string{ProtocolHTTP, ProtocolHTTPS}
var supportedProtocolMethods = map[string][]string{
	ProtocolHTTP: {
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions,
	},
	ProtocolHTTPS: {
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions,
	},
}
var supportedAuthentications = map[string][]string{
	ProtocolHTTP: {
		AuthHttpBasic,
	},
	ProtocolHTTPS: {
		AuthHttpBasic,
	},
}

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

	// Request Method
	Method string

	// Authenticaiton
	Auth Auth

	// Request Headers
	Headers map[string]string

	// Request payload
	Payload string

	// Target URL
	URL string

	// Connection timeout duration of the request in seconds
	Timeout int

	// Protocol spesific request parameters. For ex: DisableRedirects:true for Http requests
	Custom map[string]interface{}
}

type Auth struct {
	Type     string
	Username string
	Password string
}

func (si *ScenarioItem) validate() error {
	if !util.StringInSlice(si.Protocol, supportedProtocols[:]) {
		return fmt.Errorf("unsupported Protocol: %s", si.Protocol)
	}
	if !util.StringInSlice(si.Method, supportedProtocolMethods[si.Protocol][:]) {
		return fmt.Errorf("unsupported Request Method: %s", si.Method)
	}
	if si.Auth != (Auth{}) && !util.StringInSlice(si.Auth.Type, supportedAuthentications[si.Protocol][:]) {
		return fmt.Errorf("unsupported Authentication Method (%s) For Protocol (%s) ", si.Auth.Type, si.Protocol)
	}
	if si.ID == 0 {
		return fmt.Errorf("each scenario item should have an ordered unique ID")
	}
	return nil
}
