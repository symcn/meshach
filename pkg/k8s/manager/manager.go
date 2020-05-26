package manager

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mesh-operator/pkg/healthcheck"
	"github.com/mesh-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger  = logf.KBLog.WithName("controller")
	timeout <-chan time.Time
)

// key config
const (
	KeyKubeconfig = "kubeconfig.yaml"
	KeyStauts     = "status"
	ClustersAll   = "all"
)

// ClusterManagerOption ...
type ClusterManagerOption struct {
	Namespace     string
	LabelSelector map[string]string
}

// MasterClient ...
type MasterClient struct {
	KubeCli kubernetes.Interface
	manager.Manager
}

// ClusterManager ...
type ClusterManager struct {
	MasterClient
	mu             *sync.RWMutex
	Opt            *ClusterManagerOption
	clusters       []*Cluster
	PreInit        func()
	Started        bool
	ClusterAddInfo chan map[string]string
}

// DefaultClusterManagerOption ...
func DefaultClusterManagerOption(namespace string, ls map[string]string) *ClusterManagerOption {
	return &ClusterManagerOption{
		Namespace:     namespace,
		LabelSelector: ls,
	}
}

func convertToKubeconfig(cm *corev1.ConfigMap) (string, bool) {
	var kubeconfig string
	var ok bool

	if status, ok := cm.Data[KeyStauts]; ok {
		if status == string(ClusterMaintain) {
			klog.Infof("cluster name: %s status: %s", cm.Name, status)
			return "", false
		}
	}

	if kubeconfig, ok = cm.Data[KeyKubeconfig]; !ok {
		return "", false
	}

	return kubeconfig, true
}

// NewManager ...
func NewManager(cli MasterClient, opt *ClusterManagerOption) (*ClusterManager, error) {
	cMgr := &ClusterManager{
		MasterClient:   cli,
		clusters:       make([]*Cluster, 0, 4),
		mu:             &sync.RWMutex{},
		Opt:            opt,
		ClusterAddInfo: make(chan map[string]string),
	}

	err := cMgr.preStart()
	if err != nil {
		klog.Errorf("preStart cluster err: %v", err)
		return nil, err
	}

	cMgr.Started = true
	return cMgr, nil
}

// AddPreInit ...
func (m *ClusterManager) AddPreInit(preInit func()) {
	if m.PreInit != nil {
		klog.Errorf("cluster manager already have preInit func ")
	}

	m.PreInit = preInit
}

// getClusterConfigmap ...
func (m *ClusterManager) getClusterConfigmap() ([]*corev1.ConfigMap, error) {
	cms := make([]*corev1.ConfigMap, 0, 4)
	if m.Started {
		configmaps := &corev1.ConfigMapList{}
		err := m.Manager.GetClient().List(
			context.Background(),
			configmaps,
			&client.ListOptions{
				LabelSelector: labels.SelectorFromSet(m.Opt.LabelSelector),
				Namespace:     m.Opt.Namespace,
			},
		)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, err
			}

			klog.Errorf("failed to ConfigMapList ls :%v, err: %v", m.Opt.LabelSelector, err)
			return nil, err
		}
		for i := range configmaps.Items {
			cms = append(cms, &configmaps.Items[i])
		}

	} else {
		cmList, err := m.KubeCli.CoreV1().ConfigMaps(m.Opt.Namespace).List(
			metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(m.Opt.LabelSelector).String(),
			},
		)
		if err != nil {
			klog.Errorf("unable to get cluster configmap err: %v", err)
		}
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, err
			}

			klog.Errorf("failed to ConfigMapList ls :%v, err: %v", m.Opt.LabelSelector, err)
			return nil, err
		}

		for i := range cmList.Items {
			cms = append(cms, &cmList.Items[i])
		}
	}

	sort.Slice(cms, func(i, j int) bool {
		return cms[i].Name < cms[j].Name
	})
	return cms, nil
}

// GetAll get all cluster
func (m *ClusterManager) GetAll(name ...string) []*Cluster {
	m.mu.RLock()
	defer m.mu.RUnlock()

	isAll := true
	var ObserveName string
	if len(name) > 0 {
		if name[0] != ClustersAll {
			ObserveName = utils.FormatClusterName(name[0])
			isAll = false
		}
	}

	list := make([]*Cluster, 0, 4)
	for _, c := range m.clusters {
		if c.Status == ClusterOffline {
			continue
		}

		if isAll {
			list = append(list, c)
		} else if ObserveName != "" && ObserveName == c.Name {
			list = append(list, c)
			break
		}
	}

	return list
}

// Add ...
func (m *ClusterManager) Add(cluster *Cluster) error {
	if _, err := m.Get(cluster.Name); err == nil {
		return fmt.Errorf("cluster name: %s is already add to manager", cluster.Name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.clusters = append(m.clusters, cluster)
	sort.Slice(m.clusters, func(i int, j int) bool {
		return m.clusters[i].Name > m.clusters[j].Name
	})

	return nil
}

// GetClusterIndex ...
func (m *ClusterManager) GetClusterIndex(name string) (int, bool) {
	for i, r := range m.clusters {
		if r.Name == name {
			return i, true
		}
	}
	return 0, false
}

// Delete ...
func (m *ClusterManager) Delete(name string) error {
	if name == "" {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.clusters) == 0 {
		klog.Errorf("clusters list is empty, nothing to delete")
		return nil
	}

	index, ok := m.GetClusterIndex(name)
	if !ok {
		klog.Warningf("cluster:%s  is not found in the registries list, nothing to delete", name)
		return nil
	}

	clusters := m.clusters
	clusters = append(clusters[:index], clusters[index+1:]...)
	m.clusters = clusters
	klog.Infof("cluster: the cluster %s has been deleted.", name)
	return nil
}

// Get ...
func (m *ClusterManager) Get(name string) (*Cluster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name = utils.FormatClusterName(name)

	if name == "" || name == "all" {
		return nil, fmt.Errorf("single query not support: %s ", name)
	}

	var findCluster *Cluster
	for _, c := range m.clusters {
		if name == c.Name {
			findCluster = c
			break
		}
	}
	if findCluster == nil {
		return nil, fmt.Errorf("cluster: %s not found", name)
	}

	if findCluster.Status == ClusterOffline {
		return nil, fmt.Errorf("cluster: %s found, but offline", name)
	}

	return findCluster, nil
}

func (m *ClusterManager) preStart() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	configmaps, err := m.getClusterConfigmap()
	if err != nil {
		klog.Errorf("unable to get cluster configmap err: %v", err)
		return err
	}

	klog.Infof("find %d cluster from namespace: %s ls: %v ", len(configmaps), m.Opt.Namespace, m.Opt.LabelSelector)
	for _, cm := range configmaps {
		kubeconfig, ok := convertToKubeconfig(cm)
		if !ok {
			klog.Errorf("cluster: %s convertToKubeconfig err: %v", cm.Name, err)
			continue
		}

		c, err := NewCluster(cm.Name, []byte(kubeconfig), logger)
		if err != nil {
			klog.Errorf("cluster: %s new client err: %v", cm.Name, err)
			continue
		}

		if !c.healthCheck() {
			klog.Errorf("cluster: %s check offline", cm.Name)
			continue
		}

		// add field event type index must before cache start
		if err := c.Mgr.GetFieldIndexer().IndexField(
			&corev1.Event{},
			"type",
			func(rawObj runtime.Object) []string {
				event := rawObj.(*corev1.Event)
				return []string{event.Type}
			}); err != nil {
			klog.Warningf("cluster[%s] add field index event type, err: %#v", c.Name, err)
		} else {
			klog.Infof("cluster[%s] add field index envet type successfully", c.Name)
		}

		c.StartCache(ctx.Done())
		m.Add(c)
		klog.Infof("add cluster name: %s ", cm.Name)
	}

	return nil
}

func (m *ClusterManager) cluterCheck() {
	klog.V(5).Info("cluster configmap check.")
	configmaps, err := m.getClusterConfigmap()
	if err != nil {
		klog.Errorf("unable to get cluster configmap err: %v", err)
		return
	}
	// If not leader, informer resource cache is empty
	if len(configmaps) == 0 {
		return
	}

	expectList := map[string]string{}
	for _, cm := range configmaps {
		config, _ := convertToKubeconfig(cm)
		expectList[cm.Name] = config
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	currentList := map[string]*Cluster{}
	for _, c := range m.clusters {
		currentList[c.Name] = c
	}

	newClusters := make([]*Cluster, 0, 4)
	delList := currentList
	addList := map[string]string{}
	for name, conf := range expectList {
		cls, ok := currentList[name]
		if !ok {
			addList[name] = conf
			continue
		}
		delete(delList, name)
		if strings.EqualFold(conf, string(cls.RawKubeconfig)) {
			newClusters = append(newClusters, cls)
			continue
		}
		if conf == "" {
			cls.Status = ClusterMaintain
			newClusters = append(newClusters, cls)
			continue
		}
		delList[name] = cls
		addList[name] = conf
	}

	if len(delList) == 0 && len(addList) == 0 {
		return
	}

	healthHandler := healthcheck.GetHealthHandler()
	for _, cls := range delList {
		klog.Infof("delete cluster:%s connect", cls.Name)
		healthHandler.RemoveLivenessCheck(cls.Name)
		cls.Stop()
	}

	for name, conf := range addList {
		klog.Infof("create cluster:%s connect", name)
		newcls, err := m.addNewClusters(name, conf)
		if err != nil {
			return
		}
		newClusters = append(newClusters, newcls)
	}

	sort.Slice(newClusters, func(i, j int) bool {
		return newClusters[i].Name > newClusters[j].Name
	})
	m.clusters = newClusters

	timeout = time.After(time.Second * 5)
	select {
	case m.ClusterAddInfo <- addList:
	case <-timeout:
		klog.Infof("add cluster info timeout:%+v", addList)
	}
}

func (m *ClusterManager) addNewClusters(name string, kubeconfig string) (*Cluster, error) {
	// config change
	nc, err := NewCluster(name, []byte(kubeconfig), logger)
	if err != nil {
		klog.Errorf("cluster: %s new client err: %v", name, err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	nc.StartCache(ctx.Done())
	return nc, nil
}

// Start timer check cluster health
func (m *ClusterManager) Start(stopCh <-chan struct{}) error {
	if m.PreInit != nil {
		m.PreInit()
	}
	klog.Info("start cluster manager check loop ... ")
	wait.Until(m.cluterCheck, time.Minute, stopCh)

	klog.Info("close manager info")
	m.stop()
	return nil
}

func (m *ClusterManager) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cluster := range m.clusters {
		cluster.Stop()
	}
	close(m.ClusterAddInfo)
}
