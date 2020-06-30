package handler

import (
	"context"
	"fmt"
	"github.com/mesh-operator/pkg/adapter/constant"
	"github.com/mesh-operator/pkg/adapter/events"
	"github.com/mesh-operator/pkg/adapter/utils"
	v1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"strconv"
)

const (
	// FIX just for test with a fix name
	clusterName = "tcc-gz01-bj5-test"
)

// KubeV2EventHandler it used for synchronizing the events which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.
type KubeV2EventHandler struct {
	K8sMgr     *k8smanager.ClusterManager
	meshConfig *v1.MeshConfig
}

// NewKubeV2EventHander ...
func NewKubeV2EventHander(k8sMgr *k8smanager.ClusterManager) (events.EventHandler, error) {
	cluster, err := k8sMgr.Get(clusterName)
	if err != nil {
		return nil, err
	}

	mc := &v1.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sym-meshconfig",
			Namespace: defaultNamespace,
		},
	}
	err = cluster.Client.Get(context.Background(), types.NamespacedName{
		Namespace: mc.Namespace,
		Name:      mc.Name,
	}, mc)

	if err != nil {
		return nil, fmt.Errorf("Initializing mesh config has an error: %v", err)
	}

	return &KubeV2EventHandler{
		K8sMgr:     k8sMgr,
		meshConfig: mc,
	}, nil
}

// AddService ...
func (kubev2eh *KubeV2EventHandler) AddService(event *events.ServiceEvent, configuratorFinder func(s string) *events.ConfiguratorConfig) {
	klog.Infof("Kube v2 event handler: Adding a service\n%v", event)

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Transform a service event that noticed by zookeeper to a Service CRD
		// TODO we should resolve the application name from the meta data placed in a zookeeper node.
		appIdentifier := resolveAppIdentifier(event)
		if appIdentifier == "" {
			klog.Warningf("Can not find an application name with this adding event: %s", event.Service.Name)
			return nil
		}

		// loading amc CR from k8s cluster
		amc, err := kubev2eh.findAmc(appIdentifier)

		// Replacing service information belongs to this amc CR with this event.
		s := seekService(event)
		replace(s, amc)

		// meanwhile we should search a configurator for such service
		config := configuratorFinder(s.Name)
		if config == nil {
			dc := *DefaultConfigurator
			dc.Key = s.Name
			setConfig(&dc, amc, kubev2eh.meshConfig)
		} else {
			setConfig(config, amc, kubev2eh.meshConfig)
		}

		if err != nil {
			klog.Errorf("Can not find an existed amc CR: %v", err)
			amc.Spec.AppName = appIdentifier
			return kubev2eh.createAmc(amc)
		} else {
			return kubev2eh.updateAmc(amc)
		}
	})
}

func (kubev2eh *KubeV2EventHandler) AddInstance(event *events.ServiceEvent, configuratorFinder func(s string) *events.ConfiguratorConfig) {
	klog.Infof("Kube v2 event handler: Adding an instance\n%v", event.Instance)
	kubev2eh.AddService(event, configuratorFinder)
}

func (kubev2eh *KubeV2EventHandler) ReplaceInstances(event *events.ServiceEvent, configuratorFinder func(s string) *events.ConfiguratorConfig) {
	klog.Infof("Kube v2 event handler: Replacing these instances(size: %d)\n%v", len(event.Instances), event.Instances)
	kubev2eh.AddService(event, configuratorFinder)
}

// DeleteService we assume we need to remove the service Spec part of AppMeshConfig
// after received a service deleted notification.
func (kubev2eh *KubeV2EventHandler) DeleteService(event *events.ServiceEvent) {
	klog.Infof("Kube v2 event handler: Deleting a service: %s", event.Service)

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// TODO we should resolve the application name from the meta data placed in a zookeeper node.
		appIdentifier := resolveAppIdentifier(event)
		// There is a chance to delete a service with an empty instances manually, but it is not be sure that which
		// amc should be modified.
		if appIdentifier == "" {
			klog.Infof("Can not find an application name with this deleting event: %v\n", event.Service.Name)
			return nil
		}

		amc, err := kubev2eh.findAmc(appIdentifier)
		if err != nil {
			fmt.Println("amc CR can not be found, ignore it")
			return nil
		} else {
			if amc.Spec.Services != nil && len(amc.Spec.Services) > 0 {
				for i, s := range amc.Spec.Services {
					if s.Name == event.Service.Name {
						result := utils.DeleteInSlice(amc.Spec.Services, i)
						amc.Spec.Services = result.([]*v1.Service)
						break
						// TODO break? Can I assume there is no duplicate services belongs to a same amc?
					}
				}

				if len(amc.Spec.Services) == 0 {
					amc.Spec.Services = nil
				}

				return kubev2eh.updateAmc(amc)
			} else {
				fmt.Println("The services list belongs to this amc CR is empty, ignore it")
				return nil
			}
		}
	})
}

// DeleteInstance ...
func (kubev2eh *KubeV2EventHandler) DeleteInstance(event *events.ServiceEvent) {
	klog.Infof("Kube v2 event handler: deleting an instance\n%v\n", event.Instance)

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		appIdentifier := resolveAppIdentifier(event)
		amc, err := kubev2eh.findAmc(appIdentifier)
		if err != nil {
			klog.Infof("The applicatin mesh configruation can not be found with key: %s", appIdentifier)
			return nil
		}

		deleteInstance(event, amc)

		err = kubev2eh.updateAmc(amc)
		if err != nil {
			klog.Infof("Updating amc has an error: %v\n", err)
			return err
		}

		return nil
	})
}

// seekService seek a service within an event
func seekService(event *events.ServiceEvent) *events.Service {
	var svc *events.Service
	switch {
	case event.Service != nil:
		svc = event.Service
	case event.Instance != nil:
		svc = event.Instance.Service
	default:
		for k := range event.Instances {
			svc = event.Instances[k].Service
			break
		}
	}
	return svc
}

// replace Replace the whole service which belongs to this amc CR with this service entry。
func replace(svc *events.Service, amc *v1.AppMeshConfig) {
	s := convertService(svc)
	if amc.Spec.Services == nil {
		var services []*v1.Service
		services = append(services, s)
		amc.Spec.Services = services
	} else {
		for index, as := range amc.Spec.Services {
			if as.Name == s.Name {
				// No matter what happen we replacing the existed service
				amc.Spec.Services[index] = s
				// break
			}
		}
	}
}

// convertService Convert service between these two formats
func convertService(s *events.Service) *v1.Service {
	// Ports

	//var ports []*v1.Port
	//for _, p := range s.Ports {
	//	ports = append(ports, convertPort(p))
	//}

	// TODO Assuming every service can only provide an unique fixed port to adapt the dubbo case.
	service := &v1.Service{
		Name: s.Name,
		Ports: []*v1.Port{{
			Name:     constant.DubboPortName,
			Protocol: constant.DubboProtocol,
			Number:   utils.ToUint32(constant.MosnPort),
		}},
	}

	var instances []*v1.Instance
	for _, i := range s.Instances {
		ins := &v1.Instance{}
		ins.Host = utils.RemovePort(i.Host)
		ins.Port = convertPort(i.Port)
		ins.Labels = i.Labels
		ins.Labels[constant.ZoneLabel] = constant.Zone
		instances = append(instances, ins)
	}
	service.Instances = instances

	return service
}

// createAmc
func (kubev2eh *KubeV2EventHandler) createAmc(amc *v1.AppMeshConfig) error {
	// TODO
	cluster, _ := kubev2eh.K8sMgr.Get(clusterName)
	err := cluster.Client.Create(context.Background(), amc)
	klog.Infof("=The generation of amc when creating: %d", amc.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Creating an acm has an error:%v\n", err)
		return err
	}
	return nil
}

// updateAmc
func (kubev2eh *KubeV2EventHandler) updateAmc(amc *v1.AppMeshConfig) error {
	cluster, err := kubev2eh.K8sMgr.Get(clusterName)
	if err != nil {
		return err
	}
	err = cluster.Client.Update(context.Background(), amc)
	klog.Infof("=The generation of amc after updating: %d", amc.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Updating an acm has an error: %v\n", err)
		return err
	}

	return nil
}

// getAmc
func (kubev2eh *KubeV2EventHandler) getAmc(config *v1.AppMeshConfig) (*v1.AppMeshConfig, error) {
	cluster, err := kubev2eh.K8sMgr.Get(clusterName)
	if err != nil {
		return nil, err
	}
	err = cluster.Client.Get(context.Background(), types.NamespacedName{
		Namespace: config.Namespace,
		Name:      config.Name,
	}, config)
	klog.Infof("=The generation of amc when getting: %d", config.ObjectMeta.Generation)
	return config, err
}

// findAmc
func (kubev2eh KubeV2EventHandler) findAmc(appIdentifier string) (*v1.AppMeshConfig, error) {
	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIdentifier,
			Namespace: defaultNamespace,
		},
	}
	amc, err := kubev2eh.getAmc(amc)
	if err != nil {
		klog.Infof("Finding amc with name %s has an error: %v\n", appIdentifier, err)
		// TODO Is there a requirement to requeue this event?
		return nil, err
	}

	return amc, nil
}

// AddConfigEntry
func (kubev2eh *KubeV2EventHandler) AddConfigEntry(e *events.ConfigEvent, cachedServiceFinder func(s string) *events.Service) {
	klog.Infof("Kube v2 event handler: adding a configuration\n%v\n", e.Path)
	// Adding a new configuration for a service is same as changing it.
	kubev2eh.ChangeConfigEntry(e, cachedServiceFinder)
}

// ChangeConfigEntry
func (kubev2eh *KubeV2EventHandler) ChangeConfigEntry(e *events.ConfigEvent, cachedServiceFinder func(s string) *events.Service) {
	klog.Infof("Kube v2 event handler: change a configuration\n%v\n", e.Path)

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		serviceName := e.ConfigEntry.Key
		service := cachedServiceFinder(serviceName)
		appIdentifier := getAppIdentifier(service)
		if appIdentifier == "" {
			klog.Warningf("Can not find the app identified name through the cached service, service name :%s", serviceName)
			return nil
		}

		amc, err := kubev2eh.findAmc(appIdentifier)
		if err != nil {
			klog.Infof("Finding amc with name %s has an error: %v\n", appIdentifier, err)
			// TODO Is there a requirement to requeue this event?
			return nil
		}

		// utilize this configurator for such amc CR
		if e.ConfigEntry == nil || !e.ConfigEntry.Enabled {
			// TODO we really need to handle and think about the case that configuration has been disable.
			dc := *DefaultConfigurator
			dc.Key = serviceName
			setConfig(&dc, amc, kubev2eh.meshConfig)
		} else {
			setConfig(e.ConfigEntry, amc, kubev2eh.meshConfig)
		}

		return kubev2eh.updateAmc(amc)
	})
}

// DeleteConfigEntry
func (kubev2eh *KubeV2EventHandler) DeleteConfigEntry(e *events.ConfigEvent, cachedServiceFinder func(s string) *events.Service) {
	klog.Infof("Kube v2 event handler: delete a configuration\n%v", e.Path)

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// an example for the path: /dubbo/config/dubbo/com.dmall.mesh.test.PoviderDemo.configurators
		// Usually deleting event don't include the configuration data, so that we should
		// parse the zNode path to decide what is the service name.
		serviceName := utils.ResolveServiceName(e.Path)
		service := cachedServiceFinder(serviceName)
		appIdentifier := getAppIdentifier(service)
		if appIdentifier == "" {
			klog.Warningf("Can not find the app identified name through the cached service, service name :%s", serviceName)
			return nil
		}

		amc, err := kubev2eh.findAmc(appIdentifier)
		if err != nil {
			klog.Infof("Finding amc with name %s has an error: %v\n", appIdentifier, err)
			// TODO Is there a requirement to requeue this event?
			return nil
		}

		// Deleting a configuration of a service is similar to setting default configurator to this service
		dc := *DefaultConfigurator
		dc.Key = serviceName
		setConfig(&dc, amc, kubev2eh.meshConfig)

		return kubev2eh.updateAmc(amc)
	})

}

// findDefaultConfig
func findDefaultConfig(configs []events.ConfigItem) *events.ConfigItem {
	var defaultConfig *events.ConfigItem
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
func findFlagConfig(configs []events.ConfigItem) *events.ConfigItem {
	var config *events.ConfigItem
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
func matchInstance(ins *v1.Instance, configs []events.ConfigItem) (bool, *events.ConfigItem) {
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
func setConfig(c *events.ConfiguratorConfig, amc *v1.AppMeshConfig, mc *v1.MeshConfig) {
	for index, service := range amc.Spec.Services {
		// find out the service we need to process
		if service.Name == c.Key {
			s := service

			// policy's setting
			buildPolicy(s, c, mc)
			// subset's setting
			buildSubsets(s, c, mc)
			// setting source labels
			buildSourceLabels(s, c, mc)
			// Setting these instances's configuration such as weight
			buildInstanceSetting(s, c, mc)

			amc.Spec.Services[index] = s
		}
	}
}
