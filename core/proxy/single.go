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
	"net/url"
	"time"
)

const ProxyTypeSingle = "single"

func init() {
	AvailableProxyServices[ProxyTypeSingle] = &singleProxyStrategy{}
}

type singleProxyStrategy struct {
	proxyAddr *url.URL
}

func (sp *singleProxyStrategy) Init(p Proxy) error {
	sp.proxyAddr = p.Addr
	return nil
}

// Since there is a 1 proxy, return that always
func (sp *singleProxyStrategy) GetAll() []*url.URL {
	return []*url.URL{sp.proxyAddr}
}

// Since there is a 1 proxy, return that always
func (sp *singleProxyStrategy) GetProxy() *url.URL {
	return sp.proxyAddr
}

func (sp *singleProxyStrategy) ReportProxy(addr *url.URL, reason string) *url.URL {
	return sp.proxyAddr
}

func (sp *singleProxyStrategy) GetProxyCountry(addr *url.URL) string {
	return "unknown"
}

func (sp *singleProxyStrategy) GetLatency(addr *url.URL) time.Duration {
	// We may want to calculate latency for single proxy strategy also.
	return time.Duration(0)
}

func (sp *singleProxyStrategy) Done() error {
	return nil
}
