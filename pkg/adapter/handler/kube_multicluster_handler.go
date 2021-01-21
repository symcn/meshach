package handler

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/symcn/meshach/pkg/adapter/component"
	"github.com/symcn/meshach/pkg/adapter/convert"
	"github.com/symcn/meshach/pkg/adapter/metrics"
	types2 "github.com/symcn/meshach/pkg/adapter/types"
	k8smanager "github.com/symcn/meshach/pkg/k8s/manager"
	"k8s.io/klog"
)

// KubeMultiClusterEventHandler it used for synchronizing the component which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.
type KubeMultiClusterEventHandler struct {
	handlers []component.EventHandler
}

// NewKubeMultiClusterEventHandler ...
func NewKubeMultiClusterEventHandler(clustersMgr *k8smanager.ClusterManager) (component.EventHandler, error) {
	converter := &convert.DubboConverter{DefaultNamespace: defaultNamespace}
	var kubeHandlers []component.EventHandler
	for _, c := range clustersMgr.GetAll() {
		h, err := NewKubeSingleClusterEventHandler(c.Mgr, converter)
		if err != nil {
			return nil, fmt.Errorf("initializing kube handler with a manager failed: %v", err)
		}
		kubeHandlers = append(kubeHandlers, h)
	}

	return &KubeMultiClusterEventHandler{
		handlers: kubeHandlers,
	}, nil
}

// AddService ...
func (h *KubeMultiClusterEventHandler) AddService(event *types2.ServiceEvent) {
	// klog.Infof("Kube multiple clusters event handler: Adding a service: %s", event.Service.Name)
	klog.V(6).Infof("Adding a service has not been implemented yet by multiple clusters handler.")
	// kubeMceh .ReplaceInstances(event, configuratorFinder)
}

// AddInstance ...
func (h *KubeMultiClusterEventHandler) AddInstance(event *types2.ServiceEvent) {
	klog.V(6).Infof("Adding an instance has not been implemented yet by multiple clusters handler.")
}

// ReplaceInstances ...
func (h *KubeMultiClusterEventHandler) ReplaceInstances(event *types2.ServiceEvent) {
	klog.V(6).Infof("event handler for multiple clusters: Replacing these instances(size: %d)", len(event.Instances))

	metrics.SynchronizedServiceCounter.Inc()
	metrics.SynchronizedInstanceCounter.Add(float64(len(event.Instances)))
	timer := prometheus.NewTimer(metrics.ReplacingInstancesHistogram)
	defer timer.ObserveDuration()

	wg := sync.WaitGroup{}
	wg.Add(len(h.handlers))
	for _, h := range h.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()

			handler.ReplaceInstances(event)
		}(h)
	}
	wg.Wait()
}

// DeleteService we assume we need to remove the service Spec part of AppMeshConfig
// after received a service deleted notification.
func (h *KubeMultiClusterEventHandler) DeleteService(event *types2.ServiceEvent) {
	klog.V(6).Infof("event handler for multiple clusters: Deleting a service: %v", event.Service)

	metrics.DeletedServiceCounter.Inc()

	wg := sync.WaitGroup{}
	wg.Add(len(h.handlers))
	for _, h := range h.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()
			handler.DeleteService(event)
		}(h)
	}
	wg.Wait()
}

// DeleteInstance ...
func (h *KubeMultiClusterEventHandler) DeleteInstance(event *types2.ServiceEvent) {
	klog.V(6).Infof("Deleting an instance has not been implemented yet by multiple clusters handler.")
}

// ReplaceAccessorInstances ...
func (h *KubeMultiClusterEventHandler) ReplaceAccessorInstances(
	event *types2.ServiceEvent,
	getScopedServices func(s string) map[string]struct{}) {
	klog.V(6).Infof("event handler for multiple clusters: Replacing these instances(size: %d)", len(event.Instances))

	wg := sync.WaitGroup{}
	wg.Add(len(h.handlers))
	for _, h := range h.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()

			handler.ReplaceAccessorInstances(event, getScopedServices)
		}(h)
	}
	wg.Wait()
}

// AddConfigEntry ...
func (h *KubeMultiClusterEventHandler) AddConfigEntry(e *types2.ConfigEvent) {
	klog.V(6).Infof("event handler for multiple clusters: adding a configuration: %s", e.Path)
	metrics.AddedConfigurationCounter.Inc()
	// Adding a new configuration for a service is same as changing it.
	h.ChangeConfigEntry(e)
}

// ChangeConfigEntry ...
func (h *KubeMultiClusterEventHandler) ChangeConfigEntry(e *types2.ConfigEvent) {
	klog.V(6).Infof("event handler for multiple clusters: changing a configuration: %s", e.Path)

	metrics.ChangedConfigurationCounter.Inc()
	timer := prometheus.NewTimer(metrics.ChangingConfigurationHistogram)
	defer timer.ObserveDuration()

	wg := sync.WaitGroup{}
	wg.Add(len(h.handlers))
	for _, h := range h.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()

			handler.ChangeConfigEntry(e)
		}(h)
	}
	wg.Wait()
}

// DeleteConfigEntry ...
func (h *KubeMultiClusterEventHandler) DeleteConfigEntry(e *types2.ConfigEvent) {
	klog.V(6).Infof("event handler for multiple clusters: deleting a configuration %s", e.Path)

	metrics.DeletedConfigurationCounter.Inc()

	wg := sync.WaitGroup{}
	wg.Add(len(h.handlers))
	for _, h := range h.handlers {
		go func(handler component.EventHandler) {
			defer wg.Done()

			handler.DeleteConfigEntry(e)
		}(h)
	}
	wg.Wait()
}
