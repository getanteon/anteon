/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package core

import (
	"context"
	"math"
	"reflect"
	"sync"
	"time"

	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/report"
	"go.ddosify.com/ddosify/core/scenario"
	"go.ddosify.com/ddosify/core/scenario/data"
	"go.ddosify.com/ddosify/core/types"
)

const (
	// interval in millisecond
	tickerInterval = 100

	// test result status
	resultDone    = "done"
	resultStopped = "stopped"
)

type engine struct {
	hammer types.Hammer

	proxyService    proxy.ProxyService
	scenarioService *scenario.ScenarioService
	reportService   report.ReportService

	tickCounter int
	reqCountArr []int
	wg          sync.WaitGroup

	resultChan chan *types.ScenarioResult

	ctx context.Context
}

// NewEngine is the constructor of the engine.
// Hammer is used for initializing the engine itself and its' external services.
// Engine can be stopped by canceling the given ctx.
func NewEngine(ctx context.Context, h types.Hammer) (e *engine, err error) {
	err = h.Validate()
	if err != nil {
		return
	}

	ps, err := proxy.NewProxyService(h.Proxy.Strategy)
	if err != nil {
		return
	}

	rs, err := report.NewReportService(h.ReportDestination)
	if err != nil {
		return
	}

	ss := scenario.NewScenarioService()

	e = &engine{
		hammer:          h,
		ctx:             ctx,
		proxyService:    ps,
		scenarioService: ss,
		reportService:   rs,
	}

	return
}

func (e *engine) Init() (err error) {
	e.initReqCountArr()
	if err = e.proxyService.Init(e.hammer.Proxy); err != nil {
		return
	}
	// read test data
	readData, err := readTestData(e.hammer.TestDataConf)
	if err != nil {
		return err
	}
	e.hammer.Scenario.Data = readData

	if err = e.scenarioService.Init(e.ctx, e.hammer.Scenario, e.proxyService.GetAll(), scenario.ScenarioOpts{
		Debug:                  e.hammer.Debug,
		IterationCount:         e.hammer.IterationCount,
		MaxConcurrentIterCount: e.getMaxConcurrentIterCount(),
		EngineMode:             e.hammer.EngineMode,
	}); err != nil {
		return
	}

	if err = e.scenarioService.Init(e.ctx, e.hammer.Scenario, e.proxyService.GetAll(), scenario.ScenarioOpts{
		Debug:                  e.hammer.Debug,
		IterationCount:         e.hammer.IterationCount,
		MaxConcurrentIterCount: e.getMaxConcurrentIterCount(),
		EngineMode:             e.hammer.EngineMode,
	}); err != nil {
		return
	}

	if err = e.reportService.Init(e.hammer.Debug, e.hammer.SamplingRate); err != nil {
		return
	}

	return
}

func (e *engine) Start() string {
	ticker := time.NewTicker(time.Duration(tickerInterval) * time.Millisecond)
	e.resultChan = make(chan *types.ScenarioResult, e.hammer.IterationCount)
	go e.reportService.Start(e.resultChan)

	defer func() {
		ticker.Stop()
		e.stop()
	}()

	e.tickCounter = 0
	e.wg = sync.WaitGroup{}
	var mutex = &sync.Mutex{}
	for range ticker.C {
		if e.tickCounter >= len(e.reqCountArr) {
			return resultDone
		}

		select {
		case <-e.ctx.Done():
			return resultStopped
		default:
			mutex.Lock()
			e.wg.Add(e.reqCountArr[e.tickCounter])
			go e.runWorkers(e.tickCounter)
			e.tickCounter++
			mutex.Unlock()
		}
	}
	return resultDone
}

func (e *engine) runWorkers(c int) {
	for i := 1; i <= e.reqCountArr[c]; i++ {
		scenarioStartTime := time.Now()
		go func(t time.Time) {
			e.runWorker(t)
			e.wg.Done()
		}(scenarioStartTime)
	}
}

func (e *engine) runWorker(scenarioStartTime time.Time) {
	var res *types.ScenarioResult
	var err *types.RequestError

	p := e.proxyService.GetProxy()
	retryCount := 3
	for i := 1; i <= retryCount; i++ {
		res, err = e.scenarioService.Do(p, scenarioStartTime)

		if err != nil && err.Type == types.ErrorProxy {
			p = e.proxyService.ReportProxy(p, err.Reason)
			continue
		}

		if err != nil && err.Type == types.ErrorIntented {
			// Don't report intentionally created errors. Like canceled requests.
			return
		}
		break
	}

	res.Others = make(map[string]interface{})
	res.Others["hammerOthers"] = e.hammer.Others
	res.Others["proxyCountry"] = e.proxyService.GetProxyCountry(p)
	e.resultChan <- res
}

func (e *engine) stop() {
	e.wg.Wait()
	close(e.resultChan)
	<-e.reportService.DoneChan()
	e.proxyService.Done()
	e.scenarioService.Done()
}

func (e *engine) getMaxConcurrentIterCount() int {
	max := 0
	for _, v := range e.reqCountArr {
		if v > max {
			max = v
		}
	}
	return max
}

func (e *engine) initReqCountArr() {
	if e.hammer.Debug {
		e.reqCountArr = []int{1}
		return
	}
	length := int(e.hammer.TestDuration * int(time.Second/(tickerInterval*time.Millisecond)))
	e.reqCountArr = make([]int, length)

	if e.hammer.TimeRunCountMap != nil {
		e.createManualReqCountArr()
	} else {
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

func (e *engine) createManualReqCountArr() {
	tickPerSecond := int(time.Second / (tickerInterval * time.Millisecond))
	stepStartIndex := 0
	for _, t := range e.hammer.TimeRunCountMap {
		steps := make([]int, t.Duration)
		createLinearDistArr(t.Count, steps)

		for i := range steps {
			tickArrStartIndex := (i * tickPerSecond) + stepStartIndex
			tickArrEndIndex := tickArrStartIndex + tickPerSecond
			segment := e.reqCountArr[tickArrStartIndex:tickArrEndIndex]
			createLinearDistArr(steps[i], segment)
		}
		stepStartIndex += len(steps) * tickPerSecond
	}
}

func (e *engine) createLinearReqCountArr() {
	steps := make([]int, e.hammer.TestDuration)
	createLinearDistArr(e.hammer.IterationCount, steps)
	tickPerSecond := int(time.Second / (tickerInterval * time.Millisecond))
	for i := range steps {
		tickArrStartIndex := i * tickPerSecond
		tickArrEndIndex := tickArrStartIndex + tickPerSecond
		segment := e.reqCountArr[tickArrStartIndex:tickArrEndIndex]
		createLinearDistArr(steps[i], segment)
	}
}

func (e *engine) createIncrementalReqCountArr() {
	steps := createIncrementalDistArr(e.hammer.IterationCount, e.hammer.TestDuration)
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
	if quarterWaveCount == 0 {
		quarterWaveCount = 1
	}
	qWaveDuration := int(e.hammer.TestDuration / quarterWaveCount)
	reqCountPerQWave := int(e.hammer.IterationCount / quarterWaveCount)
	tickArrStartIndex := 0

	for i := 0; i < quarterWaveCount; i++ {
		if i == quarterWaveCount-1 {
			// Add remaining req count to the last wave
			reqCountPerQWave += e.hammer.IterationCount - (reqCountPerQWave * quarterWaveCount)
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
	arrLen := len(arr)
	minReqCount := int(count / arrLen)
	remaining := count - minReqCount*arrLen
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

var readTestData = func(testDataConf map[string]types.CsvConf) (map[string]types.CsvData, error) {
	// Read Data
	var readData map[string]types.CsvData
	if len(testDataConf) > 0 {
		readData = make(map[string]types.CsvData, len(testDataConf))
	}
	for k, conf := range testDataConf {
		var rows []map[string]interface{}
		var err error
		rows, err = data.ReadCsv(conf)
		if err != nil {
			return nil, err
		}
		var csvData types.CsvData
		csvData.Rows = rows

		if conf.Order == "random" {
			csvData.Random = true
		}
		readData[k] = csvData
	}

	return readData, nil
}
