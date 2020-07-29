/*
Copyright 2020 The symcn authors.

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

package app

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	meshv1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/controllers"
	"github.com/symcn/mesh-operator/pkg/healthcheck"
	k8sclient "github.com/symcn/mesh-operator/pkg/k8s/client"
	"github.com/symcn/mesh-operator/pkg/option"
	"github.com/symcn/mesh-operator/pkg/router"
	"github.com/symcn/mesh-operator/pkg/utils"
	"github.com/symcn/mesh-operator/pkg/version"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/klog"
	"k8s.io/sample-controller/pkg/signals"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger = logf.KBLog.WithName("controller")
)

// NewControllerCmd ...
func NewControllerCmd(ropt *option.RootOption) *cobra.Command {
	opt := option.DefaultControllerOption()
	cmd := &cobra.Command{
		Use:     "controller",
		Aliases: []string{"ctl"},
		Short:   "The controller is responsible for converting AppMeshConfig to istio CRs",
		Run: func(cmd *cobra.Command, args []string) {
			PrintFlags(cmd.Flags())
			v := version.GetVersion()
			fmt.Fprintf(os.Stdout, "version: %v\n", v.String())

			cfg, err := ropt.GetK8sConfig()
			if err != nil {
				klog.Fatalf("unable to get kubeconfig err: %v", err)
			}

			// Set default manager options
			rp := time.Minute * time.Duration(opt.SyncPeriod)
			options := manager.Options{
				Namespace:               ropt.Namespace,
				Scheme:                  k8sclient.GetScheme(),
				SyncPeriod:              &rp,
				MetricsBindAddress:      "0",
				LeaderElection:          opt.EnableLeaderElection,
				LeaderElectionNamespace: opt.LeaderElectionNamespace,
				LeaderElectionID:        opt.LeaderElectionID,
			}

			// Create a new manager to provide shared dependencies and start components
			mgr, err := manager.New(cfg, options)
			if err != nil {
				klog.Fatalf("unable to new manager err: %v", err)
				os.Exit(1)
			}

			// Create router for metrics
			routerOptions := &router.Options{
				GinLogEnabled:    opt.GinLogEnabled,
				MetricsEnabled:   opt.MetricsEnabled,
				GinLogSkipPath:   opt.GinLogSkipPath,
				PprofEnabled:     opt.PprofEnabled,
				Addr:             opt.HTTPAddress,
				MetricsPath:      "metrics",
				MetricsSubsystem: "controller",
			}

			healthHandler := healthcheck.GetHealthHandler()
			healthHandler.AddLivenessCheck("goroutine_threshold",
				healthcheck.GoroutineCountCheck(opt.GoroutineThreshold))

			rt := router.NewRouter(routerOptions)
			rt.AddRoutes("index", rt.DefaultRoutes())
			rt.AddRoutes("health", healthHandler.Routes())

			klog.Info("Registering Components.")
			components := &utils.Components{}
			components.Add(rt)

			// Setup Scheme for all resources
			if err := meshv1.AddToScheme(mgr.GetScheme()); err != nil {
				klog.Fatalf("add AppMeshConfig to scheme error: %+v", err)
			}
			if err := networkingv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
				klog.Fatalf("add istio CRDs to scheme error: %+v", err)
			}

			// Setup all Controllers
			if err := controllers.AddToManager(mgr, opt); err != nil {
				klog.Fatalf("unable to register controllers to the manager err: %+v", err)
			}

			stopCh := signals.SetupSignalHandler()
			klog.Infof("start custom components")
			go components.Start(stopCh)

			logger.Info("zap debug", "ResyncPeriod", opt.SyncPeriod)
			klog.Info("starting manager...")

			// Start the Cmd
			if err := mgr.Start(stopCh); err != nil {
				klog.Fatalf("error occurred while running the manager: %v", err)
			}
		},
	}

	cmd.PersistentFlags().StringVar(
		&opt.HTTPAddress,
		"http-address",
		opt.HTTPAddress,
		"The HTTP address of controller, default is :8080.",
	)
	cmd.PersistentFlags().Int32Var(
		&opt.SyncPeriod,
		"sync-period",
		opt.SyncPeriod,
		"The minimum frequency at which watched resources are reconciled.",
	)
	cmd.PersistentFlags().BoolVar(
		&opt.MetricsEnabled,
		"metrics-enables",
		opt.MetricsEnabled,
		"This parameter determines whether metrics server is enabled.",
	)
	cmd.PersistentFlags().StringVar(
		&opt.LeaderElectionNamespace,
		"leader-election-namespace",
		opt.LeaderElectionNamespace,
		"The namespace where leader election configmap resides.",
	)
	cmd.PersistentFlags().StringVar(
		&opt.LeaderElectionID,
		"leader-election-id",
		opt.LeaderElectionID,
		"The name of leader election configmap.",
	)
	cmd.PersistentFlags().BoolVar(
		&opt.EnableLeaderElection,
		"leader-enable",
		opt.EnableLeaderElection,
		"This parameter determines whether enable leader election.",
	)
	cmd.PersistentFlags().BoolVar(
		&opt.GinLogEnabled,
		"enable-ginlog",
		opt.GinLogEnabled,
		"Enabled will open gin run log.",
	)
	cmd.PersistentFlags().BoolVar(
		&opt.PprofEnabled,
		"enable-pprof",
		opt.PprofEnabled,
		"Enabled will open endpoint for go pprof.",
	)
	cmd.PersistentFlags().IntVar(
		&opt.GoroutineThreshold,
		"goroutine-threshold",
		opt.GoroutineThreshold,
		"The max Goroutine Threshold",
	)
	cmd.PersistentFlags().StringVar(
		&opt.ProxyHost,
		"proxy-host",
		opt.ProxyHost,
		"The host of dubbo proxy service.",
	)
	cmd.PersistentFlags().Int32Var(
		&opt.ProxyAttempts,
		"proxy-attempts",
		opt.ProxyAttempts,
		"Number of retries for dubbo proxy services.",
	)
	cmd.PersistentFlags().Int64Var(
		&opt.ProxyPerTryTimeout,
		"proxy-pertrytimeout",
		opt.ProxyPerTryTimeout,
		"Timeout per retry attempt for dubbo proxy service. format: 1h/1m/1s/1ms.",
	)
	cmd.PersistentFlags().StringVar(
		&opt.ProxyRetryOn,
		"proxy-retryon",
		opt.ProxyRetryOn,
		"Flag to specify whether the retries should retry to other localities.",
	)
	cmd.PersistentFlags().StringVar(
		&opt.MeshConfigName,
		"meshconfig-name",
		opt.MeshConfigName,
		"The name of MeshConfig to use.",
	)
	cmd.PersistentFlags().StringVar(
		&opt.MeshConfigNamespace,
		"meshconfig-namespace",
		opt.MeshConfigNamespace,
		"The namespace of MeshConfig to use.",
	)
	cmd.PersistentFlags().StringVar(
		&opt.SelectLabel,
		"selectlabel",
		opt.SelectLabel,
		"The key of all mesh CRs metadata labels. Default is app.",
	)
	cmd.PersistentFlags().IntVar(
		&opt.MaxConcurrentReconciles,
		"max-concurrent-reconciles",
		opt.MaxConcurrentReconciles,
		"MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 10.",
	)

	return cmd
}
