package zookeeper

import (
	"fmt"
	v1 "github.com/mesh-operator/pkg/apis/mesh/v1"
)

// CRDEventHandler it used for handling the event which has been send from the adapter client.
type CRDEventHandler struct {
}

func (ceh *CRDEventHandler) AddService(e ServiceEvent) {
	fmt.Printf("CRD event handler: Adding a service\n%v\n", e.Service)

	s := &v1.Service{
		Name:    "foo", //e.Service.name,
		AppName: "foo",
		//Ports:     e.Service.ports,
		//Instances: e.Service.instances,
		//Policy:  nil,
		//Subsets: nil,
	}
	services := []*v1.Service{}
	services = append(services, s)
	amc := &v1.AppMeshConfig{
		Spec: v1.AppMeshConfigSpec{
			Services: services,
		},
	}

	fmt.Printf("Create an AppMeshConfig CR after a service has beed created:\n%v\n", amc.Name)
}

func (ceh *CRDEventHandler) DeleteService(e ServiceEvent) {
	fmt.Printf("Don't be supported yet\n")
}

func (ceh *CRDEventHandler) AddInstance(e ServiceEvent) {
	fmt.Printf("Don't be supported yet\n")
}

func (ceh *CRDEventHandler) DeleteInstance(e ServiceEvent) {
	fmt.Printf("Don't be supported yet\n")
}
