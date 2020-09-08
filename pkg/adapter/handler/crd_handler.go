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
	if err != nil {
		klog.Errorf("Creating a ConfiguredService {%s/%s} has an error: %v", cs.Namespace, cs.Name, err)
		return err
	}
	klog.V(6).Infof("Create ConfiguredService {%s/%s} success.", cs.Namespace, cs.Name)
	return nil
}

// updateConfiguredService
func updateConfiguredService(cs *v1.ConfiguredService, c client.Client) error {
	csLock.Lock()
	defer csLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Update(context.Background(), cs)
	if err != nil {
		klog.Errorf("Updating a ConfiguredService {%s/%s} has an error: %v", cs.Namespace, cs.Name, err)
		return err
	}
	klog.V(6).Infof("Update ConfiguredService {%s/%s} success.", cs.Namespace, cs.Name)

	return nil
}

// getConfiguredService
func getConfiguredService(namespace, name string, c client.Client) (*v1.ConfiguredService, error) {
	cs := &v1.ConfiguredService{}
	err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, cs)
	if err != nil {
		klog.Errorf("Get a ConfiguredService {%s/%s} has an error:%v", namespace, name, err)
		return nil, err
	}

	klog.V(6).Infof("Get ConfiguredService {%s/%s} success.", cs.Namespace, cs.Name)
	return cs, nil
}

// deleteConfiguredService
func deleteConfiguredService(cs *v1.ConfiguredService, c client.Client) error {
	err := c.Delete(context.Background(), cs)
	if err != nil {
		klog.Errorf("Delete a ConfiguredService {%s/%s} has an error: %v", cs.Namespace, cs.Name, err)
		return err
	}

	klog.V(6).Infof("Delete ConfiguredService {%s/%s} success.", cs.Namespace, cs.Name)
	return nil
}

// createScopedAccessServices
func createScopedAccessServices(sa *v1.ServiceAccessor, c client.Client) error {
	sasLock.Lock()
	defer sasLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Create(context.Background(), sa)
	if err != nil {
		klog.Errorf("Creating a ServiceAccessor {%s/%s} has an error:%v", sa.Namespace, sa.Name, err)
		return err
	}

	klog.V(6).Infof("Create ServiceAccessor {%s/%s} success.", sa.Namespace, sa.Name)
	return nil
}

// updateScopedAccessServices
func updateScopedAccessServices(sa *v1.ServiceAccessor, c client.Client) error {
	sasLock.Lock()
	defer sasLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Update(context.Background(), sa)
	if err != nil {
		klog.Errorf("Updating a ServiceAccessor {%s/%s} has an error: %v", sa.Namespace, sa.Name, err)
		return err
	}

	klog.V(6).Infof("Updating ServiceAccessor {%s/%s} success.", sa.Namespace, sa.Name)
	return nil
}

// getScopedAccessServices
func getScopedAccessServices(namespace, name string, c client.Client) (*v1.ServiceAccessor, error) {
	sa := &v1.ServiceAccessor{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, sa); err != nil {
		klog.Errorf("Get a ServiceAccessor {%s/%s} has an error: %v", namespace, name, err)
		return nil, err
	}

	klog.V(6).Infof("Get ServiceAccessor {%s/%s} success.", namespace, name)
	return sa, nil
}

// createServiceConfig
func createServiceConfig(sc *v1.ServiceConfig, c client.Client) error {
	scLock.Lock()
	defer scLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Create(context.Background(), sc)
	if err != nil {
		klog.Errorf("Create a ServiceConfig {%s/%s} has an error: %v", sc.Namespace, sc.Name, err)
		return err
	}

	klog.V(6).Infof("Create ServiceConfig {%s/%s} success.", sc.Namespace, sc.Name)
	return nil
}

// updateServiceConfig
func updateServiceConfig(sc *v1.ServiceConfig, c client.Client) error {
	scLock.Lock()
	defer scLock.Unlock()

	// !import this method will panic concurrent map writes
	err := c.Update(context.Background(), sc)
	if err != nil {
		klog.Errorf("Update a ServiceConfig {%s/%s} has an err: %v", sc.Namespace, sc.Name, err)
		return err
	}

	klog.V(6).Infof("Update ServiceConfig {%s/%s} success.", sc.Namespace, sc.Name)
	return nil
}

// getServiceConfig
func getServiceConfig(namespace, name string, c client.Client) (*v1.ServiceConfig, error) {
	sc := &v1.ServiceConfig{}
	err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, sc)
	if err != nil {
		klog.Errorf("Get a ServiceConfig {%s/%s} has an err: %v", namespace, name, err)
		return nil, err
	}

	klog.V(6).Infof("Get ServiceConfig {%s/%s} success.", namespace, name)
	return sc, nil
}

// deleteServiceConfig
func deleteServiceConfig(sc *v1.ServiceConfig, c client.Client) error {
	scLock.Lock()
	defer scLock.Unlock()

	err := c.Delete(context.Background(), sc)
	if err != nil {
		klog.Errorf("Delete a ServiceConfig {%s/%s} has an err: %v", sc.Namespace, sc.Name, err)
		return err
	}

	klog.V(6).Infof("Delete ServiceConfig {%s/%s} success.", sc.Namespace, sc.Name)
	return nil
}
