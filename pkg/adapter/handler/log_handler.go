package handler

import (
	"github.com/mesh-operator/pkg/adapter/events"
	"k8s.io/klog"
)

// LogEventHandler Using printing the event's information as a simple handling logic.
type LogEventHandler struct {
	Name string
}

func (leh *LogEventHandler) Init() {}

func (leh *LogEventHandler) AddService(e events.ServiceEvent, configuratorFinder func(s string) *events.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Adding a service\n%v", e.Service)
}

func (leh *LogEventHandler) DeleteService(e events.ServiceEvent) {
	klog.Infof("Simple event handler: Deleting a service\n%v", e.Service)
}

func (leh *LogEventHandler) AddInstance(e events.ServiceEvent, configuratorFinder func(s string) *events.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Adding an instance\n%v", e.Instance)
}

func (leh *LogEventHandler) DeleteInstance(e events.ServiceEvent) {
	klog.Infof("Simple event handler: Deleting an instance\n%v", e.Instance)
}

func (leh *LogEventHandler) AddConfigEntry(e *events.ConfigEvent, cachedServiceFinder func(s string) *events.Service) {
	klog.Infof("Simple event handler: adding a configuration\n%v", e.Path)
}

func (leh *LogEventHandler) ChangeConfigEntry(e *events.ConfigEvent, cachedServiceFinder func(s string) *events.Service) {
	klog.Infof("Simple event handler: change a configuration\n%v", e.Path)
}

func (leh *LogEventHandler) DeleteConfigEntry(e *events.ConfigEvent, cachedServiceFinder func(s string) *events.Service) {
	klog.Infof("Simple event handler: delete a configuration\n%v", e.Path)
}

func (leh *LogEventHandler) ReplaceInstances(e events.ServiceEvent, configuratorFinder func(s string) *events.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Replacing these instances\n%v", e.Instances)
}
