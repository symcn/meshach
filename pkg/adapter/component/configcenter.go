package component

import "github.com/symcn/mesh-operator/pkg/adapter/types"

// ConfigurationCenter ...
type ConfigurationCenter interface {
	Start() error

	Events() <-chan *types.ConfigEvent

	FindConfiguratorConfig(serviceName string) *types.ConfiguratorConfig

	Stop() error
}
