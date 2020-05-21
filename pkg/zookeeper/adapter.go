package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
	"time"
)

type serviceHandler func()
type instanceHandler func()

type Adapter struct {
	client          *Client
	serviceHandlers []serviceHandler
	instanceHandler []instanceHandler
}

func NewAdapter(address string, root string) (*Adapter, error) {
	servers := strings.Split(address, ",")
	conn, _, err := zk.Connect(servers, 15*time.Second)
	if err != nil {
		return nil, err
	}

	client := NewClient(dubboRootPath, conn)
	adapter := &Adapter{
		client: client,
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
				fmt.Printf("Adding a service\n%v\n", event.Service)
			case ServiceDeleted:
				fmt.Printf("Deleting a service\n%v\n", event.Service)
			case ServiceInstanceAdded:
				fmt.Printf("Adding an instance\n%v\n", event.Instance)
			case ServiceInstanceDeleted:
				fmt.Printf("Deleting an instance\n%v\n", event.Instance)
			}
		case <-stop:
			a.client.Stop()
			return
		}
	}

}
