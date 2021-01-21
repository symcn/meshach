package convert

import (
	"net"

	"github.com/ghodss/yaml"
	v1 "github.com/symcn/meshach/api/v1alpha1"
	types2 "github.com/symcn/meshach/pkg/adapter/types"
	"github.com/symcn/meshach/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// DubboConverter ...
type DubboConverter struct {
	DefaultNamespace string
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

// ToConfiguredService Convert service between these two formats
func (dc *DubboConverter) ToConfiguredService(s *types2.ServiceEvent) *v1.ConfiguredService {
	// TODO Assuming every service can only provide an unique fixed port to adapt the dubbo case.
	cs := &v1.ConfiguredService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.FormatToDNS1123(s.Service.Name),
			Namespace: dc.DefaultNamespace,
		},
		Spec: v1.ConfiguredServiceSpec{
			OriginalName: s.Service.Name,
		},
	}

	var instances []*v1.Instance
	for _, i := range s.Instances {
		ins := &v1.Instance{}
		ins.Host = utils.RemovePort(i.Host)
		ins.Port = ToPort(i.Port)
		ins.Labels = i.Labels
		instances = append(instances, ins)
	}
	cs.Spec.Instances = instances
	// yamlbyte, err := yaml.Marshal(cs)
	// if err != nil {
	// 	klog.Errorf("Marshal yaml err:%+v", err)
	// 	return cs
	// }
	// fmt.Println(string(yamlbyte))

	return cs
}

// ToServiceConfig ...
func (dc *DubboConverter) ToServiceConfig(cc *types2.ConfiguratorConfig) *v1.ServiceConfig {
	if cc == nil || len(cc.Key) == 0 {
		klog.Infof("config's key is empty, skip it.")
		return nil
	}
	sc := v1.ServiceConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.FormatToDNS1123(cc.Key),
			Namespace: dc.DefaultNamespace,
		},
		Spec: v1.ServiceConfigSpec{
			OriginalName: cc.Key,
		},
	}

	// Policy
	policy := &v1.Policy{
		LoadBalancer: make(map[string]string),
	}
	// find out the default configuration if it presents.
	// it will be used to assemble both the service and instances without customized configurations.
	defaultConfig := findDefaultConfig(cc.Configs)
	// Setting the service's configuration such as policy
	if defaultConfig != nil && defaultConfig.Enabled {
		if t, ok := defaultConfig.Parameters["timeout"]; ok {
			policy.Timeout = t
		}
		if r, ok := defaultConfig.Parameters["retries"]; ok {
			policy.MaxRetries = utils.ToInt32(r)
		}
	}
	sc.Spec.Policy = policy

	// Instances' config
	var instanceConfigs []*v1.InstanceConfig
	for _, ci := range cc.Configs {
		if ci.Enabled && ci.Addresses[0] != "0.0.0.0" && ci.Side == "provider" {
			h, p, _ := net.SplitHostPort(ci.Addresses[0])
			instanceConfigs = append(instanceConfigs, &v1.InstanceConfig{
				Host:   h,
				Port:   &v1.Port{Name: "", Number: utils.ToUint32(p), Protocol: ""},
				Weight: utils.ToUint32(ci.Parameters["weight"]),
			})
		}
	}
	sc.Spec.Instances = instanceConfigs

	// Routes
	var routes []*v1.Destination
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
	} else {
		klog.Infof("Could not find any route config.")
	}
	sc.Spec.Route = routes

	sc.Spec.RerouteOption = &v1.RerouteOption{
		ReroutePolicy: v1.Default,
	}

	sc.Spec.CanaryRerouteOption = nil

	return &sc
}

// findDefaultConfig
func findDefaultConfig(configs []types2.ConfigItem) *types2.ConfigItem {
	var defaultConfig *types2.ConfigItem
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
func findFlagConfig(configs []types2.ConfigItem) *types2.ConfigItem {
	var config *types2.ConfigItem
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
