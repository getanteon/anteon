package proxy

import (
	"net/url"
	"strings"
	"sync"

	"ddosify.com/hammer/core/types"
)

type ProxyService interface {
	init(types.Proxy)
	GetNewProxy() *url.URL
	ReportProxy(*url.URL)
}

var once sync.Once
var service ProxyService

func CreateProxyService(p types.Proxy) (ProxyService, error) {
	if service == nil {
		once.Do(
			func() {
				if strings.EqualFold(p.Strategy, "single") {
					service = &singleProxyStrategy{}
				} 
				service.init(p)
			},
		)
	}
	return service, nil
}
