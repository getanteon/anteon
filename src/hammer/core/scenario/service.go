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

	"ddosify.com/hammer/core/scenario/requester"
	"ddosify.com/hammer/core/types"
)

type ScenarioService struct {
	scenario types.Scenario

	// Client map structure [proxy_addr][]scenarioItemRequester
	// Each proxy represents a client.
	// Each scenarioItem has a requester
	clients map[*url.URL][]scenarioItemRequester
}

func NewScenarioService(
	s types.Scenario,
	proxies []*url.URL,
	ctx context.Context) (service *ScenarioService, err error) {
	service = &ScenarioService{}
	err = service.init(s, proxies, ctx)
	return
}

func (ss *ScenarioService) init(s types.Scenario, proxies []*url.URL, ctx context.Context) (err error) {
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

			err = r.Init(si, p, ctx)
			if err != nil {
				return
			}
		}
	}
	return
}

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
