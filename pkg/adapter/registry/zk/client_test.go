package zk

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/symcn/meshach/pkg/adapter/component"
	"k8s.io/klog"
)

var _ = Describe("Test cases for zookeeper registry.", func() {
	var registry component.Registry
	var err error
	registry, err = New(*testOpt)
	Expect(err).NotTo(HaveOccurred())

	err = registry.Start()
	Expect(err).NotTo(HaveOccurred())

	BeforeEach(func() {
		klog.Infof("before process for each context")
	})

	Describe("Test for starting", func() {
		Context("consuming service event", func() {
			It("Consumer an event from the event channel", func() {
				klog.Info("ignite the registry")
				event := <-registry.ServiceEvents()
				Expect(event).ShouldNot(BeNil())
				Expect(event.EventType).ShouldNot(BeNil())
			})
		})

		Context("Consuming accessor events", func() {
			It("Consumer an event from the event channel", func() {
				klog.Info("ignite the registry")
				event := <-registry.AccessorEvents()
				Expect(event).ShouldNot(BeNil())
				Expect(event.EventType).ShouldNot(BeNil())
			})
		})
	})

	AfterEach(func() {
		klog.Infof("after process for each context")
	})

})
