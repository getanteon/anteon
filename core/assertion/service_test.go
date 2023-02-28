package assertion

import (
	"sync"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

func TestApplyAssertionsAbortsCorrectly(t *testing.T) {
	service := NewAssertionService()
	assertions := make(map[string]types.TestAssertionOpt)
	rule := "false"
	delay := 3
	assertions[rule] = types.TestAssertionOpt{
		Abort: true,
		Delay: delay,
	}
	abortChan := service.Init(assertions)

	inputChan := make(chan *types.ScenarioResult)
	go service.Start(inputChan)

	wg := sync.WaitGroup{}
	go func() {
		wg.Add(1)
		<-abortChan
		wg.Done()
	}()

	go service.ApplyAssertions()
	inputChan <- &types.ScenarioResult{}
	start := time.Now()

	wg.Wait()
	timePassed := time.Since(start).Seconds()
	if int(timePassed) != delay {
		t.Errorf("Delay, got %f, expected %d", timePassed, delay)
	}
}
