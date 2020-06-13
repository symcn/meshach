package adapter

import (
	"fmt"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
	"github.com/samuel/go-zookeeper/zk"
	"k8s.io/klog"
	"time"
)

// Adapter ...
type Adapter struct {
	registryClient RegistryClient
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

	// init multi k8s cluster manager
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

	// initializing adapter
	_, _, err = zk.Connect(opt.Address, time.Duration(opt.Timeout)*time.Second)
	if err != nil {
		return nil, err
	}
	//zkClient := zookeeper.NewClient(conn)

	var eventHandlers []EventHandler
	eventHandlers = append(eventHandlers, &SimpleEventHandler{Name: "simpleHandler"})
	eventHandlers = append(eventHandlers, &CRDEventHandler{k8sMgr: k8sMgr})

	adapter := &Adapter{
		registryClient: nil,
		eventHandlers:  eventHandlers,
		opt:            opt,
		K8sMgr:         k8sMgr,
	}

	return adapter, nil
}

// Start an adapter which is used for synchronizing services and instances to kubernetes cluster.
func (a *Adapter) Start(stop <-chan struct{}) error {
	if err := a.registryClient.Start(); err != nil {
		fmt.Printf("Start zookeeper client has an error %v\n", err)
		return err
	}

	for {
		select {
		case event := <-a.registryClient.Events():
			switch event.EventType {
			case ServiceAdded:
				for _, h := range a.eventHandlers {
					h.AddService(event)
				}
			case ServiceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteService(event)
				}
			case ServiceInstanceAdded:
				for _, h := range a.eventHandlers {
					h.AddInstance(event)
				}
			case ServiceInstanceDeleted:
				for _, h := range a.eventHandlers {
					h.DeleteInstance(event)
				}
			}
		case <-stop:
			a.registryClient.Stop()
			return nil
		}
	}

}
