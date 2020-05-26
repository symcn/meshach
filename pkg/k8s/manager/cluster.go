package manager

import (
	"strings"
	"time"

	"github.com/go-logr/logr"
	k8sclient "github.com/mesh-operator/pkg/k8s/client"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ClusterStatusType ...
type ClusterStatusType string

// These are valid status of a cluster.
const (
	// ClusterReady means the cluster is ready to accept workloads.
	ClusterReady ClusterStatusType = "Ready"
	// ClusterOffline means the cluster is temporarily down or not reachable
	ClusterOffline  ClusterStatusType = "Offline"
	ClusterMaintain ClusterStatusType = "Maintaining"
)

var (
	// SyncPeriodTime ...
	SyncPeriodTime = 1 * time.Hour
)

// Cluster ...
type Cluster struct {
	Name          string
	AliasName     string
	RawKubeconfig []byte
	Meta          map[string]string
	RestConfig    *rest.Config
	Client        client.Client
	KubeCli       kubernetes.Interface

	Log             logr.Logger
	Mgr             manager.Manager
	Cache           cache.Cache
	SyncPeriod      time.Duration
	internalStopper chan struct{}

	Status ClusterStatusType
	// Started is true if the Informers has been Started
	Started bool
}

// NewCluster ...
func NewCluster(name string, kubeconfig []byte, log logr.Logger) (*Cluster, error) {
	cluster := &Cluster{
		Name:            name,
		RawKubeconfig:   kubeconfig,
		Log:             log.WithValues("cluster", name),
		SyncPeriod:      SyncPeriodTime,
		internalStopper: make(chan struct{}),
		Started:         false,
	}

	err := cluster.initK8SClients()
	if err != nil {
		return nil, errors.Wrapf(err, "could not re-init k8s clients name:%s", name)
	}

	return cluster, nil
}

// GetName ...
func (c *Cluster) GetName() string {
	return c.Name
}

func (c *Cluster) initK8SClients() error {
	startTime := time.Now()
	cfg, err := k8sclient.NewClientConfig(c.RawKubeconfig)
	if err != nil {
		return errors.Wrapf(err, "could not get rest config name:%s", c.Name)
	}

	klog.V(5).Infof("##### cluster [%s] NewClientConfig. time taken: %v. ", c.Name, time.Since(startTime))
	c.RestConfig = cfg

	kubecli, err := kubernetes.NewForConfig(c.RestConfig)
	if err != nil {
		return errors.Wrapf(err, "could not new kubecli name:%s", c.Name)
	}

	klog.V(5).Infof("##### cluster [%s] NewClientCli. time taken: %v. ", c.Name, time.Since(startTime))
	c.KubeCli = kubecli
	o := manager.Options{
		Scheme:             k8sclient.GetScheme(),
		SyncPeriod:         &c.SyncPeriod,
		MetricsBindAddress: "0",
	}

	mgr, err := manager.New(c.RestConfig, o)
	if err != nil {
		return errors.Wrapf(err, "could not new manager name:%s", c.Name)
	}

	klog.V(5).Infof("##### cluster [%s] NewManagerCli. time taken: %v. ", c.Name, time.Since(startTime))
	c.Mgr = mgr
	c.Client = mgr.GetClient()
	c.Cache = mgr.GetCache()
	return nil
}

func (c *Cluster) healthCheck() bool {
	body, err := c.KubeCli.Discovery().RESTClient().Get().AbsPath("/healthz").Do().Raw()
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to do cluster health check for cluster %q", c.Name))
		c.Status = ClusterOffline
		return false
	}

	if !strings.EqualFold(string(body), "ok") {
		c.Status = ClusterOffline
		return false
	}
	c.Status = ClusterReady
	return true
}

// StartCache ...
func (c *Cluster) StartCache(stopCh <-chan struct{}) {
	if c.Started {
		klog.Infof("cluster name: %s cache Informers is already startd", c.Name)
		return
	}

	klog.Infof("cluster name: %s start cache Informers ", c.Name)
	go func() {
		err := c.Cache.Start(c.internalStopper)
		klog.Warningf("cluster name: %s cache Informers quit end err:%v", c.Name, err)
	}()

	c.Cache.WaitForCacheSync(stopCh)
	c.Started = true
}

// Stop ...
func (c *Cluster) Stop() {
	close(c.internalStopper)
}
