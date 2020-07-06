// Package debounce provides a debouncer func.
package debounce

import (
	"fmt"
	"time"
)

// Request ...
type Request interface {
	Merge(Request) Request
}

// Debounce ...
type Debounce struct {
	ch          chan Request
	waitTime    time.Duration     // The duration it should wait when there is no request has been put.
	maxWaitTime time.Duration     // The duration limit if there are a lot of requests is be put into continually.
	pushFn      func(req Request) // Debounced func
}

// New ...
func New(waitTime, maxWaitTime time.Duration, pushFn func(req Request)) *Debounce {
	d := &Debounce{
		ch:          make(chan Request, 0),
		waitTime:    waitTime,
		maxWaitTime: maxWaitTime,
		pushFn:      pushFn,
	}

	go d.start()
	return d
}

func (d *Debounce) start() {
	var timeChan <-chan time.Time
	var startTime time.Time
	var lastUpdateTime time.Time
	debounceEvents := 0
	var req Request
	free := true
	freeCh := make(chan struct{}, 1)

	push := func(req Request) {
		d.pushFn(req)
		freeCh <- struct{}{}
	}

	pushWorker := func() {
		lastUpdateDuration := time.Since(lastUpdateTime)
		if lastUpdateDuration >= d.waitTime || time.Since(startTime) >= d.maxWaitTime {
			if req != nil {
				free = false
				go push(req)
				req = nil
				debounceEvents = 0
			}
		} else {
			timeChan = time.After(d.waitTime - lastUpdateDuration)
			fmt.Println("Time duration is less than wait time so that it doesn't be trigger to push")
		}
	}

	for {
		select {
		case <-freeCh:
			fmt.Printf("It's time to push request when receiving data from free channel: %v\n", time.Now())
			free = true
			pushWorker()
		case r, ok := <-d.ch:
			if !ok {
				return
			}

			lastUpdateTime = time.Now()
			if debounceEvents == 0 {
				timeChan = time.After(d.waitTime)
				startTime = lastUpdateTime
			}
			debounceEvents++

			fmt.Printf("req %v,r %v\n", req, r)
			if req == nil {
				req = r
				continue
			}

			req = req.Merge(r)
		case <-timeChan:
			fmt.Printf("It's time to push request when receiving data from timeChan: %v, free: %v\n", time.Now(), free)
			if free {
				pushWorker()
			}
		}
	}

}

// Put ...
func (d *Debounce) Put(req Request) {
	fmt.Printf("Putting request:%v\n", req)
	d.ch <- req
}

// Close ...
func (d *Debounce) Close() {
	close(d.ch)
}
