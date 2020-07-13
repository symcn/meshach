package handler

import (
	"context"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/adapter/metrics"
	types2 "github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/adapter/utils"
	v1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	k8smanager "github.com/symcn/mesh-operator/pkg/k8s/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// KubeMultiClusterEventHandler it used for synchronizing the component which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.
type KubeMultiClusterEventHandler struct {
	k8sMgr     *k8smanager.ClusterManager
	meshConfig *v1.MeshConfig
}

// NewKubeMultiClusterEventHandler ...
func NewKubeMultiClusterEventHandler(k8sMgr *k8smanager.ClusterManager) (component.EventHandler, error) {
	mc := &v1.MeshConfig{}
	err := k8sMgr.MasterClient.GetClient().Get(context.Background(), types.NamespacedName{
		Name:      "sym-meshconfig",
		Namespace: defaultNamespace,
	}, mc)

	if err != nil {
		return nil, fmt.Errorf("loading mesh config has an error: %v", err)
	}

	return &KubeMultiClusterEventHandler{
		k8sMgr:     k8sMgr,
		meshConfig: mc,
	}, nil
}

// AddService ...
func (kubeMceh *KubeMultiClusterEventHandler) AddService(event *types2.ServiceEvent, configuratorFinder func(s string) *types2.ConfiguratorConfig) {
	// klog.Infof("Kube multiple clusters event handler: Adding a service: %s", event.Service.Name)
	klog.Warningf("Adding a service has not been implemented yet by multiple clusters handler.")
	// kubeMceh .ReplaceInstances(event, configuratorFinder)
}

// AddInstance ...
func (kubeMceh *KubeMultiClusterEventHandler) AddInstance(event *types2.ServiceEvent, configuratorFinder func(s string) *types2.ConfiguratorConfig) {
	klog.Warningf("Adding an instance has not been implemented yet by multiple clusters handler.")
}

// ReplaceInstances ...
func (kubeMceh *KubeMultiClusterEventHandler) ReplaceInstances(event *types2.ServiceEvent, configuratorFinder func(s string) *types2.ConfiguratorConfig) {
	klog.Infof("event handler for multiple clusters: Replacing these instances(size: %d)\n%v", len(event.Instances), event.Instances)

	metrics.SynchronizedServiceCounter.Inc()
	metrics.SynchronizedInstanceCounter.Add(float64(len(event.Instances)))
	timer := prometheus.NewTimer(metrics.ReplacingInstancesHistogram)
	defer timer.ObserveDuration()

	wg := sync.WaitGroup{}
	wg.Add(len(kubeMceh.k8sMgr.GetAll()))
	for _, cluster := range kubeMceh.k8sMgr.GetAll() {
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
					setConfig(&dc, sme, kubeMceh.meshConfig)
				} else {
					setConfig(config, sme, kubeMceh.meshConfig)
				}

				// loading sme CR from k8s cluster
				foundSme, err := get(&v1.ConfiguraredService{
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
func (kubeMceh *KubeMultiClusterEventHandler) DeleteService(event *types2.ServiceEvent) {
	klog.Infof("event handler for multiple clusters: Deleting a service: %s", event.Service)
	metrics.DeletedServiceCounter.Inc()

	wg := sync.WaitGroup{}
	wg.Add(len(kubeMceh.k8sMgr.GetAll()))
	for _, cluster := range kubeMceh.k8sMgr.GetAll() {
		go func() {
			defer wg.Done()
			retry.RetryOnConflict(retry.DefaultRetry, func() error {
				err := delete(&v1.ConfiguraredService{
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
func (kubeMceh *KubeMultiClusterEventHandler) DeleteInstance(event *types2.ServiceEvent) {
	klog.Warningf("Deleting an instance has not been implemented yet by multiple clusters handler.")
}

// AddConfigEntry ...
func (kubeMceh *KubeMultiClusterEventHandler) AddConfigEntry(e *types2.ConfigEvent, cachedServiceFinder func(s string) *types2.Service) {
	klog.Infof("event handler for multiple clusters: adding a configuration: %s", e.Path)
	metrics.AddedConfigurationCounter.Inc()
	// Adding a new configuration for a service is same as changing it.
	kubeMceh.ChangeConfigEntry(e, cachedServiceFinder)
}

// ChangeConfigEntry ...
func (kubeMceh *KubeMultiClusterEventHandler) ChangeConfigEntry(e *types2.ConfigEvent, cachedServiceFinder func(s string) *types2.Service) {
	klog.Infof("event handler for multiple clusters: changing a configuration: %s", e.Path)
	metrics.ChangedConfigurationCounter.Inc()
	timer := prometheus.NewTimer(metrics.ChangingConfigurationHistogram)
	defer timer.ObserveDuration()

	wg := sync.WaitGroup{}
	wg.Add(len(kubeMceh.k8sMgr.GetAll()))
	for _, cluster := range kubeMceh.k8sMgr.GetAll() {
		go func() {
			retry.RetryOnConflict(retry.DefaultRetry, func() error {
				serviceName := e.ConfigEntry.Key
				sme, err := get(&v1.ConfiguraredService{
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
					setConfig(&dc, sme, kubeMceh.meshConfig)
				} else {
					setConfig(e.ConfigEntry, sme, kubeMceh.meshConfig)
				}

				return update(sme, cluster.Client)
			})
		}()
	}
	wg.Wait()
}

// DeleteConfigEntry ...
func (kubeMceh *KubeMultiClusterEventHandler) DeleteConfigEntry(e *types2.ConfigEvent, cachedServiceFinder func(s string) *types2.Service) {
	klog.Infof("event handler for multiple clusters: deleting a configuration\n%s", e.Path)
	metrics.DeletedConfigurationCounter.Inc()

	wg := sync.WaitGroup{}
	wg.Add(len(kubeMceh.k8sMgr.GetAll()))
	for _, cluster := range kubeMceh.k8sMgr.GetAll() {
		go func() {
			retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// an example for the path: /dubbo/config/dubbo/com.foo.mesh.test.Demo.configurators
				// Usually deleting event don't include the configuration data, so that we should
				// parse the zNode path to decide what is the service name.
				serviceName := utils.StandardizeServiceName(utils.ResolveServiceName(e.Path))
				sme, err := get(&v1.ConfiguraredService{
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
				setConfig(&dc, sme, kubeMceh.meshConfig)

				return update(sme, cluster.Client)
			})
		}()
	}
	wg.Wait()
}
