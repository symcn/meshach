package handler

import (
	"github.com/prometheus/client_golang/prometheus"
	v1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/adapter/metrics"
	"github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// KubeSingleClusterEventHandler ...
type KubeSingleClusterEventHandler struct {
	ctrlManager manager.Manager
}

// NewKubeSingleClusterEventHandler ...
func NewKubeSingleClusterEventHandler(mgr manager.Manager) (component.EventHandler, error) {
	return &KubeSingleClusterEventHandler{
		ctrlManager: mgr,
	}, nil
}

// AddService ...
func (kubeSceh *KubeSingleClusterEventHandler) AddService(e *types.ServiceEvent) {
	klog.Warningf("Adding a service has not been implemented yet by single clusters handler.")
}

// AddInstance ...
func (kubeSceh *KubeSingleClusterEventHandler) AddInstance(e *types.ServiceEvent) {
	klog.Warningf("Adding an instance has not been implemented yet by single clusters handler.")
}

// ReplaceInstances ...
func (kubeSceh *KubeSingleClusterEventHandler) ReplaceInstances(event *types.ServiceEvent) {
	klog.Infof("event handler for a single cluster: Replacing these instances(size: %d)\n%v", len(event.Instances), event.Instances)

	metrics.SynchronizedServiceCounter.Inc()
	metrics.SynchronizedInstanceCounter.Add(float64(len(event.Instances)))
	timer := prometheus.NewTimer(metrics.ReplacingInstancesHistogram)
	defer timer.ObserveDuration()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Convert a service event that noticed by zookeeper to a Service CRD
		cs := convertToConfiguredService(event.Service)

		// loading cs CR from k8s cluster
		foundCs, err := getConfiguredService(cs.Namespace, cs.Name, kubeSceh.ctrlManager.GetClient())
		if err != nil {
			klog.Warningf("Can not find an existed cs CR: %v, then create such cs instead.", err)
			return createConfiguredService(cs, kubeSceh.ctrlManager.GetClient())
		}
		foundCs.Spec = cs.Spec
		return updateConfiguredService(foundCs, kubeSceh.ctrlManager.GetClient())
	})

}

// DeleteService ...
func (kubeSceh *KubeSingleClusterEventHandler) DeleteService(event *types.ServiceEvent) {
	klog.Infof("event handler for a single cluster: Deleting a service\n%v\n", event.Service)
	metrics.DeletedServiceCounter.Inc()
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := deleteConfiguredService(&v1.ConfiguredService{
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
			} else {
				klog.Warningf("Could't find label [app]'s value, instance: %s, skip updating the associated scope", ins.Host)
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
			Status: v1.ServiceAccessorStatus{},
		}

		retry.RetryOnConflict(retry.DefaultRetry, func() error {
			foundSas, err := getScopedAccessServices(sas.Namespace, sas.Name, kubeSceh.ctrlManager.GetClient())
			if err != nil {
				klog.Warningf("Can not find an existed asm CR: %v, then create a new one instead.", err)
				return createScopedAccessServices(sas, kubeSceh.ctrlManager.GetClient())
			}
			foundSas.Spec = sas.Spec
			return updateScopedAccessServices(foundSas, kubeSceh.ctrlManager.GetClient())
		})
	}

}

// AddConfigEntry ...
func (kubeSceh *KubeSingleClusterEventHandler) AddConfigEntry(e *types.ConfigEvent) {
	klog.Infof("event handler for a single cluster: adding a configuration: %s", e.Path)
	metrics.AddedConfigurationCounter.Inc()
	// Adding a new configuration for a service is same as changing it.
	kubeSceh.ChangeConfigEntry(e)
}

// ChangeConfigEntry ...
func (kubeSceh *KubeSingleClusterEventHandler) ChangeConfigEntry(e *types.ConfigEvent) {
	klog.Infof("event handler for a single cluster: change a configuration %s", e.Path)
	metrics.ChangedConfigurationCounter.Inc()
	timer := prometheus.NewTimer(metrics.ChangingConfigurationHistogram)
	defer timer.ObserveDuration()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		sc := convertToServiceConfig(e)
		foundSc, err := getServiceConfig(defaultNamespace, utils.FormatToDNS1123(sc.Name), kubeSceh.ctrlManager.GetClient())
		if err != nil {
			klog.Errorf("Finding ServiceConfig with name %s has an error: %v", sc.Name, err)
			if errors.IsNotFound(err) {
				klog.Infof("Could not find the ConfiguredService %s, then create it. %v", sc.Name, err)
				return createServiceConfig(sc, kubeSceh.ctrlManager.GetClient())
			}
			// TODO Is there a requirement to requeue this event?
			return nil
		}
		foundSc.Spec = sc.Spec
		return updateServiceConfig(foundSc, kubeSceh.ctrlManager.GetClient())
	})

}

// DeleteConfigEntry ...
func (kubeSceh *KubeSingleClusterEventHandler) DeleteConfigEntry(e *types.ConfigEvent) {
	klog.Infof("event handler for a single cluster: delete a configuration %s", e.Path)
	metrics.DeletedConfigurationCounter.Inc()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// an example for the path: /dubbo/config/dubbo/com.foo.mesh.test.Demo.configurators
		// Usually deleting event don't include the configuration data, so that we should
		// parse the zNode path to decide what is the service name.
		serviceName := utils.FormatToDNS1123(utils.ResolveServiceName(e.Path))
		sc, err := getServiceConfig(defaultNamespace, serviceName, kubeSceh.ctrlManager.GetClient())
		if err != nil {
			klog.Infof("Finding cs with name %s has an error: %v", serviceName, err)
			// TODO Is there a requirement to requeue this event?
			return nil
		}
		return deleteServiceConfig(sc, kubeSceh.ctrlManager.GetClient())
	})
}
