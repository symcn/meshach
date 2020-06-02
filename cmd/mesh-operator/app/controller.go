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

	"github.com/mesh-operator/pkg/apis"
	"github.com/mesh-operator/pkg/controller"
	"github.com/mesh-operator/pkg/healthcheck"
	k8sclient "github.com/mesh-operator/pkg/k8s/client"
	"github.com/mesh-operator/pkg/router"
	"github.com/mesh-operator/pkg/utils"
	"github.com/mesh-operator/pkg/version"
	"github.com/spf13/cobra"
	"k8s.io/klog"
	"k8s.io/sample-controller/pkg/signals"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger = logf.KBLog.WithName("controller")
)

// NewControllerCmd ...
func NewControllerCmd(ropt *RootOption) *cobra.Command {
	opt := utils.DefaultControllerOption()
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
			if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
				klog.Error(err, "")
				os.Exit(1)
			}

			// Setup all Controllers
			if err := controller.AddToManager(mgr, opt); err != nil {
				klog.Fatalf("unable to register controllers to the manager err: %v", err)
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
	return cmd
}
