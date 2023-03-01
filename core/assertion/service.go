package assertion

import (
	"sync"
	"time"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion"
	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/evaluator"
	"go.ddosify.com/ddosify/core/types"
)

var tickerInterval = 1000 // interval in millisecond
type AssertionService struct {
	assertions map[string]types.TestAssertionOpt // Rule -> Opts
	resultChan chan *types.ScenarioResult
	abortChan  chan struct{}
	doneChan   chan bool // TODO verbose exp
	assertEnv  *evaluator.AssertEnv
	abortTick  map[string]int // rule -> tickIndex
	mu         sync.Mutex
}

func NewAssertionService() (service *AssertionService) {
	return &AssertionService{}
}

func (as *AssertionService) Init(assertions map[string]types.TestAssertionOpt) chan struct{} {
	as.assertions = assertions
	abortChan := make(chan struct{})
	as.abortChan = abortChan
	doneChan := make(chan bool)
	as.doneChan = doneChan
	as.assertEnv = &evaluator.AssertEnv{}
	as.abortTick = make(map[string]int)
	as.mu = sync.Mutex{}
	return as.abortChan
}

func (as *AssertionService) GetTotalTimes() []int64 {
	return as.assertEnv.TotalTime
}
func (as *AssertionService) GetFailCount() int {
	return as.assertEnv.FailCount
}

func (as *AssertionService) Start(input chan *types.ScenarioResult) {
	// get iteration results ,add store them cumulatively
	firstResult := true
	for r := range input {
		as.mu.Lock()
		as.aggregate(r)
		as.mu.Unlock()

		// after first result start checking assertions
		if firstResult {
			go as.applyAssertions()
			firstResult = false
		}
	}
	as.doneChan <- as.giveFinalResult()
}

func (as *AssertionService) aggregate(r *types.ScenarioResult) {
	var iterationTime int64
	var iterFailed bool
	for _, sr := range r.StepResults {
		iterationTime += sr.Duration.Milliseconds()
		if sr.Err.Type != "" || len(sr.FailedAssertions) > 0 {
			iterFailed = true
		}
	}
	if iterFailed {
		as.assertEnv.FailCount++
	}
	as.assertEnv.TotalTime = append(as.assertEnv.TotalTime, iterationTime)
}

func (as *AssertionService) applyAssertions() {
	ticker := time.NewTicker(time.Duration(tickerInterval) * time.Millisecond)
	tickIndex := 1
	for range ticker.C {
		as.mu.Lock()
		var totalTime []int64
		totalTime = append(totalTime, as.assertEnv.TotalTime...)
		assertEnv := evaluator.AssertEnv{
			TotalTime: totalTime,
			FailCount: as.assertEnv.FailCount,
		}
		as.mu.Unlock()

		// apply assertions
		for rule, opts := range as.assertions {
			res, err := assertion.Assert(rule, &assertEnv)
			if err != nil {
				// TODO
			}
			if res == false && opts.Abort {
				// if delay is zero, immediately abort
				if opts.Delay == 0 || as.abortTick[rule] == tickIndex {
					as.abortChan <- struct{}{}
					return
				}
				if _, ok := as.abortTick[rule]; !ok {
					// schedule check at
					delayTick := (time.Duration(opts.Delay) * time.Second) / (time.Duration(tickerInterval) * time.Millisecond)
					as.abortTick[rule] = tickIndex + int(delayTick) - 1
				}
			}
		}
		tickIndex++
	}
}

// TODO return a verbose explanation
func (as *AssertionService) giveFinalResult() bool {
	// return final result
	for rule, _ := range as.assertions {
		res, err := assertion.Assert(rule, as.assertEnv)
		if err != nil {
			// TODO
		}
		if res == false {
			return false
		}
	}
	return true
}

func (as *AssertionService) Done() chan bool {
	return as.doneChan
}
