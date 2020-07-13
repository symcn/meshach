package handler

import (
	"context"

	"github.com/symcn/mesh-operator/pkg/adapter/constant"
	types2 "github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/adapter/utils"
	v1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultNamespace = "sym-admin"
	clusterName      = ""
)

// KubeEventHandler it used for synchronizing the component which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.

// convertPort Convert the port which has been defined in zookeeper library to the one that belongs to CRD.
func convertPort(port *types2.Port) *v1.Port {
	return &v1.Port{
		// Name:     port.Port,
		Name:     constant.DubboPortName,
		Protocol: port.Protocol,
		Number:   utils.ToUint32(port.Port),
	}
}

// convertService Convert service between these two formats
func convertEventToSme(s *types2.Service) *v1.ConfiguraredService {
	// TODO Assuming every service can only provide an unique fixed port to adapt the dubbo case.
	sme := &v1.ConfiguraredService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.StandardizeServiceName(s.Name),
			Namespace: defaultNamespace,
		},
		Spec: v1.ConfiguraredServiceSpec{
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
	sme.Spec.Instances = instances

	return sme
}

// create
func create(sme *v1.ConfiguraredService, c client.Client) error {
	err := c.Create(context.Background(), sme)
	klog.Infof("The generation of sme when creating: %d", sme.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Creating an sme has an error:%v\n", err)
		return err
	}
	return nil
}

// update
func update(sme *v1.ConfiguraredService, c client.Client) error {
	err := c.Update(context.Background(), sme)
	klog.Infof("The generation of sme after updating: %d", sme.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Updating a sme has an error: %v\n", err)
		return err
	}

	return nil
}

// get
func get(sme *v1.ConfiguraredService, c client.Client) (*v1.ConfiguraredService, error) {
	err := c.Get(context.Background(), types.NamespacedName{
		Namespace: sme.Namespace,
		Name:      sme.Name,
	}, sme)
	klog.Infof("The generation of sme when getting: %d", sme.ObjectMeta.Generation)
	return sme, err
}

// delete
func delete(sme *v1.ConfiguraredService, c client.Client) error {
	err := c.Delete(context.Background(), sme)
	klog.Infof("The generation of sme when getting: %d", sme.ObjectMeta.Generation)
	return err
}
