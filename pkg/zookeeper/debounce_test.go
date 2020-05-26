package zookeeper

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type MockRequest struct {
	Index int
}

func (mr *MockRequest) Merge(request Request) Request {
	fmt.Printf("Merging request:%v\n", mr.Index)
	return request
}

func Test_start(t *testing.T) {
	fmt.Printf("Start at: %v\n", time.Now())

	pushFn := func(req Request) {
		fmt.Printf("Pushing request: %v\n", req)
	}

	d := New(time.Duration(20)*time.Second, 30*time.Second, pushFn)
	go func() {
		for i := 1; i < 10; i++ {
			d.Put(&MockRequest{Index: i})
			time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
		}
	}()

	d.Start()

	time.Sleep(3 * time.Second)

}
