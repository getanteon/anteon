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

var instance *engine
var once sync.Once

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

func CreateEngine(ctx context.Context, h types.Hammer) *engine {
	if instance == nil {
		once.Do(
			func() {
				instance = &engine{hammer: h, ctx: ctx}
			},
		)
	}
	return instance
}

func (e *engine) Init() (err error) {
	if err = e.hammer.Validate(); err != nil {
		return
	}

	if instance.proxyService, err = proxy.CreateProxyService(e.hammer.Proxy); err != nil {
		return
	}

	if instance.scenarioService, err = scenario.CreateScenarioService(e.hammer.Scenario); err != nil {
		return
	}

	if instance.reportService, err = report.CreateReportService(e.hammer.ReportDestination); err != nil {
		return
	}

	instance.initReqCountArr()
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
	p := e.proxyService.GetNewProxy()
	res, err := e.scenarioService.Do(p)

	if err != nil && err.Type == types.ErrorProxy {
		e.proxyService.ReportProxy(p, err.Reason)
		fmt.Printf("ProxyErr %s\n", err.Reason)
	}

	e.responseChan <- res
}

func (e *engine) stop() {
	fmt.Println("Waiting workers to finish")
	e.wg.Wait()

	fmt.Println("Closing report chan")
	close(e.responseChan)

	fmt.Println("Waiting report done chan.")
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
func (e *engine) createCapacityReqCountArr() {
	return
}

// TODO
func (e *engine) createStressReqCountArr() {
	return
}

// TODO
func (e *engine) createSoakReqCountArr() {
	return
}
