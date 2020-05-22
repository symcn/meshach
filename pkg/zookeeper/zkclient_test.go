package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"testing"
	"time"
)

var servers = []string{"10.12.210.70:2181"}

func Test_connect(t *testing.T) {
	conn, _, err := zk.Connect(servers, 15*time.Second)
	if err != nil {
		fmt.Printf("Connect zk has an error :%v\n", err)
	}

	c := NewClient(dubboRootPath, conn)
	c.Start()

	time.Sleep(30 * time.Minute)
}
