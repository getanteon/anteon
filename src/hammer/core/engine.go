package core

import (
	"context"
	"fmt"
	"math"
	"reflect"
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
	if e.scenarioService, err = scenario.NewScenarioService(e.hammer.Scenario, proxies, e.ctx); err != nil {
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
			e.wg.Add(e.reqCountArr[e.tickCounter])
			go e.runWorkers(e.tickCounter)
			e.tickCounter++
			mutex.Unlock()
		}
	}
}

func (e *engine) runWorkers(c int) {
	for i := 1; i <= e.reqCountArr[c]; i++ {
		go func() {
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
	if err != nil && err.Type == types.ErrorConn && err.Reason == types.ReasonCtxCanceled {
		// Don't report intentionally canceled requests.
		return
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
	}
}

func (e *engine) createLinearReqCountArr() {
	createLinearDistArr(e.hammer.TotalReqCount, e.reqCountArr)
}

func (e *engine) createIncrementalReqCountArr() {
	steps := createIncrementalDistArr(e.hammer.TotalReqCount, e.hammer.TestDuration)
	tickPerSecond := int(time.Second / (tickerInterval * time.Millisecond))
	for i := range steps {
		tickArrStartIndex := i * tickPerSecond
		tickArrEndIndex := tickArrStartIndex + tickPerSecond
		segment := e.reqCountArr[tickArrStartIndex:tickArrEndIndex]
		createLinearDistArr(steps[i], segment)
	}
}

func (e *engine) createWavedReqCountArr() {
	tickPerSecond := int(time.Second / (tickerInterval * time.Millisecond))
	quarterWaveCount := int((math.Log2(float64(e.hammer.TestDuration))))
	qWaveDuration := int(e.hammer.TestDuration / quarterWaveCount)
	reqCountPerQWave := int(e.hammer.TotalReqCount / quarterWaveCount)
	tickArrStartIndex := 0

	for i := 0; i < quarterWaveCount; i++ {
		if i == quarterWaveCount-1 {
			// Add remaining req count to the last wave
			reqCountPerQWave += e.hammer.TotalReqCount - (reqCountPerQWave * quarterWaveCount)
		}

		steps := createIncrementalDistArr(reqCountPerQWave, qWaveDuration)
		if i%2 == 1 {
			reverse(steps)
		}

		for j := range steps {
			tickArrEndIndex := tickArrStartIndex + tickPerSecond
			segment := e.reqCountArr[tickArrStartIndex:tickArrEndIndex]
			createLinearDistArr(steps[j], segment)
			tickArrStartIndex += tickPerSecond
		}
	}
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

func createIncrementalDistArr(count int, len int) []int {
	steps := make([]int, len)
	sum := (len * (len + 1)) / 2
	incrementStep := int(math.Ceil(float64(sum) / float64(count)))
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

	sum = arraySum(steps)

	factor := count / sum
	remaining := count - (sum * factor)
	plus := remaining / len
	lastRemaining := remaining - (plus * len)
	for i := range steps {
		steps[i] = steps[i]*factor + plus
		if len-i-1 < lastRemaining {
			steps[i]++
		}
	}
	return steps
}

func arraySum(steps []int) int {
	sum := 0
	for i := range steps {
		sum += steps[i]
	}
	return sum
}

func reverse(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}
