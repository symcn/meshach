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
		klog.Infof("before process for each Describe")
	})

	Describe("Tests for ConfigurationService", func() {
		var cs *v1.ConfiguredService

		Context("No namespace", func() {
			BeforeEach(func() {
				cs = &v1.ConfiguredService{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       v1.ConfiguredServiceSpec{},
				}
			})

			It("get", func() {
				klog.Info("start to test method: #getConfiguredService()")
				foundCs, err := getConfiguredService(cs.Name, cs.Namespace, k8sClient)
				Expect(err).To(HaveOccurred())
				Expect(foundCs.ObjectMeta.Name).To(BeZero())
			})

			It("create", func() {
				klog.Info("start to test method: #createConfiguredService()")
				err := createConfiguredService(cs, k8sClient)
				Expect(err).To(HaveOccurred())
			})

			It("update", func() {
				klog.Info("start to test method: #updateConfiguredService()")
				err := updateConfiguredService(cs, k8sClient)
				Expect(err).To(HaveOccurred())
			})

			It("delete", func() {
				klog.Info("start to test method: #deleteConfiguredService()")
				err := deleteConfiguredService(cs, k8sClient)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Successful progress", func() {
			var cs *v1.ConfiguredService

			BeforeEach(func() {
				cs = &v1.ConfiguredService{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "default",
					},
					Spec: v1.ConfiguredServiceSpec{
						OriginalName:         "Foo",
						Instances:            nil,
						MeshConfigGeneration: 0,
					},
				}

				createConfiguredService(cs, k8sClient)
			})

			It("Retrieving the exist cs", func() {
				foundSc, err := getConfiguredService("default", "foo", k8sClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(foundSc.Spec.OriginalName).Should(Equal("Foo"))
			})

			It("Deleting the exist cs", func() {
				err := deleteConfiguredService(cs, k8sClient)
				Expect(err).ToNot(HaveOccurred())

				foundSc, err := getConfiguredService("default", "foo", k8sClient)
				// could not retrieve the cs
				Expect(err).To(HaveOccurred())
				Expect(foundSc.Name).Should(BeZero())
			})

			AfterEach(func() {
				deleteConfiguredService(cs, k8sClient)
			})
		})
	})

	Describe("Tests for ServiceConfig", func() {
		Context("No namespace", func() {
			var sc *v1.ServiceConfig

			BeforeEach(func() {
				sc = &v1.ServiceConfig{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
				}
			})

			It("get", func() {
				klog.Info("start to test method: #getServiceConfig()")
				foundCs, err := getServiceConfig(sc.Name, sc.Namespace, k8sClient)
				Expect(err).To(HaveOccurred())
				Expect(foundCs.ObjectMeta.Name).To(BeZero())
			})

			It("create", func() {
				klog.Info("start to test method: #createServiceConfig()")
				err := createServiceConfig(sc, k8sClient)
				Expect(err).To(HaveOccurred())
			})

			It("update", func() {
				klog.Info("start to test method: #updateServiceConfig()")
				err := updateServiceConfig(sc, k8sClient)
				Expect(err).To(HaveOccurred())
			})

			It("delete", func() {
				klog.Info("start to test method: #deleteServiceConfig()")
				err := deleteServiceConfig(sc, k8sClient)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Successful progress", func() {
			var sc *v1.ServiceConfig

			BeforeEach(func() {
				sc = &v1.ServiceConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "default",
					},
					Spec: v1.ServiceConfigSpec{OriginalName: "Bar"},
				}

				err := createServiceConfig(sc, k8sClient)
				klog.Errorf("creating before each It has an error: %v", err)
			})

			It("get", func() {
				klog.Info("start to test method: #getServiceConfig()")
				foundSc, err := getServiceConfig(sc.Namespace, sc.Name, k8sClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(foundSc.ObjectMeta.Name).To(Equal("bar"))
			})

			It("create", func() {
				klog.Info("start to test method: #createServiceConfig()")
				err := createServiceConfig(sc, k8sClient)
				Expect(err).To(HaveOccurred())
			})

			It("delete", func() {
				err := deleteServiceConfig(sc, k8sClient)
				Expect(err).ToNot(HaveOccurred())

				_, err = getServiceConfig(sc.Name, sc.Namespace, k8sClient)
				Expect(err).To(HaveOccurred())
			})

			AfterEach(func() {
				deleteServiceConfig(sc, k8sClient)
			})
		})
	})

	AfterEach(func() {
		klog.Infof("after process for each context")
	})

})
