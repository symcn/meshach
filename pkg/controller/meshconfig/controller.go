package meshconfig

import (
	"context"

	meshv1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	"github.com/mesh-operator/pkg/option"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_meshconfig")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new MeshConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, opt *option.ControllerOption) error {
	return add(mgr, newReconciler(mgr, opt))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, opt *option.ControllerOption) reconcile.Reconciler {
	return &ReconcileMeshConfig{client: mgr.GetClient(), scheme: mgr.GetScheme(), opt: opt}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("meshconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource MeshConfig
	err = c.Watch(&source.Kind{Type: &meshv1.MeshConfig{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileMeshConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileMeshConfig{}

// ReconcileMeshConfig reconciles a MeshConfig object
type ReconcileMeshConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	opt    *option.ControllerOption
}

// Reconcile reads that state of the cluster for a MeshConfig object and makes changes based on the state read
func (r *ReconcileMeshConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling MeshConfig")

	// Fetch the MeshConfig instance
	instance := &meshv1.MeshConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	amcList := &meshv1.AppMeshConfigList{}
	err = r.client.List(context.TODO(), amcList, &client.ListOptions{Namespace: corev1.NamespaceAll})
	if err != nil {
		klog.Infof("Get AppMeshConfig error: %s", err)
	}
	for i := range amcList.Items {
		amc := amcList.Items[i]
		if amc.Spec.MeshConfigGeneration != instance.Generation {
			amc.Spec.MeshConfigGeneration = instance.Generation
			err := r.client.Update(context.TODO(), &amc)
			if err != nil {
				klog.Errorf("Update AppMeshConfig[%s/%s] in MeshConfig reconcile error: %+v",
					amc.Namespace, amc.Name, err)
			}
		}
	}

	return reconcile.Result{}, nil
}
