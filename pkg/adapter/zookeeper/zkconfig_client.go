package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

type ZkConfigClient struct {
	conn             *zk.Conn
	out              chan interface{}
	rootPathCache    *pathCache
	configPathCaches map[string]*pathCache
}

func NewConfigClient(conn *zk.Conn) *ZkConfigClient {
	cc := &ZkConfigClient{
		conn:             conn,
		out:              make(chan interface{}),
		rootPathCache:    nil,
		configPathCaches: make(map[string]*pathCache),
	}

	return cc
}

func (cc *ZkConfigClient) Start() error {
	rootpc, err := newPathCache(cc.conn, ConfiguratorPath)
	if err != nil {
		return err
	}
	cc.rootPathCache = rootpc
	go cc.eventLoop()

	// FIXME just for debug
	go func() {
		tick := time.Tick(10 * time.Second)
		for {
			select {
			case <-tick:
				fmt.Printf("The cache of root path for zk configuration client: \n%v\n", cc.rootPathCache.cached)
			}
		}
	}()

	return nil
}

// eventLoop
func (cc *ZkConfigClient) eventLoop() {
	for event := range cc.rootPathCache.events() {
		switch event.eventType {
		case pathCacheEventAdded:
			cc.getData(event.path)
			break
		case pathCacheEventChanged:
			cc.getData(event.path)
			break
		case pathCacheEventDeleted:
			// TODO Deleting configurations about this service in the CR
			cc.rootPathCache.cached[event.path] = false
			break
		default:
			fmt.Printf("can not support event type yet: %v\n", event.eventType)
		}
	}
}

func (cc *ZkConfigClient) Events() <-chan interface{} {
	return nil
}

// getData
func (cc *ZkConfigClient) getData(path string) string {
	var data string
	dataBytes, _, err := cc.conn.Get(path)
	if err != nil {
		fmt.Printf("Get data with path %s has an error: %v\n", path, err)
		return data
	}
	data = string(dataBytes)
	fmt.Printf("Get data with path %s: %v\n", path, data)
	return data
}

func (cc *ZkConfigClient) Stop() error {
	return nil
}
