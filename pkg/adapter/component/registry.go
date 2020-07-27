package component

import "github.com/symcn/mesh-operator/pkg/adapter/types"

// Registry ...
type Registry interface {
	Start() error

	// Received the notification component indicates services or instances has been modified.
	ServiceEvents() <-chan *types.ServiceEvent

	// Received all accessors that accesses a service.
	AccessorEvents() <-chan *types.ServiceEvent

	// If you need to make the application name explicit, you can find it with a service name from registry client.
	// FindAppIdentifier(serviceName string) string

	// GetCachedScopedMapping Retrieve the services with the scoped key from the cache.
	GetCachedScopedMapping(scopedKey string) map[string]struct{}

	GetCachedService(serviceName string) *types.Service

	Stop()
}
