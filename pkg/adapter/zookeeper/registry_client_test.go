package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"testing"
	"time"
)

var ZkServers = []string{
	// local
	//"10.12.210.70:2181",
	// dev
	"10.248.224.37:2181",
	// dev - dsf
	//"10.248.224.25:2181",

}

func Test_connect(t *testing.T) {
	conn, _, err := zk.Connect(ZkServers, 15*time.Second)
	if err != nil {
		fmt.Printf("Connect zk has an error :%v\n", err)
	}

	rc := NewRegistryClient(conn)
	rc.Start()

	time.Sleep(30 * time.Minute)
}
