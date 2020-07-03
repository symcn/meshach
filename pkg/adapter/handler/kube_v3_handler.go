package handler

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/symcn/mesh-operator/pkg/adapter/metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"

	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/adapter/constant"
	types2 "github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/adapter/utils"
	v1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	k8smanager "github.com/symcn/mesh-operator/pkg/k8s/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// KubeV3EventHandler it used for synchronizing the component which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.
type KubeV3EventHandler struct {
	k8sMgr     *k8smanager.ClusterManager
	meshConfig *v1.MeshConfig
}

// NewKubeV3EventHandler ...
func NewKubeV3EventHandler(k8sMgr *k8smanager.ClusterManager) (component.EventHandler, error) {
	mc := &v1.MeshConfig{}
	err := k8sMgr.MasterClient.GetClient().Get(context.Background(), types.NamespacedName{
		Name:      "sym-meshconfig",
		Namespace: defaultNamespace,
	}, mc)

	if err != nil {
		return nil, fmt.Errorf("loading mesh config has an error: %v", err)
	}

	return &KubeV3EventHandler{
		k8sMgr:     k8sMgr,
		meshConfig: mc,
	}, nil
}

// AddService ...
func (kubev3eh *KubeV3EventHandler) AddService(event *types2.ServiceEvent, configuratorFinder func(s string) *types2.ConfiguratorConfig) {
	// klog.Infof("Kube v3 event handler: Adding a service: %s", event.Service.Name)
	klog.Warningf("Adding a service has not been implemented yet by v3 handler.")
	// kubev3eh.ReplaceInstances(event, configuratorFinder)
}

// AddInstance ...
func (kubev3eh *KubeV3EventHandler) AddInstance(event *types2.ServiceEvent, configuratorFinder func(s string) *types2.ConfiguratorConfig) {
	klog.Warningf("Adding an instance has not been implemented yet by v3 handler.")
}

// ReplaceInstances ...
func (kubev3eh *KubeV3EventHandler) ReplaceInstances(event *types2.ServiceEvent, configuratorFinder func(s string) *types2.ConfiguratorConfig) {
	klog.Infof("Kube v3 event handler: Replacing these instances(size: %d)\n%v", len(event.Instances), event.Instances)

	metrics.SynchronizedServiceCounter.Inc()
	metrics.SynchronizedInstanceCounter.Add(float64(len(event.Instances)))
	timer := prometheus.NewTimer(metrics.ReplacingInstancesHistogram)
	defer timer.ObserveDuration()

	wg := sync.WaitGroup{}
	wg.Add(len(kubev3eh.k8sMgr.GetAll()))
	for _, cluster := range kubev3eh.k8sMgr.GetAll() {
		go func() {
			defer wg.Done()

			retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// Convert a service event that noticed by zookeeper to a Service CRD
				sme := convertEventToSme(event.Service)

				// meanwhile we should search a configurator for such service
				config := configuratorFinder(event.Service.Name)
				if config == nil {
					dc := *DefaultConfigurator
					dc.Key = event.Service.Name
					setConfig(&dc, sme, kubev3eh.meshConfig)
				} else {
					setConfig(config, sme, kubev3eh.meshConfig)
				}

				// loading sme CR from k8s cluster
				foundSme, err := get(&v1.ServiceMeshEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.StandardizeServiceName(event.Service.Name),
						Namespace: defaultNamespace,
					},
				}, cluster.Client)
				if err != nil {
					klog.Warningf("Can not find an existed sme CR: %v, then create such sme instead.", err)
					return create(sme, cluster.Client)
				}
				foundSme.Spec = sme.Spec
				return update(foundSme, cluster.Client)
			})
		}()
	}
	wg.Wait()
}

// DeleteService we assume we need to remove the service Spec part of AppMeshConfig
// after received a service deleted notification.
func (kubev3eh *KubeV3EventHandler) DeleteService(event *types2.ServiceEvent) {
	klog.Infof("Kube v3 event handler: Deleting a service: %s", event.Service)
	metrics.DeletedServiceCounter.Inc()

	wg := sync.WaitGroup{}
	wg.Add(len(kubev3eh.k8sMgr.GetAll()))
	for _, cluster := range kubev3eh.k8sMgr.GetAll() {
		go func() {
			defer wg.Done()
			retry.RetryOnConflict(retry.DefaultRetry, func() error {
				err := delete(&v1.ServiceMeshEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.StandardizeServiceName(event.Service.Name),
						Namespace: defaultNamespace,
					},
				}, cluster.Client)

				return err
			})
		}()
	}
	wg.Wait()
}

// DeleteInstance ...
func (kubev3eh *KubeV3EventHandler) DeleteInstance(event *types2.ServiceEvent) {
	klog.Warningf("Deleting an instance has not been implemented yet by v3 handler.")
}

// convertService Convert service between these two formats
func convertEventToSme(s *types2.Service) *v1.ServiceMeshEntry {
	// TODO Assuming every service can only provide an unique fixed port to adapt the dubbo case.
	sme := &v1.ServiceMeshEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.StandardizeServiceName(s.Name),
			Namespace: defaultNamespace,
		},
		Spec: v1.ServiceMeshEntrySpec{
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
func create(sme *v1.ServiceMeshEntry, c client.Client) error {
	err := c.Create(context.Background(), sme)
	klog.Infof("The generation of sme when creating: %d", sme.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Creating an sme has an error:%v\n", err)
		return err
	}
	return nil
}

// update
func update(sme *v1.ServiceMeshEntry, c client.Client) error {
	err := c.Update(context.Background(), sme)
	klog.Infof("The generation of sme after updating: %d", sme.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Updating a sme has an error: %v\n", err)
		return err
	}

	return nil
}

// get
func get(sme *v1.ServiceMeshEntry, c client.Client) (*v1.ServiceMeshEntry, error) {
	err := c.Get(context.Background(), types.NamespacedName{
		Namespace: sme.Namespace,
		Name:      sme.Name,
	}, sme)
	klog.Infof("The generation of sme when getting: %d", sme.ObjectMeta.Generation)
	return sme, err
}

// delete
func delete(sme *v1.ServiceMeshEntry, c client.Client) error {
	err := c.Delete(context.Background(), sme)
	klog.Infof("The generation of sme when getting: %d", sme.ObjectMeta.Generation)
	return err
}

// AddConfigEntry ...
func (kubev3eh *KubeV3EventHandler) AddConfigEntry(e *types2.ConfigEvent, cachedServiceFinder func(s string) *types2.Service) {
	klog.Infof("Kube v3 event handler: adding a configuration: %s", e.Path)
	metrics.AddedConfigurationCounter.Inc()
	// Adding a new configuration for a service is same as changing it.
	kubev3eh.ChangeConfigEntry(e, cachedServiceFinder)
}

// ChangeConfigEntry ...
func (kubev3eh *KubeV3EventHandler) ChangeConfigEntry(e *types2.ConfigEvent, cachedServiceFinder func(s string) *types2.Service) {
	klog.Infof("Kube v3 event handler: changing a configuration: %s", e.Path)
	metrics.ChangedConfigurationCounter.Inc()
	timer := prometheus.NewTimer(metrics.ChangingConfigurationHistogram)
	defer timer.ObserveDuration()

	wg := sync.WaitGroup{}
	wg.Add(len(kubev3eh.k8sMgr.GetAll()))
	for _, cluster := range kubev3eh.k8sMgr.GetAll() {
		go func() {
			retry.RetryOnConflict(retry.DefaultRetry, func() error {
				serviceName := e.ConfigEntry.Key
				sme, err := get(&v1.ServiceMeshEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.StandardizeServiceName(serviceName),
						Namespace: defaultNamespace,
					},
				}, cluster.Client)
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

				return update(sme, cluster.Client)
			})
		}()
	}
	wg.Wait()
}

// DeleteConfigEntry ...
func (kubev3eh *KubeV3EventHandler) DeleteConfigEntry(e *types2.ConfigEvent, cachedServiceFinder func(s string) *types2.Service) {
	klog.Infof("Kube v3 event handler: deleting a configuration\n%s", e.Path)
	metrics.DeletedConfigurationCounter.Inc()

	wg := sync.WaitGroup{}
	wg.Add(len(kubev3eh.k8sMgr.GetAll()))
	for _, cluster := range kubev3eh.k8sMgr.GetAll() {
		go func() {
			retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// an example for the path: /dubbo/config/dubbo/com.foo.mesh.test.Demo.configurators
				// Usually deleting event don't include the configuration data, so that we should
				// parse the zNode path to decide what is the service name.
				serviceName := utils.StandardizeServiceName(utils.ResolveServiceName(e.Path))
				sme, err := get(&v1.ServiceMeshEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      serviceName,
						Namespace: defaultNamespace,
					},
				}, cluster.Client)
				if err != nil {
					klog.Infof("Finding sme with name %s has an error: %v", serviceName, err)
					// TODO Is there a requirement to requeue this event?
					return nil
				}

				// Deleting a configuration of a service is similar to setting default configurator to this service
				dc := *DefaultConfigurator
				dc.Key = serviceName
				setConfig(&dc, sme, kubev3eh.meshConfig)

				return update(sme, cluster.Client)
			})
		}()
	}
	wg.Wait()
}
