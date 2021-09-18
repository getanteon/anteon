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
	"context"
	"fmt"
	"net/url"
	"strings"

	"ddosify.com/hammer/core/types"
)

type Requester interface {
	Init(types.ScenarioItem, *url.URL, context.Context) error
	Send() *types.ResponseItem
}

func NewRequester(s types.ScenarioItem) (requester Requester, err error) {
	if strings.EqualFold(s.Protocol, "http") ||
		strings.EqualFold(s.Protocol, "https") {
		requester = &httpRequester{}
	} else {
		err = fmt.Errorf("unsupported requester")
	}
	return
}
