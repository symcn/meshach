package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/mesh-operator/pkg/adapter/component"
	"github.com/mesh-operator/pkg/adapter/constant"
	"github.com/mesh-operator/pkg/adapter/utils"
	v1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

var defaultNamespace = "sym-admin"

// KubeEventHandler it used for synchronizing the component which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.
type KubeEventHandler struct {
	K8sMgr *k8smanager.ClusterManager
}

func (kubeeh *KubeEventHandler) Init() {}

// AddService ...
func (kubeeh *KubeEventHandler) AddService(se component.ServiceEvent, configuratorFinder func(s string) *component.ConfiguratorConfig) {
	klog.Infof("CRD event handler: Adding a service\n%v\n", se.Service)

	// Transform a service event that noticed by zookeeper to a Service CRD
	// TODO we should resolve the application name from the meta data placed in a zookeeper node.
	appIdentifier := resolveAppIdentifier(&se)
	if appIdentifier == "" {
		klog.Infof("Can not find an application name with this adding event: %v\n", se.Service.Name)
		return
	}

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIdentifier,
			Namespace: defaultNamespace,
		},
	}

	_, err := kubeeh.GetAmc(amc)
	putService(&se, amc)
	if err != nil {
		klog.Infof("Can not find an existed amc CR: %v\n", err)
		kubeeh.CreateAmc(amc)
	} else {
		kubeeh.UpdateAmc(amc)
	}

	klog.Infof("Create or update an AppMeshConfig CR after a service has beed created: %s\n", amc.Name)
}

// DeleteService we assume we need to remove the service Spec part of AppMeshConfig
// after received a service deleted notification.
func (kubeeh *KubeEventHandler) DeleteService(se component.ServiceEvent) {
	klog.Infof("CRD event handler: Deleting a service: %s\n", se.Service.Name)

	// TODO we should resolve the application name from the meta data placed in a zookeeper node.
	appIdentifier := resolveAppIdentifier(&se)
	// There is a chance to delete a service with an empty instances manually, but it is not be sure that which
	// amc should be modified.
	if appIdentifier == "" {
		klog.Infof("Can not find an application name with this deleting event: %v\n", se.Service.Name)
		return
	}

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIdentifier,
			Namespace: defaultNamespace,
		},
	}

	_, err := kubeeh.GetAmc(amc)
	if err != nil {
		fmt.Println("amc CR can not be found, ignore it")
		return
	} else {
		if amc.Spec.Services != nil && len(amc.Spec.Services) > 0 {
			for i, s := range amc.Spec.Services {
				if s.Name == se.Service.Name {
					result := utils.DeleteInSlice(amc.Spec.Services, i)
					amc.Spec.Services = result.([]*v1.Service)
					break
					// TODO break? Can I assume there is no duplicate services belongs to a same amc?
				}
			}

			if len(amc.Spec.Services) == 0 {
				amc.Spec.Services = nil
			}

			kubeeh.UpdateAmc(amc)
		} else {
			fmt.Println("The services list belongs to this amc CR is empty, ignore it")
			return
		}
	}

}

// AddInstance ...
func (kubeeh *KubeEventHandler) AddInstance(ie component.ServiceEvent, configuratorFinder func(s string) *component.ConfiguratorConfig) {
	klog.Infof("CRD event handler: Adding an instance\n%v\n", ie.Instance)

	// TODO we should resolve the application name from the meta data placed in a zookeeper node.
	appIdentifier := resolveAppIdentifier(&ie)

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIdentifier,
			Namespace: defaultNamespace,
		},
	}

	_, err := kubeeh.GetAmc(amc)
	putInstance(&ie, amc)
	if err != nil {
		klog.Infof("Can not get an exist amc CR: %v\n", err)
		kubeeh.CreateAmc(amc)
	} else {
		kubeeh.UpdateAmc(amc)
	}

	klog.Infof("Create or update an AppMeshConfig CR after an instance has beed added:%s\n", amc.Name)
}

// DeleteInstance ...
func (kubeeh *KubeEventHandler) DeleteInstance(ie component.ServiceEvent) {
	klog.Infof("CRD event handler: deleting an instance\n%v\n", ie.Instance)

	appIdentifier := resolveAppIdentifier(&ie)
	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIdentifier,
			Namespace: defaultNamespace,
		},
	}
	_, err := kubeeh.GetAmc(amc)
	if err != nil {
		klog.Infof("The applicatin mesh configruation can not be found with key: %s", appIdentifier)
		return
	}

	deleteInstance(&ie, amc)

	kubeeh.UpdateAmc(amc)
}

// CreateAmc ...
func (kubeeh *KubeEventHandler) CreateAmc(amc *v1.AppMeshConfig) {
	cluster, err := kubeeh.K8sMgr.Get(clusterName)
	if err != nil {
		klog.Errorln(err)
		return
	}

	err = cluster.Client.Create(context.Background(), amc)
	if err != nil {
		klog.Errorf("Creating an acm has an error:%v", err)
		return
	}
}

// UpdateAmc ...
func (kubeeh *KubeEventHandler) UpdateAmc(amc *v1.AppMeshConfig) {
	cluster, err := kubeeh.K8sMgr.Get(clusterName)
	if err != nil {
		klog.Errorln(err)
		return
	}

	err = cluster.Client.Update(context.Background(), amc)
	if err != nil {
		klog.Errorf("Updating an acm has an error: %v", err)
		return
	}
}

// GetAmc ...
func (kubeeh *KubeEventHandler) GetAmc(config *v1.AppMeshConfig) (*v1.AppMeshConfig, error) {
	cluster, err := kubeeh.K8sMgr.Get(clusterName)
	if err != nil {
		return nil, err
	}

	err = cluster.Client.Get(context.Background(), types.NamespacedName{
		Namespace: config.Namespace,
		Name:      config.Name,
	}, config)
	return config, err
}

// putService Put a service derived from a service event into the application mesh config.
func putService(se *component.ServiceEvent, amc *v1.AppMeshConfig) {
	// Ports
	var ports []*v1.Port
	for _, p := range se.Service.Ports {
		ports = append(ports, convertPort(p))
	}
	s := &v1.Service{
		Name:  se.Service.Name,
		Ports: ports,
		//Instances: e.Service.instances,
		//Policy:  nil,
		//Subsets: nil,
	}

	if amc.Spec.Services == nil {
		var services []*v1.Service
		services = append(services, s)
		amc.Spec.Services = services
	} else {
		var hasExist = false
		for _, as := range amc.Spec.Services {
			if as.Name == s.Name {
				hasExist = true
				//TODO should we update the details of this service without instances
				break
			}
		}

		if !hasExist {
			amc.Spec.Services = append(amc.Spec.Services, s)
		}
	}
}

// putInstance put an instance into the application mesh config
func putInstance(ie *component.ServiceEvent, amc *v1.AppMeshConfig) {
	i := &v1.Instance{}
	i.Host = utils.RemovePort(ie.Instance.Host)
	i.Port = convertPort(ie.Instance.Port)
	i.Labels = ie.Instance.Labels

	var s *v1.Service
	// Ports
	var ports []*v1.Port
	for _, p := range ie.Instance.Service.Ports {
		ports = append(ports, convertPort(p))
	}
	s = &v1.Service{
		Name:  ie.Instance.Service.Name,
		Ports: ports,
	}

	// Put the service if it is not present
	if amc.Spec.Services == nil {
		var services []*v1.Service
		services = append(services, s)
		amc.Spec.Services = services
	} else {
		var hasExist = false
		for _, as := range amc.Spec.Services {
			if as.Name == s.Name {
				hasExist = true
				s = as
				//TODO should we update the details of this service without instances
				break
			}
		}

		if !hasExist {
			amc.Spec.Services = append(amc.Spec.Services, s)
		}
	}

	// Put the instance if it is present.
	if s.Instances == nil {
		var instances []*v1.Instance
		instances = append(instances, i)
		s.Instances = instances
	} else {
		var hasExist = false
		for index, ins := range s.Instances {
			if ins.Host == i.Host && ins.Port.Number == i.Port.Number {
				hasExist = true
				// replace the current instance by the newest one.
				s.Instances[index] = i
				klog.Infof("Instance %v has exist in a service\n", i)
				break
			}
		}
		if !hasExist {
			s.Instances = append(s.Instances, i)
		}
	}
}

// deleteInstance Remove an instance from the amc CR
func deleteInstance(ie *component.ServiceEvent, amc *v1.AppMeshConfig) {
	instance := &v1.Instance{
		Host: utils.RemovePort(ie.Instance.Host),
		Port: convertPort(ie.Instance.Port),
	}

	if amc.Spec.Services == nil || len(amc.Spec.Services) == 0 {
		klog.Infof("The List of services who will be changed by removing an instance is empty.")
		return
	}

	for _, s := range amc.Spec.Services {
		// Can not find a service name in an instance.
		//if s.Name != ie.Instance.Service.name {
		//	continue
		//}

		if s.Instances == nil || len(s.Instances) == 0 {
			klog.Infof("The list of instances who will be change by removing an instance is empty.")
		}

		for index, i := range s.Instances {
			if i.Host == instance.Host && i.Port.Number == instance.Port.Number {
				result := utils.DeleteInSlice(s.Instances, index)
				s.Instances = result.([]*v1.Instance)
				// TODO Can I assume there is not duplicate instances belongs to a same service.
				break
			}
		}

		if len(s.Instances) == 0 {
			s.Instances = nil
		}
	}
}

// convertPort Convert the port which has been defined in zookeeper library to the one that belongs to CRD.
func convertPort(port *component.Port) *v1.Port {
	return &v1.Port{
		// Name:     port.Port,
		Name:     constant.DubboPortName,
		Protocol: port.Protocol,
		Number:   utils.ToUint32(port.Port),
	}
}

// resolveAppIdentifier Resolve the application code that was used as the key of an amc CR
// from the instance belongs to a service event.
func resolveAppIdentifier(e *component.ServiceEvent) string {
	vi := findValidInstance(e)
	if vi == nil {
		klog.Errorf("Can not find a valid instance with this event %v.", e)
		return ""

		// it will use foo as the default application name with an origin dubbo SDK.
		//return "foo"
	}

	appIdentifier := findAppIdentifier(vi)
	return appIdentifier
}

// FindAppIdentifier
func findAppIdentifier(i *component.Instance) string {
	if i != nil && i.Labels != nil {
		if appCodeLabelValue, ok := i.Labels[constant.AppCodeLabel]; ok {
			return strings.ToLower(appCodeLabelValue + "-" + i.Labels[constant.ProjectCodeLabel])
		}

		return strings.ToLower(i.Labels[constant.ApplicationLabel])
	}
	return ""
}

// getAppIdentifier
func getAppIdentifier(s *component.Service) string {
	var appName string
	if s == nil {
		klog.Infof("Can not get the application identifier with an empty service.\n")
		return appName
	}

	if s.Instances == nil || len(s.Instances) == 0 {
		klog.Infof("Can not find any instance from a service %s which has an empty instances list.\n", s.Name)
		return appName
	}

	for _, ins := range s.Instances {
		id := findAppIdentifier(ins)
		if id != "" {
			return id
		}
	}
	return appName

}

// findValidInstance because the application name was defined at an instance,
// we must try to find out an valid instance who always is the first one.
func findValidInstance(e *component.ServiceEvent) *component.Instance {
	if e == nil {
		klog.Infof("Service event is nil when start to find a valid instance from it.\n")
		return nil
	}

	if e.Instance != nil {
		return e.Instance
	}

	if e.Service == nil || e.Service.Instances == nil || len(e.Service.Instances) == 0 {
		klog.Warningf("The instances list of this service is nil or empty when start to find valid instance from it.")
		return nil
	}

	for _, value := range e.Service.Instances {
		if value != nil && value.Labels != nil {
			//&& value.Labels[constant.ApplicationLabel] != ""
			return value
		}
	}

	return nil
}

// AddConfigEntry
func (kubeeh *KubeEventHandler) AddConfigEntry(e *component.ConfigEvent, cachedServiceFinder func(s string) *component.Service) {
	klog.Infof("Kube event handler: adding a configuration\n%v\n", e.Path)

	serviceName := e.ConfigEntry.Key
	service := cachedServiceFinder(serviceName)
	appIdentifier := getAppIdentifier(service)

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIdentifier,
			Namespace: defaultNamespace,
		},
	}
	_, err := kubeeh.GetAmc(amc)
	if err != nil {
		klog.Infof("Finding amc with name %s has an error: %v\n", appIdentifier, err)
		// TODO Is there a requirement to requeue this event?
	} else {
		//for _, ci := range cc.Configs {
		//for address := range ci.Addresses {
		//
		//}
		//}
	}

}

func (kubeeh *KubeEventHandler) ChangeConfigEntry(e *component.ConfigEvent, cachedServiceFinder func(s string) *component.Service) {
	klog.Infof("Kube event handler: change a configuration\n%v\n", e.Path)

	serviceName := e.ConfigEntry.Key
	service := cachedServiceFinder(serviceName)
	appIdentifier := getAppIdentifier(service)

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIdentifier,
			Namespace: defaultNamespace,
		},
	}
	_, err := kubeeh.GetAmc(amc)
	if err != nil {
		klog.Infof("Finding amc with name %s has an error: %v\n", appIdentifier, err)
		// TODO Is there a requirement to requeue this event?
	} else {
		//for _, ci := range cc.Configs {
		//for address := range ci.Addresses {
		//
		//}
		//}
	}
}

func (kubeeh *KubeEventHandler) DeleteConfigEntry(e *component.ConfigEvent, cachedServiceFinder func(s string) *component.Service) {
	klog.Infof("Kube event handler: delete a configuration\n%v", e.Path)
}

func (kubeeh *KubeEventHandler) ReplaceInstances(e component.ServiceEvent, configuratorFinder func(s string) *component.ConfiguratorConfig) {
	klog.Infof("Simple event handler: Replacing these instances\n%v", e.Instances)
}
