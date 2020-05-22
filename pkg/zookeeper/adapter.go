package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
	"time"
)

type Adapter struct {
	zkClient      *ZkClient
	eventHandlers []EventHandler
}

func NewAdapter(address string, root string) (*Adapter, error) {
	servers := strings.Split(address, ",")
	conn, _, err := zk.Connect(servers, 15*time.Second)
	if err != nil {
		return nil, err
	}

	zkClient := NewClient(dubboRootPath, conn)
	eventHandlers := []EventHandler{}
	eventHandlers = append(eventHandlers, &SimpleEventHandler{Name: "testHandler"})
	eventHandlers = append(eventHandlers, &CRDEventHandler{})
	adapter := &Adapter{
		zkClient:      zkClient,
		eventHandlers: eventHandlers,
	}
	return adapter, nil
}

func (a *Adapter) Run(stop <-chan struct{}) {
	if err := a.zkClient.Start(); err != nil {
		fmt.Printf("Start client has an error %v\n", err)
		return
	}

	for {
		select {
		case event := <-a.zkClient.Events():
			switch event.EventType {
			case ServiceAdded:
				for _, h := range a.eventHandlers {
					h.AddService(event)
				}
			case ServiceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteService(event)
				}
			case ServiceInstanceAdded:
				for _, h := range a.eventHandlers {
					h.AddInstance(event)
				}
			case ServiceInstanceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteInstance(event)
				}
			}
		case <-stop:
			a.zkClient.Stop()
			return
		}
	}

}
