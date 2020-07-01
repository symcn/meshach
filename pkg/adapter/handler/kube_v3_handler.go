package handler

import (
	"context"
	"fmt"
	"github.com/mesh-operator/pkg/adapter/component"
	"github.com/mesh-operator/pkg/adapter/constant"
	"github.com/mesh-operator/pkg/adapter/utils"
	v1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

const (
	// FIX just for test with a fix name
	clusterName = "tcc-gz01-bj5-test"
)

// KubeV3EventHandler it used for synchronizing the component which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.
type KubeV3EventHandler struct {
	k8sMgr     *k8smanager.ClusterManager
	meshConfig *v1.MeshConfig
}

// NewKubev3EventHander ...
func NewKubeV3EventHandler(k8sMgr *k8smanager.ClusterManager) (component.EventHandler, error) {
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
		return nil, fmt.Errorf("initializing mesh config has an error: %v", err)
	}

	return &KubeV3EventHandler{
		k8sMgr:     k8sMgr,
		meshConfig: mc,
	}, nil
}

// AddService ...
func (kubev3eh *KubeV3EventHandler) AddService(event *component.ServiceEvent, configuratorFinder func(s string) *component.ConfiguratorConfig) {
	klog.Infof("Kube v3 event handler: Adding a service: %s", event.Service.Name)
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Convert a service event that noticed by zookeeper to a Service CRD
		sme := convertEventToSme(event.Service)
		// loading sme CR from k8s cluster
		sme, err := kubev3eh.get(sme)

		// meanwhile we should search a configurator for such service
		config := configuratorFinder(event.Service.Name)
		if config == nil {
			dc := *DefaultConfigurator
			dc.Key = event.Service.Name
			setConfig(&dc, sme, kubev3eh.meshConfig)
		} else {
			setConfig(config, sme, kubev3eh.meshConfig)
		}

		if err != nil {
			klog.Warningf("Can not find an existed sme CR: %v, then create such sme instead.", err)
			return kubev3eh.create(sme)
		} else {
			return kubev3eh.update(sme)
		}
	})
}

func (kubev3eh *KubeV3EventHandler) AddInstance(event *component.ServiceEvent, configuratorFinder func(s string) *component.ConfiguratorConfig) {
	klog.Warningf("Adding an instance has not been implemented yet by v3 handler.")
}

func (kubev3eh *KubeV3EventHandler) ReplaceInstances(event *component.ServiceEvent, configuratorFinder func(s string) *component.ConfiguratorConfig) {
	klog.Infof("Kube v3 event handler: Replacing these instances(size: %d)\n%v", len(event.Instances), event.Instances)
	kubev3eh.AddService(event, configuratorFinder)
}

// DeleteService we assume we need to remove the service Spec part of AppMeshConfig
// after received a service deleted notification.
func (kubev3eh *KubeV3EventHandler) DeleteService(event *component.ServiceEvent) {
	klog.Infof("Kube v3 event handler: Deleting a service: %s", event.Service)
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := kubev3eh.delete(&v1.ServiceMeshEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.StandardizeServiceName(event.Service.Name),
				Namespace: defaultNamespace,
			},
		})

		return err
	})
}

// DeleteInstance ...
func (kubev3eh *KubeV3EventHandler) DeleteInstance(event *component.ServiceEvent) {
	klog.Warningf("Deleting an instance has not been implemented yet by v3 handler.")
}

// convertService Convert service between these two formats
func convertEventToSme(s *component.Service) *v1.ServiceMeshEntry {
	// TODO Assuming every service can only provide an unique fixed port to adapt the dubbo case.
	sme := &v1.ServiceMeshEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.StandardizeServiceName(s.Name),
			Namespace: defaultNamespace,
		},
		Spec: v1.ServiceMeshEntrySpec{
			Name: s.Name,
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
		ins.Labels = i.Labels
		ins.Labels[constant.ZoneLabel] = constant.Zone
		instances = append(instances, ins)
	}
	sme.Spec.Instances = instances

	return sme
}

// createAmc
func (kubev3eh *KubeV3EventHandler) create(sme *v1.ServiceMeshEntry) error {
	// TODO
	cluster, _ := kubev3eh.k8sMgr.Get(clusterName)
	err := cluster.Client.Create(context.Background(), sme)
	klog.Infof("The generation of sme when creating: %d", sme.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Creating an sme has an error:%v\n", err)
		return err
	}
	return nil
}

// updateAmc
func (kubev3eh *KubeV3EventHandler) update(sme *v1.ServiceMeshEntry) error {
	cluster, err := kubev3eh.k8sMgr.Get(clusterName)
	if err != nil {
		return err
	}
	err = cluster.Client.Update(context.Background(), sme)
	klog.Infof("The generation of sme after updating: %d", sme.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Updating a sme has an error: %v\n", err)
		return err
	}

	return nil
}

// getAmc
func (kubev3eh *KubeV3EventHandler) get(sme *v1.ServiceMeshEntry) (*v1.ServiceMeshEntry, error) {
	cluster, err := kubev3eh.k8sMgr.Get(clusterName)
	if err != nil {
		return nil, err
	}
	err = cluster.Client.Get(context.Background(), types.NamespacedName{
		Namespace: sme.Namespace,
		Name:      sme.Name,
	}, sme)
	klog.Infof("The generation of sme when getting: %d", sme.ObjectMeta.Generation)
	return sme, err
}

// getAmc
func (kubev3eh *KubeV3EventHandler) delete(sme *v1.ServiceMeshEntry) error {
	cluster, err := kubev3eh.k8sMgr.Get(clusterName)
	if err != nil {
		return err
	}
	err = cluster.Client.Delete(context.Background(), sme)
	klog.Infof("The generation of sme when getting: %d", sme.ObjectMeta.Generation)
	return err
}

// AddConfigEntry
func (kubev3eh *KubeV3EventHandler) AddConfigEntry(e *component.ConfigEvent, cachedServiceFinder func(s string) *component.Service) {
	klog.Infof("Kube v3 event handler: adding a configuration\n%v\n", e.Path)
	// Adding a new configuration for a service is same as changing it.
	kubev3eh.ChangeConfigEntry(e, cachedServiceFinder)
}

// ChangeConfigEntry
func (kubev3eh *KubeV3EventHandler) ChangeConfigEntry(e *component.ConfigEvent, cachedServiceFinder func(s string) *component.Service) {
	klog.Infof("Kube v3 event handler: change a configuration\n%v", e.Path)
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		serviceName := e.ConfigEntry.Key
		sme, err := kubev3eh.get(&v1.ServiceMeshEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.StandardizeServiceName(serviceName),
				Namespace: defaultNamespace,
			},
		})
		if err != nil {
			klog.Infof("Finding sme with name %s has an error: %v", serviceName, err)
			// TODO Is there a requirement to requeue this event?
			return nil
		}

		// utilize this configurator for such amc CR
		if e.ConfigEntry == nil || !e.ConfigEntry.Enabled {
			// TODO we really need to handle and think about the case that configuration has been disable.
			dc := *DefaultConfigurator
			dc.Key = serviceName
			setConfig(&dc, sme, kubev3eh.meshConfig)
		} else {
			setConfig(e.ConfigEntry, sme, kubev3eh.meshConfig)
		}

		return kubev3eh.update(sme)
	})
}

// DeleteConfigEntry
func (kubev3eh *KubeV3EventHandler) DeleteConfigEntry(e *component.ConfigEvent, cachedServiceFinder func(s string) *component.Service) {
	klog.Infof("Kube v3 event handler: delete a configuration\n%v", e.Path)
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// an example for the path: /dubbo/config/dubbo/com.dmall.mesh.test.PoviderDemo.configurators
		// Usually deleting event don't include the configuration data, so that we should
		// parse the zNode path to decide what is the service name.
		serviceName := utils.StandardizeServiceName(e.ConfigEntry.Key)
		sme, err := kubev3eh.get(&v1.ServiceMeshEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: defaultNamespace,
			},
		})
		if err != nil {
			klog.Infof("Finding sme with name %s has an error: %v", serviceName, err)
			// TODO Is there a requirement to requeue this event?
			return nil
		}

		// Deleting a configuration of a service is similar to setting default configurator to this service
		dc := *DefaultConfigurator
		dc.Key = serviceName
		setConfig(&dc, sme, kubev3eh.meshConfig)

		return kubev3eh.update(sme)
	})
}
