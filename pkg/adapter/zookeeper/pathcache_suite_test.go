package zookeeper

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/symcn/mesh-operator/pkg/option"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "PathCache test suite", []Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter))
	By("Test case for the functions of PathCache")
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("Tearing down")
})

var testOpt = &option.Registry{
	Type:    "zk",
	Address: []string{"10.12.214.41:2181"},
	Timeout: 30000,
}
