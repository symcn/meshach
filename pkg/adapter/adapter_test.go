package adapter

import (
	"strings"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/symcn/mesh-operator/pkg/option"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"

	_ "github.com/symcn/mesh-operator/pkg/adapter/configcenter/zk"
	_ "github.com/symcn/mesh-operator/pkg/adapter/registry/zk"
)

func Test_Start(t *testing.T) {
	// if we didn't find any namespaces configuration, for example, namespaces is a empty string,
	// we'll use all-namespaces instead.
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		//log.Error(err, "Failed to get watch namespace")
		//os.Exit(1)
	}

	// Get a config to talk to the apiserver
	//cfg, err := config.GetConfig()
	//if err != nil {
	//	os.Exit(1)
	//}

	// Set default manager option
	opts := ctrlmanager.Options{
		Namespace: namespace,
		//MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More Info: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if strings.Contains(namespace, ",") {
		opts.Namespace = ""
		opts.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
	}

	//// Create a new manager to provide shared dependencies and start components
	//mgr, err := ctrlmanager.New(cfg, opts)
	//if err != nil {
	//	os.Exit(1)
	//}

	opt := option.DefaultAdapterOption()

	//kubeCli, err := kubernetes.NewForConfig(cfg)
	//opt.MasterCli = k8smanager.MasterClient{
	//	KubeCli: kubeCli,
	//	Manager: mgr,
	//}
	//
	//nodes, err := kubeCli.CoreV1().Nodes().List(metav1.ListOptions{})
	//fmt.Printf("nodes : %v\n", nodes.Items[0].Name)
	//clusters := k8smanager.ClusterManager.GetAll("")
	//fmt.Sprintf("Start an adaptor has an error: %v\n", clusters)

	adapter, err := NewAdapter(opt)
	if err != nil {
		klog.Errorf("Create an adapter has an error: %v\n", err)
		return
	}

	stop := make(chan struct{})
	err = adapter.Start(stop)
	if err != nil {
		klog.Errorf("Start an adaptor has an error: %v\n", err)
		return
	}

	time.Sleep(30 * time.Minute)
	stop <- struct{}{}
	time.Sleep(5 * time.Second)
}
