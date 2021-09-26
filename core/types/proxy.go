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
	"net/url"

	"ddosify.com/hammer/core/util"
)

const (
	ProxyTypeSingle        = "single"
)

var AvailableProxyStrategies = [...]string{ProxyTypeSingle}

type Proxy struct {
	// Stragy of the proxy usage.
	Strategy string

	// Set this field if ProxyStrategy is single
	Addr *url.URL
}

func (p *Proxy) validate() error {
	if !util.StringInSlice(p.Strategy, AvailableProxyStrategies[:]) {
		return fmt.Errorf("Unsupported Proxy Strategy: %s", p.Strategy)
	}
	return nil
}
