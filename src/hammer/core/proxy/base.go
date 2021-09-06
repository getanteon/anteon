package proxy

import (
	"net/url"
	"strings"

	"ddosify.com/hammer/core/types"
)

type ProxyService interface {
	init(types.Proxy)
	GetAll() []*url.URL
	GetProxy() *url.URL
	ReportProxy(addr *url.URL, reason string) *url.URL
	GetProxyCountry(*url.URL) string
}

func NewProxyService(p types.Proxy) (service ProxyService, err error) {
	if strings.EqualFold(p.Strategy, "single") {
		service = &singleProxyStrategy{}
	}
	service.init(p)

	return service, nil
}
