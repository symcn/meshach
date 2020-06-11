/*
Copyright 2020 The Symcn Authors.

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

// Package appmeshconfig ...
package appmeshconfig

import (
	"context"

	meshv1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	"github.com/mesh-operator/pkg/option"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

const (
	controllerName         = "appMeshConfig-controller"
	appLabelKey            = "app"
	matchLabelKey          = "sym-zone"
	workloadSelectLabelKey = "service"
	workloadGroupLabelKey  = "sym-group"
	workloadZoneLabelKey   = "sym-zone"
	httpRouteName          = "dubbo-http-route"
	proxyRouteName         = "dubbo-proxy-route"
)

var log = logf.Log.WithName(controllerName)

// Add creates a new AppMeshConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, opt *option.ControllerOption) error {
	return add(mgr, newReconciler(mgr, opt))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, opt *option.ControllerOption) reconcile.Reconciler {
	return &ReconcileAppMeshConfig{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		opt:    opt,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AppMeshConfig
	err = c.Watch(&source.Kind{Type: &meshv1.AppMeshConfig{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resources and requeue the owner AppMeshConfig
	err = c.Watch(&source.Kind{Type: &networkingv1beta1.WorkloadEntry{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &meshv1.AppMeshConfig{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &networkingv1beta1.VirtualService{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &meshv1.AppMeshConfig{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{
		Type: &networkingv1beta1.DestinationRule{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &meshv1.AppMeshConfig{},
		})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{
		Type: &networkingv1beta1.ServiceEntry{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &meshv1.AppMeshConfig{},
		})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileAppMeshConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileAppMeshConfig{}

// ReconcileAppMeshConfig reconciles a AppMeshConfig foundect
type ReconcileAppMeshConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads foundects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	opt    *option.ControllerOption
}

// Reconcile reads that state of the cluster for a AppMeshConfig foundect and makes changes based on the state read
// and what is in the AppMeshConfig.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAppMeshConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Infof("Reconciling AppMeshConfig: %s/%s", request.Namespace, request.Name)
	ctx := context.TODO()
	// Fetch the AppMeshConfig instance
	instance := &meshv1.AppMeshConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request foundect not found, could have been deleted after reconcile request.
			// Owned foundects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the foundect - requeue the request.
		return reconcile.Result{}, err
	}

	// TODO(haidong): Is it requeue request after Spec changed 5 seconds later to update Status?
	for _, svc := range instance.Spec.Services {
		if err := r.reconcileWorkloadEntry(ctx, instance, svc); err != nil {
			return reconcile.Result{}, err
		}
		if err := r.reconcileServiceEntry(ctx, instance, svc); err != nil {
			return reconcile.Result{}, err
		}
		if err := r.reconcileDestinationRule(ctx, instance, svc); err != nil {
			return reconcile.Result{}, err
		}
		if err := r.reconcileVirtualService(ctx, instance, svc); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Modify Status
	status := r.buildStatus(instance)
	if !equality.Semantic.DeepEqual(status, instance.Status) {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			status.DeepCopyInto(&instance.Status)
			t := metav1.Now()
			instance.Status.LastUpdateTime = &t

			updateErr := r.client.Status().Update(ctx, instance)
			if updateErr == nil {
				klog.V(4).Infof("%s/%s update status[%s] successfully",
					request.Namespace, request.Name, instance.Status.Phase)
				return nil
			}

			getErr := r.client.Get(ctx, request.NamespacedName, instance)
			if getErr != nil {
				klog.Errorf("%s/%s update get AppMeshConfig failed, err: %+v", request.Namespace, request.Name)
				return getErr
			}
			return updateErr
		})
	}

	klog.Infof("End Reconciliation, AppMeshConfig: %s/%s.", request.Namespace, request.Name)
	return reconcile.Result{}, nil
}
