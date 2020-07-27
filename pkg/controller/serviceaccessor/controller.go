package serviceaccessor

import (
	"context"
	"fmt"

	meshv1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	"github.com/symcn/mesh-operator/pkg/option"
	"github.com/symcn/mesh-operator/pkg/utils"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_serviceaccessor")

// Add creates a new ServiceAccessor Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, opt *option.ControllerOption) error {
	return add(mgr, newReconciler(mgr, opt))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, opt *option.ControllerOption) reconcile.Reconciler {
	return &ReconcileServiceAccessor{client: mgr.GetClient(), scheme: mgr.GetScheme(), opt: opt}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("serviceaccessor-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ServiceAccessor
	err = c.Watch(&source.Kind{Type: &meshv1.ServiceAccessor{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{
		Type: &networkingv1beta1.Sidecar{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &meshv1.ServiceAccessor{},
		})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileServiceAccessor implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileServiceAccessor{}

// ReconcileServiceAccessor reconciles a ServiceAccessor object
type ReconcileServiceAccessor struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client     client.Client
	scheme     *runtime.Scheme
	opt        *option.ControllerOption
	meshConfig *meshv1.MeshConfig
}

// Reconcile reads that state of the cluster for a ServiceAccessor object and makes changes based on the state read
// and what is in the ServiceAccessor.Spec
func (r *ReconcileServiceAccessor) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Infof("Reconciling ServiceAccessor: %s/%s", request.Namespace, request.Name)
	ctx := context.TODO()

	// Fetch the MeshConfig
	err := r.getMeshConfig(ctx)
	if err != nil {
		klog.Errorf("Get cluster MeshConfig[%s/%s] error: %+v",
			r.opt.MeshConfigNamespace, r.opt.MeshConfigName, err)
		return reconcile.Result{}, err
	}

	// Fetch the ServiceAccessor instance
	instance := &meshv1.ServiceAccessor{}
	err = r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Get namespace of app pods
	namespaces := r.getAppNamespace(instance.Name)
	for namespace := range namespaces {
		klog.V(6).Infof("create sidecar in namespace: %s", namespace)
		r.reconcileSidecar(ctx, namespace, instance)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileServiceAccessor) reconcileSidecar(ctx context.Context, namespace string, cr *meshv1.ServiceAccessor) error {
	sidecar := r.buildSidecar(namespace, cr.Name, cr)
	// NOTE: can not set Reference cross difference namespaces
	// if err := controllerutil.SetControllerReference(cr, sidecar, r.scheme); err != nil {
	// 	klog.Errorf("SetControllerReference error: %v", err)
	// 	return err
	// }

	found := &networkingv1beta1.Sidecar{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: cr.Name}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			err := r.client.Create(ctx, sidecar)
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

			updateErr := r.client.Update(ctx, found)
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

func (r *ReconcileServiceAccessor) getMeshConfig(ctx context.Context) error {
	meshConfig := &meshv1.MeshConfig{}
	err := r.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: r.opt.MeshConfigNamespace,
			Name:      r.opt.MeshConfigName,
		},
		meshConfig,
	)
	if err != nil {
		return err
	}
	r.meshConfig = meshConfig
	klog.V(6).Infof("Get cluster MeshConfig: %+v", meshConfig)
	return nil
}

func (r *ReconcileServiceAccessor) buildHosts(namespace string, accessHosts []string) []string {
	var hosts []string
	if len(r.meshConfig.Spec.SidecarDefaultHosts) > 0 {
		hosts = append(hosts, r.meshConfig.Spec.SidecarDefaultHosts...)
	}

	for _, svc := range accessHosts {
		hosts = append(hosts, fmt.Sprintf("%s/%s", namespace, utils.FormatToDNS1123(svc)))
	}
	return hosts
}

func (r *ReconcileServiceAccessor) buildEgress(cr *meshv1.ServiceAccessor) []*v1beta1.IstioEgressListener {
	hosts := r.buildHosts(cr.Namespace, cr.Spec.AccessHosts)
	return []*v1beta1.IstioEgressListener{{
		Hosts: hosts,
	}}
}

func (r *ReconcileServiceAccessor) buildSidecar(namespace, name string, cr *meshv1.ServiceAccessor) *networkingv1beta1.Sidecar {
	egress := r.buildEgress(cr)
	return &networkingv1beta1.Sidecar{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.Sidecar{
			WorkloadSelector: &v1beta1.WorkloadSelector{
				Labels: map[string]string{
					r.meshConfig.Spec.SidecarSelectLabel: name,
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

func (r *ReconcileServiceAccessor) getAppNamespace(appName string) map[string]struct{} {
	namespaces := make(map[string]struct{})
	pods := &corev1.PodList{}
	labels := &client.MatchingLabels{r.meshConfig.Spec.SidecarSelectLabel: appName}
	opts := &client.ListOptions{}
	labels.ApplyToList(opts)

	err := r.client.List(context.TODO(), pods, opts)
	if err != nil {
		klog.Warningf("Get pods error when create Sidecar[%s]: %+v", appName, err)
	}

	if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			namespaces[pod.Namespace] = struct{}{}
		}
	} else {
		klog.Warningf("No pods founds, skip create Sidecar[%s]", err)
	}

	return namespaces
}
