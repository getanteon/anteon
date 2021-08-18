package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ddosify.com/hammer/core/proxy"
	"ddosify.com/hammer/core/report"
	"ddosify.com/hammer/core/request"
	"ddosify.com/hammer/core/types"
)

const (
	// internval in milisecond
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
	reportService  report.ReportService

	tickCounter int
	reqCountArr []int

	responseChan chan *types.Response

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

				if engine.reportService, err = report.CreateReportService(h.ReportDestination); err != nil {
					return
				}

				engine.initReqCountArr()
			},
		)
	}
	return
}

func (e *Engine) Start() {
	ticker := time.NewTicker(time.Duration(tickerInterval) * time.Millisecond)
	e.responseChan = make(chan *types.Response, e.hammer.TotalReqCount)
	go e.reportService.Start(e.responseChan)

	defer func() {
		ticker.Stop()
		e.stop()
	}()

	e.tickCounter = 0
	for range ticker.C {
		if e.tickCounter >= len(e.reqCountArr) {
			return
		}

		for i := 1; i <= e.reqCountArr[e.tickCounter]; i++ {
			select {
			case <-e.ctx.Done():
				return
			default:
				go func() {
					e.runWorker()
				}()
			}
		}

		e.tickCounter++
	}
}

func (e *Engine) runWorker() {
	p := e.proxyService.GetNewProxy()
	res, err := e.requestService.Send(p)

	if err != nil {
		if reqError, ok := err.(*types.Error); ok {
			switch reqError.Type {
			case types.ErrorProxy:
				e.proxyService.ReportProxy(p, reqError.Reason)
			}
		}
	}
	e.responseChan <- res
}

func (e *Engine) stop() {
	fmt.Println("Closing report chan")
	close(e.responseChan)

	fmt.Println("Waiting done chan.")
	<-e.reportService.DoneChan()

	e.reportService.Report()
}

func (e *Engine) initReqCountArr() {
	if e.hammer.TimeReqCountMap != nil {
		fmt.Println("initReqCountArr from TimeReqCountMap")
	} else {
		length := int(e.hammer.TestDuration * int(time.Second/(tickerInterval*time.Millisecond)))
		e.reqCountArr = make([]int, length)

		switch e.hammer.LoadType {
		case types.LoadTypeLinear:
			e.createLinearReqCountArr()
		case types.LoadTypeCapacity:
			e.createCapacityReqCountArr()
		case types.LoadTypeStress:
			e.createStressReqCountArr()
		case types.LoadTypeSoak:
			e.createSoakReqCountArr()
		}
		// fmt.Println(e.reqCountArr)
	}
}

func (e *Engine) createLinearReqCountArr() {
	minReqCount := int(e.hammer.TotalReqCount / len(e.reqCountArr))
	remaining := e.hammer.TotalReqCount - minReqCount*len(e.reqCountArr)
	for i := range e.reqCountArr {
		plusOne := 0
		if i < remaining {
			plusOne = 1
		}
		reqCount := minReqCount + plusOne
		e.reqCountArr[i] = reqCount
	}
}

// TODO
func (e *Engine) createCapacityReqCountArr() {
	return
}

// TODO
func (e *Engine) createStressReqCountArr() {
	return
}

// TODO
func (e *Engine) createSoakReqCountArr() {
	return
}
