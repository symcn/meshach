package adapter

import (
	"github.com/mesh-operator/pkg/adapter/events"
)

type RegistryClient interface {
	Start() error

	// Received the notification events indicates services or instances has been modified.
	Events() <-chan events.ServiceEvent

	// If you need to make the application name explicit, you can find it with a service name from registry client.
	FindAppIdentifier(serviceName string) string

	Stop()
}
