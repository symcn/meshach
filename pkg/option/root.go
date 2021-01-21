package option

import (
	"github.com/pkg/errors"
	k8sclient "github.com/symcn/meshach/pkg/k8s/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

// RootOption ...
type RootOption struct {
	Kubeconfig       string
	ConfigContext    string
	Namespace        string
	DefaultNamespace string
	DevelopmentMode  bool
	QPS              float32
	Burst            int
}

// DefaultRootOption ...
func DefaultRootOption() *RootOption {
	return &RootOption{
		Namespace:       corev1.NamespaceAll,
		DevelopmentMode: true,
		QPS:             120,
		Burst:           120,
	}
}

// GetK8sConfig ...
func (r *RootOption) GetK8sConfig() (*rest.Config, error) {
	config, err := k8sclient.GetConfigWithContext(r.Kubeconfig, r.ConfigContext)
	if err != nil {
		return nil, errors.Wrap(err, "could not get k8s config")
	}

	return k8sclient.SetDefaultQPS(config, r.QPS, r.Burst), nil
}

// GetKubeInterface ...
func (r *RootOption) GetKubeInterface() (kubernetes.Interface, error) {
	cfg, err := r.GetK8sConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not get k8s config")
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}

	return kubeCli, nil
}

// GetKubeInterfaceOrDie ...
func (r *RootOption) GetKubeInterfaceOrDie() kubernetes.Interface {
	kubeCli, err := r.GetKubeInterface()
	if err != nil {
		klog.Fatalf("unable to get kube interface err: %v", err)
	}

	return kubeCli
}
