package adapter

import (
	"fmt"
	"github.com/mesh-operator/pkg/adapter/events"
)

// All events comes from adapter needs to be handle by various event handler.
type EventHandler interface {

	// AddService you should handle the event described that a service has been created
	AddService(event events.ServiceEvent)

	// DeleteService you should handle the event describe that a service has been removed
	DeleteService(event events.ServiceEvent)

	// AddInstance you should handle the event described that an instance has been registered
	AddInstance(event events.ServiceEvent)

	// AddInstance you should handle the event describe that an instance has been unregistered
	DeleteInstance(event events.ServiceEvent)
}

// SimpleEventHandler Using printing the event's information as a simple handling logic.
type SimpleEventHandler struct {
	Name string
}

func (ceh *SimpleEventHandler) AddService(e events.ServiceEvent) {
	fmt.Printf("Simple event handler: Adding a service\n%v\n", e.Service)
}

func (ceh *SimpleEventHandler) DeleteService(e events.ServiceEvent) {
	fmt.Printf("Simple event handler: Deleting a service\n%v\n", e.Service)
}

func (ceh *SimpleEventHandler) AddInstance(e events.ServiceEvent) {
	fmt.Printf("Simple event handler: Adding an instance\n%v\n", e.Instance)
}

func (ceh *SimpleEventHandler) DeleteInstance(e events.ServiceEvent) {
	fmt.Printf("Simple event handler: Deleting an instance\n%v\n", e.Instance)
}
