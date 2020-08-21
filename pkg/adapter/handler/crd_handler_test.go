package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/symcn/mesh-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var _ = Describe("Test cases for crd handler.", func() {
	BeforeEach(func() {
		klog.Infof("before process for each context")
	})

	Describe("Test for starting", func() {
		Context("createConfiguredService", func() {
			It("", func() {
				klog.Info("start to test method: #createConfiguredService()")
				cs := &v1.ConfiguredService{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       v1.ConfiguredServiceSpec{},
					Status:     v1.ConfiguredServiceStatus{},
				}

				err := createConfiguredService(cs, k8sClient)
				Expect(err).To(HaveOccurred())
			})
		})

	})

	AfterEach(func() {
		klog.Infof("after process for each context")
	})

})
