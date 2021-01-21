package handler

import (
	"fmt"
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/symcn/meshach/api/v1alpha1"
	v1 "github.com/symcn/meshach/api/v1alpha1"
	"github.com/symcn/meshach/pkg/adapter/component"
	"github.com/symcn/meshach/pkg/adapter/metrics"
	"github.com/symcn/meshach/pkg/adapter/types"
	"github.com/symcn/meshach/pkg/utils"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

// KubeSingleClusterEventHandler ...
type KubeSingleClusterEventHandler struct {
	ctrlMgr   ctrlmanager.Manager
	converter component.Converter
}

// NewKubeSingleClusterEventHandler ...
func NewKubeSingleClusterEventHandler(ctrlMgr ctrlmanager.Manager, converter component.Converter) (component.EventHandler, error) {

	// add informer cache
	ctrlMgr.GetCache().GetInformer(&v1alpha1.ConfiguredService{})
	ctrlMgr.GetCache().GetInformer(&v1alpha1.ServiceConfig{})
	ctrlMgr.GetCache().GetInformer(&v1alpha1.ServiceAccessor{})
	// ctrlMgr.GetCache().GetInformer(&v1alpha1.ConfiguredServiceList{})
	// ctrlMgr.GetCache().GetInformer(&v1alpha1.ServiceConfigList{})
	// ctrlMgr.GetCache().GetInformer(&v1alpha1.ServiceAccessorList{})

	klog.Info("starting the control manager")
	stopCh := utils.SetupSignalHandler()
	go func() {
		if err := ctrlMgr.Start(stopCh); err != nil {
			klog.Fatalf("start to run the controllers manager, err: %v", err)
		}
	}()
	for !ctrlMgr.GetCache().WaitForCacheSync(stopCh) {
		klog.Warningf("Waiting for caching objects to informer")
		time.Sleep(1 * time.Second)
	}
	klog.Infof("caching objects to informer is successful")

	return &KubeSingleClusterEventHandler{
		ctrlMgr:   ctrlMgr,
		converter: converter,
	}, nil
}

// AddService ...
func (h *KubeSingleClusterEventHandler) AddService(e *types.ServiceEvent) {
	klog.V(6).Infof("Adding a service has not been implemented yet by single clusters handler.")
}

// AddInstance ...
func (h *KubeSingleClusterEventHandler) AddInstance(e *types.ServiceEvent) {
	klog.V(6).Infof("Adding an instance has not been implemented yet by single clusters handler.")
}

// ReplaceInstances ...
func (h *KubeSingleClusterEventHandler) ReplaceInstances(event *types.ServiceEvent) {
	klog.V(6).Infof("event handler for a single cluster: Replacing these instances(size: %d)\n", len(event.Instances))

	metrics.SynchronizedServiceCounter.Inc()
	metrics.SynchronizedInstanceCounter.Add(float64(len(event.Instances)))
	timer := prometheus.NewTimer(metrics.ReplacingInstancesHistogram)
	defer timer.ObserveDuration()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Convert a service event that notified by zookeeper to a Service CRD
		cs := h.converter.ToConfiguredService(event)

		// loading cs CR from k8s cluster
		foundCs, err := getConfiguredService(cs.Namespace, cs.Name, h.ctrlMgr.GetClient())
		if err != nil {
			if errors.IsNotFound(err) {
				return createConfiguredService(cs, h.ctrlMgr.GetClient())
			}
			// klog.Errorf("Can not find an existed cs {%s/%s} CR: %v, then create such cs instead.", cs.Namespace, cs.Name, err)
			klog.Errorf("Get a ConfiguredService {%s/%s} has an error:%v", cs.Namespace, cs.Name, err)
			return err
		}
		foundCs.Spec = cs.Spec
		return updateConfiguredService(foundCs, h.ctrlMgr.GetClient())
	})

}

// DeleteService ...
func (h *KubeSingleClusterEventHandler) DeleteService(event *types.ServiceEvent) {
	klog.V(6).Infof("event handler for a single cluster: Deleting a service: %v", event.Service)
	metrics.DeletedServiceCounter.Inc()
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := deleteConfiguredService(&v1.ConfiguredService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.FormatToDNS1123(event.Service.Name),
				Namespace: defaultNamespace,
			},
		}, h.ctrlMgr.GetClient())
		return err
	})

}

// DeleteInstance ...
func (h *KubeSingleClusterEventHandler) DeleteInstance(e *types.ServiceEvent) {
	klog.V(6).Infof("Deleting an instance has not been implemented yet by single clusters handler.")
}

// ReplaceAccessorInstances ...
func (h *KubeSingleClusterEventHandler) ReplaceAccessorInstances(
	e *types.ServiceEvent,
	getScopedServices func(s string) map[string]struct{}) {

	// klog.V(6).Infof("event handler for a single cluster: replacing the accessor's instances: %s", e.Service.Name)
	metrics.ReplacedAccessorInstancesCounter.Inc()

	instances := e.Instances
	changedScopes := make(map[string]struct{})
	for _, ins := range instances {
		if ins != nil {
			sk, ok := ins.Labels["app"]
			if ok {
				changedScopes[sk] = struct{}{}
			} else {
				klog.V(6).Infof("Could't find label [app]'s value, instance: %s, skip updating the associated scope", ins.Host)
			}
		}
	}

	for changedScope := range changedScopes {
		scopedMapping := getScopedServices(changedScope)
		var accessedServices []string
		for s := range scopedMapping {
			accessedServices = append(accessedServices, s)
		}
		sort.Strings(accessedServices)

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
			foundSas, err := getScopedAccessServices(sas.Namespace, sas.Name, h.ctrlMgr.GetClient())
			if err != nil {
				klog.Warningf("Can not find an existed asm {%s/%s} CR: %v, then create a new one instead.", sas.Namespace, sas.Name, err)
				if errors.IsNotFound(err) {
					return createScopedAccessServices(sas, h.ctrlMgr.GetClient())
				}
				return err
			}
			if equality.Semantic.DeepEqual(foundSas.Spec, sas.Spec) {
				// not changed, return directly
				return nil
			}
			foundSas.Spec = sas.Spec
			sas.Spec.DeepCopyInto(&foundSas.Spec)
			return updateScopedAccessServices(foundSas, h.ctrlMgr.GetClient())
		})
	}
}

// AddConfigEntry ...
func (h *KubeSingleClusterEventHandler) AddConfigEntry(e *types.ConfigEvent) {
	klog.V(6).Infof("event handler for a single cluster: adding a configuration: %s", e.Path)

	metrics.AddedConfigurationCounter.Inc()
	// Adding a new configuration for a service is same as changing it.
	h.ChangeConfigEntry(e)
}

// ChangeConfigEntry ...
func (h *KubeSingleClusterEventHandler) ChangeConfigEntry(e *types.ConfigEvent) {
	klog.V(6).Infof("event handler for a single cluster: change a configuration %s", e.Path)

	metrics.ChangedConfigurationCounter.Inc()
	timer := prometheus.NewTimer(metrics.ChangingConfigurationHistogram)
	defer timer.ObserveDuration()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		sc := h.converter.ToServiceConfig(e.ConfigEntry)
		if sc == nil {
			return fmt.Errorf("config[%s]'s event has an empty config key, skip it", e.Path)
		}

		foundSc, err := getServiceConfig(defaultNamespace, utils.FormatToDNS1123(sc.Name), h.ctrlMgr.GetClient())
		if err != nil {
			klog.Errorf("Finding ServiceConfig with name %s has an error: %v", sc.Name, err)
			if errors.IsNotFound(err) {
				klog.Errorf("Could not find the ConfiguredService %s, then create it. %v", sc.Name, err)
				return createServiceConfig(sc, h.ctrlMgr.GetClient())
			}
			// TODO Is there a requirement to requeue this event?
			return err
		}

		foundSc.Spec = sc.Spec
		return updateServiceConfig(foundSc, h.ctrlMgr.GetClient())
	})

}

// DeleteConfigEntry ...
func (h *KubeSingleClusterEventHandler) DeleteConfigEntry(e *types.ConfigEvent) {
	klog.V(6).Infof("event handler for a single cluster: delete a configuration %s", e.Path)

	metrics.DeletedConfigurationCounter.Inc()

	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// an example for the path: /dubbo/config/dubbo/com.foo.mesh.test.Demo.configurators
		// Usually deleting event don't include the configuration data, so that we should
		// parse the zNode path to decide what is the service name.
		serviceName := utils.FormatToDNS1123(utils.ResolveServiceName(e.Path))
		sc, err := getServiceConfig(defaultNamespace, serviceName, h.ctrlMgr.GetClient())
		if err != nil {
			klog.Errorf("Finding cs with name %s has an error: %v", serviceName, err)
			// TODO Is there a requirement to requeue this event?
			return err
		}
		return deleteServiceConfig(sc, h.ctrlMgr.GetClient())
	})
}
