package events

type Registry interface {
	Start() error

	// Received the notification events indicates services or instances has been modified.
	Events() <-chan *ServiceEvent

	// If you need to make the application name explicit, you can find it with a service name from registry client.
	// FindAppIdentifier(serviceName string) string

	GetCachedService(serviceName string) *Service

	Stop()
}
