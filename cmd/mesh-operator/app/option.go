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
	k8sclient "github.com/mesh-operator/pkg/k8s/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

// RootOption ...
type RootOption struct {
	Kubeconfig       string
	ConfigContext    string
	Namespace        string
	DefaultNamespace string
	DevelopmentMode  bool
}

// DefaultRootOption ...
func DefaultRootOption() *RootOption {
	return &RootOption{
		Namespace:       corev1.NamespaceAll,
		DevelopmentMode: true,
	}
}

// DksCli ...
type DksCli struct {
	RootCmd *cobra.Command
	Opt     *RootOption
}

// NewDksCli ...
func NewDksCli(opt *RootOption) *DksCli {
	return &DksCli{
		Opt: opt,
	}
}

// GetK8sConfig ...
func (c *DksCli) GetK8sConfig() (*rest.Config, error) {
	config, err := k8sclient.GetConfigWithContext(c.Opt.Kubeconfig, c.Opt.ConfigContext)
	if err != nil {
		return nil, errors.Wrap(err, "could not get k8s config")
	}

	return config, nil
}

// GetKubeInterface ...
func (c *DksCli) GetKubeInterface() (kubernetes.Interface, error) {
	cfg, err := c.GetK8sConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not get k8s config")
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}

	return kubeCli, nil
}

// GetKubeInterfaceOrDie ...
func (c *DksCli) GetKubeInterfaceOrDie() kubernetes.Interface {
	kubeCli, err := c.GetKubeInterface()
	if err != nil {
		klog.Fatalf("unable to get kube interface err: %v", err)
	}

	return kubeCli
}
