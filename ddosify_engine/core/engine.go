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
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"reflect"
	"slices"
	"sync"
	"time"

	"go.ddosify.com/ddosify/v2/core/assertion"

	"go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/proxy"
	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/scenario"
	"go.ddosify.com/ddosify/v2/core/scenario/data"
	"go.ddosify.com/ddosify/v2/core/types"
	"go.ddosify.com/ddosify/v2/core/util"
)

const (
	// interval in millisecond
	tickerInterval = 100

	// test result status
	resultDone    = "done"
	resultStopped = "stopped"
	resultAborted = "aborted"
)

type engine struct {
	hammer types.Hammer

	proxyService    proxy.ProxyService
	reportService   report.ReportService
	scenarioService *scenario.ScenarioService

	// for assertion
	aborter     assertion.Aborter
	asserter    assertion.Asserter
	resListener assertion.ResultListener

	tickCounter int
	reqCountArr []int

	// load_spread poisson: each iteration's start from the schedule's start, sorted, and the next one to start
	arrivals    []time.Duration
	nextArrival int

	// manual_load stages by tick, set only when a stage is named or a warm-up
	stages []tickStage

	observers []live.Observer
	collector *live.Collector
	wg        sync.WaitGroup

	// when the ticker started; tick c is meant to start iterations at startedAt + (c+1) intervals
	startedAt time.Time

	resultReportChan chan *types.ScenarioResult
	resultAssertChan chan *types.ScenarioResult

	abortChan   <-chan struct{}
	testSuccess bool

	// ctx ends the schedule: an interrupt. reqCtx is the requests' and cut ends them.
	ctx    context.Context
	reqCtx context.Context
	cut    context.CancelFunc
}

type EngineServices struct {
	Aborter     assertion.Aborter
	Asserter    assertion.Asserter
	ResListener assertion.ResultListener

	ProxyServ  proxy.ProxyService
	ReportServ report.ReportService

	// Observers get the live snapshots and the final report; the CLI sets them
	Observers []live.Observer
}

var InitEngineServices = func(h types.Hammer) (*EngineServices, error) {
	// Initialize things here and pass interfaces to NewEngine which it depends ?
	// this piece can change between implementations
	as := assertion.NewDefaultAssertionService()
	as.Init(h.Assertions)

	// TODO: remove reflection ?
	ps, err := proxy.NewProxyService(h.Proxy.Strategy)
	if err != nil {
		return nil, err
	}
	err = ps.Init(h.Proxy)
	if err != nil {
		return nil, err
	}

	// TODO: remove reflection ?
	rs, err := report.NewReportService(h.ReportDestination)
	if err != nil {
		return nil, err
	}
	if err = rs.Init(h.Debug, h.SamplingRate); err != nil {
		return nil, err
	}
	if ss, ok := rs.(report.SLOSetter); ok && h.SLO != nil {
		ss.SetSLO(h.SLO)
	}
	if cs, ok := rs.(report.CapacitySetter); ok && h.Capacity != nil {
		cs.SetCapacity(h.Capacity)
	}
	if h.RequestLog && !h.Debug {
		if rl, ok := rs.(report.RequestLogSetter); ok {
			rl.SetRequestLog(true)
		} else {
			report.Warn(fmt.Sprintf("request_log writes its lines with the stdout-json output only, not with %s", h.ReportDestination))
		}
	}

	return &EngineServices{
		// defaultAssertionService as implements all
		Aborter:     as,
		Asserter:    as,
		ResListener: as,

		ProxyServ:  ps,
		ReportServ: rs,
	}, nil
}

// NewEngine is the constructor of the engine.
// Hammer is used for initializing the engine itself and its' external services.
// Engine can be stopped by canceling the given ctx.
func NewEngine(ctx context.Context, h types.Hammer,
	services *EngineServices) (e *engine, err error) {
	ss := scenario.NewScenarioService()
	reqCtx, cut := context.WithCancel(context.Background())

	e = &engine{
		hammer:          h,
		ctx:             ctx,
		reqCtx:          reqCtx,
		cut:             cut,
		proxyService:    services.ProxyServ,
		scenarioService: ss,
		reportService:   services.ReportServ,

		// for assertion
		aborter:     services.Aborter,
		resListener: services.ResListener,
		asserter:    services.Asserter,
		observers:   services.Observers,
	}

	return
}

func (e *engine) IsTestFailed() bool {
	return !e.testSuccess
}

func (e *engine) Init() (err error) {
	// read test data
	readData, err := readTestData(e.hammer.TestDataConf)
	if err != nil {
		return err
	}
	e.hammer.Scenario.Data = readData

	e.initReqCountArr()

	var initialCookies []*http.Cookie
	if e.hammer.CookiesEnabled && len(e.hammer.Cookies) > 0 {
		initialCookies, err = createInitialCookies(e.hammer.Cookies)
		if err != nil {
			return err
		}
	}

	for _, st := range e.hammer.Scenario.Steps {
		if st.LLM != nil && st.LLM.Chat != nil {
			st.LLM.Chat.Seed = e.hammer.Seed
		}
	}

	if err = e.scenarioService.Init(e.reqCtx, e.hammer.Scenario, e.proxyService.GetAll(), scenario.ScenarioOpts{
		Debug:                  e.hammer.Debug,
		IterationCount:         e.hammer.IterationCount,
		MaxConcurrentIterCount: e.getMaxConcurrentIterCount(),
		EngineMode:             e.hammer.EngineMode,
		InitialCookies:         initialCookies,
		Seed:                   e.hammer.Seed,
		StopOnFailure:          e.hammer.StopOnFailure,
	}); err != nil {
		return
	}

	e.abortChan = e.aborter.AbortChan()

	if len(e.observers) > 0 {
		e.collector = live.NewCollector(e.plan(), e.hammer.SLO, e.observers)
	}
	return
}

func (e *engine) plan() live.Plan {
	tickDur := time.Duration(tickerInterval) * time.Millisecond
	p := live.Plan{
		Duration:   time.Duration(len(e.reqCountArr)) * tickDur,
		Iterations: arraySum(e.reqCountArr),
		LoadType:   e.hammer.LoadType,
		EngineMode: e.hammer.EngineMode,
		Schedule:   append([]int(nil), e.reqCountArr...),
		SLO:        e.hammer.SLO,
	}
	if e.closedLoop() {
		p.Duration = time.Duration(e.hammer.TestDuration) * time.Second // the caps; 0 when no stage has one
		p.Iterations = e.hammer.IterationCount
		p.ClosedLoop = true
		for i, t := range e.hammer.TimeRunCountMap {
			p.Stages = append(p.Stages, live.Stage{Index: i, Name: t.Name, Warmup: t.Warmup,
				Iterations: t.Count, Concurrency: t.Concurrency, Cap: time.Duration(t.Duration) * time.Second})
		}
	}
	for _, ts := range e.stages {
		p.Stages = append(p.Stages, live.Stage{Index: ts.stage.Index, Name: ts.stage.Name, Warmup: ts.stage.Warmup,
			Start: time.Duration(ts.start) * tickDur, End: time.Duration(ts.end) * tickDur,
			Iterations: arraySum(e.reqCountArr[ts.start:ts.end])})
	}
	for _, s := range e.hammer.Scenario.Steps {
		info := live.StepInfo{ID: s.ID, Name: s.Name, Method: s.Method, URL: s.URL, Streamed: s.Stream != nil}
		if s.LLM != nil {
			info.CacheBust, info.SharedPrefixTokens, info.TokenSource = s.LLM.CacheBust, s.LLM.SharedPrefixTokens, s.LLM.TokenSource
		}
		p.Steps = append(p.Steps, info)
	}
	return p
}

func (e *engine) Start() (status string) {
	ticker := time.NewTicker(time.Duration(tickerInterval) * time.Millisecond)
	e.resultReportChan = make(chan *types.ScenarioResult, e.hammer.IterationCount)
	e.resultAssertChan = make(chan *types.ScenarioResult, e.hammer.IterationCount)

	var testResultChan <-chan assertion.TestAssertionResult
	if e.runAssertionsInEngine() {
		// run test wide assertions in parallel
		testResultChan = e.asserter.ResultChan()
	}

	if len(e.hammer.Assertions) > 0 { // test-wide assertions given
		go e.resListener.Start(e.resultAssertChan)
	}

	if dp, ok := e.reportService.(report.DeferredPrinter); ok && e.collector != nil {
		dp.DeferPrint()
	}
	go e.reportService.Start(e.resultReportChan, testResultChan)

	if e.collector != nil {
		e.collector.Start()
	}

	defer func() {
		ticker.Stop()
		status = e.stop(status)
	}()

	e.tickCounter = 0
	e.startedAt = time.Now()
	e.wg = sync.WaitGroup{}
	if e.closedLoop() {
		return e.runClosedLoop()
	}
	var mutex = &sync.Mutex{}
	for range ticker.C {
		if e.tickCounter >= len(e.reqCountArr) {
			return resultDone
		}

		select {
		case <-e.ctx.Done():
			return resultStopped
		case <-e.abortChan:
			e.testSuccess = false
			return resultAborted
		default:
			mutex.Lock()
			n := e.reqCountArr[e.tickCounter]
			e.wg.Add(n)
			var at []time.Duration
			if e.arrivals != nil {
				at = e.arrivals[e.nextArrival : e.nextArrival+n]
				e.nextArrival += n
			}
			go e.runWorkers(e.tickCounter, at)
			e.tickCounter++
			mutex.Unlock()
		}
	}
	return resultDone
}

// runWorkers starts the iterations of tick c. at, with load_spread poisson, holds their starts
// from the schedule's start: each waits for its own, which is also its intended start.
func (e *engine) runWorkers(c int, at []time.Duration) {
	stage := e.stageOf(c)
	if at != nil {
		e.runArrivals(stage, at)
		return
	}
	intended := e.startedAt.Add(time.Duration(c+1) * tickerInterval * time.Millisecond)
	for i := 1; i <= e.reqCountArr[c]; i++ {
		scenarioStartTime := time.Now()
		go func(t time.Time) {
			e.runWorker(t, stage, intended)
			e.wg.Done()
		}(scenarioStartTime)
	}
}

func (e *engine) runArrivals(stage *types.LoadStage, at []time.Duration) {
	// tick c fires at startedAt + (c+1) intervals, so the schedule's start is one interval in
	zero := e.startedAt.Add(tickerInterval * time.Millisecond)
	var timer *time.Timer
	for i, off := range at {
		intended := zero.Add(off)
		if d := time.Until(intended); d > 0 {
			if timer == nil {
				timer = time.NewTimer(d)
			} else {
				timer.Reset(d)
			}
			select {
			case <-timer.C:
			case <-e.ctx.Done():
				// interrupted: the schedule ends, and these iterations never start
				timer.Stop()
				for range at[i:] {
					e.wg.Done()
				}
				return
			}
		}
		go func() {
			e.runWorker(time.Now(), stage, intended)
			e.wg.Done()
		}()
	}
	if timer != nil {
		timer.Stop()
	}
}

func (e *engine) closedLoop() bool {
	return !e.hammer.Debug && e.hammer.TimeRunCountMap.ClosedLoop()
}

// runClosedLoop runs the stages one after the other. A stage starts its count of iterations with
// at most its concurrency in flight, a new one as soon as one ends, as vllm bench serve does with
// --max-concurrency and --num-prompts; it ends when all have ended, or at its duration when it has
// one, after which it starts no more and lets those in flight finish.
func (e *engine) runClosedLoop() string {
	for i, t := range e.hammer.TimeRunCountMap {
		stage := &types.LoadStage{Index: i, Name: t.Name, Warmup: t.Warmup, Concurrency: t.Concurrency}
		if e.collector != nil {
			e.collector.EnterStage(i)
		}
		if status := e.runStage(stage, t.Count, time.Duration(t.Duration)*time.Second); status != "" {
			return status
		}
	}
	return resultDone
}

// runStage returns "" when the stage ended, or the status that ended the test.
func (e *engine) runStage(stage *types.LoadStage, count int, limit time.Duration) string {
	var capped <-chan time.Time
	if limit > 0 {
		t := time.NewTimer(limit)
		defer t.Stop()
		capped = t.C
	}
	slots := make(chan struct{}, stage.Concurrency)
	var inStage sync.WaitGroup
	interrupted := func() string {
		select {
		case <-e.ctx.Done():
			return resultStopped
		case <-e.abortChan:
			e.testSuccess = false
			return resultAborted
		default:
			return ""
		}
	}
launch:
	for n := 0; n < count; n++ {
		select {
		case slots <- struct{}{}:
		case <-e.ctx.Done():
			return resultStopped
		case <-e.abortChan:
			e.testSuccess = false
			return resultAborted
		case <-capped:
			break launch
		}
		if status := interrupted(); status != "" {
			<-slots
			return status
		}
		select {
		case <-capped:
			<-slots
			break launch
		default:
		}
		e.wg.Add(1)
		inStage.Add(1)
		go func(start time.Time) {
			defer func() {
				<-slots
				inStage.Done()
				e.wg.Done()
			}()
			e.runWorker(start, stage, start)
		}(time.Now())
	}
	done := make(chan struct{})
	go func() {
		inStage.Wait()
		close(done)
	}()
	select {
	case <-done:
		return ""
	case <-e.ctx.Done():
		return resultStopped
	case <-e.abortChan:
		e.testSuccess = false
		return resultAborted
	}
}

type tickStage struct {
	start, end int // ticks, end not inclusive
	stage      types.LoadStage
}

func (e *engine) stageOf(tick int) *types.LoadStage {
	for i := range e.stages {
		if tick >= e.stages[i].start && tick < e.stages[i].end {
			return &e.stages[i].stage
		}
	}
	return nil
}

func (e *engine) initStages() {
	tagged := false
	for _, t := range e.hammer.TimeRunCountMap {
		if t.Name != "" || t.Warmup {
			tagged = true
		}
	}
	if !tagged || e.hammer.Debug {
		return
	}
	tickPerSecond := int(time.Second / (tickerInterval * time.Millisecond))
	start := 0
	for i, t := range e.hammer.TimeRunCountMap {
		end := start + t.Duration*tickPerSecond
		e.stages = append(e.stages, tickStage{start: start, end: end,
			stage: types.LoadStage{Index: i, Name: t.Name, Warmup: t.Warmup}})
		start = end
	}
}

func (e *engine) runWorker(scenarioStartTime time.Time, stage *types.LoadStage, intended time.Time) {
	if e.collector != nil {
		e.collector.IterationStarted()
	}
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

		break
	}
	if res == nil || (len(res.StepResults) == 0 && err == nil) { // the requesters could not be built, or every step was skipped
		if e.collector != nil {
			e.collector.IterationDone(nil, stage, 0)
		}
		return
	}

	res.Others = make(map[string]interface{})
	if err != nil && err.Type == types.ErrorIntented {
		// cut while in flight: reported as interrupted, out of the latencies
		res.Others[types.OthersInterrupted] = true
	}
	res.Others["hammerOthers"] = e.hammer.Others
	res.Others["proxyCountry"] = e.proxyService.GetProxyCountry(p)
	if stage != nil {
		res.Others["loadStage"] = *stage
	}
	if !e.startedAt.IsZero() {
		res.Others[types.OthersIntendedStart] = intended
	}
	if e.collector != nil {
		e.collector.IterationDone(res, stage, time.Since(scenarioStartTime))
	}
	e.resultReportChan <- res

	if len(e.hammer.Assertions) > 0 {
		e.resultAssertChan <- res
	}
}

func (e *engine) runAssertionsInEngine() bool {
	return e.hammer.SingleMode && len(e.hammer.Assertions) > 0
}

// Cut ends the requests in flight now, without waiting for the drain. Safe to call more than once.
func (e *engine) Cut() {
	e.cut()
}

// drain waits for the iterations in flight. An interrupt, during the schedule or after it, makes
// the status stopped and gives them hammer.Drain to finish before they are cut.
func (e *engine) drain(status string) string {
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()
	interrupted := e.ctx.Done()
	var deadline <-chan time.Time
	for {
		select {
		case <-done:
			return status
		case <-interrupted:
			interrupted = nil
			if status != resultAborted {
				status = resultStopped
			}
			if e.hammer.Drain <= 0 {
				e.cut()
			} else {
				deadline = time.After(e.hammer.Drain)
			}
		case <-deadline:
			deadline = nil
			e.cut()
		}
	}
}

func (e *engine) stop(status string) string {
	status = e.drain(status)
	defer e.cut()
	if rs, ok := e.reportService.(report.RunStatusSetter); ok {
		rs.SetRunStatus(status)
	}
	if ss, ok := e.reportService.(report.StartSetter); ok && !e.startedAt.IsZero() {
		ss.SetStartedAt(e.startedAt)
	}
	close(e.resultReportChan)
	close(e.resultAssertChan)
	e.proxyService.Done()
	e.scenarioService.Done()

	if len(e.hammer.Assertions) > 0 { // if results are listened, wait
		<-e.resListener.DoneChan()
	}

	e.testSuccess = <-e.reportService.DoneChan()

	if e.collector != nil {
		var result *report.Result
		if rp, ok := e.reportService.(report.ResultProvider); ok {
			result = rp.Result()
		}
		e.collector.Finish(status, e.testSuccess, result)
		if dp, ok := e.reportService.(report.DeferredPrinter); ok {
			dp.PrintReport()
		}
	}
	return status
}

func (e *engine) getMaxConcurrentIterCount() int {
	if e.closedLoop() {
		return e.hammer.TimeRunCountMap.MaxConcurrency()
	}
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
	if e.closedLoop() {
		return
	}
	length := int(e.hammer.TestDuration * int(time.Second/(tickerInterval*time.Millisecond)))
	e.reqCountArr = make([]int, length)

	if e.hammer.TimeRunCountMap != nil {
		e.createManualReqCountArr()
		e.initStages()
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

func (e *engine) evenSpread() bool {
	return e.hammer.LoadSpread == types.LoadSpreadEven
}

func (e *engine) poissonSpread() bool {
	return e.hammer.LoadSpread == types.LoadSpreadPoisson
}

// poissonOffsets draws count start times in [0, d), sorted: a Poisson process conditioned on its
// count, so a stage starts exactly count iterations at independent, exponentially spaced times.
func poissonOffsets(count int, d time.Duration, r *rand.Rand) []time.Duration {
	offs := make([]time.Duration, count)
	for i := range offs {
		o := time.Duration(r.Float64() * float64(d))
		if o >= d {
			o = d - 1
		}
		offs[i] = o
	}
	slices.Sort(offs)
	return offs
}

// stageRand is the source of stage i's arrivals: the same for a seed, and independent across stages.
func stageRand(seed int64, stage int) *rand.Rand {
	return rand.New(rand.NewSource(int64(util.SeedMix(seed, uint64(stage), 0))))
}

// addArrivals puts a stage's poisson arrivals, starting at tick start, into the ticks and arrivals.
func (e *engine) addArrivals(stage, count, start int, d time.Duration) {
	tickDur := time.Duration(tickerInterval) * time.Millisecond
	base := time.Duration(start) * tickDur
	for _, o := range poissonOffsets(count, d, stageRand(e.hammer.Seed, stage)) {
		e.reqCountArr[start+int(o/tickDur)]++
		e.arrivals = append(e.arrivals, base+o)
	}
}

func (e *engine) createManualReqCountArr() {
	tickPerSecond := int(time.Second / (tickerInterval * time.Millisecond))
	stepStartIndex := 0
	if e.poissonSpread() {
		e.arrivals = make([]time.Duration, 0, e.hammer.IterationCount)
	}
	for i, t := range e.hammer.TimeRunCountMap {
		if e.poissonSpread() {
			e.addArrivals(i, t.Count, stepStartIndex, time.Duration(t.Duration)*time.Second)
			stepStartIndex += t.Duration * tickPerSecond
			continue
		}
		if e.evenSpread() {
			end := stepStartIndex + t.Duration*tickPerSecond
			createEvenDistArr(t.Count, e.reqCountArr[stepStartIndex:end])
			stepStartIndex = end
			continue
		}
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
	if e.poissonSpread() {
		e.arrivals = make([]time.Duration, 0, e.hammer.IterationCount)
		e.addArrivals(0, e.hammer.IterationCount, 0, time.Duration(e.hammer.TestDuration)*time.Second)
		return
	}
	if e.evenSpread() {
		createEvenDistArr(e.hammer.IterationCount, e.reqCountArr)
		return
	}
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
		e.spreadInSecond(steps[i], segment)
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
			e.spreadInSecond(steps[j], segment)
			tickArrStartIndex += tickPerSecond
		}
	}
}

func (e *engine) spreadInSecond(count int, ticks []int) {
	if e.evenSpread() {
		createEvenDistArr(count, ticks)
		return
	}
	createLinearDistArr(count, ticks)
}

// createEvenDistArr puts count iterations at even intervals over arr, the first one at arr[0].
// 30 over 3000 ticks is one every 100 ticks, where createLinearDistArr fills the first 30.
func createEvenDistArr(count int, arr []int) {
	n := int64(len(arr))
	if n == 0 {
		return
	}
	c := int64(count)
	ceilDiv := func(a, b int64) int64 { return (a + b - 1) / b }
	for i := int64(0); i < n; i++ {
		arr[i] = int(ceilDiv((i+1)*c, n) - ceilDiv(i*c, n))
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

func parseRawCookie(cookie string) []*http.Cookie {
	header := http.Header{}
	header.Add("Set-Cookie", cookie)
	req := http.Response{Header: header}
	return req.Cookies()
}

var createInitialCookies = func(cookies []types.CustomCookie) ([]*http.Cookie, error) {
	initialCookies := make([]*http.Cookie, 0, len(cookies))
	for _, c := range cookies {
		var ck *http.Cookie
		if c.Raw != "" {
			cookies := parseRawCookie(c.Raw)
			if len(cookies) == 0 {
				return nil, fmt.Errorf("cookie could not be parsed, got : %s", c.Raw)
			}
			ck = cookies[0]
		} else {
			var expires time.Time
			if c.Expires != "" {
				var err error
				expires, err = time.Parse(time.RFC1123, c.Expires)
				if err != nil {
					return nil, fmt.Errorf("error parsing cookie expiry: %s", err)
				}
			}

			ck = &http.Cookie{
				Name:       c.Name,
				Value:      c.Value,
				Path:       c.Path,
				Domain:     c.Domain,
				Expires:    expires,
				RawExpires: c.Expires,
				MaxAge:     c.MaxAge,
				Secure:     c.Secure,
				HttpOnly:   c.HttpOnly,
				Raw:        c.Raw,

				// below fields not used
				SameSite: 0,
				Unparsed: []string{},
			}
		}

		initialCookies = append(initialCookies, ck)
	}

	return initialCookies, nil
}
