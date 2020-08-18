package adapter

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/symcn/mesh-operator/pkg/option"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Adapter suite", []Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter))
	By("bootstrapping")
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("Tearing down")
})

var testOpt = &option.AdapterOption{
	EventHandlers: option.EventHandlers{
		EnableK8s:        true,
		IsMultiClusters:  false,
		ClusterOwner:     "sym-admin",
		ClusterNamespace: "sym-admin",
		EnableDebugLog:   false,
	},
	Registry: option.Registry{
		Type:    "zk",
		Address: []string{"devzk.dmall.com:2181"},
		Timeout: 0,
	},
	Configuration: option.Configuration{
		Type:    "zk",
		Address: []string{"devzk.dmall.com:2181"},
		Timeout: 15,
	},
}
