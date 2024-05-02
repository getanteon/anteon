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

import "fmt"

// Constants for custom error types and reasons
const (
	// Types
	ErrorProxy          = "proxyError"
	ErrorConn           = "connectionError"
	ErrorUnkown         = "unknownError"
	ErrorIntented       = "intentedError" // Errors for created intentionally
	ErrorDns            = "dnsError"
	ErrorParse          = "parseError"
	ErrorAddr           = "addressError"
	ErrorInvalidRequest = "invalidRequestError"

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

// RequestError is our custom error struct created in the requester.Requester implementations.
type RequestError struct {
	Type   string
	Reason string
}

// Custom error message method of ScenarioError
func (e *RequestError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Reason)
}

type ScenarioValidationError struct { // UnWrappable
	msg        string
	wrappedErr error
}

func (sc ScenarioValidationError) Error() string {
	return sc.msg
}

func (sc ScenarioValidationError) Unwrap() error {
	return sc.wrappedErr
}

type EnvironmentNotDefinedError struct { // UnWrappable
	msg        string
	wrappedErr error
}

func (sc EnvironmentNotDefinedError) Error() string {
	return sc.msg
}

func (sc EnvironmentNotDefinedError) Unwrap() error {
	return sc.wrappedErr
}

type CaptureConfigError struct { // UnWrappable
	msg        string
	wrappedErr error
}

func (sc CaptureConfigError) Error() string {
	return sc.msg
}

func (sc CaptureConfigError) Unwrap() error {
	return sc.wrappedErr
}

type FailedAssertion struct {
	Rule     string
	Received map[string]interface{}
	Reason   string
}
