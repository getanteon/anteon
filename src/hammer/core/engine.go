package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ddosify.com/hammer/core/proxy"
	"ddosify.com/hammer/core/report"
	"ddosify.com/hammer/core/scenario"
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

	proxyService    proxy.ProxyService
	scenarioService *scenario.ScenarioService
	reportService   report.ReportService

	tickCounter int
	reqCountArr []int
	wg          sync.WaitGroup

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

				if engine.scenarioService, err = scenario.CreateScenarioService(h.Scenario); err != nil {
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

	e.tickCounter = -1
	e.wg = sync.WaitGroup{}
	for range ticker.C {
		e.tickCounter++
		if e.tickCounter >= len(e.reqCountArr) {
			return
		}

		select {
		case <-e.ctx.Done():
			return
		default:
			go e.runWorkers()
		}
	}
}

func (e *Engine) runWorkers() {
	for i := 1; i <= e.reqCountArr[e.tickCounter]; i++ {
		go func() {
			e.wg.Add(1)
			e.runWorker()
			e.wg.Done()
		}()
	}
}

func (e *Engine) runWorker() {
	p := e.proxyService.GetNewProxy()
	res, err := e.scenarioService.Do(p)

	if err != nil {
		if reqError, ok := err.(*types.Error); ok {
			switch reqError.Type {
			case types.ErrorProxy:
				e.proxyService.ReportProxy(p, reqError.Reason)
			}
		}
		return
	}
	// fmt.Println("Sendin res to response chan.")
	e.responseChan <- res
}

func (e *Engine) stop() {
	fmt.Println("Waiting workers to finish")
	e.wg.Wait()

	fmt.Println("Closing report chan")
	close(e.responseChan)

	fmt.Println("Waiting report done chan.")
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
