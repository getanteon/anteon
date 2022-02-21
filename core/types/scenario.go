/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package types

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.ddosify.com/ddosify/core/util"
)

// Constants for Scenario field values
const (
	// Constants of the Protocol types
	ProtocolHTTP  = "HTTP"
	ProtocolHTTPS = "HTTPS"

	// Constants of the Auth types
	AuthHttpBasic = "basic"

	// Max sleep in ms (90s)
	maxSleep = 90000
)

// SupportedProtocols should be updated whenever a new requester.Requester interface implemented
var SupportedProtocols = [...]string{ProtocolHTTP, ProtocolHTTPS}
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

// Scenario struct contains a list of ScenarioItem so scenario.ScenarioService can execute the scenario step by step.
type Scenario struct {
	Scenario []ScenarioItem
}

func (s *Scenario) validate() error {
	stepIds := make(map[int16]struct{}, len(s.Scenario))
	for _, si := range s.Scenario {
		if err := si.validate(); err != nil {
			return err
		}

		if _, ok := stepIds[si.ID]; ok {
			return fmt.Errorf("duplicate step id: %d", si.ID)
		}
		stepIds[si.ID] = struct{}{}
	}
	return nil
}

// ScenarioItem represents one step of a Scenario.
// This struct should be able to include all necessary data in a network packet for SupportedProtocols.
type ScenarioItem struct {
	// ID of the Item. Should be given by the client.
	ID int16

	// Name of the Item.
	Name string

	// Protocol of the requests.
	Protocol string

	// Request Method
	Method string

	// Authentication
	Auth Auth

	// Request Headers
	Headers map[string]string

	// Request payload
	Payload string

	// Target URL
	URL string

	// Connection timeout duration of the request in seconds
	Timeout int

	// Sleep duration after running the step. Can be a time range like "300-500" or an exact duration like "350" in ms
	Sleep string

	// Protocol spesific request parameters. For ex: DisableRedirects:true for Http requests
	Custom map[string]interface{}
}

// Auth struct should be able to include all necessary authentication realated data for supportedAuthentications.
type Auth struct {
	Type     string
	Username string
	Password string
}

func (si *ScenarioItem) validate() error {
	if !util.StringInSlice(si.Protocol, SupportedProtocols[:]) {
		return fmt.Errorf("unsupported Protocol: %s", si.Protocol)
	}
	if !util.StringInSlice(si.Method, supportedProtocolMethods[si.Protocol][:]) {
		return fmt.Errorf("unsupported Request Method: %s", si.Method)
	}
	if si.Auth != (Auth{}) && !util.StringInSlice(si.Auth.Type, supportedAuthentications[si.Protocol][:]) {
		return fmt.Errorf("unsupported Authentication Method (%s) For Protocol (%s) ", si.Auth.Type, si.Protocol)
	}
	if si.ID == 0 {
		return fmt.Errorf("step ID should be greater than zero")
	}
	if si.Sleep != "" {
		sleep := strings.Split(si.Sleep, "-")

		// Avoid invalid syntax like "-300-500"
		if len(sleep) > 2 {
			return fmt.Errorf("sleep expression is not valid: %s", si.Sleep)
		}

		// Validate string to int conversion
		for _, s := range sleep {
			dur, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("sleep is not valid: %s", si.Sleep)
			}

			if dur > maxSleep {
				return fmt.Errorf("maximum sleep limit exceeded. provided: %d ms, maximum: %d ms", dur, maxSleep)
			}
		}
	}
	return nil
}
