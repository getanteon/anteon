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

package proxy

import (
	"fmt"
	"net/url"
	"reflect"
	"time"
)

var AvailableProxyServices = make(map[string]ProxyService)

// Proxy struct is used for initializing the ProxyService implementations.
type Proxy struct {
	// Stragy of the proxy usage.
	Strategy string

	// Set this field if ProxyStrategy is single
	Addr *url.URL

	// Dynamic field for other proxy strategies.
	Others map[string]interface{}
}

// ProvideService is the interface that abstracts different proxy implementations.
// Strategy field in types.Proxy determines which implementation to use.
type ProxyService interface {
	Init(Proxy) error
	GetAll() []*url.URL
	GetProxy() *url.URL
	ReportProxy(addr *url.URL, reason string) *url.URL
	GetProxyCountry(*url.URL) string
	GetLatency(*url.URL) time.Duration
	Done() error
}

// NewProxyService is the factory method of the ProxyService.
func NewProxyService(s string) (service ProxyService, err error) {
	if val, ok := AvailableProxyServices[s]; ok {
		// Create a new object from the service type
		service = reflect.New(reflect.TypeOf(val).Elem()).Interface().(ProxyService)
	} else {
		err = fmt.Errorf("unsupported proxy strategy: %s", s)
	}

	return
}
