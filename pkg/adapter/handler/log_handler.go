package handler

import (
	"github.com/mesh-operator/pkg/adapter/component"
	"k8s.io/klog"
)

// LogEventHandler Using printing the event's information as a simple handling logic.
type LogEventHandler struct {
}

func NewLogEventHandler() (component.EventHandler, error) {
	return &LogEventHandler{}, nil
}

func (leh *LogEventHandler) AddService(e *component.ServiceEvent, configuratorFinder func(s string) *component.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Adding a service\n%v\n", e.Service)
}

func (leh *LogEventHandler) DeleteService(e *component.ServiceEvent) {
	klog.Infof("Simple event handler: Deleting a service\n%v\n", e.Service)
}

func (leh *LogEventHandler) AddInstance(e *component.ServiceEvent, configuratorFinder func(s string) *component.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Adding an instance\n%v\n", e.Instance)
}

func (leh *LogEventHandler) ReplaceInstances(e *component.ServiceEvent, configuratorFinder func(s string) *component.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Replacing these instances\n%v\n", e.Instances)
}

func (leh *LogEventHandler) DeleteInstance(e *component.ServiceEvent) {
	klog.Infof("Simple event handler: Deleting an instance\n%v\n", e.Instance)
}

func (leh *LogEventHandler) AddConfigEntry(e *component.ConfigEvent, cachedServiceFinder func(s string) *component.Service) {
	klog.Infof("Simple event handler: adding a configuration\n%v", e.Path)
}

func (leh *LogEventHandler) ChangeConfigEntry(e *component.ConfigEvent, cachedServiceFinder func(s string) *component.Service) {
	klog.Infof("Simple event handler: change a configuration\n%v", e.Path)
}

func (leh *LogEventHandler) DeleteConfigEntry(e *component.ConfigEvent, cachedServiceFinder func(s string) *component.Service) {
	klog.Infof("Simple event handler: delete a configuration\n%v", e.Path)
}
