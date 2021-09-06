package proxy

import (
	"net/url"

	"ddosify.com/hammer/core/types"
)

type singleProxyStrategy struct {
	proxyAddr *url.URL
}

func (sp *singleProxyStrategy) init(p types.Proxy) {
	sp.proxyAddr = p.Addr
}

// Since there is a 1 proxy, return that always
func (sp *singleProxyStrategy) GetAll() []*url.URL {
	return []*url.URL{sp.proxyAddr}
}

// Since there is a 1 proxy, return that always
func (sp *singleProxyStrategy) GetProxy() *url.URL {
	return sp.proxyAddr
}

func (sp *singleProxyStrategy) ReportProxy(addr *url.URL, reason string) *url.URL {
	return sp.proxyAddr
}

func (sp *singleProxyStrategy) GetProxyCountry(addr *url.URL) string {
	return "unkown"
}
