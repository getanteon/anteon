package report

import (
	"fmt"

	"ddosify.com/hammer/core/types"
)

type stdout struct {
	doneChan chan struct{}
	result   []*types.ResponseItem
}

func (s *stdout) init() {
	s.doneChan = make(chan struct{})
}

func (s *stdout) Start(input chan *types.Response) {
	for r := range input {

		for _, rr := range r.ResponseItems {
			fmt.Printf("[Stdout]Report service resp receieved: %v\n", rr)
			s.result = append(s.result, rr)
		}
	}

	s.doneChan <- struct{}{}
}

func (s *stdout) Report() {
	fmt.Printf("Reported! %d items\n", len(s.result))
}

func (s *stdout) DoneChan() <-chan struct{} {
	return s.doneChan
}
