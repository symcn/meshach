package zookeeper

import (
	"fmt"
	"time"
)

type Request interface {
	Merge(Request) Request
}

type Debounce struct {
	ch          chan Request
	waitTime    time.Duration
	maxWaitTime time.Duration
	pushFn      func(req Request)
}

func New(waitTime, maxWaitTime time.Duration, pushFn func(req Request)) *Debounce {
	d := &Debounce{
		ch:          make(chan Request, 0),
		waitTime:    waitTime,
		maxWaitTime: maxWaitTime,
		pushFn:      pushFn,
	}

	go d.Start()
	return d
}

func (d *Debounce) Start() {
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
		eventDelay := time.Since(startTime)
		quitTime := time.Since(lastUpdateTime)
		if quitTime >= d.waitTime || eventDelay >= d.maxWaitTime {
			if req != nil {
				free = false
				go push(req)
				req = nil
				debounceEvents = 0
			}
		} else {
			timeChan = time.After(d.waitTime - quitTime)
		}
	}

	for {
		select {
		case <-freeCh:
			fmt.Printf("It's time to push by receiving data form free channel: %v\n", time.Now())
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

			fmt.Printf("req %v,r:%v\n", req, r)
			if req == nil {
				req = r
				continue
			}

			req = req.Merge(r)
		case <-timeChan:
			//if !ok{
			//	continue
			//}

			fmt.Printf("It's time to push by receiving data from timeChan: %v\n", time.Now())
			if free {
				pushWorker()
			}
		}
	}

}

func (d *Debounce) Put(req Request) {
	fmt.Printf("Puting request:%v\n", req)
	d.ch <- req
}

func (d *Debounce) Close() {
	close(d.ch)
}
