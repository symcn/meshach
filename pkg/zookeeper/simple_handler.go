package zookeeper

import (
	"fmt"
)

// SimpleEventHandler Using printing the event's information as a simple handling logic.
type SimpleEventHandler struct {
	Name string
}

func (ceh *SimpleEventHandler) AddService(e ServiceEvent) {
	fmt.Printf("Simple event handler: Adding a service\n%v\n", e.Service)
}

func (ceh *SimpleEventHandler) DeleteService(e ServiceEvent) {
	fmt.Printf("Simple event handler: Deleting a service\n%v\n", e.Service)
}

func (ceh *SimpleEventHandler) AddInstance(e ServiceEvent) {
	fmt.Printf("Simple event handler: Adding an instance\n%v\n", e.Instance)
}

func (ceh *SimpleEventHandler) DeleteInstance(e ServiceEvent) {
	fmt.Printf("Simple event handler: Deleting an instance\n%v\n", e.Instance)
}
