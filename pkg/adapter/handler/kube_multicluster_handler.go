package handler

import (
	"context"
	"fmt"
	"sync"

	v1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/adapter/configcenter"
	"k8s.io/apimachinery/pkg/types"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/adapter/metrics"
	types2 "github.com/symcn/mesh-operator/pkg/adapter/types"
	k8smanager "github.com/symcn/mesh-operator/pkg/k8s/manager"
	"k8s.io/klog"
)

// KubeMultiClusterEventHandler it used for synchronizing the component which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.
type KubeMultiClusterEventHandler struct {
	k8sMgr        *k8smanager.ClusterManager
	configBuilder configcenter.ConfigBuilder
	handlers      []component.EventHandler
}

// NewKubeMultiClusterEventHandler ...
func NewKubeMultiClusterEventHandler(k8sMgr *k8smanager.ClusterManager) (component.EventHandler, error) {
	mc := &v1.MeshConfig{}
	err := k8sMgr.MasterClient.GetClient().Get(context.Background(), types.NamespacedName{
		Name:      meshConfigName,
		Namespace: defaultNamespace,
	}, mc)
	if err != nil {
		return nil, fmt.Errorf("loading mesh config has an error: %v", err)
	}

	var kubeHandlers []component.EventHandler
	for _, c := range k8sMgr.GetAll() {
		h, err := NewKubeSingleClusterEventHandler(c.Mgr, mc)
		if err != nil {
			klog.Errorf("initializing kube handler with a manager failed: %v", err)
			return nil, err
		}

		kubeHandlers = append(kubeHandlers, h)
	}

	return &KubeMultiClusterEventHandler{
		k8sMgr:   k8sMgr,
		handlers: kubeHandlers,
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
	wg.Add(len(kubeMceh.handlers))
	for _, h := range kubeMceh.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()

			handler.ReplaceInstances(event, configuratorFinder)
		}(h)
	}
	wg.Wait()
}

// DeleteService we assume we need to remove the service Spec part of AppMeshConfig
// after received a service deleted notification.
func (kubeMceh *KubeMultiClusterEventHandler) DeleteService(event *types2.ServiceEvent) {
	klog.Infof("event handler for multiple clusters: Deleting a service: \n%v", event.Service)
	metrics.DeletedServiceCounter.Inc()

	wg := sync.WaitGroup{}
	wg.Add(len(kubeMceh.handlers))
	for _, h := range kubeMceh.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()
			handler.DeleteService(event)
		}(h)
	}
	wg.Wait()
}

// DeleteInstance ...
func (kubeMceh *KubeMultiClusterEventHandler) DeleteInstance(event *types2.ServiceEvent) {
	klog.Warningf("Deleting an instance has not been implemented yet by multiple clusters handler.")
}

// ReplaceAccessorInstances ...
func (kubeMceh *KubeMultiClusterEventHandler) ReplaceAccessorInstances(event *types2.ServiceEvent,
	getScopedServices func(s string) map[string]struct{}) {
	klog.Infof("event handler for multiple clusters: Replacing these instances(size: %d)\n%v", len(event.Instances), event.Instances)

	wg := sync.WaitGroup{}
	wg.Add(len(kubeMceh.handlers))
	for _, h := range kubeMceh.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()

			handler.ReplaceAccessorInstances(event, getScopedServices)
		}(h)
	}
	wg.Wait()
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
	wg.Add(len(kubeMceh.handlers))
	for _, h := range kubeMceh.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()

			handler.ChangeConfigEntry(e, cachedServiceFinder)
		}(h)
	}
	wg.Wait()
}

// DeleteConfigEntry ...
func (kubeMceh *KubeMultiClusterEventHandler) DeleteConfigEntry(e *types2.ConfigEvent, cachedServiceFinder func(s string) *types2.Service) {
	klog.Infof("event handler for multiple clusters: deleting a configuration %s", e.Path)
	metrics.DeletedConfigurationCounter.Inc()

	wg := sync.WaitGroup{}
	wg.Add(len(kubeMceh.handlers))
	for _, h := range kubeMceh.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()

			handler.DeleteConfigEntry(e, cachedServiceFinder)
		}(h)
	}
	wg.Wait()
}
