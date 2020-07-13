package handler

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/adapter/metrics"
	"github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/adapter/utils"
	v1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// KubeSingleClusterEventHandler ...
type KubeSingleClusterEventHandler struct {
	meshConfig  *v1.MeshConfig
	ctrlManager manager.Manager
}

// NewKubeSingleClusterEventHandler ...
func NewKubeSingleClusterEventHandler(mgr manager.Manager) (component.EventHandler, error) {
	mc := &v1.MeshConfig{}
	err := mgr.GetClient().Get(context.Background(), client.ObjectKey{
		Name:      "sym-meshconfig",
		Namespace: defaultNamespace,
	}, mc)
	if err != nil {
		return nil, fmt.Errorf("loading mesh config has an error: %v", err)
	}

	return &KubeSingleClusterEventHandler{
		ctrlManager: mgr,
		meshConfig:  mc,
	}, nil
}

// AddService ...
func (kubeSceh *KubeSingleClusterEventHandler) AddService(e *types.ServiceEvent, configuratorFinder func(s string) *types.ConfiguratorConfig) {
	klog.Warningf("Adding a service has not been implemented yet by single clusters handler.")
}

// AddInstance ...
func (kubeSceh *KubeSingleClusterEventHandler) AddInstance(e *types.ServiceEvent, configuratorFinder func(s string) *types.ConfiguratorConfig) {
	klog.Warningf("Adding an instance has not been implemented yet by single clusters handler.")
}

// ReplaceInstances ...
func (kubeSceh *KubeSingleClusterEventHandler) ReplaceInstances(event *types.ServiceEvent, configuratorFinder func(s string) *types.ConfiguratorConfig) {
	klog.Infof("event handler for a single cluster: Replacing these instances(size: %d)\n%v", len(event.Instances), event.Instances)

	metrics.SynchronizedServiceCounter.Inc()
	metrics.SynchronizedInstanceCounter.Add(float64(len(event.Instances)))
	timer := prometheus.NewTimer(metrics.ReplacingInstancesHistogram)
	defer timer.ObserveDuration()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Convert a service event that noticed by zookeeper to a Service CRD
		sme := convertEventToSme(event.Service)

		// meanwhile we should search a configurator for such service
		config := configuratorFinder(event.Service.Name)
		if config == nil {
			dc := *DefaultConfigurator
			dc.Key = event.Service.Name
			setConfig(&dc, sme, kubeSceh.meshConfig)
		} else {
			setConfig(config, sme, kubeSceh.meshConfig)
		}

		// loading sme CR from k8s cluster
		foundSme, err := get(&v1.ConfiguraredService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.StandardizeServiceName(event.Service.Name),
				Namespace: defaultNamespace,
			},
		}, kubeSceh.ctrlManager.GetClient())
		if err != nil {
			klog.Warningf("Can not find an existed sme CR: %v, then create such sme instead.", err)
			return create(sme, kubeSceh.ctrlManager.GetClient())
		}
		foundSme.Spec = sme.Spec
		return update(foundSme, kubeSceh.ctrlManager.GetClient())
	})

}

// DeleteService ...
func (kubeSceh *KubeSingleClusterEventHandler) DeleteService(event *types.ServiceEvent) {
	klog.Infof("event handler for a single cluster: Deleting a service\n%v\n", event.Service)
	metrics.DeletedServiceCounter.Inc()
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := delete(&v1.ConfiguraredService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.StandardizeServiceName(event.Service.Name),
				Namespace: defaultNamespace,
			},
		}, kubeSceh.ctrlManager.GetClient())
		return err
	})

}

// DeleteInstance ...
func (kubeSceh *KubeSingleClusterEventHandler) DeleteInstance(e *types.ServiceEvent) {
	klog.Warningf("Deleting an instance has not been implemented yet by single clusters handler.")
}

// AddConfigEntry ...
func (kubeSceh *KubeSingleClusterEventHandler) AddConfigEntry(e *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service) {
	klog.Infof("event handler for a single cluster: adding a configuration: %s", e.Path)
	metrics.AddedConfigurationCounter.Inc()
	// Adding a new configuration for a service is same as changing it.
	kubeSceh.ChangeConfigEntry(e, cachedServiceFinder)
}

// ChangeConfigEntry ...
func (kubeSceh *KubeSingleClusterEventHandler) ChangeConfigEntry(e *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service) {
	klog.Infof("event handler for a single cluster: change a configuration\n%v", e.Path)
	metrics.ChangedConfigurationCounter.Inc()
	timer := prometheus.NewTimer(metrics.ChangingConfigurationHistogram)
	defer timer.ObserveDuration()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		serviceName := e.ConfigEntry.Key
		sme, err := get(&v1.ConfiguraredService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.StandardizeServiceName(serviceName),
				Namespace: defaultNamespace,
			},
		}, kubeSceh.ctrlManager.GetClient())
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
			setConfig(&dc, sme, kubeSceh.meshConfig)
		} else {
			setConfig(e.ConfigEntry, sme, kubeSceh.meshConfig)
		}

		return update(sme, kubeSceh.ctrlManager.GetClient())
	})

}

// DeleteConfigEntry ...
func (kubeSceh *KubeSingleClusterEventHandler) DeleteConfigEntry(e *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service) {
	klog.Infof("event handler for a single cluster: delete a configuration\n%v", e.Path)
	metrics.DeletedConfigurationCounter.Inc()

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
		}, kubeSceh.ctrlManager.GetClient())
		if err != nil {
			klog.Infof("Finding sme with name %s has an error: %v", serviceName, err)
			// TODO Is there a requirement to requeue this event?
			return nil
		}

		// Deleting a configuration of a service is similar to setting default configurator to this service
		dc := *DefaultConfigurator
		dc.Key = serviceName
		setConfig(&dc, sme, kubeSceh.meshConfig)

		return update(sme, kubeSceh.ctrlManager.GetClient())
	})

}
