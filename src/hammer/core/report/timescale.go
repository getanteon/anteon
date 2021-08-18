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
		fmt.Printf("Report service resp receieved: %s\n", r.RequestID)
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
