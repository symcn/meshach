package zookeeper

import (
	"fmt"
	"testing"
	"time"
)

func Test_Start(t *testing.T) {
	opt := Option{
		Address: []string{"10.12.210.70:2181"},
		Timeout: 15,
	}
	adapter, err := NewAdapter(&opt)

	if err != nil {
		fmt.Sprintf("Start a controller has an error: %v\n", err)
	}

	stop := make(chan struct{})
	adapter.Run(stop)

	time.Sleep(30 * time.Minute)

	stop <- struct{}{}
	time.Sleep(5 * time.Second)

}
