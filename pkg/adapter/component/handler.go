package component

// All component comes from adapter needs to be handle by various event handler.
type EventHandler interface {
	// AddService you should handle the event described that a service has been created
	AddService(event *ServiceEvent, configuratorFinder func(s string) *ConfiguratorConfig)
	// DeleteService you should handle the event describe that a service has been removed
	DeleteService(event *ServiceEvent)
	// AddInstance you should handle the event described that an instance has been registered
	AddInstance(event *ServiceEvent, configuratorFinder func(s string) *ConfiguratorConfig)
	// AddInstance you should handle the event described that an instance has been registered
	ReplaceInstances(event *ServiceEvent, configuratorFinder func(s string) *ConfiguratorConfig)
	// AddInstance you should handle the event describe that an instance has been unregistered
	DeleteInstance(event *ServiceEvent)

	// AddConfigEntry you should handle the event depicted that a dynamic configuration has been added
	AddConfigEntry(event *ConfigEvent, cachedServiceFinder func(s string) *Service)
	// ChangeConfigEntry you should handle the event depicted that a dynamic configuration has been changed
	ChangeConfigEntry(event *ConfigEvent, cachedServiceFinder func(s string) *Service)
	// DeleteConfigEntry you should handle the event depicted that a dynamic configuration has been deleted
	DeleteConfigEntry(event *ConfigEvent, cachedServiceFinder func(s string) *Service)
}
