package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ddosify.com/hammer/core/proxy"
	"ddosify.com/hammer/core/request"
	"ddosify.com/hammer/core/types"
)

const (
	tickerInterval = 100
	// QPS?
	// maxReq?
)

var hammer *Engine
var once sync.Once

type Engine struct {
	hammer types.Hammer

	proxyService   proxy.ProxyService
	requestService request.RequestService

	tickCounter int
	reqCountArr []int

	ctx context.Context
}

func CreateEngine(ctx context.Context, h types.Hammer) (engine *Engine, err error) {
	if engine == nil {
		once.Do(
			func() {
				engine = &Engine{hammer: h, ctx: ctx}
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
	fmt.Println("Hammerizing...")

	ticker := time.NewTicker(time.Duration(tickerInterval) * time.Millisecond)
	e.tickCounter = 0
	e.reqCountArr = []int{10, 20, 30, 20, 10, 20, 30, 20, 10}

	defer func() {
		fmt.Println("Stopping the ticker")
		ticker.Stop()
		e.stop()
	}()

	for range ticker.C {
		if e.tickCounter >= len(e.reqCountArr) {
			fmt.Println("All request has been sent")
			return
		}

		select {
		case <-e.ctx.Done():
			fmt.Println(("Stop signal received.."))
			return
		default:
			e.runWorkers()
		}

		e.tickCounter++
	}
}

func (e *Engine) runWorkers() {
	for i := 1; i <= e.reqCountArr[e.tickCounter]; i++ {
		go func() {
			e.runWorker()
		}()
	}
}

func (e *Engine) runWorker() {
	p := e.proxyService.GetNewProxy()
	res, err := e.requestService.Send(p)

	if err != nil {
		if custom_err, ok := err.(*types.Error); ok {

			switch custom_err.Type {
			case types.ErrorProxy:
				e.proxyService.ReportProxy(p, custom_err.Reason)
			}

		}
	} else {
		fmt.Println("Res:", res)
	}
}

func (e *Engine) stop() {
	fmt.Println("Engine Finished.")
}
