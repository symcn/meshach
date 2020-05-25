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

package client

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// GetConfigWithContext returns kubernetes config based on the current environment.
// If fpath is provided, loads configuration from that file. Otherwise,
// GetConfig uses default strategy to load configuration from $KUBECONFIG,
// .kube/config, or just returns in-cluster config.
func GetConfigWithContext(kubeconfigPath, kubeContext string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		rules.ExplicitPath = kubeconfigPath
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: kubeContext}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
}

// GetRawConfig creates a raw clientcmd api config
func GetRawConfig(kubeconfigPath, kubeContext string) (api.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		rules.ExplicitPath = kubeconfigPath
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: kubeContext}
	clientConfig := clientcmd.
		NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	return clientConfig.RawConfig()
}

// GetConfig ...
func GetConfig(kubeconfigPath string) (*rest.Config, error) {
	// If a flag is specified with the config location, use that
	if len(kubeconfigPath) > 0 {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	// If an env variable is specified with the config locaiton, use that
	if len(os.Getenv("KUBECONFIG")) > 0 {
		return clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	}
	// If no explicit location, try the in-cluster config
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory
	if usr, err := user.Current(); err == nil {
		if c, err := clientcmd.BuildConfigFromFlags(
			"", filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("could not locate a kubeconfig")
}

// NewClientConfig creates a Kubernetes client config from raw kube config.
func NewClientConfig(kubeConfig []byte) (*rest.Config, error) {
	if kubeConfig == nil {
		return nil, errors.New("kube config is empty")
	}
	apiconfig, err := clientcmd.Load(kubeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kubernetes API config")
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build client config from API config")
	}

	// Adjust our client's rate limits based on the number of controllers we are running.
	if cfg.QPS == 0.0 {
		cfg.QPS = 40.0
		cfg.Burst = 60
	}

	return cfg, nil
}

// NewClientFromConfig creates a new Kubernetes client from config.
func NewClientFromConfig(config *rest.Config) (kubernetes.Interface, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client for config")
	}

	return client, nil
}
