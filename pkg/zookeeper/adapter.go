package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
	"time"
)

type Adapter struct {
	client        *Client
	eventHandlers []EventHandler
}

func NewAdapter(address string, root string) (*Adapter, error) {
	servers := strings.Split(address, ",")
	conn, _, err := zk.Connect(servers, 15*time.Second)
	if err != nil {
		return nil, err
	}

	client := NewClient(dubboRootPath, conn)
	eventHandlers := []EventHandler{}
	eventHandlers = append(eventHandlers, &SimpleEventHandler{Name: "testHandler"})
	eventHandlers = append(eventHandlers, &CRDEventHandler{})
	adapter := &Adapter{
		client:        client,
		eventHandlers: eventHandlers,
	}
	return adapter, nil
}

func (a *Adapter) Run(stop <-chan struct{}) {
	if err := a.client.Start(); err != nil {
		fmt.Printf("Start client has an error %v\n", err)
		return
	}

	for {
		select {
		case event := <-a.client.Events():
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
			a.client.Stop()
			return
		}
	}

}
