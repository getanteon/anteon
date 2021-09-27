/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package report

import (
	"fmt"
	"time"

	"go.ddosify.com/ddosify/core/types"
)

type timescale struct {
	doneChan chan struct{}
}

func (t *timescale) Init() (err error) {
	t.doneChan = make(chan struct{})
	return
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
