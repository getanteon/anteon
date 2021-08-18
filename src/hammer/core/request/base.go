package request

import (
	"net/url"
	"strings"
	"sync"

	"ddosify.com/hammer/core/types"
)

type request struct {
	types.Packet
	types.Scenario
}

// TODO: each request should have request_id to trace/debug etc.
type RequestService interface {
	init(types.Packet, types.Scenario)
	Send(proxyAddr *url.URL) (*types.Response, error)
}

var once sync.Once
var service RequestService

func CreateRequestService(p types.Packet, s types.Scenario) (RequestService, error) {
	if service == nil {
		once.Do(
			func() {
				if strings.EqualFold(p.Protocol, "http") ||
					strings.EqualFold(p.Protocol, "https") {
					service = &httpRequest{}
				}
				service.init(p, s)
			},
		)
	}
	return service, nil
}
