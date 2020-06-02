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

// Package app contains all the commands that operator need to start.
package app

import (
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog"
)

// GetRootCmd returns the root of the cobra command-tree.
func GetRootCmd(args []string) *cobra.Command {
	opt := DefaultRootOption()
	rootCmd := &cobra.Command{
		Use:               "operator",
		Short:             "It's in charge of all things about implements of Service Mesh",
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		Run:               runHelp,
	}

	rootCmd.SetArgs(args)
	rootCmd.PersistentFlags().StringVarP(&opt.Kubeconfig, "kubeconfig", "c", "", "Kubernetes configuration file")
	rootCmd.PersistentFlags().StringVar(&opt.ConfigContext, "context", "", "The name of the kubeconfig context to use")
	rootCmd.PersistentFlags().StringVarP(&opt.Namespace, "namespace", "n", opt.Namespace, "Config namespace")

	// Make sure that klog logging variables are initialized so that we can
	// update them from this file.
	klog.InitFlags(nil)

	// Make sure klog (used by the client-go dependency) logs to stderr, as it
	// will try to log to directories that may not exist in the cilium-operator
	// container (/tmp) and cause the cilium-operator to exit.
	flag.Set("logtostderr", "true")

	AddFlags(rootCmd)
	rootCmd.AddCommand(NewAdapterCmd(opt))
	rootCmd.AddCommand(NewControllerCmd(opt))
	rootCmd.AddCommand(NewCmdVersion())
	return rootCmd
}

// AddFlags ...
func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

// PrintFlags logs the flags in the flagset
func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		klog.Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}
