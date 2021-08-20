package report

import (
	"fmt"
	"time"

	"ddosify.com/hammer/core/types"
)

type stdout struct {
	doneChan chan struct{}
}

func (s *stdout) init() {
	s.doneChan = make(chan struct{})
}

func (s *stdout) Start(input chan *types.Response) {
	for r := range input {
		for _, rr := range r.ResponseItems {
			fmt.Printf("[Stdout]Report service resp receieved: %s\n", rr.RequestID)
		}
	}

	time.Sleep(2 * time.Second)
	s.doneChan <- struct{}{}
}

func (s *stdout) Report() {
	fmt.Println("Reported!")
}

func (s *stdout) DoneChan() <-chan struct{} {
	return s.doneChan
}
