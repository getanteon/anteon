package types

import (
	"fmt"
	"net/url"

	"ddosify.com/hammer/core/util"
)

var availableProxyStrategies = [...]string{"single"}

type Proxy struct {
	// Stragy of the proxy usage.
	Strategy string

	// Set this field if ProxyStrategy is single
	Addr *url.URL
}

func (p *Proxy) validate() error {
	if !util.StringInSlice(p.Strategy, availableProxyStrategies[:]) {
		return fmt.Errorf("Unsupported Porxy Strategy: %s", p.Strategy)
	}
	return nil
}
