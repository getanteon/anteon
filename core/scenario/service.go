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

package scenario

import (
	"context"
	"net/url"
	"time"

	"go.ddosify.com/ddosify/core/scenario/requester"
	"go.ddosify.com/ddosify/core/types"
)

// ScenarioService encapsulates proxy/scenario/requester information and runs the scenario.
type ScenarioService struct {
	// Client map structure [proxy_addr][]scenarioItemRequester
	// Each proxy represents a client.
	// Each scenarioItem has a requester
	clients map[*url.URL][]scenarioItemRequester
}

// Constructor of the ScenarioService.
func NewScenarioService() *ScenarioService {
	return &ScenarioService{}
}

// Initialize the ScenarioService.clients with the given types.Scenario and proxies.
// Passes the given ctx to the underlying requestor so we are able to control to the life of each request.
func (ss *ScenarioService) Init(ctx context.Context, s types.Scenario, proxies []*url.URL) (err error) {
	ss.clients = make(map[*url.URL][]scenarioItemRequester, len(proxies))
	for _, p := range proxies {
		ss.clients[p] = []scenarioItemRequester{}
		for _, si := range s.Scenario {
			var r requester.Requester
			r, err = requester.NewRequester(si)
			if err != nil {
				return
			}
			ss.clients[p] = append(ss.clients[p], scenarioItemRequester{scenarioItemID: si.ID, requester: r})

			err = r.Init(ctx, si, p)
			if err != nil {
				return
			}
		}
	}
	return
}

// Executes the scenario for the given proxy.
// Returns "types.Response" filled by the requester of the given Proxy
// Returns error only if types.Response.Err.Type is types.ErrorProxy or types.ErrorIntented
func (ss *ScenarioService) Do(proxy *url.URL) (response *types.Response, err *types.RequestError) {
	response = &types.Response{ResponseItems: []*types.ResponseItem{}}
	response.StartTime = time.Now()
	response.ProxyAddr = proxy
	for _, sr := range ss.clients[proxy] {
		res := sr.requester.Send()
		if res.Err.Type == types.ErrorProxy || res.Err.Type == types.ErrorIntented {
			err = &res.Err
		}
		response.ResponseItems = append(response.ResponseItems, res)
	}
	return
}

type scenarioItemRequester struct {
	scenarioItemID int16
	requester      requester.Requester
}
