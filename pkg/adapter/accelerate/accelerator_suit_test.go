package accelerate

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/symcn/meshach/pkg/adapter/component"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
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

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
})
