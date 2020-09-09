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
func (h *LogEventHandler) AddService(e *types.ServiceEvent) {
	klog.Infof("Simple event handler: Adding a service\n%v\n", e.Service)
}

// DeleteService ...
func (h *LogEventHandler) DeleteService(e *types.ServiceEvent) {
	klog.Infof("Simple event handler: Deleting a service\n%v\n", e.Service)
}

// AddInstance ...
func (h *LogEventHandler) AddInstance(e *types.ServiceEvent) {
	klog.Infof("Simple event handler: Adding an instance\n%v\n", e.Instance)
}

// ReplaceInstances ...
func (h *LogEventHandler) ReplaceInstances(e *types.ServiceEvent) {
	klog.Infof("Simple event handler: Replacing these instances\n%v\n", e.Instances)
}

// DeleteInstance ...
func (h *LogEventHandler) DeleteInstance(e *types.ServiceEvent) {
	klog.Infof("Simple event handler: Deleting an instance\n%v\n", e.Instance)
}

// ReplaceAccessorInstances ...
func (h *LogEventHandler) ReplaceAccessorInstances(e *types.ServiceEvent, getScopedServices func(s string) map[string]struct{}) {
	klog.Infof("Simple event handler: Replacing the accessors' instances\n%v\n", e.Instance)
}

// AddConfigEntry ...
func (h *LogEventHandler) AddConfigEntry(e *types.ConfigEvent) {
	klog.Infof("Simple event handler: adding a configuration\n%v", e.Path)
}

// ChangeConfigEntry ...
func (h *LogEventHandler) ChangeConfigEntry(e *types.ConfigEvent) {
	klog.Infof("Simple event handler: change a configuration\n%v", e.Path)
}

// DeleteConfigEntry ...
func (h *LogEventHandler) DeleteConfigEntry(e *types.ConfigEvent) {
	klog.Infof("Simple event handler: delete a configuration\n%v", e.Path)
}
