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

package serviceaccessor

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/option"
	"github.com/symcn/mesh-operator/pkg/utils"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Reconciler reconciles a ServiceAccessor object
type Reconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Opt        *option.ControllerOption
	MeshConfig *meshv1alpha1.MeshConfig
}

// +kubebuilder:rbac:groups=mesh.symcn.com,resources=*,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=*,verbs=get;list;watch;create;update;patch;delete

// Reconcile ...
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	klog.Infof("Reconciling ServiceAccessor: %s/%s", req.Namespace, req.Name)
	ctx := context.TODO()

	// Fetch the MeshConfig
	err := r.getMeshConfig(ctx)
	if err != nil {
		klog.Errorf("Get cluster MeshConfig[%s/%s] error: %+v",
			r.Opt.MeshConfigNamespace, r.Opt.MeshConfigName, err)
		return ctrl.Result{}, err
	}

	// Fetch the ServiceAccessor instance
	list := &meshv1alpha1.ServiceAccessorList{}
	fileds := &client.MatchingFields{"name": req.Name}
	opts := &client.ListOptions{}
	fileds.ApplyToList(opts)

	err = r.List(ctx, list, opts)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("no serviceaccessor list [%s]", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		klog.Errorf("list serviceaccessor[%s] err: %+v", req.NamespacedName, err)
		return ctrl.Result{}, err
	}

	for i := range list.Items {
		instance := &list.Items[i]
		// Get namespace of app pods
		namespaces := r.getAppNamespace(instance.Name)
		for _, namespace := range namespaces {
			klog.Infof("create sidecar in namespace: %s", namespace)
			r.reconcileSidecar(ctx, namespace, instance)
		}
	}

	klog.Infof("End Reconciliation, ServiceAccessor: %s/%s.", req.Namespace, req.Name)
	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileSidecar(ctx context.Context, namespace string, sa *meshv1alpha1.ServiceAccessor) error {
	sidecar := r.buildSidecar(namespace, sa)
	// NOTE: can not set Reference cross difference namespaces
	// if err := controllerutil.SetControllerReference(sa, sidecar, r.scheme); err != nil {
	// 	klog.Errorf("SetControllerReference error: %v", err)
	// 	return err
	// }

	found := &networkingv1beta1.Sidecar{}
	err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: sidecar.Name}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			err := r.Create(ctx, sidecar)
			if err != nil {
				klog.Errorf("Create Sidecar[%s,%s] error: %+v", sidecar.Namespace, sidecar.Name, err)
				return err
			}
		}
		return err
	}

	if compareSidecar(sidecar, found) {
		klog.Infof("Update Sidecar, Namespace: %s, Name: %s", found.Namespace, found.Name)
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			sidecar.Spec.DeepCopyInto(&found.Spec)
			found.Finalizers = sidecar.Finalizers
			found.Labels = sidecar.ObjectMeta.Labels

			updateErr := r.Update(ctx, found)
			if updateErr == nil {
				klog.V(4).Infof("%s/%s update Sidecar successfully", sidecar.Namespace, sidecar.Name)
				return nil
			}
			return updateErr
		})
		if err != nil {
			klog.Warningf("Update Sidecar [%s] spec failed, err: %+v", sidecar.Name, err)
			return err
		}
	}
	return nil
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

func (r *Reconciler) buildHosts(namespace string, accessHosts []string) []string {
	var hosts []string
	if len(r.MeshConfig.Spec.SidecarDefaultHosts) > 0 {
		hosts = append(hosts, r.MeshConfig.Spec.SidecarDefaultHosts...)
	}

	for _, svc := range accessHosts {
		hosts = append(hosts, fmt.Sprintf("%s/%s", namespace, utils.FormatToDNS1123(svc)))
	}
	return hosts
}

func (r *Reconciler) buildEgress(sa *meshv1alpha1.ServiceAccessor) []*v1beta1.IstioEgressListener {
	hosts := r.buildHosts(sa.Namespace, sa.Spec.AccessHosts)
	return []*v1beta1.IstioEgressListener{{
		Hosts: hosts,
	}}
}

func (r *Reconciler) buildSidecar(namespace string, sa *meshv1alpha1.ServiceAccessor) *networkingv1beta1.Sidecar {
	egress := r.buildEgress(sa)
	return &networkingv1beta1.Sidecar{
		ObjectMeta: v1.ObjectMeta{
			Name:      sa.Name,
			Namespace: namespace,
		},
		Spec: v1beta1.Sidecar{
			WorkloadSelector: &v1beta1.WorkloadSelector{
				Labels: map[string]string{
					r.MeshConfig.Spec.SidecarSelectLabel: sa.Name,
				},
			},
			Egress: egress,
			OutboundTrafficPolicy: &v1beta1.OutboundTrafficPolicy{
				Mode: v1beta1.OutboundTrafficPolicy_ALLOW_ANY,
			},
		},
	}
}

func compareSidecar(new, old *networkingv1beta1.Sidecar) bool {
	if !equality.Semantic.DeepEqual(new.ObjectMeta.Finalizers, old.ObjectMeta.Finalizers) {
		return true
	}

	if !equality.Semantic.DeepEqual(new.ObjectMeta.Labels, old.ObjectMeta.Labels) {
		return true
	}

	if !equality.Semantic.DeepEqual(new.Spec, old.Spec) {
		return true
	}
	return false
}

func (r *Reconciler) getAppNamespace(appName string) []string {
	var namespaces []string
	pods := &corev1.PodList{}
	labels := &client.MatchingLabels{r.MeshConfig.Spec.SidecarSelectLabel: appName}
	opts := &client.ListOptions{}
	labels.ApplyToList(opts)

	err := r.List(context.TODO(), pods, opts)
	if err != nil {
		klog.Warningf("Get pods error when create Sidecar[%s]: %+v", appName, err)
	}

	if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			namespaces = append(namespaces, pod.Namespace)
		}
	} else {
		klog.Warningf("No pods founds, skip create Sidecar[%s]", appName)
	}

	return namespaces
}

// SetupWithManager ...
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		&meshv1alpha1.ServiceAccessor{},
		"name",
		func(rawObj runtime.Object) []string {
			sa := rawObj.(*meshv1alpha1.ServiceAccessor)
			return []string{sa.Name}
		},
	); err != nil {
		klog.Warningf("add field index serviceaccessor.name, err: %#v", err)
	} else {
		klog.Info("add field index serviceaccessor.name successfully")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&meshv1alpha1.ServiceAccessor{}).
		Watches(
			&source.Kind{Type: &networkingv1beta1.Sidecar{}},
			&handler.EnqueueRequestForObject{},
		).
		Complete(r)
}
