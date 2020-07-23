package configcenter

import (
	"github.com/symcn/mesh-operator/pkg/adapter/types"
	v1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
)

// ConfigBuilder ...
type ConfigBuilder interface {
	// GetGlobalConfig This config is always used as a global configuration such as global policy, etc.
	GetGlobalConfig() *v1.MeshConfig

	// GetDefaultConfig We'll use this configurator as default if the creating service has not a configurator.
	GetDefaultConfig() *types.ConfiguratorConfig

	// BuildPolicy ...
	BuildPolicy(cs *v1.ConfiguredService, cc *types.ConfiguratorConfig) *v1.ConfiguredService

	// BuildSubsets ...
	BuildSubsets(cs *v1.ConfiguredService, cc *types.ConfiguratorConfig) *v1.ConfiguredService

	// BuildSourceLabels ...
	BuildSourceLabels(cs *v1.ConfiguredService, cc *types.ConfiguratorConfig) *v1.ConfiguredService

	// BuildInstanceSetting ...
	BuildInstanceSetting(cs *v1.ConfiguredService, cc *types.ConfiguratorConfig) *v1.ConfiguredService

	// SetConfig ...
	SetConfig(cs *v1.ConfiguredService, cc *types.ConfiguratorConfig)
}
