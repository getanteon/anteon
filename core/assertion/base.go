package assertion

import (
	"go.ddosify.com/ddosify/core/types"
)

type Aborter interface {
	AbortChan() <-chan struct{}
}

type ResultListener interface {
	Start(input <-chan *types.ScenarioResult)
	DoneChan() <-chan struct{} // indicates processing of results are done
}

type Asserter interface {
	ResultChan() <-chan TestAssertionResult
}
