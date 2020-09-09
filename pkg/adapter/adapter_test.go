package adapter

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "github.com/symcn/mesh-operator/pkg/adapter/configcenter/zk"
	_ "github.com/symcn/mesh-operator/pkg/adapter/registry/zk"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Launching a test case via command line:
// go run main.go adapter --is-multi-clusters false --reg-addr devzk.dmall.com:2181 --conf-addr devzk.dmall.com:2181 --acc-size 5 -v 4

var _ = Describe("Test cases for adapter", func() {
	var adapter *Adapter
	var err error
	var done chan struct{}
	BeforeEach(
		func() {
			logf.Log.Info("Start testing the adapter")
			// configcenter.Registry("zk", New)
			adapter, err = NewAdapter(testOpt)
			Expect(err).ToNot(HaveOccurred())

			done = make(chan struct{})
		})

	Describe("Testing creating a new adapter", func() {
		JustBeforeEach(func() {
			logf.Log.Info("Before creating an adapter")
		}, 5)

		AfterEach(func() {
			logf.Log.Info("After creating an adapter")
			// done <- struct{}{}
		}, 5)

		Context("Creating context", func() {
			It("creating", func() {
				logf.Log.Info("Creating a new adapter")
				error := adapter.Start(done)
				Expect(error).ToNot(HaveOccurred())
			})
		})
	})

})
