package zookeeper

import (
	"fmt"
	"github.com/mesh-operator/pkg/adapter/constant"
	zkClient "github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog"
	"testing"
	"time"
)

func Test_watch(t *testing.T) {
	conn, _, err := zkClient.Connect(constant.ZkServers, time.Duration(15)*time.Second)
	if err != nil || conn == nil {
		klog.Errorf("Get zookeeper client has an error: %v", err)
	}

	//go func() {
	p, err := NewPathCache(conn, DubboRootPath, "MOCK", true)
	if err != nil {
		fmt.Printf("Create a new pathcache for [%s] has an err: %v\n", DubboRootPath, err)
		assert.EqualError(t, err, "zk: node does not exist")
		return
	}
	klog.Infof("Created a path cache : %s\n", p.Path)
	//}()

	time.Sleep(30 * time.Minute)
}
