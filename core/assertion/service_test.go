package assertion

import (
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

func TestApplyAssertionsAbortsCorrectly(t *testing.T) {
	service := NewDefaultAssertionService()
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
	wg.Add(1)
	go func() {
		<-abortChan
		wg.Done()
	}()

	inputChan <- &types.ScenarioResult{}
	start := time.Now()

	wg.Wait()
	timePassed := time.Since(start).Seconds()
	if int(timePassed) != delay {
		t.Errorf("Delay, got %f, expected %d", timePassed, delay)
	}
}

func TestServiceKeepsIterationTimes(t *testing.T) {
	service := NewDefaultAssertionService()
	assertions := make(map[string]types.TestAssertionOpt)
	rule := "false"
	delay := 3
	assertions[rule] = types.TestAssertionOpt{
		Abort: false,
		Delay: delay,
	}
	_ = service.Init(assertions)

	inputChan := make(chan *types.ScenarioResult)
	go service.Start(inputChan)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-service.ResultChan()
		wg.Done()
	}()

	expectedIterationTimes := SortableInt64Slice{}
	for i := 0; i < 10; i++ {
		iterTime := time.Duration(((i * 5) % 4) * int(time.Millisecond))
		expectedIterationTimes = append(expectedIterationTimes, iterTime.Milliseconds())
		inputChan <- &types.ScenarioResult{
			StepResults: []*types.ScenarioStepResult{
				{
					StepID:   1,
					Duration: iterTime,
				},
			},
		}
	}
	sort.Sort(expectedIterationTimes)
	close(inputChan)

	wg.Wait()

	iterationTimes := service.GetTotalTimes()

	if !reflect.DeepEqual(iterationTimes, []int64(expectedIterationTimes)) {
		t.Errorf("TestServiceKeepsIterationTimes, cumulative data store failed")
	}
}

func TestServiceKeepsFailCount(t *testing.T) {
	service := NewDefaultAssertionService()
	assertions := make(map[string]types.TestAssertionOpt)
	_ = service.Init(assertions)

	inputChan := make(chan *types.ScenarioResult)
	go service.Start(inputChan)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-service.ResultChan()
		wg.Done()
	}()

	N := 10
	// 2*N times failed iteration result
	for i := 0; i < N; i++ {
		inputChan <- &types.ScenarioResult{
			StepResults: []*types.ScenarioStepResult{
				{
					StepID: 1,
					FailedAssertions: []types.FailedAssertion{
						{
							Rule:     "failed assertion expression",
							Received: map[string]interface{}{},
							Reason:   "",
						},
					},
				},
			},
		}
		inputChan <- &types.ScenarioResult{
			StepResults: []*types.ScenarioStepResult{
				{
					StepID: 1,
					Err: types.RequestError{
						Type:   "server error type",
						Reason: "",
					},
				},
			},
		}
	}
	close(inputChan)

	wg.Wait()

	failCount := service.GetFailCount()

	if failCount != 2*N {
		t.Errorf("TestServiceKeepsFailCount, expected : %d, got : %d", 2*N, failCount)
	}
}

type SortableInt64Slice []int64

func (a SortableInt64Slice) Len() int           { return len(a) }
func (a SortableInt64Slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortableInt64Slice) Less(i, j int) bool { return a[i] < a[j] }
