/*
Copyright 2019 The dks authors.

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
	"time"

	k8sclient "github.com/mesh-operator/pkg/k8s/client"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
	zk "github.com/mesh-operator/pkg/zookeeper"
	"github.com/spf13/cobra"
	"k8s.io/klog"
	"k8s.io/sample-controller/pkg/signals"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewAdapterCmd ...
func NewAdapterCmd(cli *DksCli) *cobra.Command {
	opt := zk.DefaultOption()
	cmd := &cobra.Command{
		Use:     "adapter",
		Aliases: []string{"adapter"},
		Short:   "Adapters configured for different registry center",
		Run: func(cmd *cobra.Command, args []string) {
			PrintFlags(cmd.Flags())

			cfg, err := cli.GetK8sConfig()
			if err != nil {
				klog.Fatalf("unable to get kubeconfig err: %v", err)
			}

			rp := time.Second * 120
			mgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
				Scheme:             k8sclient.GetScheme(),
				MetricsBindAddress: "0",
				LeaderElection:     false,
				// Port:               9443,
				SyncPeriod: &rp,
			})
			if err != nil {
				klog.Fatalf("unable to new manager err: %v", err)
			}

			opt.MasterCli = k8smanager.MasterClient{
				KubeCli: cli.GetKubeInterfaceOrDie(),
				Manager: mgr,
			}

			adapter, err := zk.NewAdapter(opt)
			if err != nil {
				klog.Fatalf("unable to NewAdapter err: %v", err)
			}

			stopCh := signals.SetupSignalHandler()
			mgr.Add(adapter)
			mgr.Add(adapter.K8sMgr)

			klog.Info("starting manager")
			if err := mgr.Start(stopCh); err != nil {
				klog.Fatalf("problem start running manager err: %v", err)
			}
		},
	}

	cmd.PersistentFlags().StringArrayVar(
		&opt.Address,
		"zk-addr",
		opt.Address,
		"the zookeeper address pool")

	cmd.PersistentFlags().StringVar(
		&opt.Root,
		"zk-root",
		opt.Root,
		"the zookeeper root")

	cmd.PersistentFlags().Int64Var(
		&opt.Timeout,
		"zk-timeout",
		opt.Timeout,
		"the zookeeper session timeout second")

	cmd.PersistentFlags().StringVar(
		&opt.ClusterOwner,
		"cluster-owner",
		opt.ClusterOwner,
		"the labels that multiple cluster manager used for select clusters")

	cmd.PersistentFlags().StringVar(
		&opt.ClusterNamespace,
		"cluster-namespace",
		opt.ClusterNamespace,
		"the namesapce that multiple cluster manager uses when selecting the cluster configmaps")
	return cmd
}
