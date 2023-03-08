package assertion

import (
	"go.ddosify.com/ddosify/core/types"
)

type Aborter interface {
	AbortChan() chan struct{}
}
type ResultListener interface {
	Start(input chan *types.ScenarioResult)
}

type Asserter interface {
	ResultChan() chan TestAssertionResult
}
