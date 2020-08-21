package zk

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"k8s.io/klog"
)

var _ = Describe("Test cases for zookeeper configuration center.", func() {
	var configCenter component.ConfigurationCenter
	var err error
	configCenter, err = New(*testOpt)
	Expect(err).NotTo(HaveOccurred())

	err = configCenter.Start()
	Expect(err).NotTo(HaveOccurred())

	BeforeEach(func() {
		klog.Infof("before process for each context")
	})

	Describe("Test for starting", func() {
		Context("consuming config events", func() {
			It("Consumer an event from the event channel", func() {
				klog.Info("ignite the registry")
				event := <-configCenter.Events()
				Expect(event).ShouldNot(BeNil())
				Expect(event.EventType).ShouldNot(BeNil())
			})
		})

	})

	AfterEach(func() {
		klog.Infof("after process for each context")
	})

})
