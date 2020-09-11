/*


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

package configuredservice

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/option"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestConfiguredService(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"ConfiguredService Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "config", "crd", "istio"),
		},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = meshv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = networkingv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var testOpt = &option.ControllerOption{
	HTTPAddress:             ":8080",
	SyncPeriod:              120,
	MetricsEnabled:          true,
	GinLogEnabled:           true,
	GinLogSkipPath:          []string{"/ready", "/live"},
	EnableLeaderElection:    true,
	LeaderElectionID:        "mesh-operator-test-lock",
	LeaderElectionNamespace: "mesh-test",
	PprofEnabled:            true,
	GoroutineThreshold:      1000,
	ProxyHost:               "mesh.test.proxy",
	ProxyAttempts:           3,
	ProxyPerTryTimeout:      2,
	ProxyRetryOn:            "gateway-error,connect-failure,refused-stream",
	MeshConfigName:          "mc-test-case",
	MeshConfigNamespace:     "mesh-test",
	SelectLabel:             "service",
	EnableAppMeshConfig:     true,
	EnableConfiguredService: true,
	EnableIstioConfig:       true,
	EnableMeshConfig:        true,
	EnableServiceAccessor:   true,
	EnableServiceConfig:     true,
	WatchIstioCRD:           true,
}

func getTestMeshConfig() *meshv1alpha1.MeshConfig {
	return &meshv1alpha1.MeshConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mc-test-case",
			Namespace: "mesh-test",
		},
		Spec: meshv1alpha1.MeshConfigSpec{
			MatchSourceLabelKeys:   []string{"test-group"},
			WorkloadEntryLabelKeys: []string{"test-group"},
			MeshLabelsRemap: map[string]string{
				"flag": "test-group",
			},
			GlobalSubsets: []*meshv1alpha1.Subset{blueSubset, greenSubset},
			GlobalPolicy: &meshv1alpha1.Policy{
				LoadBalancer: map[string]string{
					"simple": "ROUND_ROBIN",
				},
				MaxConnections: 100,
				Timeout:        "5s",
				MaxRetries:     3,
			},
		},
	}
}

var (
	blueSubset = &meshv1alpha1.Subset{
		Name: "blue",
		Policy: &meshv1alpha1.Policy{
			LoadBalancer: map[string]string{
				"simple": "aa",
			},
			MaxConnections: 10,
			Timeout:        "10s",
			MaxRetries:     2,
		},
		Labels: map[string]string{"test-group": "blue"},
	}
	greenSubset = &meshv1alpha1.Subset{
		Name: "green",
		Policy: &meshv1alpha1.Policy{
			LoadBalancer: map[string]string{
				"simple": "aa",
			},
			MaxConnections: 10,
			Timeout:        "10s",
			MaxRetries:     2,
		},
		Labels: map[string]string{"test-group": "green"},
	}
	canarySubset = &meshv1alpha1.Subset{
		Name: "canary",
		Policy: &meshv1alpha1.Policy{
			LoadBalancer: map[string]string{
				"simple": "aa",
			},
			MaxConnections: 10,
			Timeout:        "10s",
			MaxRetries:     2,
		},
		Labels:   map[string]string{"test-group": "green"},
		IsCanary: true,
	}
	redSubset = &meshv1alpha1.Subset{
		Name: "red",
		Policy: &meshv1alpha1.Policy{
			LoadBalancer: map[string]string{
				"simple": "aa",
			},
			MaxConnections: 10,
			Timeout:        "10s",
			MaxRetries:     2,
		},
		Labels: map[string]string{"test-group": "red"},
	}
)

// getFakeClient return a fake client to mock API calls.
func getFakeClient(objs ...runtime.Object) client.Client {
	return fake.NewFakeClient(objs...)
}

func getFakeReconciler(objs ...runtime.Object) *Reconciler {
	return &Reconciler{
		Client:     getFakeClient(objs...),
		Log:        ctrl.Log.WithName("controllers").WithName("configuredservice"),
		Scheme:     getFakeScheme(),
		Opt:        testOpt,
		MeshConfig: getTestMeshConfig(),
	}
}

func getMockClientReconcile(mockClient client.Client) *Reconciler {
	return &Reconciler{
		Client:     mockClient,
		Log:        ctrl.Log.WithName("controllers").WithName("configuredservice"),
		Scheme:     getFakeScheme(),
		Opt:        testOpt,
		MeshConfig: getTestMeshConfig(),
	}
}

// getFakeScheme register operator types with the runtime scheme.
func getFakeScheme() *runtime.Scheme {
	s := scheme.Scheme
	err := meshv1alpha1.AddToScheme(s)
	if err != nil {
		klog.Errorf("[test] add meshv1alpha1 to scheme error: %+v", err)
		return s
	}

	err = networkingv1beta1.AddToScheme(s)
	if err != nil {
		klog.Errorf("[test] add networkingv1beta1 to scheme error: %+v", err)
		return s
	}
	return s
}
