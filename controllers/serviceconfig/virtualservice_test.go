package serviceconfig

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	// . "github.com/symcn/mesh-operator/test"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("test virtualservice", func() {
	Describe("without mock client", func() {
		Context("test compare virtualservice", func() {
			It("return true while finalizers is not equal", func() {
				new := &networkingv1beta1.VirtualService{ObjectMeta: v1.ObjectMeta{Finalizers: []string{"test.finalizers.a"}}}
				old := &networkingv1beta1.VirtualService{}
				result := compareVirtualService(new, old)
				Expect(result).To(Equal(true))
			}, timeout)

			It("return true while labels is not equal", func() {
				m := map[string]string{"app": "test"}
				new := &networkingv1beta1.VirtualService{ObjectMeta: v1.ObjectMeta{Labels: m}}
				old := &networkingv1beta1.VirtualService{}
				result := compareVirtualService(new, old)
				Expect(result).To(Equal(true))
			}, timeout)

			It("return false", func() {
				se := &networkingv1beta1.VirtualService{}
				result := compareVirtualService(se, se)
				Expect(result).To(Equal(false))
			}, timeout)
		})
	})
})
