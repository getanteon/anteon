package assertion

import (
	"go.ddosify.com/ddosify/core/types"
)

var AvailableAssertionServices = make(map[string]AssertionService)

// AssertionService is the interface that abstracts different assertion implementations.
type AssertionService interface {
	ResultChan() chan TestAssertionResult
	Start(input chan *types.ScenarioResult)
	AbortChan() chan struct{}
}
