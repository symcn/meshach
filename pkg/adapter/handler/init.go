package handler

import (
	"fmt"
	"time"

	"github.com/symcn/meshach/pkg/adapter/component"
	"github.com/symcn/meshach/pkg/adapter/convert"
	k8sclient "github.com/symcn/meshach/pkg/k8s/client"
	k8smanager "github.com/symcn/meshach/pkg/k8s/manager"
	"github.com/symcn/meshach/pkg/option"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

// Init the handler initialization
func Init(opt option.EventHandlers) ([]component.EventHandler, error) {
	var eventHandlers []component.EventHandler
	// If this flag has been set as true, it means you want to synchronize all services to a kubernetes cluster.
	if opt.EnableK8s {
		cfg, err := buildRestConfig(opt)
		if err != nil {
			klog.Fatal(err)
		}
		klog.Infof("QPS: %+v, Burst: %+v", cfg.QPS, cfg.Burst)

		kubeCli, err := buildClientSet(cfg)
		if err != nil {
			klog.Fatal(err)
		}

		ctrlMgr, err := buildCtrlManager(cfg)
		if err != nil {
			klog.Fatal(err)
		}

		var eh component.EventHandler
		if opt.IsMultiClusters {
			eh, err = buildMultiClusterEventHandler(opt, ctrlMgr, kubeCli)
		} else {
			eh, err = buildSingleClusterEventHandler(opt, ctrlMgr, kubeCli)
		}
		if err != nil {
			klog.Fatalf("Build cluster event handler err:%v", err)
		}

		eventHandlers = append(eventHandlers, eh)
	}

	if opt.EnableDebugLog {
		logHandler, err := NewLogEventHandler()
		if err != nil {
			return nil, err
		}

		eventHandlers = append(eventHandlers, logHandler)
	}

	return eventHandlers, nil
}

func buildRestConfig(opt option.EventHandlers) (*rest.Config, error) {
	// deciding which kubeconfig we shall use.
	var cfg *rest.Config
	var err error
	if opt.Kubeconfig == "" {
		cfg, err = config.GetConfig()
	} else {
		cfg, err = k8sclient.GetConfigWithContext(opt.Kubeconfig, opt.ConfigContext)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to load the default kubeconfig, err: %v", err)
	}
	return k8sclient.SetDefaultQPS(cfg, opt.QPS, opt.Burst), nil
}

func buildClientSet(cfg *rest.Config) (*kubernetes.Clientset, error) {
	// initializing kube client with the config we has decided
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes Clientse: %v", err)
	}
	return kubeCli, nil
}

func buildCtrlManager(cfg *rest.Config) (ctrlmanager.Manager, error) {
	// initializing control manager with the config
	rp := time.Second * 120
	ctrlMgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
		Scheme:             k8sclient.GetScheme(),
		MetricsBindAddress: "0",
		LeaderElection:     false,
		// Port:               9443,
		SyncPeriod: &rp,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create a manager, err: %v", err)
	}
	return ctrlMgr, nil
}

func buildMultiClusterEventHandler(opt option.EventHandlers, ctrlMgr ctrlmanager.Manager, kubeCli *kubernetes.Clientset) (component.EventHandler, error) {
	// it need to synchronize services to the clusters we found with a configmap which is used for
	// defining these clusters
	masterClient := k8smanager.MasterClient{
		KubeCli: kubeCli,
		Manager: ctrlMgr,
	}
	// initializing multiple k8s cluster manager
	klog.Info("start to initializing multiple cluster managers ... ")
	labels := map[string]string{
		"ClusterOwner": opt.ClusterOwner,
	}
	clustersMgrOpt := k8smanager.DefaultClusterManagerOption(opt.ClusterNamespace, labels)
	if opt.ClusterNamespace != "" {
		clustersMgrOpt.Namespace = opt.ClusterNamespace
	}
	clustersMgr, err := k8smanager.NewClusterManager(masterClient, clustersMgrOpt)
	if err != nil {
		return nil, fmt.Errorf("unable to create a new k8s manager, err: %v", err)
	}

	// initializing the handlers that you decide to utilize
	kubeMultiHandler, err := NewKubeMultiClusterEventHandler(clustersMgr)
	if err != nil {
		return nil, err
	}
	klog.Info("event handler for synchronizing to multiple clusters has been initialized.")
	return kubeMultiHandler, nil
}

func buildSingleClusterEventHandler(opt option.EventHandlers, ctrlMgr ctrlmanager.Manager, kubeCli *kubernetes.Clientset) (component.EventHandler, error) {
	converter := convert.DubboConverter{DefaultNamespace: defaultNamespace}
	kubeSingleHandler, err := NewKubeSingleClusterEventHandler(ctrlMgr, &converter)
	if err != nil {
		return nil, fmt.Errorf("Initializing an event handler for synchronizing to multiple clusters has an error: %v", err)
	}
	klog.Info("event handler for synchronizing to single clusters has been initialized.")
	return kubeSingleHandler, nil
}
