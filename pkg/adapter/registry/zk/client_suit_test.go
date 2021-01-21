package zk

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/symcn/meshach/pkg/option"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Registry test suite", []Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter))
	By("Test case for registry is bootstrapping")
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("Tearing down")
})

var testOpt = &option.Registry{
	Type:    "zk",
	Address: []string{"devzk.dmall.com:2181"},
	Timeout: 30000,
}
