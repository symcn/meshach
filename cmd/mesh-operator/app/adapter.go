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

	zk "github.com/mesh-operator/pkg/zookeeper"
	"github.com/spf13/cobra"
)

// NewAdapterCmd ...
func NewAdapterCmd() *cobra.Command {
	opt := zk.DefaultOption()
	cmd := &cobra.Command{
		Use:     "adapter",
		Aliases: []string{"adapter"},
		Short:   "Adapters configured for different registry center",
		Run: func(cmd *cobra.Command, args []string) {
			PrintFlags(cmd.Flags())
			adapter, _ := zk.NewAdapter(opt)
			stop := make(chan struct{})
			adapter.Run(stop)

			time.Sleep(30 * time.Minute)
			stop <- struct{}{}
			time.Sleep(5 * time.Second)
		},
	}
	cmd.PersistentFlags().StringArrayVar(&opt.Address, "zk-addr", opt.Address, "the zookeeper address pool")
	cmd.PersistentFlags().StringVar(&opt.Root, "zk-root", opt.Root, "the zookeeper root")
	cmd.PersistentFlags().Int64Var(&opt.Timeout, "zk-timeout", opt.Timeout, "the zookeeper session timeout second")
	return cmd
}
