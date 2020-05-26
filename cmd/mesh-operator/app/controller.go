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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mesh-operator/pkg/apis"
	"github.com/mesh-operator/pkg/controller"
	"github.com/mesh-operator/pkg/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"k8s.io/sample-controller/pkg/signals"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ControllerOption ...
type ControllerOption struct {
	MetricsHost         string
	MetricsPort         int32
	OperatorMetricsPort int32
}

// DefaultControllerOption ...
func DefaultControllerOption() *ControllerOption {
	return &ControllerOption{
		MetricsHost:         "0.0.0.0",
		MetricsPort:         8383,
		OperatorMetricsPort: 8686,
	}
}

// NewControllerCmd ...
func NewControllerCmd(ropt *RootOption) *cobra.Command {
	opt := DefaultControllerOption()
	cmd := &cobra.Command{
		Use:     "controller",
		Aliases: []string{"ctl"},
		Short:   "The controller is responsible for converting AppMeshConfig to istio CRs",
		Run: func(cmd *cobra.Command, args []string) {
			PrintFlags(cmd.Flags())
			v := version.GetVersion()
			fmt.Fprintf(os.Stdout, "version: %v\n", v.String())

			// we'll use all-namespaces instead.
			namespace, err := k8sutil.GetWatchNamespace()
			if err != nil {
				klog.Error(err, "Failed to get watch namespace")
			}

			// Get a config to talk to the apiserver
			cfg, err := config.GetConfig()
			if err != nil {
				klog.Error(err, "get kubeconfig error.")
				os.Exit(1)
			}

			ctx := context.TODO()
			// Become the leader before proceeding
			err = leader.Become(ctx, "mesh-operator-lock")
			if err != nil {
				klog.Error(err, "")
				os.Exit(1)
			}

			// Set default manager options
			options := manager.Options{
				Namespace:          namespace,
				MetricsBindAddress: fmt.Sprintf("%s:%d", opt.MetricsHost, opt.MetricsPort),
			}
			if strings.Contains(namespace, ",") {
				options.Namespace = ""
				options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
			}

			// Create a new manager to provide shared dependencies and start components
			mgr, err := manager.New(cfg, options)
			if err != nil {
				klog.Error(err, "")
				os.Exit(1)
			}

			klog.Info("Registering Components.")

			// Setup Scheme for all resources
			if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
				klog.Error(err, "")
				os.Exit(1)
			}

			// Setup all Controllers
			if err := controller.AddToManager(mgr); err != nil {
				klog.Error(err, "")
				os.Exit(1)
			}

			// Add the Metrics Service
			addMetrics(ctx, cfg, opt)

			klog.Info("Starting the Cmd.")
			// Start the Cmd
			if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
				klog.Error(err, "Manager exited non-zero")
				os.Exit(1)
			}

			// cfg, err := ropt.GetK8sConfig()
			// if err != nil {
			// 	klog.Fatalf("unable to get kubeconfig err: %v", err)
			// }

			// rp := time.Second * 120
			// mgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
			// 	Scheme:             k8sclient.GetScheme(),
			// 	MetricsBindAddress: "0",
			// 	LeaderElection:     false,
			// 	// Port:               9443,
			// 	SyncPeriod: &rp,
			// })
			// if err != nil {
			// 	klog.Fatalf("unable to new manager err: %v", err)
			// }

			// opt.MasterCli = k8smanager.MasterClient{
			// 	KubeCli: ropt.GetKubeInterfaceOrDie(),
			// 	Manager: mgr,
			// }
		},
	}
	return cmd
}

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cfg *rest.Config, opt *ControllerOption) {
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			klog.Info("Skipping CR metrics server creation; not running in a cluster.")
			return
		}
	}

	if err := serveCRMetrics(cfg, operatorNs, opt); err != nil {
		klog.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: opt.MetricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: opt.MetricsPort}},
		{Port: opt.OperatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: opt.OperatorMetricsPort}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		klog.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}

	// The ServiceMonitor is created in the same namespace where the operator is deployed
	_, err = metrics.CreateServiceMonitors(cfg, operatorNs, services)
	if err != nil {
		klog.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			klog.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://opt.MetricsHost:opt.OperatorMetricsPort".
func serveCRMetrics(cfg *rest.Config, operatorNs string, opt *ControllerOption) error {
	// The function below returns a list of filtered operator/CR specific GVKs. For more control, override the GVK list below
	// with your own custom logic. Note that if you are adding third party API schemas, probably you will need to
	// customize this implementation to avoid permissions issues.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}

	// The metrics will be generated from the namespaces which are returned here.
	// NOTE that passing nil or an empty list of namespaces in GenerateAndServeCRMetrics will result in an error.
	ns, err := kubemetrics.GetNamespacesForMetrics(operatorNs)
	if err != nil {
		return err
	}

	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, opt.MetricsHost, opt.OperatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}
