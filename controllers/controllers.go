package controllers

import (
	"github.com/symcn/meshach/controllers/appmeshconfig"
	"github.com/symcn/meshach/controllers/configuredservice"
	"github.com/symcn/meshach/controllers/istioconfig"
	"github.com/symcn/meshach/controllers/meshconfig"
	"github.com/symcn/meshach/controllers/serviceaccessor"
	"github.com/symcn/meshach/controllers/serviceconfig"
	"github.com/symcn/meshach/pkg/option"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
)

// AddToManager ...
func AddToManager(mgr ctrl.Manager, opt *option.ControllerOption) error {
	if opt.EnableAppMeshConfig {
		if err := (&appmeshconfig.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("AppMeshConfig"),
			Scheme: mgr.GetScheme(),
			Opt:    opt,
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "AppMeshConfig")
			return err
		}
	}

	if opt.EnableMeshConfig {
		if err := (&meshconfig.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("MeshConfig"),
			Scheme: mgr.GetScheme(),
			Opt:    opt,
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "MeshConfig")
			return err
		}
	}

	if opt.EnableIstioConfig {
		if err := (&istioconfig.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("IstioConfig"),
			Scheme: mgr.GetScheme(),
			Opt:    opt,
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "IstioConfig")
			return err
		}
	}

	if opt.EnableConfiguredService {
		if err := (&configuredservice.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("ConfiguredService"),
			Scheme: mgr.GetScheme(),
			Opt:    opt,
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "ConfiguredService")
			return err
		}
	}

	if opt.EnableServiceAccessor {
		if err := (&serviceaccessor.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("ServiceAccessor"),
			Scheme: mgr.GetScheme(),
			Opt:    opt,
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "ServiceAccessor")
			return err
		}
	}

	if opt.EnableServiceConfig {
		if err := (&serviceconfig.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("ServiceConfig"),
			Scheme: mgr.GetScheme(),
			Opt:    opt,
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "ServiceConfig")
			return err
		}
	}
	return nil
}
