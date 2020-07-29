package handler

import (
	"github.com/symcn/mesh-operator/pkg/adapter/configcenter"

	"github.com/prometheus/client_golang/prometheus"
	v1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/adapter/metrics"
	"github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// KubeSingleClusterEventHandler ...
type KubeSingleClusterEventHandler struct {
	ctrlManager   manager.Manager
	configBuilder configcenter.ConfigBuilder
}

// NewKubeSingleClusterEventHandler ...
func NewKubeSingleClusterEventHandler(mgr manager.Manager, mc *v1.MeshConfig) (component.EventHandler, error) {
	dcb := &DubboConfiguratorBuilder{
		globalConfig:  mc,
		defaultConfig: DubboDefaultConfig,
	}

	return &KubeSingleClusterEventHandler{
		ctrlManager:   mgr,
		configBuilder: dcb,
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
func (kubeSceh *KubeSingleClusterEventHandler) ReplaceInstances(event *types.ServiceEvent,
	configuratorFinder func(s string) *types.ConfiguratorConfig) {
	klog.Infof("event handler for a single cluster: Replacing these instances(size: %d)\n%v", len(event.Instances), event.Instances)

	metrics.SynchronizedServiceCounter.Inc()
	metrics.SynchronizedInstanceCounter.Add(float64(len(event.Instances)))
	timer := prometheus.NewTimer(metrics.ReplacingInstancesHistogram)
	defer timer.ObserveDuration()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Convert a service event that noticed by zookeeper to a Service CRD
		cs := convertEvent(event.Service)

		var config *types.ConfiguratorConfig
		if configuratorFinder == nil {
			config = nil
		} else {
			// meanwhile we should search a configurator for such service
			config = configuratorFinder(event.Service.Name)
		}
		if config == nil {
			kubeSceh.configBuilder.SetConfig(cs, kubeSceh.configBuilder.GetDefaultConfig())
		} else {
			kubeSceh.configBuilder.SetConfig(cs, config)
		}

		// loading cs CR from k8s cluster
		foundCs, err := get(&v1.ConfiguredService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.FormatToDNS1123(event.Service.Name),
				Namespace: defaultNamespace,
			},
		}, kubeSceh.ctrlManager.GetClient())
		if err != nil {
			klog.Warningf("Can not find an existed cs CR: %v, then create such cs instead.", err)
			return create(cs, kubeSceh.ctrlManager.GetClient())
		}
		foundCs.Spec = cs.Spec
		return update(foundCs, kubeSceh.ctrlManager.GetClient())
	})

}

// DeleteService ...
func (kubeSceh *KubeSingleClusterEventHandler) DeleteService(event *types.ServiceEvent) {
	klog.Infof("event handler for a single cluster: Deleting a service\n%v\n", event.Service)
	metrics.DeletedServiceCounter.Inc()
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := delete(&v1.ConfiguredService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.FormatToDNS1123(event.Service.Name),
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

// ReplaceAccessorInstances ...
func (kubeSceh *KubeSingleClusterEventHandler) ReplaceAccessorInstances(e *types.ServiceEvent,
	getScopedServices func(s string) map[string]struct{}) {
	klog.Infof("event handler for a single cluster: replacing the accessor's instances: %s", e.Service.Name)
	metrics.ReplacedAccessorInstancesCounter.Inc()

	instances := e.Instances
	changedScopes := make(map[string]struct{})
	for _, ins := range instances {
		if ins != nil {
			sk, ok := ins.Labels["app"]
			if ok {
				changedScopes[sk] = struct{}{}
			}
		}
	}

	for changedScope := range changedScopes {
		scopedMapping := getScopedServices(changedScope)
		var accessedServices []string
		for s := range scopedMapping {
			accessedServices = append(accessedServices, s)
		}

		sas := &v1.ServiceAccessor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.FormatToDNS1123(changedScope),
				Namespace: defaultNamespace,
			},
			Spec: v1.ServiceAccessorSpec{
				AccessHosts: accessedServices,
			},
		}

		retry.RetryOnConflict(retry.DefaultRetry, func() error {
			foundSas, err := getScopedAccessServices(&v1.ServiceAccessor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.FormatToDNS1123(changedScope),
					Namespace: defaultNamespace,
				},
			}, kubeSceh.ctrlManager.GetClient())
			if err != nil {
				klog.Warningf("Can not find an existed asm CR: %v, then create a new one instead.", err)
				return createScopedAccessServices(foundSas, kubeSceh.ctrlManager.GetClient())
			}
			foundSas.Spec = sas.Spec
			return updateScopedAccessServices(foundSas, kubeSceh.ctrlManager.GetClient())
		})
	}

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
	klog.Infof("event handler for a single cluster: change a configuration %s", e.Path)
	metrics.ChangedConfigurationCounter.Inc()
	timer := prometheus.NewTimer(metrics.ChangingConfigurationHistogram)
	defer timer.ObserveDuration()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		serviceName := e.ConfigEntry.Key
		cs, err := get(&v1.ConfiguredService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.FormatToDNS1123(serviceName),
				Namespace: defaultNamespace,
			},
		}, kubeSceh.ctrlManager.GetClient())
		if err != nil {
			klog.Infof("Finding cs with name %s has an error: %v", serviceName, err)
			// TODO Is there a requirement to requeue this event?
			return nil
		}

		// utilize this configurator for such cs CR
		if e.ConfigEntry == nil || !e.ConfigEntry.Enabled {
			// TODO we really need to handle and think about the case that configuration has been disable.
			kubeSceh.configBuilder.SetConfig(cs, kubeSceh.configBuilder.GetDefaultConfig())
		} else {
			kubeSceh.configBuilder.SetConfig(cs, e.ConfigEntry)
		}

		return update(cs, kubeSceh.ctrlManager.GetClient())
	})

}

// DeleteConfigEntry ...
func (kubeSceh *KubeSingleClusterEventHandler) DeleteConfigEntry(e *types.ConfigEvent, cachedServiceFinder func(s string) *types.Service) {
	klog.Infof("event handler for a single cluster: delete a configuration %s", e.Path)
	metrics.DeletedConfigurationCounter.Inc()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// an example for the path: /dubbo/config/dubbo/com.foo.mesh.test.Demo.configurators
		// Usually deleting event don't include the configuration data, so that we should
		// parse the zNode path to decide what is the service name.
		serviceName := utils.FormatToDNS1123(utils.ResolveServiceName(e.Path))
		cs, err := get(&v1.ConfiguredService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: defaultNamespace,
			},
		}, kubeSceh.ctrlManager.GetClient())
		if err != nil {
			klog.Infof("Finding cs with name %s has an error: %v", serviceName, err)
			// TODO Is there a requirement to requeue this event?
			return nil
		}

		// Deleting a configuration of a service is similar to setting default configurator to this service
		kubeSceh.configBuilder.SetConfig(cs, kubeSceh.configBuilder.GetDefaultConfig())

		return update(cs, kubeSceh.ctrlManager.GetClient())
	})

}
