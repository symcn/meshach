package adapter

import (
	"github.com/symcn/mesh-operator/pkg/adapter/accelerate"
	// terminal: $ go tool pprof http://localhost:6066/debug/pprof/{heap,allocs,block,cmdline,goroutine,mutex,profile,threadcreate,trace}
	// web:
	// 1、http://localhost:8081/ui
	// 2、http://localhost:6066/debug/charts
	// 3、http://localhost:6066/debug/pprof
	_ "net/http/pprof"

	"net/http"

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
	go http.ListenAndServe(constant.HTTPPort, nil)
	klog.Infof("Started HTTP server for providing some features such as exposing metrics and pprof on port: %s", constant.HTTPPort)

	// Using an accelerator can improve the efficient of interacting with k8s api server
	accelerator := accelerate.NewAccelerator(a.opt.EventHandlers.AcceleratorSize, stop)
	klog.Infof("Initializing an accelerator with a channels size %d", a.opt.EventHandlers.AcceleratorSize)

	for {
		select {
		case se := <-a.registryClient.ServiceEvents():
			klog.V(6).Infof("Registry component which has been received by adapter: %s, type: %v", se.Service.Name, se.EventType)
			switch se.EventType {
			case types.ServiceAdded:
				for _, h := range a.eventHandlers {
					h.AddService(se)
				}
			case types.ServiceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteService(se)
				}
			case types.ServiceInstanceAdded:
				for _, h := range a.eventHandlers {
					h.AddInstance(se)
				}
			case types.ServiceInstancesReplace:
				for _, h := range a.eventHandlers {
					handler := h
					accelerator.Accelerate(func() {
						handler.ReplaceInstances(se)
					}, se.Service.Name)
				}
			case types.ServiceInstanceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteInstance(se)
				}
			}
		case ae := <-a.registryClient.AccessorEvents():
			klog.V(6).Infof("Accessor which has been received by adapter: %v", ae)
			switch ae.EventType {
			case types.ServiceInstancesReplace:
				for _, h := range a.eventHandlers {
					handler := h
					accelerator.Accelerate(func() {
						handler.ReplaceAccessorInstances(ae, a.registryClient.GetCachedScopedMapping)
					}, ae.Service.Name)
				}
			default:
				klog.Warningf("The event with %v type has not been support yet.", ae.EventType)
			}
		case ce := <-a.configClient.Events():
			klog.V(6).Infof("Configuration component which has been received by adapter: %v", ce)
			switch ce.EventType {
			case types.ConfigEntryAdded:
				for _, h := range a.eventHandlers {
					h.AddConfigEntry(ce)
				}
			case types.ConfigEntryChanged:
				for _, h := range a.eventHandlers {
					h.ChangeConfigEntry(ce)
				}
			case types.ConfigEntryDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteConfigEntry(ce)
				}
			}
		case <-stop:
			a.registryClient.Stop()
			a.configClient.Stop()
			return nil
		}
	}
}
