package handler

import (
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/symcn/mesh-operator/pkg/adapter/constant"
	"github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/adapter/utils"
	v1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	"k8s.io/klog"
)

// DubboDefaultConfig it is used when the service has not a customized configurator
var DubboDefaultConfig = &types.ConfiguratorConfig{
	ConfigVersion: "2.7",
	Scope:         "service",
	Key:           constant.DefaultConfigName,
	Enabled:       true,
	Configs: []types.ConfigItem{
		{
			Type:       "service",
			Enabled:    true,
			Addresses:  []string{"0.0.0.0"},
			Parameters: map[string]string{"retries": "3", "timeout": "200"},
			Side:       "provider",
			// Applications:      nil,
			// ProviderAddresses: nil,
			// Services:          nil,
		}, {
			Type:       "service",
			Enabled:    false,
			Addresses:  []string{"0.0.0.0"},
			Parameters: map[string]string{
				//"flag_config": "flags:\n- key: blue\n weight: 60\n- key: green\n weight: 40\nmanual: true\n",
			},
			Side: "consumer",
			// Applications:      nil,
			// ProviderAddresses: nil,
			// Services:          nil,
		},
	},
}

// DubboConfiguratorBuilder it is just an implementing for dubbo.
type DubboConfiguratorBuilder struct {
	globalConfig  *v1.MeshConfig
	defaultConfig *types.ConfiguratorConfig
}

// FlagConfigParameter just for customized setting of dubbo
type FlagConfigParameter struct {
	Flags  []*Flag
	Manual bool
}

// Flag ...
type Flag struct {
	Key    string
	Weight int32
}

// GetGlobalConfig ...
func (dcb *DubboConfiguratorBuilder) GetGlobalConfig() *v1.MeshConfig {
	return dcb.globalConfig
}

// GetDefaultConfig ...
func (dcb *DubboConfiguratorBuilder) GetDefaultConfig() *types.ConfiguratorConfig {
	return dcb.defaultConfig
}

// BuildPolicy configurator of this service belongs to the zNode with path looks like following:
// e.g. /dubbo/config/dubbo/fooService.configurator
func (dcb *DubboConfiguratorBuilder) BuildPolicy(cs *v1.ConfiguraredService, cc *types.ConfiguratorConfig) *v1.ConfiguraredService {
	cs.Spec.Policy = &v1.Policy{
		LoadBalancer:   dcb.GetGlobalConfig().Spec.GlobalPolicy.LoadBalancer,
		MaxConnections: dcb.GetGlobalConfig().Spec.GlobalPolicy.MaxConnections,
		Timeout:        dcb.GetGlobalConfig().Spec.GlobalPolicy.Timeout,
		MaxRetries:     dcb.GetGlobalConfig().Spec.GlobalPolicy.MaxRetries,
	}

	// find out the default configuration if it presents.
	// it will be used to assemble both the service and instances without customized configurations.
	defaultConfig := findDefaultConfig(cc.Configs)
	// Setting the service's configuration such as policy
	if defaultConfig != nil && defaultConfig.Enabled {
		if t, ok := defaultConfig.Parameters["timeout"]; ok {
			cs.Spec.Policy.Timeout = t
		}
		if r, ok := defaultConfig.Parameters["retries"]; ok {
			cs.Spec.Policy.MaxRetries = utils.ToInt32(r)
		}
	}

	return cs
}

// BuildSubsets ...
func (dcb *DubboConfiguratorBuilder) BuildSubsets(cs *v1.ConfiguraredService, cc *types.ConfiguratorConfig) *v1.ConfiguraredService {
	cs.Spec.Subsets = dcb.GetGlobalConfig().Spec.GlobalSubsets
	return cs
}

// BuildSourceLabels ...
func (dcb *DubboConfiguratorBuilder) BuildSourceLabels(cs *v1.ConfiguraredService, cc *types.ConfiguratorConfig) *v1.ConfiguraredService {
	var sls []*v1.SourceLabels
	for _, subset := range dcb.GetGlobalConfig().Spec.GlobalSubsets {
		sl := &v1.SourceLabels{
			Name:   subset.Name,
			Labels: subset.Labels,
		}
		// header
		h := make(map[string]string)
		h[constant.SourceLabelZoneName] = constant.ZoneValue
		sl.Headers = h

		// route
		// The dynamic configuration has the highest priority if the manual is true
		var routes []*v1.Destination
		// By default the consumer can only visit the providers
		// whose group is same as itself.
		for _, ss := range dcb.GetGlobalConfig().Spec.GlobalSubsets {
			d := &v1.Destination{Subset: ss.Name}
			if ss.Name == sl.Name {
				d.Weight = 100
			} else {
				d.Weight = 0
			}
			routes = append(routes, d)
		}
		// setting flag configurator
		flagConfig := findFlagConfig(cc.Configs)
		if flagConfig != nil {
			fc, ok := flagConfig.Parameters["flag_config"]
			if ok {
				klog.Infof("Flag config: %s", fc)
				fcp := &FlagConfigParameter{}
				err := yaml.Unmarshal([]byte(fc), fcp)
				if err != nil {
					klog.Errorf("Parsing the flag_config parameter has an error: %v", err)
				} else if flagConfig.Enabled && fcp.Manual {
					// clear the default routes firstly
					routes = routes[:0]
					for _, f := range fcp.Flags {
						routes = append(routes, &v1.Destination{
							Subset: f.Key,
							Weight: f.Weight,
						})
					}
				}
			}
		}

		sl.Route = routes
		sls = append(sls, sl)
	}
	cs.Spec.Policy.SourceLabels = sls
	return cs
}

// BuildInstanceSetting ...
func (dcb *DubboConfiguratorBuilder) BuildInstanceSetting(cs *v1.ConfiguraredService, cc *types.ConfiguratorConfig) *v1.ConfiguraredService {
	for index, ins := range cs.Spec.Instances {
		if matched, c := matchInstance(ins, cc.Configs); matched {
			cs.Spec.Instances[index].Weight = utils.ToUint32(c.Parameters["weight"])
		} else {
			cs.Spec.Instances[index].Weight = 100
		}
	}
	return cs
}

// findDefaultConfig
func findDefaultConfig(configs []types.ConfigItem) *types.ConfigItem {
	var defaultConfig *types.ConfigItem
	for _, c := range configs {
		if c.Side == "provider" {
			for _, a := range c.Addresses {
				if a == "0.0.0.0" {
					defaultConfig = &c
					return defaultConfig
				}
			}
		}
	}
	return defaultConfig
}

// findFlagConfig
func findFlagConfig(configs []types.ConfigItem) *types.ConfigItem {
	var config *types.ConfigItem
	for _, c := range configs {
		if c.Side == "consumer" {
			for _, a := range c.Addresses {
				if a == "0.0.0.0" {
					config = &c
					return config
				}
			}
		}
	}
	return config
}

// matchInstance
func matchInstance(ins *v1.Instance, configs []types.ConfigItem) (bool, *types.ConfigItem) {
	for _, cc := range configs {
		for _, adds := range cc.Addresses {
			if ins.Host+":"+strconv.FormatInt(int64(ins.Port.Number), 10) == adds {
				// found an customized configuration for this instance.
				return true, &cc
			}
		}
	}
	return false, nil
}

// SetConfig ...
func (dcb *DubboConfiguratorBuilder) SetConfig(cs *v1.ConfiguraredService, cc *types.ConfiguratorConfig) {
	// policy's setting
	dcb.BuildPolicy(cs, cc)
	// subset's setting
	dcb.BuildSubsets(cs, cc)
	// setting source labels
	dcb.BuildSourceLabels(cs, cc)
	// Setting these instances's configuration such as weight
	dcb.BuildInstanceSetting(cs, cc)
}
