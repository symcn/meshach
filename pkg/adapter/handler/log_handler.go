package handler

import (
	"fmt"

	"github.com/mesh-operator/pkg/adapter/events"
)

// LogEventHandler Using printing the event's information as a simple handling logic.
type LogEventHandler struct {
}

func NewLogEventHandler() (events.EventHandler, error) {
	return &LogEventHandler{}, nil
}

func (leh *LogEventHandler) AddService(e *events.ServiceEvent, configuratorFinder func(s string) *events.ConfiguratorConfig) {
	fmt.Printf("Simple event handler: Adding a service\n%v\n", e.Service)
}

func (leh *LogEventHandler) DeleteService(e *events.ServiceEvent) {
	fmt.Printf("Simple event handler: Deleting a service\n%v\n", e.Service)
}

func (leh *LogEventHandler) AddInstance(e *events.ServiceEvent, configuratorFinder func(s string) *events.ConfiguratorConfig) {
	fmt.Printf("Simple event handler: Adding an instance\n%v\n", e.Instance)
}

func (leh *LogEventHandler) ReplaceInstances(e *events.ServiceEvent, configuratorFinder func(s string) *events.ConfiguratorConfig) {
	fmt.Printf("Simple event handler: Replacing these instances\n%v\n", e.Instances)
}

func (leh *LogEventHandler) DeleteInstance(e *events.ServiceEvent) {
	fmt.Printf("Simple event handler: Deleting an instance\n%v\n", e.Instance)
}

func (leh *LogEventHandler) AddConfigEntry(e *events.ConfigEvent, cachedServiceFinder func(s string) *events.Service) {
	fmt.Printf("Simple event handler: adding a configuration\n%v\n", e.Path)
}

func (leh *LogEventHandler) ChangeConfigEntry(e *events.ConfigEvent, cachedServiceFinder func(s string) *events.Service) {
	fmt.Printf("Simple event handler: change a configuration\n%v\n", e.Path)
}

func (leh *LogEventHandler) DeleteConfigEntry(e *events.ConfigEvent, cachedServiceFinder func(s string) *events.Service) {
	fmt.Printf("Simple event handler: delete a configuration\n%v\n", e.Path)
}
