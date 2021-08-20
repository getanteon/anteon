package report

import (
	"fmt"
	"time"

	"ddosify.com/hammer/core/types"
)

type timescale struct {
	doneChan chan struct{}
}

func (t *timescale) init() {
	t.doneChan = make(chan struct{})
}

func (t *timescale) Start(input chan *types.Response) {
	for r := range input {
		for _, rr := range r.ResponseItems {
			fmt.Printf("[Timescale]Report service resp receieved: %s\n", rr.RequestID)
		}
	}

	time.Sleep(2 * time.Second)
	t.doneChan <- struct{}{}
}

func (t *timescale) Report() {
	fmt.Println("Reported!")
}

func (t *timescale) DoneChan() <-chan struct{} {
	return t.doneChan
}
