package scenario

import (
	"net/url"
	"sync"

	"ddosify.com/hammer/core/scenario/requester"
	"ddosify.com/hammer/core/types"
)

type ScenarioService struct {
	scenario   types.Scenario
	requesters map[int16]requester.Requester
}

var once sync.Once
var service *ScenarioService

func CreateScenarioService(s types.Scenario) (service *ScenarioService, err error) {
	if service == nil {
		once.Do(
			func() {
				service = &ScenarioService{}
				err = service.init(s)
			},
		)
	}
	return
}

func (ss *ScenarioService) init(s types.Scenario) (err error) {
	ss.requesters = make(map[int16]requester.Requester, len(s.Scenario))
	for _, si := range s.Scenario {
		ss.requesters[si.ID], err = requester.NewRequester(si)
		if err != nil {
			return
		}
		err = ss.requesters[si.ID].Init(si)
	}
	return
}

func (ss *ScenarioService) Do(proxy *url.URL) (*types.Response, error) {
	response := &types.Response{ResponseItems: []*types.ResponseItem{}}
	for _, r := range ss.requesters {
		res, err := r.Send(proxy)
		if err != nil {
			return nil, err
		}
		response.ResponseItems = append(response.ResponseItems, res)
	}
	return response, nil
}
