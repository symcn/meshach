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

package appmeshconfig

import (
	"context"

	"github.com/go-logr/logr"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/option"
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

// Reconciler reconciles a AppMeshConfig object
type Reconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Opt        *option.ControllerOption
	MeshConfig *meshv1alpha1.MeshConfig
}

// +kubebuilder:rbac:groups=mesh.symcn.com,resources=appmeshconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mesh.symcn.com,resources=appmeshconfigs/status,verbs=get;update;patch

// Reconcile ...
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	klog.Infof("Reconciling AppMeshConfig: %s/%s", req.Namespace, req.Name)
	ctx := context.Background()

	// Fetch the MeshConfig
	err := r.getMeshConfig(ctx)
	if err != nil {
		klog.Errorf("Get cluster MeshConfig[%s/%s] error: %+v",
			r.Opt.MeshConfigNamespace, r.Opt.MeshConfigName, err)
		return ctrl.Result{}, err
	}

	// Fetch the AppMeshConfig instance
	instance := &meshv1alpha1.AppMeshConfig{}
	err = r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request foundect not found, could have been deleted after reconcile req.
			// Owned foundects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.Infof("Can't found AppMeshConfig[%s/%s], requeue...", req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		// Error reading the foundect - requeue the req.
		return ctrl.Result{}, err
	}

	klog.Infof("Update AppMeshConfig[%s/%s] status...", req.Namespace, req.Name)
	err = r.updateStatus(ctx, req, instance)
	if err != nil {
		klog.Errorf("%s/%s update AppMeshConfig failed, err: %+v", req.Namespace, req.Name, err)
		return ctrl.Result{}, err
	}

	klog.Infof("End Reconciliation, AppMeshConfig: %s/%s.", req.Namespace, req.Name)
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
	klog.V(4).Infof("Get cluster MeshConfig: %+v", meshConfig)
	return nil
}

// SetupWithManager ...
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meshv1alpha1.AppMeshConfig{}).
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
