package zookeeper

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/mesh-operator/pkg/adapter/events"
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

type ZkConfigClient struct {
	conn          *zk.Conn
	out           chan *events.ConfigEvent
	configEntries map[string]*events.ConfiguratorConfig
	rootPathCache *pathCache
}

func NewConfigClient(conn *zk.Conn) *ZkConfigClient {
	cc := &ZkConfigClient{
		conn:          conn,
		out:           make(chan *events.ConfigEvent),
		configEntries: make(map[string]*events.ConfiguratorConfig),
		rootPathCache: nil,
	}

	return cc
}

func (cc *ZkConfigClient) Start() error {
	rpc, err := newPathCache(cc.conn, ConfiguratorPath)
	if err != nil {
		return err
	}
	cc.rootPathCache = rpc
	go cc.eventLoop()

	// FIXME just for debug
	var debug = true
	if debug {
		go func() {
			tick := time.Tick(10 * time.Second)
			for {
				select {
				case <-tick:
					fmt.Printf("Observing cache of configuration client\n  flags: %v\n  configs: %v\n",
						cc.rootPathCache.cached, cc.configEntries)
				}
			}
		}()
	}

	return nil
}

// eventLoop
func (cc *ZkConfigClient) eventLoop() {
	for event := range cc.rootPathCache.events() {
		var data []byte
		var ce *events.ConfigEvent
		switch event.eventType {
		case pathCacheEventAdded:
			data = cc.getData(event.path)
			config := &events.ConfiguratorConfig{}
			err := yaml.Unmarshal([]byte(data), config)
			if err != nil {
				fmt.Printf("Parsing the configuration data to a defined struct has an error: %v\n", err)
				continue
			}

			cc.configEntries[config.Key] = config
			ce = &events.ConfigEvent{
				EventType:   events.ConfigEntryAdded,
				Path:        event.path,
				ConfigEntry: config,
			}
			go cc.notify(ce)
			break
		case pathCacheEventChanged:
			data = cc.getData(event.path)
			config := &events.ConfiguratorConfig{}
			err := yaml.Unmarshal([]byte(data), config)
			if err != nil {
				fmt.Printf("Parsing the configuration data to a defined struct has an error: %v\n", err)
				continue
			}
			cc.configEntries[config.Key] = config
			ce = &events.ConfigEvent{
				EventType:   events.ConfigEntryChanged,
				Path:        event.path,
				ConfigEntry: config,
			}
			go cc.notify(ce)
			break
		case pathCacheEventDeleted:
			// TODO Deleting configurations about this service in the CR
			cc.rootPathCache.cached[event.path] = false
			ce = &events.ConfigEvent{
				EventType: events.ConfigEntryDeleted,
				Path:      event.path,
			}
			go cc.notify(ce)
			break
		default:
			fmt.Printf("can not support event type yet: %v\n", event.eventType)
		}
	}
}

func (cc *ZkConfigClient) Events() <-chan *events.ConfigEvent {
	return cc.out
}

// getData
func (cc *ZkConfigClient) getData(path string) []byte {
	data, _, err := cc.conn.Get(path)
	if err != nil {
		fmt.Printf("Get data with path %s has an error: %v\n", path, err)
		return data
	}

	//fmt.Printf("Get data with path %s: \n%v\n", path, data)
	return data
}

func (cc *ZkConfigClient) Stop() error {
	return nil
}

// notify
func (cc *ZkConfigClient) notify(event *events.ConfigEvent) {
	cc.out <- event
}
