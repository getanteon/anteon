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

package requester

import (
	"errors"
	"net"
	"net/url"
	"testing"

	"go.ddosify.com/ddosify/v2/core/types"
)

func TestFetchErrTypeKeepsTheCause(t *testing.T) {
	u := func(cause error) error {
		return &url.Error{Op: "Post", URL: "https://api.example.internal/chats", Err: cause}
	}
	lookup := func(e string, timeout bool) error {
		return &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: e, Name: "api.example.internal", Server: "10.0.0.2:53", IsTimeout: timeout}}
	}
	cases := []struct {
		name   string
		cause  error
		reason string
	}{
		{"a name that does not resolve", lookup("no such host", false), "dns lookup api.example.internal: no such host"},
		// the resolver's error carries a random local port; the reason must not, or every request is a kind of its own
		{"a resolver that timed out is still a name", lookup("read udp 10.0.0.3:5353->10.0.0.2:53: i/o timeout", true), "dns lookup api.example.internal: timeout"},
		{"refused keeps its class", errors.New("dial tcp 10.0.0.1:443: connect: connection refused"), types.ReasonConnRefused},
		{"a read timeout keeps its class", errors.New("read tcp 10.0.0.3:5000->10.0.0.1:443: i/o timeout"), types.ReasonReadTimeout},
	}
	for _, c := range cases {
		err := u(c.cause)
		got := fetchErrType(err)
		if got.Type != types.ErrorConn || got.Reason != c.reason || got.Detail != err.Error() {
			t.Errorf("%s: %+v", c.name, got)
		}
	}
}
