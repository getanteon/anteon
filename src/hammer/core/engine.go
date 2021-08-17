package core

import (
	"fmt"
	"sync"

	"ddosify.com/hammer/core/proxy"
	"ddosify.com/hammer/core/request"
	"ddosify.com/hammer/core/types"
)

var hammer *Engine
var once sync.Once

type Engine struct {
	hammer types.Hammer

	proxyService   proxy.ProxyService
	requestService request.RequestService

	stopChan chan struct{}
}

func CreateEngine(h types.Hammer) (engine *Engine, err error) {
	if engine == nil {
		once.Do(
			func() {
				engine = &Engine{hammer: h}
				if err := h.Validate(); err != nil {
					return
				}

				if engine.proxyService, err = proxy.CreateProxyService(h.Proxy); err != nil {
					return
				}

				if engine.requestService, err = request.CreateRequestService(h.Packet, h.Scenario); err != nil {
					return
				}

			},
		)
	}
	return
}

func (e *Engine) Start() {
	fmt.Println("Starting to hammerizing...")
	// http.ProxyURL(e.proxyService.GetNewProxy())
	e.requestService.Send(e.proxyService.GetNewProxy())
}

func (e *Engine) Stop() {
	fmt.Println("Hammer stopped.")
}
