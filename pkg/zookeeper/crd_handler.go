package zookeeper

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"

	v1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
)

// CRDEventHandler it used for handling the event which has been send from the adapter client.
type CRDEventHandler struct {
	k8sMgr *k8smanager.ClusterManager
}

// AddService ...
func (ceh *CRDEventHandler) AddService(se ServiceEvent) {
	fmt.Printf("CRD event handler: Adding a service\n%v\n", se.Service)

	// Transform a service event that noticed by zookeeper to a Service CRD
	// TODO we should resolve the application name from the meta data placed in a zookeeper node.
	appName := "foo"

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: "default",
		},
	}

	_, err := ceh.GetAmc(amc)
	putService(&se, amc)
	if err != nil {
		fmt.Printf("Can not get an exist amc :%v\n", err)
		ceh.CreateAmc(amc)
	} else {
		ceh.UpdateAmc(amc)
	}

	fmt.Printf("Create or update an AppMeshConfig CR after a service has beed created:\n%v\n", amc.Name)
}

// DeleteService ...
func (ceh *CRDEventHandler) DeleteService(e ServiceEvent) {
	fmt.Printf("Don't be supported yet\n")
}

// AddInstance ...
func (ceh *CRDEventHandler) AddInstance(ie ServiceEvent) {
	fmt.Printf("CRD event handler: Adding an instance\n%v\n", ie.Instance)

	// TODO we should resolve the application name from the meta data placed in a zookeeper node.
	appName := "foo"

	amc := &v1.AppMeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: "default",
		},
	}

	_, err := ceh.GetAmc(amc)
	putInstance(&ie, amc)
	if err != nil {
		fmt.Printf("Can not get an exist amc :%v\n", err)
		ceh.CreateAmc(amc)
	} else {
		ceh.UpdateAmc(amc)
	}

	fmt.Printf("Create or update an AppMeshConfig CR after an instance has beed added:\n%v\n", amc.Name)

}

// DeleteInstance ...
func (ceh *CRDEventHandler) DeleteInstance(e ServiceEvent) {
	fmt.Printf("Don't be supported yet\n")
}

func (ceh *CRDEventHandler) CreateAmc(amc *v1.AppMeshConfig) {
	// TODO
	cluster, _ := ceh.k8sMgr.Get("tcc-gz01-bj5-test")
	err := cluster.Client.Create(context.Background(), amc)
	if err != nil {
		fmt.Printf("Creating an acm has an error:%v\n", err)
		return
	}
}

func (ceh *CRDEventHandler) UpdateAmc(amc *v1.AppMeshConfig) {
	// TODO
	cluster, _ := ceh.k8sMgr.Get("tcc-gz01-bj5-test")
	err := cluster.Client.Update(context.Background(), amc)
	if err != nil {
		fmt.Printf("Updating an acm has an error:%v\n", err)
		return
	}
}

func (ceh *CRDEventHandler) GetAmc(config *v1.AppMeshConfig) (*v1.AppMeshConfig, error) {
	cluster, _ := ceh.k8sMgr.Get("tcc-gz01-bj5-test")
	key, _ := client.ObjectKeyFromObject(config)
	err := cluster.Client.Get(context.Background(), key, config)
	return config, err
}

// putService Put a service derived from a service event into the application mesh config.
func putService(se *ServiceEvent, amc *v1.AppMeshConfig) {
	// Ports
	port, _ := strconv.ParseInt(se.Service.ports[0].Port, 10, 32)
	portInt := uint32(port)
	ports := []*v1.Port{}
	ports = append(ports, &v1.Port{
		Protocol: se.Service.ports[0].Protocol,
		Number:   portInt,
	})
	// Instances
	//for _,i :=range e.Instance

	s := &v1.Service{
		Name:    se.Service.name,
		AppName: "foo",
		Ports:   ports,
		//Instances: e.Service.instances,
		//Policy:  nil,
		//Subsets: nil,
	}

	if amc.Spec.Services == nil {
		services := []*v1.Service{}
		services = append(services, s)
		amc.Spec.Services = services
	} else {
		var hasExist = false
		for _, amcs := range amc.Spec.Services {
			if amcs.Name == s.Name {
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
func putInstance(ie *ServiceEvent, amc *v1.AppMeshConfig) {
	i := &v1.Instance{}
	i.Host = removePort(ie.Instance.Host)
	port, _ := strconv.ParseInt(ie.Instance.Port.Port, 10, 32)
	i.Port = &v1.Port{
		Name:     ie.Instance.Port.Port,
		Protocol: ie.Instance.Port.Protocol,
		Number:   uint32(port),
	}

	var s *v1.Service
	// Ports
	sport, _ := strconv.ParseInt(ie.Instance.Service.ports[0].Port, 10, 32)
	portInt := uint32(sport)
	ports := []*v1.Port{}
	ports = append(ports, &v1.Port{
		Protocol: ie.Instance.Service.ports[0].Protocol,
		Number:   portInt,
	})

	s = &v1.Service{
		Name:    ie.Instance.Service.name,
		AppName: "foo",
		Ports:   ports,
	}

	//Put service
	if amc.Spec.Services == nil {
		services := []*v1.Service{}
		services = append(services, s)
		amc.Spec.Services = services
	} else {
		var hasExist = false
		for _, amcs := range amc.Spec.Services {
			if amcs.Name == s.Name {
				hasExist = true
				s = amcs
				//TODO should we update the details of this service without instances
				break
			}
		}

		if !hasExist {
			amc.Spec.Services = append(amc.Spec.Services, s)
		}
	}

	// Put instance
	if s.Instances == nil {
		instances := []*v1.Instance{}
		instances = append(instances, i)
		s.Instances = instances
	} else {
		var hasExist = false
		for _, ins := range s.Instances {
			if ins.Host == i.Host && ins.Port.Number == i.Port.Number {
				hasExist = true
				fmt.Printf("Instance %v has been exist in a service\n", i)
				break
			}
		}
		if !hasExist {
			s.Instances = append(s.Instances, i)
		}
	}

}

// Removing the port part of a service name is necessary due to istio requirement.
// 127.0.0.1:10000 -> 127.0.0.1
func removePort(addressWithPort string) string {
	host, _, err := net.SplitHostPort(addressWithPort)
	if err != nil {
		fmt.Printf("Split host and port for a service name has an errt:%v", err)
		// returning the original address instead if the address has a incorrect format
		return addressWithPort
	}
	return host
}
