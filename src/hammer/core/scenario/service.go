package scenario

import (
	"net/url"
	"sync"
	"time"

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

func (ss *ScenarioService) Do(proxy *url.URL) (response *types.Response, err *types.RequestError) {
	response = &types.Response{ResponseItems: []*types.ResponseItem{}}
	response.StartTime = time.Now()
	for _, r := range ss.requesters {
		res := r.Send(proxy)
		if res.Err.Type == types.ErrorProxy {
			err = &res.Err
			return
		}
		response.ResponseItems = append(response.ResponseItems, res)
	}
	response.EndTime = time.Now()
	return
}
