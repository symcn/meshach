package handler

import (
	"context"
	"fmt"
	"github.com/mesh-operator/pkg/adapter/events"
	v1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

var defaultNamespace = "default"

// CRDEventHandler it used for handling the events which has been send from the adapter client.
type CRDEventHandler struct {
	K8sMgr *k8smanager.ClusterManager
}

// AddService ...
func (ceh *CRDEventHandler) AddService(se events.ServiceEvent) {
	fmt.Printf("CRD event handler: Adding a service\n%v\n", se.Service)

	// Transform a service event that noticed by zookeeper to a Service CRD
	// TODO we should resolve the application name from the meta data placed in a zookeeper node.
	appCode := resolveAppCode(&se)
	if appCode == "" {
		fmt.Printf("Can not find an application name with this adding event: %v\n", se.Service.Name)
		return
	}

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appCode,
			Namespace: defaultNamespace,
		},
	}

	_, err := ceh.GetAmc(amc)
	putService(&se, amc)
	if err != nil {
		fmt.Printf("Can not find an existed amc CR: %v\n", err)
		ceh.CreateAmc(amc)
	} else {
		ceh.UpdateAmc(amc)
	}

	fmt.Printf("Create or update an AppMeshConfig CR after a service has beed created: %s\n", amc.Name)
}

// DeleteService we assume we need to remove the service Spec part of AppMeshConfig
// after received a service deleted notification.
func (ceh *CRDEventHandler) DeleteService(se events.ServiceEvent) {
	fmt.Printf("CRD event handler: Deleting a service: %s\n", se.Service.Name)

	// TODO we should resolve the application name from the meta data placed in a zookeeper node.
	appCode := resolveAppCode(&se)
	// There is a chance to delete a service with an empty instances manually, but it is not be sure that which
	// amc should be modified.
	if appCode == "" {
		fmt.Printf("Can not find an application name with this deleting event: %v\n", se.Service.Name)
		return
	}

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appCode,
			Namespace: defaultNamespace,
		},
	}

	_, err := ceh.GetAmc(amc)
	if err != nil {
		fmt.Println("amc CR can not be found, ignore it")
		return
	} else {
		if amc.Spec.Services != nil && len(amc.Spec.Services) > 0 {
			for i, s := range amc.Spec.Services {
				if s.Name == se.Service.Name {
					result := DeleteInSlice(amc.Spec.Services, i)
					amc.Spec.Services = result.([]*v1.Service)
					break
					// TODO break? Can I assume there is no duplicate services belongs to a same amc?
				}
			}

			if len(amc.Spec.Services) == 0 {
				amc.Spec.Services = nil
			}

			ceh.UpdateAmc(amc)
		} else {
			fmt.Println("The services list belongs to this amc CR is empty, ignore it")
			return
		}
	}

}

// AddInstance ...
func (ceh *CRDEventHandler) AddInstance(ie events.ServiceEvent) {
	fmt.Printf("CRD event handler: Adding an instance\n%v\n", ie.Instance)

	// TODO we should resolve the application name from the meta data placed in a zookeeper node.
	appCode := resolveAppCode(&ie)

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appCode,
			Namespace: defaultNamespace,
		},
	}

	_, err := ceh.GetAmc(amc)
	putInstance(&ie, amc)
	if err != nil {
		fmt.Printf("Can not get an exist amc CR: %v\n", err)
		ceh.CreateAmc(amc)
	} else {
		ceh.UpdateAmc(amc)
	}

	fmt.Printf("Create or update an AppMeshConfig CR after an instance has beed added:%s\n", amc.Name)
}

// DeleteInstance ...
func (ceh *CRDEventHandler) DeleteInstance(ie events.ServiceEvent) {
	fmt.Printf("CRD event handler: deleting an instance\n%v\n", ie.Instance)

	appCode := resolveAppCode(&ie)
	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appCode,
			Namespace: defaultNamespace,
		},
	}
	_, err := ceh.GetAmc(amc)
	if err != nil {
		fmt.Printf("The applicatin mesh configruation can not be found with key: %s", appCode)
		return
	} else {
		deleteInstance(&ie, amc)
	}

	ceh.UpdateAmc(amc)
}

// CreateAmc
func (ceh *CRDEventHandler) CreateAmc(amc *v1.AppMeshConfig) {
	// TODO
	cluster, _ := ceh.K8sMgr.Get("tcc-gz01-bj5-test")
	err := cluster.Client.Create(context.Background(), amc)
	if err != nil {
		fmt.Printf("Creating an acm has an error:%v\n", err)
		return
	}
}

// UpdateAmc
func (ceh *CRDEventHandler) UpdateAmc(amc *v1.AppMeshConfig) {
	// TODO
	cluster, _ := ceh.K8sMgr.Get("tcc-gz01-bj5-test")
	err := cluster.Client.Update(context.Background(), amc)
	if err != nil {
		fmt.Printf("Updating an acm has an error: %v\n", err)
		return
	}
}

// GetAmc
func (ceh *CRDEventHandler) GetAmc(config *v1.AppMeshConfig) (*v1.AppMeshConfig, error) {
	cluster, _ := ceh.K8sMgr.Get("tcc-gz01-bj5-test")
	key, _ := client.ObjectKeyFromObject(config)
	err := cluster.Client.Get(context.Background(), key, config)
	return config, err
}

// putService Put a service derived from a service event into the application mesh config.
func putService(se *events.ServiceEvent, amc *v1.AppMeshConfig) {
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
func putInstance(ie *events.ServiceEvent, amc *v1.AppMeshConfig) {
	i := &v1.Instance{}
	i.Host = removePort(ie.Instance.Host)
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
				fmt.Printf("Instance %v has exist in a service\n", i)
				break
			}
		}
		if !hasExist {
			s.Instances = append(s.Instances, i)
		}
	}
}

// deleteInstance
func deleteInstance(ie *events.ServiceEvent, amc *v1.AppMeshConfig) {
	instance := &v1.Instance{
		Host: removePort(ie.Instance.Host),
		Port: convertPort(ie.Instance.Port),
	}

	if amc.Spec.Services == nil || len(amc.Spec.Services) == 0 {
		fmt.Printf("The List of services who will be changed by removing an instance is empty.")
		return
	}

	for _, s := range amc.Spec.Services {
		// Can not find a service name in an instance.
		//if s.Name != ie.Instance.Service.name {
		//	continue
		//}

		if s.Instances == nil || len(s.Instances) == 0 {
			fmt.Printf("The list of instances who will be change by removing an instance is empty.")
		}

		for index, i := range s.Instances {
			if i.Host == instance.Host && i.Port.Number == instance.Port.Number {
				result := DeleteInSlice(s.Instances, index)
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

// Removing the port part of a service name is necessary due to istio requirement.
// 127.0.0.1:10000 -> 127.0.0.1
func removePort(addressWithPort string) string {
	host, _, err := net.SplitHostPort(addressWithPort)
	if err != nil {
		fmt.Printf("Split host and port for a service name has an error:%v\n", err)
		// returning the original address instead if the address has a incorrect format
		return addressWithPort
	}
	return host
}

// toInt32 Convert a string variable to integer with 32 bit size.
func toUint32(portStr string) uint32 {
	port, _ := strconv.ParseInt(portStr, 10, 32)
	return uint32(port)
}

// convertPort Convert the port which has been defined in zookeeper library to the one that belongs to CRD.
func convertPort(port *events.Port) *v1.Port {
	return &v1.Port{
		Name:     port.Port,
		Protocol: port.Protocol,
		Number:   toUint32(port.Port),
	}
}

// Delete an element from a Slice with an index.
// return the original parameter as the result instead if it is not a slice.
func DeleteInSlice(s interface{}, index int) interface{} {
	value := reflect.ValueOf(s)
	if value.Kind() == reflect.Slice {
		//|| value.Kind() == reflect.Array {
		result := reflect.AppendSlice(value.Slice(0, index), value.Slice(index+1, value.Len()))
		return result.Interface()
	} else {
		fmt.Printf("Only a slice can be passed into this method for deleting an element of it.")
		return s
	}
}

// resolveAppCode Resolve the application code that was used as the key of an amc CR
// from the instance belongs to a service event.
func resolveAppCode(e *events.ServiceEvent) string {
	vi := findValidInstance(e)
	if vi == nil {
		fmt.Printf("Can not find a valid instance with this event.")
		return ""

		// it will use foo as the default application name with an origin dubbo SDK.
		//return "foo"
	}

	appCode := vi.Labels["application"]
	return appCode
}

// findValidInstance because the application name just belongs to an instance,
// we must try to find out an valid instance.
func findValidInstance(e *events.ServiceEvent) *events.Instance {
	if e == nil {
		fmt.Printf("Service event is nil when start to find a valid instance from it.\n")
		return nil
	}

	if e.Instance != nil {
		return e.Instance
	}

	if e.Service == nil || e.Service.Instances == nil || len(e.Service.Instances) == 0 {
		fmt.Printf("The instances list of this service is nil or empty when start to find valid instance from it.\n")
		return nil
	}

	for _, value := range e.Service.Instances {
		if value != nil && value.Labels != nil && value.Labels["application"] != "" {
			return value
		}
	}

	return nil
}