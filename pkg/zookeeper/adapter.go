package zookeeper

import (
	"fmt"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

// Adapter ...
type Adapter struct {
	zkClient      *ZkClient
	eventHandlers []EventHandler
}

// Option ...
type Option struct {
	Address []string
	Root    string
	Timeout int64
}

// DefaultOption ...
func DefaultOption() *Option {
	return &Option{
		Timeout: 15,
		Root:    dubboRootPath,
	}
}

// NewAdapter ...
func NewAdapter(opt *Option) (*Adapter, error) {
	conn, _, err := zk.Connect(opt.Address, time.Duration(opt.Timeout)*time.Second)
	if err != nil {
		return nil, err
	}
	rootPath := dubboRootPath
	if opt.Root != "" {
		rootPath = opt.Root
	}
	zkClient := NewClient(rootPath, conn)

	eventHandlers := []EventHandler{}
	eventHandlers = append(eventHandlers, &SimpleEventHandler{Name: "simpleHandler"})
	eventHandlers = append(eventHandlers, &CRDEventHandler{})
	adapter := &Adapter{
		zkClient:      zkClient,
		eventHandlers: eventHandlers,
	}
	return adapter, nil
}

// Run ...
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
