package core

import (
	"fmt"
	"sync"

	"ddosify.com/hammer/core/proxy"
	"ddosify.com/hammer/core/types"
)

var hammer *Engine
var once sync.Once

type Engine struct {
	context types.Hammer

	proxyService proxy.ProxyService

	stopChan chan struct{}
}

func CreateEngine(h types.Hammer) (hammer *Engine, err error) {
	if hammer == nil {
		once.Do(
			func() {
				hammer = &Engine{context: h}
				if err := h.Validate(); err != nil {
					return
				}
			},
		)
	}
	return
}

func (h *Engine) Start() {
	fmt.Println("Starting to hammerizing...")
	fmt.Println(h)
}

func (h *Engine) Stop() {
	fmt.Println("Hammer stopped.")
}
