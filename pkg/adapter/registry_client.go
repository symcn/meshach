package adapter

import (
	"github.com/mesh-operator/pkg/adapter/events"
)

type RegistryClient interface {
	Start() error

	// Received the notification events indicates services or instances has been modified.
	Events() <-chan events.ServiceEvent

	Stop()
}
