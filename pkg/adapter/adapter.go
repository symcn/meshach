package adapter

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/utils"
	"github.com/mesh-operator/pkg/adapter/configcenter"
	"github.com/mesh-operator/pkg/adapter/events"
	"github.com/mesh-operator/pkg/adapter/handler"
	"github.com/mesh-operator/pkg/adapter/options"
	"github.com/mesh-operator/pkg/adapter/registrycenter"
	"k8s.io/klog"
)

// Adapter ...
type Adapter struct {
	opt            *options.Option
	registryClient events.Registry
	configClient   events.ConfigurationCenter
	eventHandlers  []events.EventHandler
}

// NewAdapter ...
func NewAdapter(opt *options.Option) (*Adapter, error) {
	// TODO init health check handler
	// TODO init router

	// Initializing event handlers
	eventHandlers, err := handler.Init(opt.EventHandlers)
	if err != nil {
		return nil, err
	}

	// Initializing registry client
	registryClient, err := registrycenter.GetRegistry(opt.Registry)
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

// Start an adapter which is used for synchronizing services and instances to kubernetes cluster.
func (a *Adapter) Start(stop <-chan struct{}) error {
	klog.Info("start adapter")
	if err := a.registryClient.Start(); err != nil {
		klog.Errorf("Start a registry center's client has an error: %v", err)
		return err
	}

	if err := a.configClient.Start(); err != nil {
		klog.Errorf("Start a configuration center's client has an error: %v", err)
		return err
	}

	for {
		select {
		case event := <-a.registryClient.Events():
			klog.Infof("Registry events which has been reveived by adapter: %v", event)
			switch event.EventType {
			case events.ServiceAdded:
				uuid := utils.GetUUID()
				klog.Infof("Start to handle event - ADD SERVICE %s", uuid)
				for _, h := range a.eventHandlers {
					h.AddService(event, a.configClient.FindConfiguratorConfig)
				}
				klog.Infof("end handling event - ADD SERVICE %s", uuid)
			case events.ServiceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteService(event)
				}
			case events.ServiceInstanceAdded:
				uuid := utils.GetUUID()
				klog.Infof("Start to handle event - ADD INSTANCE %s, %s", uuid, event.Instance.Host)
				for _, h := range a.eventHandlers {
					h.AddInstance(event, a.configClient.FindConfiguratorConfig)
				}
				klog.Infof("end handling event - ADD INSTANCE %s", uuid)
			case events.ServiceInstancesReplace:
				uuid := utils.GetUUID()
				klog.Infof("Start to handle event - REPLACES INSTANCES %s, %d", uuid, len(event.Instances))
				for _, h := range a.eventHandlers {
					h.ReplaceInstances(event, a.configClient.FindConfiguratorConfig)
				}
				klog.Infof("end handling event - REPLACES INSTANCES %s", uuid)
			case events.ServiceInstanceDeleted:
				uuid := utils.GetUUID()
				klog.Infof("Start to handle event - DELETE INSTANCE %s, %s", uuid, event.Instance.Host)
				for _, h := range a.eventHandlers {
					h.DeleteInstance(event)
				}
				klog.Infof("end handling event - DELETE INSTANCE %s", uuid)
			}
		case ce := <-a.configClient.Events():
			klog.Infof("Configuration events which has been reveived by adapter: %v", ce)
			switch ce.EventType {
			case events.ConfigEntryAdded:
				for _, h := range a.eventHandlers {
					h.AddConfigEntry(ce, a.registryClient.GetCachedService)
				}
			case events.ConfigEntryChanged:
				for _, h := range a.eventHandlers {
					h.ChangeConfigEntry(ce, a.registryClient.GetCachedService)
				}
			case events.ConfigEntryDeleted:
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
