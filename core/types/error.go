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

const (
	// Type
	ErrorProxy  = "proxyError"
	ErrorConn   = "connectionError"
	ErrorUnkown = "unkownError"

	// Errors for created intentionally
	ErrorIntented = "intentedError"

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
