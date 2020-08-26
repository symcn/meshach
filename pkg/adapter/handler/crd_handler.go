package handler

import (
	"context"
	"sync"

	// "github.com/ghodss/yaml"
	v1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultNamespace = "sym-admin"
	clusterName      = ""
	meshConfigName   = "sym-meshconfig"
	csLock           sync.Mutex
	sasLock          sync.Mutex
	scLock           sync.Mutex
)

// KubeEventHandler it used for synchronizing the component which has been send by the adapter client
// to a kubernetes cluster which has an istio controller there.
// It usually uses a CRD group to depict both registered services and instances.

// createConfiguredService
func createConfiguredService(cs *v1.ConfiguredService, c client.Client) error {
	csLock.Lock()
	defer csLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Create(context.Background(), cs)
	klog.Infof("The generation of cs when creating: %d", cs.ObjectMeta.Generation)
	if err != nil {
		klog.Errorf("Creating an cs has an error:%v\n", err)
		return err
	}
	return nil
}

// updateConfiguredService
func updateConfiguredService(cs *v1.ConfiguredService, c client.Client) error {
	csLock.Lock()
	defer csLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Update(context.Background(), cs)
	klog.Infof("The generation of  after updating: %d", cs.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Updating a ConfiguredService has an error: %v\n", err)
		return err
	}

	return nil
}

// getConfiguredService
func getConfiguredService(namespace, name string, c client.Client) (*v1.ConfiguredService, error) {
	cs := &v1.ConfiguredService{}
	err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, cs)
	klog.Infof("The generation of ConfiguredService when getting: %d", cs.ObjectMeta.Generation)
	return cs, err
}

// deleteConfiguredService
func deleteConfiguredService(cs *v1.ConfiguredService, c client.Client) error {
	err := c.Delete(context.Background(), cs)
	klog.Infof("The generation of ConfiguredService when getting: %d", cs.ObjectMeta.Generation)
	return err
}

// createScopedAccessServices
func createScopedAccessServices(sa *v1.ServiceAccessor, c client.Client) error {
	sasLock.Lock()
	defer sasLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Create(context.Background(), sa)
	klog.Infof("The generation of ServiceAccessor when creating: %d", sa.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Creating an ServiceAccessor has an error:%v\n", err)
		return err
	}
	return nil
}

// updateScopedAccessServices
func updateScopedAccessServices(sa *v1.ServiceAccessor, c client.Client) error {
	sasLock.Lock()
	defer sasLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Update(context.Background(), sa)
	klog.Infof("The generation of ServiceAccessor after updating: %d", sa.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Updating a ServiceAccessor has an error: %v\n", err)
		return err
	}

	return nil
}

// getScopedAccessServices
func getScopedAccessServices(namespace, name string, c client.Client) (*v1.ServiceAccessor, error) {
	sa := &v1.ServiceAccessor{}
	err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, sa)
	klog.Infof("The generation of ServiceAccessor when getting: %d", sa.ObjectMeta.Generation)
	return sa, err
}

// createServiceConfig
func createServiceConfig(sc *v1.ServiceConfig, c client.Client) error {
	scLock.Lock()
	defer scLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Create(context.Background(), sc)
	klog.Infof("The generation of ServiceConfig when creating: %d", sc.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Creating an ServiceConfig has an error:%v\n", err)
		return err
	}
	return nil
}

// updateServiceConfig
func updateServiceConfig(sc *v1.ServiceConfig, c client.Client) error {
	scLock.Lock()
	defer scLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Update(context.Background(), sc)
	klog.Infof("The generation of ServiceConfig after updating: %d", sc.ObjectMeta.Generation)
	if err != nil {
		klog.Infof("Updating a ServiceConfig has an error: %v\n", err)
		return err
	}

	return nil
}

// getServiceConfig
func getServiceConfig(namespace, name string, c client.Client) (*v1.ServiceConfig, error) {
	sc := &v1.ServiceConfig{}
	err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, sc)
	klog.Infof("The generation of ServiceConfig when getting: %d", sc.ObjectMeta.Generation)
	return sc, err
}

// deleteServiceConfig
func deleteServiceConfig(sc *v1.ServiceConfig, c client.Client) error {
	err := c.Delete(context.Background(), sc)
	klog.Infof("The generation of ServiceConfig when getting: %d", sc.ObjectMeta.Generation)
	return err
}
