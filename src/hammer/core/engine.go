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

type engine struct {
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

func NewEngine(ctx context.Context, h types.Hammer) *engine {
	return &engine{hammer: h, ctx: ctx}
}

func (e *engine) Init() (err error) {
	if err = e.hammer.Validate(); err != nil {
		return
	}

	if e.proxyService, err = proxy.NewProxyService(e.hammer.Proxy); err != nil {
		return
	}

	proxies := e.proxyService.GetAll()
	if e.scenarioService, err = scenario.NewScenarioService(e.hammer.Scenario, proxies); err != nil {
		return
	}

	if e.reportService, err = report.NewReportService(e.hammer.ReportDestination); err != nil {
		return
	}

	e.initReqCountArr()
	return nil
}

func (e *engine) Start() {
	ticker := time.NewTicker(time.Duration(tickerInterval) * time.Millisecond)
	e.responseChan = make(chan *types.Response, e.hammer.TotalReqCount)
	go e.reportService.Start(e.responseChan)

	defer func() {
		ticker.Stop()
		e.stop()
	}()

	e.tickCounter = 0
	e.wg = sync.WaitGroup{}
	var mutex = &sync.Mutex{}
	for range ticker.C {
		if e.tickCounter >= len(e.reqCountArr) {
			return
		}

		select {
		case <-e.ctx.Done():
			return
		default:
			mutex.Lock()
			go e.runWorkers(e.tickCounter)
			e.tickCounter++
			mutex.Unlock()
		}
	}
}

func (e *engine) runWorkers(c int) {
	for i := 1; i <= e.reqCountArr[c]; i++ {
		go func() {
			e.wg.Add(1)
			e.runWorker()
			e.wg.Done()
		}()
	}
}

func (e *engine) runWorker() {
	p := e.proxyService.GetProxy()
	res, err := e.scenarioService.Do(p)

	if err != nil && err.Type == types.ErrorProxy {
		e.proxyService.ReportProxy(p, err.Reason)
		// fmt.Printf("ProxyErr %s\n", err.Reason)
	}

	e.responseChan <- res
}

func (e *engine) stop() {
	// fmt.Println("Waiting workers to finish")
	e.wg.Wait()

	// fmt.Println("Closing report chan")
	close(e.responseChan)

	// fmt.Println("Waiting report done chan.")
	<-e.reportService.DoneChan()

	e.reportService.Report()
}

func (e *engine) initReqCountArr() {
	if e.hammer.TimeReqCountMap != nil {
		fmt.Println("initReqCountArr from TimeReqCountMap")
	} else {
		length := int(e.hammer.TestDuration * int(time.Second/(tickerInterval*time.Millisecond)))
		e.reqCountArr = make([]int, length)

		switch e.hammer.LoadType {
		case types.LoadTypeLinear:
			e.createLinearReqCountArr()
		case types.LoadTypeIncremental:
			e.createIncrementalReqCountArr()
		case types.LoadTypeWaved:
			e.createWavedReqCountArr()
		}
		// fmt.Println(e.reqCountArr)
	}
}

func (e *engine) createLinearReqCountArr() {
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
func (e *engine) createIncrementalReqCountArr() {
	return
}

// TODO
func (e *engine) createWavedReqCountArr() {
	return
}
