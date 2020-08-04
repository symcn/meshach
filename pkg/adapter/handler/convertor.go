package handler

import (
	v1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/adapter/constant"
	types2 "github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

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

// convertToConfiguredService Convert service between these two formats
func convertToConfiguredService(s *types2.Service) *v1.ConfiguredService {
	// TODO Assuming every service can only provide an unique fixed port to adapt the dubbo case.
	cs := &v1.ConfiguredService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.FormatToDNS1123(s.Name),
			Namespace: defaultNamespace,
		},
		Spec: v1.ConfiguredServiceSpec{
			OriginalName: s.Name,
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
	// yamlbyte, err := yaml.Marshal(cs)
	// if err != nil {
	// 	klog.Errorf("Marshal yaml err:%+v", err)
	// 	return cs
	// }
	// fmt.Println("debug yaml======================")
	// fmt.Println(string(yamlbyte))
	// fmt.Println("================================")

	return cs
}

// convertToServiceConfig
func convertToServiceConfig(event *types2.ConfigEvent) *v1.ServiceConfig {
	return nil
}
