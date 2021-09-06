package scenario

import (
	"net/url"
	"time"

	"ddosify.com/hammer/core/scenario/requester"
	"ddosify.com/hammer/core/types"
)

type ScenarioService struct {
	scenario types.Scenario

	// Client map structure [proxy_addr][scenarioItemID][requester]
	// Each proxy represents a client.
	// Each scenarioItem has a requester
	clients map[*url.URL]map[int16]requester.Requester
}

func NewScenarioService(s types.Scenario, proxies []*url.URL) (service *ScenarioService, err error) {
	service = &ScenarioService{}
	err = service.init(s, proxies)
	return
}

func (ss *ScenarioService) init(s types.Scenario, proxies []*url.URL) (err error) {
	ss.clients = make(map[*url.URL]map[int16]requester.Requester, len(proxies))
	for _, p := range proxies {
		ss.clients[p] = make(map[int16]requester.Requester)
		for _, si := range s.Scenario {
			ss.clients[p][si.ID], err = requester.NewRequester(si)
			if err != nil {
				return
			}

			err = ss.clients[p][si.ID].Init(si, p)
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
	for _, r := range ss.clients[proxy] {
		res := r.Send()
		if res.Err.Type == types.ErrorProxy {
			err = &res.Err
		}
		response.ResponseItems = append(response.ResponseItems, res)
	}
	return
}
