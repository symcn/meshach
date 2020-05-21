package zookeeper

// All events comes from adapter needs to be handle by various event handler.
type EventHandler interface {

	// AddService you should handle the event described that a service has been created
	AddService(event ServiceEvent)

	// DeleteService you should handle the event describe that a service has been removed
	DeleteService(event ServiceEvent)

	// AddInstance you should handle the event described that an instance has been registered
	AddInstance(event ServiceEvent)

	// AddInstance you should handle the event describe that an instance has been unregistered
	DeleteInstance(event ServiceEvent)
}
