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
func (sp *singleProxyStrategy) GetNewProxy() *url.URL {
	return sp.proxyAddr
}

func (sp *singleProxyStrategy) ReportProxy(*url.URL) {
	return
}
