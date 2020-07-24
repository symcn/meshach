package serviceaccessor

import (
	"context"
	"fmt"

	meshv1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	"github.com/symcn/mesh-operator/pkg/option"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Found Sidecar with app label
	label := r.meshConfig.Spec.SidecarSelectLabel
	sidecarName, ok := instance.Labels[label]
	if !ok {
		klog.Infof("%s/%s no [%s] labels found, skip to create Sidecar", request.Namespace, request.Name, label)
		return reconcile.Result{}, nil
	}

	found := &networkingv1beta1.Sidecar{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: sidecarName}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			sidecar := r.buildSidecar(sidecarName, instance)
			err := r.client.Create(ctx, sidecar)
			if err != nil {
				klog.Errorf("Create Sidecar[%s,%s] error: %+v", sidecar.Namespace, sidecar.Name, err)
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Update Sidecar

	return reconcile.Result{}, nil
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

func (r *ReconcileServiceAccessor) buildHosts(namespace string, accessServices []string) []string {
	var hosts []string
	if len(r.meshConfig.Spec.SidecarDefaultHosts) > 0 {
		hosts = append(hosts, r.meshConfig.Spec.SidecarDefaultHosts...)
	}

	for _, svc := range accessServices {
		hosts = append(hosts, fmt.Sprintf("%s/%s", namespace, svc))
	}

	return hosts
}

func (r *ReconcileServiceAccessor) buildEgress(cr *meshv1.ServiceAccessor) []*v1beta1.IstioEgressListener {
	hosts := r.buildHosts(cr.Namespace, cr.Spec.AccessHosts)
	return []*v1beta1.IstioEgressListener{{
		// Port:  &v1beta1.Port{},
		// Bind:  "",
		Hosts: hosts,
	}}
}

func (r *ReconcileServiceAccessor) buildSidecar(name string, cr *meshv1.ServiceAccessor) *networkingv1beta1.Sidecar {
	// ingress := []*v1beta1.IstioIngressListener{{
	// 	Port: &v1beta1.Port{
	// 		Number:   20882,
	// 		Protocol: "HTTP",
	// 		Name:     "dubbo-ingress",
	// 	},
	// 	Bind:            "127.0.0.1",
	// 	DefaultEndpoint: "127.0.0.1:20882",
	// }}
	egress := r.buildEgress(cr)
	return &networkingv1beta1.Sidecar{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
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
