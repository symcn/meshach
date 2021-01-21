package component

import (
	v1 "github.com/symcn/meshach/api/v1alpha1"
	types2 "github.com/symcn/meshach/pkg/adapter/types"
)

// Converter a series of operation for convert meshach's model to CRD
// it allows that there are differences between various registry such as zk, nanos, etc.
// finally all services and configs will be convert to the unified models we have defined.
type Converter interface {
	// ToConfiguredService convert service to cs
	ToConfiguredService(s *types2.ServiceEvent) *v1.ConfiguredService

	// ToServiceConfig
	ToServiceConfig(cc *types2.ConfiguratorConfig) *v1.ServiceConfig
}
