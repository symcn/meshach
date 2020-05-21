package zookeeper

import (
	"fmt"
	"testing"
	"time"
)

func Test_StartController(t *testing.T) {
	adapter, err := NewAdapter("10.12.210.70:2181", "")

	if err != nil {
		fmt.Sprintf("Start a controller has an error: %v\n", err)
	}

	stop := make(chan struct{})
	adapter.Run(stop)

	time.Sleep(30 * time.Minute)

	stop <- struct{}{}
	time.Sleep(5 * time.Second)

}
