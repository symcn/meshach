package accelerate

import (
	"fmt"
	"hash/fnv"
)

// Accelerator putting your events and handling process into this accelerator to execute them in concurrency
type Accelerator struct {
	size     int
	channels []chan *ExeRequest
	done     <-chan struct{}
}

// ExeRequest ...
type ExeRequest struct {
	fn func()
}

// NewAccelerator initializing a new accelerator
func NewAccelerator(size int, done <-chan struct{}) *Accelerator {
	ac := &Accelerator{
		size: size,
		done: done,
	}

	for i := 0; i < size; i++ {
		arChan := make(chan *ExeRequest)
		ac.channels = append(ac.channels, arChan)

		go func(c chan *ExeRequest) {
			for {
				select {
				case r := <-c:
					fmt.Printf("Take an accelerate request from a buffer : %v\n", r)
					r.fn()
				case <-done:
					return
				}

			}
		}(arChan)
	}

	return ac
}

// Accelerate The purpose of this accelerator is twofold:
// 1.dividing the events into a group of channels by calculating hash code of event's name
// 2.there is one goroutine for each channel is waiting for handling the accelerate request
func (ac *Accelerator) Accelerate(fn func(), hashKey string) {
	// The case that channels' size is less than 0 means this function could be execute in current goroutine.
	if ac.size <= 0 {
		fn()
	} else {
		hashcode := FNV32a(hashKey)
		c := ac.channels[int(hashcode)%ac.size]

		go func(fn func()) {
			c <- &ExeRequest{
				fn: fn,
			}
		}(fn)
	}
}

// FNV32a calculating the hash code of a text
func FNV32a(text string) uint32 {
	algorithm := fnv.New32a()
	algorithm.Write([]byte(text))
	return algorithm.Sum32()
}
