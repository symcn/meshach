package configuraredservice

import (
	"context"

	meshv1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	"github.com/symcn/mesh-operator/pkg/option"
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

const (
	controllerName = "configuraredservice-controller"
	httpRouteName  = "dubbo-http-route"
	proxyRouteName = "dubbo-proxy-route"
)

var log = logf.Log.WithName("controller_configuraredservice")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ConfiguraredService Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, opt *option.ControllerOption) error {
	return add(mgr, newReconciler(mgr, opt), opt)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, opt *option.ControllerOption) reconcile.Reconciler {
	return &ReconcileConfiguraredService{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		opt:    opt,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, opt *option.ControllerOption) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: opt.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ConfiguraredService
	err = c.Watch(&source.Kind{Type: &meshv1.ConfiguraredService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{
		Type: &networkingv1beta1.WorkloadEntry{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &meshv1.ConfiguraredService{},
		})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{
		Type: &networkingv1beta1.VirtualService{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &meshv1.ConfiguraredService{},
		})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{
		Type: &networkingv1beta1.DestinationRule{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &meshv1.ConfiguraredService{},
		})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{
		Type: &networkingv1beta1.ServiceEntry{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &meshv1.ConfiguraredService{},
		})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileConfiguraredService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileConfiguraredService{}

// ReconcileConfiguraredService reconciles a ConfiguraredService object
type ReconcileConfiguraredService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client     client.Client
	scheme     *runtime.Scheme
	opt        *option.ControllerOption
	meshConfig *meshv1.MeshConfig
}

// Reconcile reads that state of the cluster for a ConfiguraredService object and makes changes based on the state read
// and what is in the ConfiguraredService.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileConfiguraredService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Infof("Reconciling ConfiguraredService: %s/%s", request.Namespace, request.Name)
	ctx := context.TODO()

	// Fetch the MeshConfig
	err := r.getMeshConfig(ctx)
	if err != nil {
		klog.Errorf("Get cluster MeshConfig[%s/%s] error: %+v",
			r.opt.MeshConfigNamespace, r.opt.MeshConfigName, err)
		return reconcile.Result{}, err
	}

	// Fetch the ConfiguraredService instance
	instance := &meshv1.ConfiguraredService{}
	err = r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request foundect not found, could have been deleted after reconcile request.
			// Owned foundects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.Infof("Can't found ConfiguraredService[%s/%s], requeue...", request.Namespace, request.Name)
			return reconcile.Result{}, nil
		}
		// Error reading the foundect - requeue the request.
		return reconcile.Result{}, err
	}

	// Set finalizers
	deleteAmcFinalizer := "appmeshconfig.finalizers"
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(instance.ObjectMeta.Finalizers, deleteAmcFinalizer) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, deleteAmcFinalizer)
			if err := r.client.Update(ctx, instance); err != nil {
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(instance.ObjectMeta.Finalizers, deleteAmcFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteAmc(ctx, instance); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return reconcile.Result{}, err
			}

			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, deleteAmcFinalizer)
			if err := r.client.Update(ctx, instance); err != nil {
				return reconcile.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return reconcile.Result{}, nil
	}

	// Distribute Istio Config
	if err := r.reconcileWorkloadEntry(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.reconcileServiceEntry(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.reconcileDestinationRule(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.reconcileVirtualService(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}

	// Update Status
	klog.Infof("Update ConfiguraredService[%s/%s] status...", request.Namespace, request.Name)
	err = r.updateStatus(ctx, request, instance)
	if err != nil {
		klog.Errorf("%s/%s update ConfiguraredService failed, err: %+v", request.Namespace, request.Name, err)
		return reconcile.Result{}, err
	}

	// Reconcile AppMeshConfig
	klog.Infof("Reconcile AppMeshConfig[%s/%s]", request.Namespace, request.Name)
	err = r.reconcileAmc(ctx, instance)
	if err != nil {
		klog.Errorf("%s/%s create AppMeshConfig failed, err: %+v", request.Namespace, request.Name, err)
	}

	klog.Infof("End Reconciliation, ConfiguraredService: %s/%s.", request.Namespace, request.Name)
	return reconcile.Result{}, nil
}

func (r *ReconcileConfiguraredService) getMeshConfig(ctx context.Context) error {
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
	klog.V(4).Infof("Get cluster MeshConfig: %+v", meshConfig)
	return nil
}

func (r *ReconcileConfiguraredService) reconcileAmc(ctx context.Context, cs *meshv1.ConfiguraredService) error {
	name, ok := cs.Labels["app"]
	if !ok {
		klog.Infof("Can not found app label in ConfiguraredService[%s], skip create AppMeshConfig.", cs.Name)
		return nil
	}

	found := &meshv1.AppMeshConfig{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: cs.Namespace, Name: name}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Can't found AppMeshConfig[%s/%s], create...", cs.Namespace, name)
			amc := r.buildAmc(cs, name)
			err = r.client.Create(ctx, amc)
			if err != nil {
				klog.Errorf("Create AppMeshConfig[%s/%s] error: %+v", cs.Namespace, name, err)
				return err
			}
			return nil
		}
		klog.Errorf("Get AppMeshConfig[%s/%s] error: %+v", cs.Namespace, name, err)
		return err
	}

	err = r.client.Update(ctx, r.updateAmc(found, cs))
	if err != nil {
		klog.Errorf("Update AppMeshConfig[%s/%s] error: %+v", cs.Namespace, name, err)
		return err
	}
	return nil
}

func (r *ReconcileConfiguraredService) buildAmc(cs *meshv1.ConfiguraredService, name string) *meshv1.AppMeshConfig {
	return &meshv1.AppMeshConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: cs.Namespace,
			// Labels:    map[string]string{"": ""},
		},
		Spec: meshv1.AppMeshConfigSpec{
			Services: []*meshv1.Service{{
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

func (r *ReconcileConfiguraredService) updateAmc(amc *meshv1.AppMeshConfig, cs *meshv1.ConfiguraredService) *meshv1.AppMeshConfig {
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
		amc.Spec.Services = append(amc.Spec.Services, &meshv1.Service{
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

func (r *ReconcileConfiguraredService) deleteAmc(ctx context.Context, cs *meshv1.ConfiguraredService) error {
	name, ok := cs.Labels["app"]
	if !ok {
		klog.Infof("Can not found app label in ConfiguraredService[%s], skip delete AppMeshConfig.", cs.Name)
		return nil
	}

	found := &meshv1.AppMeshConfig{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: cs.Namespace, Name: name}, found)
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
			if err := r.client.Delete(ctx, found); err != nil {
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

	if err := r.client.Update(ctx, found); err != nil {
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
