package assertion

import (
	"go.ddosify.com/ddosify/core/types"
)

type AssertionService struct {
	assertions []types.TestAssertion
	resultChan chan *types.ScenarioResult
	abortChan  chan struct{}
}

func NewAssertionService() (service *AssertionService) {
	return &AssertionService{}
}

func (as *AssertionService) Init(assertions []types.TestAssertion) chan struct{} {
	as.assertions = assertions
	abortChan := make(chan struct{})
	as.abortChan = abortChan
	return as.abortChan
}

func (as *AssertionService) Start(input chan *types.ScenarioResult) {
	// get iteration results cumulatively and calculate rules

}
