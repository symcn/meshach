package component

import (
	v1 "github.com/symcn/mesh-operator/api/v1alpha1"
	types2 "github.com/symcn/mesh-operator/pkg/adapter/types"
)

// Converter ...
type Converter interface {
	ToConfiguredService(s *types2.Service) *v1.ConfiguredService
	ToServiceConfig(cc *types2.ConfiguratorConfig) *v1.ServiceConfig
}
