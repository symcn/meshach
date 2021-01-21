package handler

import (
	"github.com/symcn/meshach/pkg/adapter/component"
	"github.com/symcn/meshach/pkg/adapter/convert"
	k8sclient "github.com/symcn/meshach/pkg/k8s/client"
	"github.com/symcn/meshach/pkg/utils"
	"k8s.io/klog"
	"path/filepath"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	meshv1alpha1 "github.com/symcn/meshach/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var singleClusterHandler component.EventHandler

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Registry test suite", []Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "config", "crd", "bases"),
			// filepath.Join("..", "..", "..", "config", "crd", "istio"),
		},
	}

	var err error
	// If you plan to start testEnv you should install kubebuilder first to this path below:
	// /usr/local/kubebuilder/bin
	// etcd
	// kube-apiserver
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = meshv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// err = networkingv1beta1.AddToScheme(scheme.Scheme)
	// Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	// initializing control manager with the config
	rp := time.Second * 120
	ctrlMgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
		Scheme:             k8sclient.GetScheme(),
		MetricsBindAddress: "0",
		LeaderElection:     false,
		// Port:               9443,
		SyncPeriod: &rp,
	})
	if err != nil {
		klog.Fatalf("unable to create a manager, err: %v", err)
	}

	klog.Info("starting the control manager")
	stopCh := utils.SetupSignalHandler()
	go func() {
		if err := ctrlMgr.Start(stopCh); err != nil {
			klog.Fatalf("problem start running manager err: %v", err)
		}
	}()
	for !ctrlMgr.GetCache().WaitForCacheSync(stopCh) {
		klog.Warningf("Waiting for caching objects to informer")
		time.Sleep(5 * time.Second)
	}
	klog.Infof("caching objects to informer is successful")

	singleClusterHandler, err = NewKubeSingleClusterEventHandler(ctrlMgr, &convert.DubboConverter{DefaultNamespace: defaultNamespace})
	Expect(err).ToNot(HaveOccurred())
	Expect(singleClusterHandler).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
