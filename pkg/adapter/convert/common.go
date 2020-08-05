package convert

import (
	v1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/adapter/constant"
	types2 "github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/utils"
	"k8s.io/klog"
)

// ToPort Convert the port which has been defined in zookeeper library to the one that belongs to CRD.
func ToPort(port *types2.Port) *v1.Port {
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
