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

package serviceconfig

import (
	"context"

	"github.com/go-logr/logr"
	meshv1alpha1 "github.com/symcn/meshach/api/v1alpha1"
	"github.com/symcn/meshach/pkg/option"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
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

// Reconciler reconciles a ServiceConfig object
type Reconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Opt        *option.ControllerOption
	MeshConfig *meshv1alpha1.MeshConfig
}

// +kubebuilder:rbac:groups=mesh.symcn.com,resources=serviceconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mesh.symcn.com,resources=serviceconfigs/status,verbs=get;update;patch

// Reconcile ...
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	klog.Infof("Reconciling ServiceConfig: %s/%s", req.Namespace, req.Name)
	ctx := context.TODO()

	// Fetch the ServiceConfig instance
	instance := &meshv1alpha1.ServiceConfig{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(6).Infof("Can't found ServiceConfig[%s/%s], skip...", req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the MeshConfig
	err = r.getMeshConfig(ctx)
	if err != nil {
		klog.Errorf("Get cluster MeshConfig[%s/%s] error: %+v",
			r.Opt.MeshConfigNamespace, r.Opt.MeshConfigName, err)
		return ctrl.Result{}, err
	}

	// Distribute Istio Config
	if err := r.reconcileWorkloadEntry(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileDestinationRule(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileVirtualService(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Update Status
	klog.Infof("Update ServiceConfig[%s/%s] status...", req.Namespace, req.Name)
	err = r.updateStatus(ctx, req, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		klog.Errorf("%s/%s update ServiceConfig status failed, err: %+v", req.Namespace, req.Name, err)
		return ctrl.Result{}, err
	}

	klog.Infof("End Reconciliation, ServiceConfig: %s/%s.", req.Namespace, req.Name)
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

// SetupWithManager ...
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meshv1alpha1.ServiceConfig{}).
		Watches(
			&source.Kind{Type: &networkingv1beta1.VirtualService{}},
			&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &meshv1alpha1.ServiceConfig{}},
		).
		Watches(
			&source.Kind{Type: &networkingv1beta1.DestinationRule{}},
			&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &meshv1alpha1.ServiceConfig{}},
		).
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
