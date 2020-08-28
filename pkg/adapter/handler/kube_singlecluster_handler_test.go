package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/symcn/mesh-operator/pkg/adapter/types"
	"k8s.io/klog"
)

var _ = Describe("Test cases for the handler of a single cluster ", func() {
	Describe("Process a replacing event", func() {
		Context("", func() {
			event := types.ServiceEvent{
				EventType: types.ServiceAdded,
				Service: &types.Service{
					Name: "foo",
					//Ports:     nil,
					//Instances: nil,
				},
				//Instance:  nil,
				//Instances: nil,
			}

			BeforeEach(func() {

			})

			AfterEach(func() {
				klog.Infof("After process for each Describe")
			})

			It("", func() {
				klog.Infof("Before process for each Describe")
				singleClusterHandler.ReplaceInstances(&event)
				foundCs, err := getConfiguredService(defaultNamespace, "foo", k8sClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(foundCs).ToNot(BeNil())
				Expect(foundCs.ObjectMeta.Name).To(Equal("foo"))
			})
		})

	})

})
