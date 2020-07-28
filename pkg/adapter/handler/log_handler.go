package handler

import (
	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/adapter/types"
	"k8s.io/klog"
)

// LogEventHandler Using printing the event's information as a simple handling logic.
type LogEventHandler struct {
}

// NewLogEventHandler ...
func NewLogEventHandler() (component.EventHandler, error) {
	return &LogEventHandler{}, nil
}

// AddService ...
func (leh *LogEventHandler) AddService(e *types.ServiceEvent, configuratorFinder func(s string) *types.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Adding a service\n%v\n", e.Service)
}

// DeleteService ...
func (leh *LogEventHandler) DeleteService(e *types.ServiceEvent) {
	klog.Infof("Simple event handler: Deleting a service\n%v\n", e.Service)
}

// AddInstance ...
func (leh *LogEventHandler) AddInstance(e *types.ServiceEvent, configuratorFinder func(s string) *types.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Adding an instance\n%v\n", e.Instance)
}

// ReplaceInstances ...
func (leh *LogEventHandler) ReplaceInstances(e *types.ServiceEvent, configuratorFinder func(s string) *types.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Replacing these instances\n%v\n", e.Instances)
}

// DeleteInstance ...
func (leh *LogEventHandler) DeleteInstance(e *types.ServiceEvent) {
	klog.Infof("Simple event handler: Deleting an instance\n%v\n", e.Instance)
}

// ReplaceAccessorInstances ...
func (leh *LogEventHandler) ReplaceAccessorInstances(e *types.ServiceEvent, getScopedServices func(s string) map[string]struct{}) {
	klog.Infof("Simple event handler: Replacing the accessors' instances\n%v\n", e.Instance)
}

// AddConfigEntry ...
func (leh *LogEventHandler) AddConfigEntry(e *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service) {
	klog.Infof("Simple event handler: adding a configuration\n%v", e.Path)
}

// ChangeConfigEntry ...
func (leh *LogEventHandler) ChangeConfigEntry(e *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service) {
	klog.Infof("Simple event handler: change a configuration\n%v", e.Path)
}

// DeleteConfigEntry ...
func (leh *LogEventHandler) DeleteConfigEntry(e *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service) {
	klog.Infof("Simple event handler: delete a configuration\n%v", e.Path)
}
