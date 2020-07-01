package handler

import (
	"github.com/mesh-operator/pkg/adapter/component"
	"github.com/mesh-operator/pkg/adapter/options"
	k8sclient "github.com/mesh-operator/pkg/k8s/client"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"k8s.io/sample-controller/pkg/signals"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
)

// Init the handler initialization
func Init(opt options.EventHandlers) ([]component.EventHandler, error) {
	var eventHandlers []component.EventHandler
	// If this flag has been set as true, it means you want to synchronize all services to a kubernetes cluster.
	if opt.EnableK8s {
		var cfg *rest.Config
		var err error
		if opt.Kubeconfig == "" {
			cfg, err = config.GetConfig()
		} else {
			cfg, err = k8sclient.GetConfigWithContext(opt.Kubeconfig, opt.ConfigContext)
		}
		if err != nil {
			klog.Fatalf("unable to load the default kubeconfig, err: %v", err)

		}

		rp := time.Second * 120
		mgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
			Scheme:             k8sclient.GetScheme(),
			MetricsBindAddress: "0",
			LeaderElection:     false,
			// Port:               9443,
			SyncPeriod: &rp,
		})
		if err != nil {
			klog.Fatalf("unable to create a manager, err: %v", err)
		}

		kubeCli, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			klog.Fatalf("failed to get kubernetes Clientset: %v", err)
		}
		masterClient := k8smanager.MasterClient{
			KubeCli: kubeCli,
			Manager: mgr,
		}

		// initializing multiple k8s cluster manager
		klog.Info("start to initializing multiple cluster managers ... ")
		labels := map[string]string{
			"ClusterOwner": opt.ClusterOwner,
		}
		mgrOpt := k8smanager.DefaultClusterManagerOption(opt.ClusterNamespace, labels)
		if opt.ClusterNamespace != "" {
			mgrOpt.Namespace = opt.ClusterNamespace
		}
		k8sMgr, err := k8smanager.NewManager(masterClient, mgrOpt)
		if err != nil {
			klog.Fatalf("unable to create a new k8s manager, err: %v", err)
		}

		stopCh := signals.SetupSignalHandler()
		// mgr.Add(adp)
		// mgr.Add(adp.K8sMgr)

		klog.Info("starting manager")
		go func() {
			if err := mgr.Start(stopCh); err != nil {
				klog.Fatalf("problem start running manager err: %v", err)
			}
		}()

		// add kube v2 handler
		//kubeh, err := NewKubeV2EventHandler(k8sMgr)
		//if err != nil {
		//	return nil, err
		//}
		// add kube v2 handler
		kubeh, err := NewKubeV3EventHandler(k8sMgr)
		if err != nil {
			return nil, err
		}

		eventHandlers = append(eventHandlers, kubeh)
	}

	if opt.EnableDebugLog {
		logh, err := NewLogEventHandler()
		if err != nil {
			return nil, err
		}

		eventHandlers = append(eventHandlers, logh)
	}

	return eventHandlers, nil
}
