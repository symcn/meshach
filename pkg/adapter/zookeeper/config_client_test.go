package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"testing"
	"time"
)

func Test_Start_zk_config_client(t *testing.T) {
	conn, _, err := zk.Connect(ZkServers, 15*time.Second)
	if err != nil {
		fmt.Printf("Connect zk has an error :%v\n", err)
	}

	zkcc := NewConfigClient(conn)
	zkcc.Start()

	//events := zkcc.rootPathCache.events()
	//for {
	//	select {
	//	case e := <-events:
	//		fmt.Printf("event : %v\n", e)
	//	}
	//}

	time.Sleep(30 * time.Minute)
}
