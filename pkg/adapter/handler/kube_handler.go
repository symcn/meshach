package handler

import (
	"context"
	"sync"

	v1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/adapter/constant"
	types2 "github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultNamespace = "sym-admin"
	clusterName      = ""
	meshConfigName   = "sym-meshconfig"
	l                sync.Mutex
	sasl             sync.Mutex
)

// KubeEventHandler it used for synchronizing the component which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.

// convertPort Convert the port which has been defined in zookeeper library to the one that belongs to CRD.
func convertPort(port *types2.Port) *v1.Port {
	if port == nil {
		klog.Warningf("Could not convert a Port due to a nil value")
		return &v1.Port{
			Name:     constant.DubboPortName,
			Protocol: "dubbo",
			Number:   0,
		}
	}

	return &v1.Port{
		// Name:     port.Port,
		Name:     constant.DubboPortName,
		Protocol: port.Protocol,
		Number:   utils.ToUint32(port.Port),
	}
}

// convertService Convert service between these two formats
func convertEvent(s *types2.Service) *v1.ConfiguredService {
	// TODO Assuming every service can only provide an unique fixed port to adapt the dubbo case.
	cs := &v1.ConfiguredService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.FormatToDNS1123(s.Name),
			Namespace: defaultNamespace,
		},
		Spec: v1.ConfiguredServiceSpec{
			OriginalName: s.Name,
			Ports: []*v1.Port{{
				Name:     constant.DubboPortName,
				Protocol: constant.DubboProtocol,
				Number:   utils.ToUint32(constant.MosnPort),
			}},
		},
	}

	var instances []*v1.Instance
	for _, i := range s.Instances {
		ins := &v1.Instance{}
		ins.Host = utils.RemovePort(i.Host)
		ins.Port = convertPort(i.Port)
		ins.Labels = make(map[string]string)
		for k, v := range i.Labels {
			ins.Labels[k] = v
		}
		ins.Labels[constant.InstanceLabelZoneName] = constant.ZoneValue
		instances = append(instances, ins)
	}
	cs.Spec.Instances = instances

	return cs
}

// create
func create(cs *v1.ConfiguredService, c client.Client) error {
	l.Lock()
	defer l.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Create(context.Background(), cs)
	klog.Infof("The generation of cs when creating: %d", cs.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Creating an cs has an error:%v\n", err)
		return err
	}
	return nil
}

// update
func update(cs *v1.ConfiguredService, c client.Client) error {
	l.Lock()
	defer l.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Update(context.Background(), cs)
	klog.Infof("The generation of cs after updating: %d", cs.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Updating a cs has an error: %v\n", err)
		return err
	}

	return nil
}

// get
func get(cs *v1.ConfiguredService, c client.Client) (*v1.ConfiguredService, error) {
	err := c.Get(context.Background(), types.NamespacedName{
		Namespace: cs.Namespace,
		Name:      cs.Name,
	}, cs)
	klog.Infof("The generation of cs when getting: %d", cs.ObjectMeta.Generation)
	return cs, err
}

// delete
func delete(cs *v1.ConfiguredService, c client.Client) error {
	err := c.Delete(context.Background(), cs)
	klog.Infof("The generation of cs when getting: %d", cs.ObjectMeta.Generation)
	return err
}

// createScopedAccessServices
func createScopedAccessServices(sa *v1.ServiceAccessor, c client.Client) error {
	sasl.Lock()
	defer sasl.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Create(context.Background(), sa)
	klog.Infof("The generation of sa when creating: %d", sa.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Creating an sa has an error:%v\n", err)
		return err
	}
	return nil
}

// updateScopedAccessServices
func updateScopedAccessServices(sa *v1.ServiceAccessor, c client.Client) error {
	sasl.Lock()
	defer sasl.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Update(context.Background(), sa)
	klog.Infof("The generation of sa after updating: %d", sa.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Updating a sa has an error: %v\n", err)
		return err
	}

	return nil
}

// getScopedAccessServices
func getScopedAccessServices(sa *v1.ServiceAccessor, c client.Client) (*v1.ServiceAccessor, error) {
	err := c.Get(context.Background(), types.NamespacedName{
		Namespace: sa.Namespace,
		Name:      sa.Name,
	}, sa)
	klog.Infof("The generation of sa when getting: %d", sa.ObjectMeta.Generation)
	return sa, err
}
