package core

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"ddosify.com/hammer/core/proxy"
	"ddosify.com/hammer/core/report"
	"ddosify.com/hammer/core/scenario"
	"ddosify.com/hammer/core/types"
)

const (
	// interval in milisecond
	tickerInterval = 100
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
	}

	e.responseChan <- res
}

func (e *engine) stop() {
	e.wg.Wait()
	close(e.responseChan)
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
	createLinearDistArr(e.hammer.TotalReqCount, e.reqCountArr)
}

func createLinearDistArr(count int, arr []int) {
	len := len(arr)
	minReqCount := int(count / len)
	remaining := count - minReqCount*len
	for i := range arr {
		plusOne := 0
		if i < remaining {
			plusOne = 1
		}
		reqCount := minReqCount + plusOne
		arr[i] = reqCount
	}
}

func (e *engine) createIncrementalReqCountArr() {
	steps := make([]int, e.hammer.TestDuration)
	sum := (e.hammer.TestDuration * (e.hammer.TestDuration + 1)) / 2
	incrementStep := int(math.Ceil(float64(sum) / float64(e.hammer.TotalReqCount)))
	val := 0
	for i := range steps {
		if i > 0 {
			val = steps[i-1]
		}

		if i%incrementStep == 0 {
			steps[i] = val + 1
		} else {
			steps[i] = val
		}
	}

	sum = 0
	for i := range steps {
		sum += steps[i]
	}

	factor := e.hammer.TotalReqCount / sum
	remaining := e.hammer.TotalReqCount - (sum * factor)
	plus := remaining / len(steps)
	lastRemaining := remaining - (plus * len(steps))
	for i := range steps {
		steps[i] = steps[i]*factor + plus
		if len(steps)-i-1 < lastRemaining {
			steps[i]++
		}

		tickArrStartIndex := i * 10
		createLinearDistArr(steps[i], e.reqCountArr[tickArrStartIndex:tickArrStartIndex+10])
	}

}

// TODO
func (e *engine) createWavedReqCountArr() {
	return
}
