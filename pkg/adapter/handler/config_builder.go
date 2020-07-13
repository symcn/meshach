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

// DefaultConfigurator for the service without a customized configurator
var DefaultConfigurator = &types.ConfiguratorConfig{
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

// buildPolicy
func buildPolicy(cs *v1.ConfiguraredService, e *types.ConfiguratorConfig, mc *v1.MeshConfig) *v1.ConfiguraredService {
	cs.Spec.Policy = &v1.Policy{
		LoadBalancer:   mc.Spec.GlobalPolicy.LoadBalancer,
		MaxConnections: mc.Spec.GlobalPolicy.MaxConnections,
		Timeout:        mc.Spec.GlobalPolicy.Timeout,
		MaxRetries:     mc.Spec.GlobalPolicy.MaxRetries,
	}

	// find out the default configuration if it presents.
	// it will be used to assemble both the service and instances without customized configurations.
	defaultConfig := findDefaultConfig(e.Configs)
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

// buildSubsets
func buildSubsets(cs *v1.ConfiguraredService, e *types.ConfiguratorConfig, mc *v1.MeshConfig) *v1.ConfiguraredService {
	cs.Spec.Subsets = mc.Spec.GlobalSubsets
	return cs
}

// buildSourceLabels
func buildSourceLabels(cs *v1.ConfiguraredService, e *types.ConfiguratorConfig, mc *v1.MeshConfig) *v1.ConfiguraredService {
	var sls []*v1.SourceLabels
	for _, subset := range mc.Spec.GlobalSubsets {
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
		for _, ss := range mc.Spec.GlobalSubsets {
			d := &v1.Destination{Subset: ss.Name}
			if ss.Name == sl.Name {
				d.Weight = 100
			} else {
				d.Weight = 0
			}
			routes = append(routes, d)
		}
		// setting flag configurator
		flagConfig := findFlagConfig(e.Configs)
		if flagConfig != nil {
			fc, ok := flagConfig.Parameters["flag_config"]
			if ok {
				klog.Infof("Flag config: %s", fc)
				fcp := &types.FlagConfigParameter{}
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

// buildInstanceSetting
func buildInstanceSetting(cs *v1.ConfiguraredService, e *types.ConfiguratorConfig, mc *v1.MeshConfig) *v1.ConfiguraredService {
	for index, ins := range cs.Spec.Instances {
		if matched, c := matchInstance(ins, e.Configs); matched {
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

// setConfig
func setConfig(c *types.ConfiguratorConfig, cs *v1.ConfiguraredService, mc *v1.MeshConfig) {
	// find out the service we need to process
	if cs.Name == utils.StandardizeServiceName(c.Key) {
		// policy's setting
		buildPolicy(cs, c, mc)
		// subset's setting
		buildSubsets(cs, c, mc)
		// setting source labels
		buildSourceLabels(cs, c, mc)
		// Setting these instances's configuration such as weight
		buildInstanceSetting(cs, c, mc)
	} else {
		klog.Warningf("Set configuration failed: the cs's name [%s] is difference from the configurator's name [%s]",
			cs.Name, c.Key)
	}
}
