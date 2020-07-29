/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package configuredservice

import (
	"context"

	"github.com/go-logr/logr"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/option"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	httpRouteName    = "dubbo-http-route"
	defaultRouteName = "dubbo-default-route"
)

// Reconciler reconciles a ConfiguredService object
type Reconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Opt        *option.ControllerOption
	MeshConfig *meshv1alpha1.MeshConfig
}

// +kubebuilder:rbac:groups=mesh.symcn.com,resources=configuredservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mesh.symcn.com,resources=configuredservices/status,verbs=get;update;patch

// Reconcile ...
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	klog.Infof("Reconciling ConfiguredService: %s/%s", req.Namespace, req.Name)
	ctx := context.TODO()

	// Fetch the MeshConfig
	err := r.getMeshConfig(ctx)
	if err != nil {
		klog.Errorf("Get cluster MeshConfig[%s/%s] error: %+v",
			r.Opt.MeshConfigNamespace, r.Opt.MeshConfigName, err)
		return ctrl.Result{}, err
	}

	// Fetch the ConfiguredService instance
	instance := &meshv1alpha1.ConfiguredService{}
	err = r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request foundect not found, could have been deleted after reconcile req.
			// Owned foundects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.Infof("Can't found ConfiguredService[%s/%s], requeue...", req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		// Error reading the foundect - requeue the req.
		return ctrl.Result{}, err
	}

	// Set finalizers
	deleteAmcFinalizer := "appmeshconfig.finalizers"
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(instance.ObjectMeta.Finalizers, deleteAmcFinalizer) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, deleteAmcFinalizer)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(instance.ObjectMeta.Finalizers, deleteAmcFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteAmc(ctx, instance); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, deleteAmcFinalizer)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Distribute Istio Config
	if err := r.reconcileWorkloadEntry(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileServiceEntry(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileDestinationRule(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileVirtualService(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Update Status
	klog.Infof("Update ConfiguredService[%s/%s] status...", req.Namespace, req.Name)
	err = r.updateStatus(ctx, req, instance)
	if err != nil {
		klog.Errorf("%s/%s update ConfiguredService failed, err: %+v", req.Namespace, req.Name, err)
		return ctrl.Result{}, err
	}

	// Reconcile AppMeshConfig
	klog.Infof("Reconcile AppMeshConfig[%s/%s]", req.Namespace, req.Name)
	err = r.reconcileAmc(ctx, instance)
	if err != nil {
		klog.Errorf("%s/%s create AppMeshConfig failed, err: %+v", req.Namespace, req.Name, err)
	}

	klog.Infof("End Reconciliation, ConfiguredService: %s/%s.", req.Namespace, req.Name)
	return ctrl.Result{}, nil
}

func (r *Reconciler) getMeshConfig(ctx context.Context) error {
	meshConfig := &meshv1alpha1.MeshConfig{}
	err := r.Get(
		ctx,
		types.NamespacedName{
			Namespace: r.Opt.MeshConfigNamespace,
			Name:      r.Opt.MeshConfigName,
		},
		meshConfig,
	)
	if err != nil {
		return err
	}
	r.MeshConfig = meshConfig
	klog.V(6).Infof("Get cluster MeshConfig: %+v", meshConfig)
	return nil
}

func (r *Reconciler) reconcileAmc(ctx context.Context, cs *meshv1alpha1.ConfiguredService) error {
	name, ok := cs.Labels["app"]
	if !ok {
		klog.Infof("Can not found app label in ConfiguredService[%s], skip create AppMeshConfig.", cs.Name)
		return nil
	}

	found := &meshv1alpha1.AppMeshConfig{}
	err := r.Get(ctx, types.NamespacedName{Namespace: cs.Namespace, Name: name}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Can't found AppMeshConfig[%s/%s], create...", cs.Namespace, name)
			amc := r.buildAmc(cs, name)
			err = r.Create(ctx, amc)
			if err != nil {
				klog.Errorf("Create AppMeshConfig[%s/%s] error: %+v", cs.Namespace, name, err)
				return err
			}
			return nil
		}
		klog.Errorf("Get AppMeshConfig[%s/%s] error: %+v", cs.Namespace, name, err)
		return err
	}

	err = r.Update(ctx, r.updateAmc(found, cs))
	if err != nil {
		klog.Errorf("Update AppMeshConfig[%s/%s] error: %+v", cs.Namespace, name, err)
		return err
	}
	return nil
}

func (r *Reconciler) buildAmc(cs *meshv1alpha1.ConfiguredService, name string) *meshv1alpha1.AppMeshConfig {
	return &meshv1alpha1.AppMeshConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: cs.Namespace,
			// Labels:    map[string]string{"": ""},
		},
		Spec: meshv1alpha1.AppMeshConfigSpec{
			Services: []*meshv1alpha1.Service{{
				Name:         cs.Name,
				OriginalName: cs.Spec.OriginalName,
				Ports:        cs.Spec.Ports,
				Instances:    cs.Spec.Instances,
				Policy:       cs.Spec.Policy,
				Subsets:      cs.Spec.Subsets,
			}},
		},
	}
}

func (r *Reconciler) updateAmc(amc *meshv1alpha1.AppMeshConfig, cs *meshv1alpha1.ConfiguredService) *meshv1alpha1.AppMeshConfig {
	found := false
	for _, svc := range amc.Spec.Services {
		if svc.Name == cs.Name {
			found = true
			svc.Subsets = cs.Spec.Subsets
			svc.Ports = cs.Spec.Ports
			svc.Instances = cs.Spec.Instances
			svc.Policy = cs.Spec.Policy
			svc.OriginalName = cs.Spec.OriginalName
		}
	}
	if !found {
		amc.Spec.Services = append(amc.Spec.Services, &meshv1alpha1.Service{
			Name:         cs.Name,
			OriginalName: cs.Spec.OriginalName,
			Ports:        cs.Spec.Ports,
			Instances:    cs.Spec.Instances,
			Policy:       cs.Spec.Policy,
			Subsets:      cs.Spec.Subsets,
		})
	}
	return amc
}

func (r *Reconciler) deleteAmc(ctx context.Context, cs *meshv1alpha1.ConfiguredService) error {
	name, ok := cs.Labels["app"]
	if !ok {
		klog.Infof("Can not found app label in ConfiguredService[%s], skip delete AppMeshConfig.", cs.Name)
		return nil
	}

	found := &meshv1alpha1.AppMeshConfig{}
	err := r.Get(ctx, types.NamespacedName{Namespace: cs.Namespace, Name: name}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Can't found AppMeshConfig[%s/%s] when deleting it", cs.Namespace, name)
			return nil
		}
		klog.Errorf("Get AppMeshConfig[%s/%s] error: %+v", cs.Namespace, name, err)
		return err
	}

	for _, svc := range found.Spec.Services {
		if len(found.Spec.Services) == 0 {
			if err := r.Delete(ctx, found); err != nil {
				klog.Errorf("Delete AppMeshConfig[%s] error: %+v", name, err)
				return err
			}
			return nil
		}
		if svc.Name == cs.Name {
			continue
		}
		found.Spec.Services = append(found.Spec.Services, svc)
	}

	if err := r.Update(ctx, found); err != nil {
		klog.Errorf("Update AppMeshConfig[%s] error: %+v", name, err)
		return err
	}
	return nil

}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// SetupWithManager ...
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meshv1alpha1.ConfiguredService{}).
		Watches(
			&source.Kind{Type: &networkingv1beta1.WorkloadEntry{}},
			&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &meshv1alpha1.ConfiguredService{}},
		).
		Watches(
			&source.Kind{Type: &networkingv1beta1.VirtualService{}},
			&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &meshv1alpha1.ConfiguredService{}},
		).
		Watches(
			&source.Kind{Type: &networkingv1beta1.DestinationRule{}},
			&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &meshv1alpha1.ConfiguredService{}},
		).
		Watches(
			&source.Kind{Type: &networkingv1beta1.ServiceEntry{}},
			&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &meshv1alpha1.ConfiguredService{}},
		).
		Complete(r)
}
