package adapter

import (
	"fmt"
	"time"

	"github.com/mesh-operator/pkg/adapter/events"
	"github.com/mesh-operator/pkg/adapter/handler"
	"github.com/mesh-operator/pkg/adapter/zookeeper"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
	"github.com/samuel/go-zookeeper/zk"
	"k8s.io/klog"
)

// Adapter ...
type Adapter struct {
	registryClient RegistryClient
	configClient   ConfigurationCenterClient
	eventHandlers  []EventHandler
	opt            *Option
	K8sMgr         *k8smanager.ClusterManager
}

// Option ...
type Option struct {
	Address          []string
	Timeout          int64
	ClusterOwner     string
	ClusterNamespace string
	MasterCli        k8smanager.MasterClient
}

// DefaultOption ...
func DefaultOption() *Option {
	return &Option{
		Timeout:          15,
		ClusterOwner:     "sym-admin",
		ClusterNamespace: "sym-admin",
	}
}

// NewAdapter ...
func NewAdapter(opt *Option) (*Adapter, error) {
	// TODO init health check handler
	// TODO init router

	// initializing multiple k8s cluster manager
	klog.Info("start to initializing multiple cluster managers ... ")
	labels := map[string]string{
		"ClusterOwner": opt.ClusterOwner,
	}
	mgrOpt := k8smanager.DefaultClusterManagerOption(opt.ClusterNamespace, labels)
	if opt.ClusterNamespace != "" {
		mgrOpt.Namespace = opt.ClusterNamespace
	}
	k8sMgr, err := k8smanager.NewManager(opt.MasterCli, mgrOpt)
	if err != nil {
		klog.Fatalf("unable to create a new k8s manager, err: %v", err)
	}
	//k8sMgr.GetAll()

	// Initializing the registry client that you want to use.
	rConn, _, err := zk.Connect(opt.Address, time.Duration(opt.Timeout)*time.Second)
	if err != nil {
		klog.Errorf("Initializing a registry client has an error: %v\n", err)
		return nil, err
	}
	registryClient := zookeeper.NewRegistryClient(rConn)

	// Initializing the a client to connect to configuration center
	cConn, _, err := zk.Connect(opt.Address, time.Duration(opt.Timeout)*time.Second)
	if err != nil {
		klog.Errorf("Initializing a registry client has an error: %v\n", err)
		return nil, err
	}
	configClient := zookeeper.NewConfigClient(cConn)

	// initializing adapter
	var eventHandlers []EventHandler
	//eventHandlers = append(eventHandlers, &SimpleEventHandler{Name: "simpleHandler"})
	//eventHandlers = append(eventHandlers, &handler.LogEventHandler{Name: "logging events."})
	//eventHandlers = append(eventHandlers, &handler.KubeEventHandler{K8sMgr: k8sMgr})
	kubev2Handler := &handler.KubeV2EventHandler{K8sMgr: k8sMgr}
	kubev2Handler.Init()
	eventHandlers = append(eventHandlers, kubev2Handler)

	adapter := &Adapter{
		opt:            opt,
		K8sMgr:         k8sMgr,
		registryClient: registryClient,
		configClient:   configClient,
		eventHandlers:  eventHandlers,
	}

	return adapter, nil
}

// Start an adapter which is used for synchronizing services and instances to kubernetes cluster.
func (a *Adapter) Start(stop <-chan struct{}) error {
	if err := a.registryClient.Start(); err != nil {
		fmt.Printf("Start a registry center's client has an error: %v\n", err)
		return err
	}

	if err := a.configClient.Start(); err != nil {
		fmt.Printf("Start a configuration center's client has an error: %v\n", err)
		return err
	}

	for {
		select {
		case event := <-a.registryClient.Events():
			switch event.EventType {
			case events.ServiceAdded:
				for _, h := range a.eventHandlers {
					h.AddService(event, a.configClient.FindConfiguratorConfig)
				}
			case events.ServiceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteService(event)
				}
			case events.ServiceInstanceAdded:
				for _, h := range a.eventHandlers {
					h.AddInstance(event, a.configClient.FindConfiguratorConfig)
				}
			case events.ServiceInstanceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteInstance(event)
				}
			}
		case ce := <-a.configClient.Events():
			fmt.Printf("%v\n", ce)
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
			return nil
		}
	}

}
