package component

import "github.com/symcn/mesh-operator/pkg/adapter/types"

// EventHandler described all component comes from adapter needs to be handle by various event handler.
type EventHandler interface {
	// AddService you should handle the event described that a service has been created
	AddService(event *types.ServiceEvent, configuratorFinder func(s string) *types.ConfiguratorConfig)
	// DeleteService you should handle the event describe that a service has been removed
	DeleteService(event *types.ServiceEvent)
	// AddInstance you should handle the event described that an instance has been registered
	AddInstance(event *types.ServiceEvent, configuratorFinder func(s string) *types.ConfiguratorConfig)
	// AddInstance you should handle the event described that an instance has been registered
	ReplaceInstances(event *types.ServiceEvent, configuratorFinder func(s string) *types.ConfiguratorConfig)
	// AddInstance you should handle the event describe that an instance has been unregistered
	DeleteInstance(event *types.ServiceEvent)

	// AddConfigEntry you should handle the event depicted that a dynamic configuration has been added
	AddConfigEntry(event *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service)
	// ChangeConfigEntry you should handle the event depicted that a dynamic configuration has been changed
	ChangeConfigEntry(event *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service)
	// DeleteConfigEntry you should handle the event depicted that a dynamic configuration has been deleted
	DeleteConfigEntry(event *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service)
}
