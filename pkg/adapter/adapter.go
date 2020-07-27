package adapter

import (
	"net/http"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/adapter/configcenter"
	"github.com/symcn/mesh-operator/pkg/adapter/constant"
	"github.com/symcn/mesh-operator/pkg/adapter/handler"
	"github.com/symcn/mesh-operator/pkg/adapter/registry"
	"github.com/symcn/mesh-operator/pkg/adapter/types"
	"github.com/symcn/mesh-operator/pkg/option"
	"k8s.io/klog"
)

// Adapter ...
type Adapter struct {
	opt            *option.AdapterOption
	registryClient component.Registry
	configClient   component.ConfigurationCenter
	eventHandlers  []component.EventHandler
}

// NewAdapter ...
func NewAdapter(opt *option.AdapterOption) (*Adapter, error) {
	// TODO init health check handler
	// TODO init router

	// Initializing event handlers
	eventHandlers, err := handler.Init(opt.EventHandlers)
	if err != nil {
		return nil, err
	}

	// Initializing registry client
	registryClient, err := registry.GetRegistry(opt.Registry)
	if err != nil {
		return nil, err
	}

	// Initializing config client
	configClient, err := configcenter.GetRegistry(opt.Configuration)
	if err != nil {
		return nil, err
	}

	adapter := &Adapter{
		opt:            opt,
		registryClient: registryClient,
		configClient:   configClient,
		eventHandlers:  eventHandlers,
	}

	return adapter, nil
}

// Start start an adapter which is used for synchronizing services and instances to kubernetes cluster.
func (a *Adapter) Start(stop <-chan struct{}) error {
	klog.Info("start adapter")

	// Start registry client
	if err := a.registryClient.Start(); err != nil {
		klog.Errorf("Start a registry center's client has an error: %v", err)
		return err
	}
	klog.Info("Registry client started.")

	// Start configuration client
	if err := a.configClient.Start(); err != nil {
		klog.Errorf("Start a configuration center's client has an error: %v", err)
		return err
	}
	klog.Info("Configuration client started.")

	// Prometheus HTTP server
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(constant.PromHTTPPort, nil)
	klog.Infof("Started prometheus HTTP server on port: %s", constant.PromHTTPPort)

	for {
		select {
		case event := <-a.registryClient.ServiceEvents():
			klog.Infof("Registry component which has been received by adapter: %s", event.Service.Name)
			switch event.EventType {
			case types.ServiceAdded:
				uuid := utils.GetUUID()
				klog.Infof("Start to handle event - ADD SERVICE with uuid: %s", uuid)
				for _, h := range a.eventHandlers {
					h.AddService(event, a.configClient.FindConfiguratorConfig)
				}
				klog.Infof("end handling event - ADD SERVICE with uuid: %s", uuid)
			case types.ServiceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteService(event)
				}
			case types.ServiceInstanceAdded:
				uuid := utils.GetUUID()
				klog.Infof("Start to handle event - ADD INSTANCE with uuid: %s, %s", uuid, event.Instance.Host)
				for _, h := range a.eventHandlers {
					h.AddInstance(event, a.configClient.FindConfiguratorConfig)
				}
				klog.Infof("end handling event - ADD INSTANCE with uuid: %s", uuid)
			case types.ServiceInstancesReplace:
				uuid := utils.GetUUID()
				klog.Infof("Start to handle event - REPLACES INSTANCES with uuid: %s, %d", uuid, len(event.Instances))
				for _, h := range a.eventHandlers {
					h.ReplaceInstances(event, a.configClient.FindConfiguratorConfig)
				}
				klog.Infof("end handling event - REPLACES INSTANCES with uuid: %s", uuid)
			case types.ServiceInstanceDeleted:
				uuid := utils.GetUUID()
				klog.Infof("Start to handle event - DELETE INSTANCE with uuid: %s, %s", uuid, event.Instance.Host)
				for _, h := range a.eventHandlers {
					h.DeleteInstance(event)
				}
				klog.Infof("end handling event - DELETE INSTANCE with uuid: %s", uuid)
			}
		case ae := <-a.registryClient.AccessorEvents():
			klog.Infof("Accessor which has been received by adapter: %v", ae)
			switch ae.EventType {
			case types.ServiceInstancesReplace:
				for _, h := range a.eventHandlers {
					h.ReplaceAccessorInstances(ae, a.registryClient.GetCachedScopedMapping)
				}
			default:
				klog.Warningf("The event with %v type has not been support yet.", ae.EventType)
			}
		case ce := <-a.configClient.Events():
			klog.Infof("Configuration component which has been received by adapter: %v", ce)
			switch ce.EventType {
			case types.ConfigEntryAdded:
				for _, h := range a.eventHandlers {
					h.AddConfigEntry(ce, a.registryClient.GetCachedService)
				}
			case types.ConfigEntryChanged:
				for _, h := range a.eventHandlers {
					h.ChangeConfigEntry(ce, a.registryClient.GetCachedService)
				}
			case types.ConfigEntryDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteConfigEntry(ce, a.registryClient.GetCachedService)
				}
			}
		case <-stop:
			a.registryClient.Stop()
			a.configClient.Stop()
			return nil
		}
	}
}
