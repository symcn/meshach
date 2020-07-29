package controllers

import (
	"github.com/symcn/mesh-operator/controllers/appmeshconfig"
	"github.com/symcn/mesh-operator/controllers/configuredservice"
	"github.com/symcn/mesh-operator/controllers/istioconfig"
	"github.com/symcn/mesh-operator/controllers/meshconfig"
	"github.com/symcn/mesh-operator/controllers/serviceaccessor"
	"github.com/symcn/mesh-operator/pkg/option"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
)

// AddToManager ...
func AddToManager(mgr ctrl.Manager, opt *option.ControllerOption) error {
	if err := (&appmeshconfig.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("AppMeshConfig"),
		Scheme: mgr.GetScheme(),
		Opt:    opt,
	}).SetupWithManager(mgr); err != nil {
		klog.Error(err, "unable to create controller", "controller", "AppMeshConfig")
		return err
	}
	if err := (&meshconfig.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("MeshConfig"),
		Scheme: mgr.GetScheme(),
		Opt:    opt,
	}).SetupWithManager(mgr); err != nil {
		klog.Error(err, "unable to create controller", "controller", "MeshConfig")
		return err
	}
	if err := (&istioconfig.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("IstioConfig"),
		Scheme: mgr.GetScheme(),
		Opt:    opt,
	}).SetupWithManager(mgr); err != nil {
		klog.Error(err, "unable to create controller", "controller", "IstioConfig")
		return err
	}
	if err := (&configuredservice.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ConfiguredService"),
		Scheme: mgr.GetScheme(),
		Opt:    opt,
	}).SetupWithManager(mgr); err != nil {
		klog.Error(err, "unable to create controller", "controller", "ConfiguredService")
		return err
	}
	if err := (&serviceaccessor.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ServiceAccessor"),
		Scheme: mgr.GetScheme(),
		Opt:    opt,
	}).SetupWithManager(mgr); err != nil {
		klog.Error(err, "unable to create controller", "controller", "ServiceAccessor")
		return err
	}
	return nil
}
