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
	"net/http"
	"net/url"

	"go.ddosify.com/ddosify/core/scenario/scripting/injection"
	"go.ddosify.com/ddosify/core/types"
)

// Requester is the interface that abstracts different protocols' request sending implementations.
// Protocol field in the types.ScenarioStep determines which requester implementation to use.
type Requester interface {
	Type() string
	Done()
}

type HttpRequesterI interface {
	Init(ctx context.Context, ss types.ScenarioStep, url *url.URL, debug bool, ei *injection.EnvironmentInjector) error
	Send(client *http.Client, envs map[string]interface{}) *types.ScenarioStepResult // should use its own client if client is nil
}

// NewRequester is the factory method of the Requester.
func NewRequester(s types.ScenarioStep) (requester Requester, err error) {
	requester = &HttpRequester{} // we have only HttpRequester type for now, add check for rpc in future
	return
}
