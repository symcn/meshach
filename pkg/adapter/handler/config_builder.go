package handler

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/mesh-operator/pkg/adapter/constant"
	"github.com/mesh-operator/pkg/adapter/events"
	"github.com/mesh-operator/pkg/adapter/utils"
	v1 "github.com/mesh-operator/pkg/apis/mesh/v1"
)

// Default configurator for the service without a customized configurator
var DefaultConfigurator = &events.ConfiguratorConfig{
	ConfigVersion: "2.7",
	Scope:         "service",
	Key:           constant.DefaultConfigName,
	Enabled:       true,
	Configs: []events.ConfigItem{
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
func buildPolicy(s *v1.Service, e *events.ConfiguratorConfig, mc *v1.MeshConfig) *v1.Service {
	s.Policy = &v1.Policy{
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
			s.Policy.Timeout = t
		}
		if r, ok := defaultConfig.Parameters["retries"]; ok {
			s.Policy.MaxRetries = utils.ToInt32(r)
		}
	}

	return s
}

// buildSubsets
func buildSubsets(s *v1.Service, e *events.ConfiguratorConfig, mc *v1.MeshConfig) *v1.Service {
	s.Subsets = mc.Spec.GlobalSubsets
	return s
}

// buildSourceLabels
func buildSourceLabels(s *v1.Service, e *events.ConfiguratorConfig, mc *v1.MeshConfig) *v1.Service {
	var sls []*v1.SourceLabels
	for _, subset := range mc.Spec.GlobalSubsets {
		sl := &v1.SourceLabels{
			Name:   subset.Name,
			Labels: subset.Labels,
		}
		// header
		h := make(map[string]string)
		h["sym-zone"] = constant.Zone
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
				fmt.Printf("%s\n", fc)
				fcp := &events.FlagConfigParameter{}
				err := yaml.Unmarshal([]byte(fc), fcp)
				if err != nil {
					fmt.Printf("Parsing the flag_config parameter has an error: %v\n", err)
				} else {
					if flagConfig.Enabled && fcp.Manual {
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
		}

		sl.Route = routes
		sls = append(sls, sl)
	}
	s.Policy.SourceLabels = sls
	return s
}

// buildInstanceSetting
func buildInstanceSetting(s *v1.Service, e *events.ConfiguratorConfig, mc *v1.MeshConfig) *v1.Service {
	for index, ins := range s.Instances {
		if matched, c := matchInstance(ins, e.Configs); matched {
			s.Instances[index].Weight = utils.ToUint32(c.Parameters["weight"])
		} else {
			s.Instances[index].Weight = 100
		}
	}
	return s
}
