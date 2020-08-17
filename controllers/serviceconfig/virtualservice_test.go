package serviceconfig

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// . "github.com/symcn/mesh-operator/test"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("test virtualservice", func() {
	var (
		sc                *meshv1alpha1.ServiceConfig
		cs                *meshv1alpha1.ConfiguredService
		vs                *networkingv1beta1.VirtualService
		todelete          *networkingv1beta1.VirtualService
		rerouteCsNoBlue   *meshv1alpha1.ConfiguredService
		rerouteCsNoCanary *meshv1alpha1.ConfiguredService
		normalRoute       []*meshv1alpha1.Destination
	)

	BeforeEach(func() {
		normalRoute = []*meshv1alpha1.Destination{
			{Subset: "blue", Weight: 40},
			{Subset: "green", Weight: 60},
		}
		sc = &meshv1alpha1.ServiceConfig{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.reconcile.virtualservice",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ServiceConfigSpec{
				OriginalName: "Normal.Reconcile.VirtualService",
				Policy: &meshv1alpha1.Policy{
					LoadBalancer: map[string]string{
						"simple": "RANDOM",
					},
					MaxConnections: 100,
					Timeout:        "20s",
					MaxRetries:     3,
				},
				Route: normalRoute,
				Instances: []*meshv1alpha1.InstanceConfig{
					{
						Host: "1.1.1.1",
						Port: &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host: "1.1.1.2",
						Port: &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host: "1.1.1.3",
						Port: &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host: "1.1.1.4",
						Port: &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
				},
				RerouteOption: &meshv1alpha1.RerouteOption{
					ReroutePolicy: meshv1alpha1.Default,
				},
				CanaryRerouteOption: &meshv1alpha1.RerouteOption{
					ReroutePolicy: meshv1alpha1.Default,
				},
				MeshConfigGeneration: 0,
			},
		}

		cs = &meshv1alpha1.ConfiguredService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.reconcile.virtualservice",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName: "Normal.Reconcile.VirtualService",
				Instances: []*meshv1alpha1.Instance{
					{
						Host:   "1.1.1.1",
						Labels: map[string]string{"test": "ccc", "flag": "red"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host:   "1.1.1.2",
						Labels: map[string]string{"test": "ccc", "flag": "green"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host:   "1.1.1.3",
						Labels: map[string]string{"test": "ccc", "flag": "canary"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host:   "1.1.1.4",
						Labels: map[string]string{"test": "ccc", "test-group": "blue"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
				},
				MeshConfigGeneration: 0,
			},
		}

		rerouteCsNoBlue = &meshv1alpha1.ConfiguredService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.reconcile.virtualservice",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName: "Normal.Reconcile.VirtualService",
				Instances: []*meshv1alpha1.Instance{
					{
						Host:   "1.1.1.1",
						Labels: map[string]string{"test": "ccc", "flag": "red"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host:   "1.1.1.2",
						Labels: map[string]string{"test": "ccc", "flag": "green"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host:   "1.1.1.3",
						Labels: map[string]string{"test": "ccc", "flag": "canary"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
				},
				MeshConfigGeneration: 0,
			},
		}

		rerouteCsNoCanary = &meshv1alpha1.ConfiguredService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.reconcile.virtualservice",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName: "Normal.Reconcile.VirtualService",
				Instances: []*meshv1alpha1.Instance{
					{
						Host:   "1.1.1.1",
						Labels: map[string]string{"flag": "red"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host:   "1.1.1.2",
						Labels: map[string]string{"flag": "green"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host:   "1.1.1.4",
						Labels: map[string]string{"test-group": "blue"},
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
				},
				MeshConfigGeneration: 0,
			},
		}

		vs = &networkingv1beta1.VirtualService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.reconcile.virtualservice",
				Namespace: "mesh-test",
				Labels: map[string]string{
					"service": "Normal.Reconcile.VirtualService",
				},
			},
			Spec: v1beta1.VirtualService{
				Hosts: []string{"toupdate.reconcile.virtualservice"},
			},
		}

		todelete = &networkingv1beta1.VirtualService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "todelete.reconcile.virtualservice",
				Namespace: "mesh-test",
				Labels: map[string]string{
					"service": "Normal.Reconcile.VirtualService",
				},
			},
			Spec: v1beta1.VirtualService{
				Hosts: []string{"toupdate.reconcile.virtualservice"},
			},
		}
	}, timeout)

	Describe("test reroute logic", func() {
		Context("when canary group is empty", func() {
			It("and not in actural subsets, success", func() {
				r := Reconciler{
					Client:     getFakeClient(getTestMeshConfig(), sc, rerouteCsNoCanary, vs),
					Scheme:     getFakeScheme(),
					Log:        nil,
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				err := r.reconcileVirtualService(context.Background(), sc)
				Expect(err).NotTo(HaveOccurred())

				By("check updated virtualservice")
				found := &networkingv1beta1.VirtualService{}
				err = r.Get(context.Background(),
					types.NamespacedName{Namespace: "mesh-test", Name: "normal.reconcile.virtualservice"},
					found)

				By("reroute canary subset weight to 0, check")
				want := &v1beta1.HTTPRouteDestination{
					Destination: &v1beta1.Destination{
						Host:   "normal.reconcile.virtualservice",
						Subset: "canary",
					},
					Weight: 100,
				}
				Expect(err).NotTo(HaveOccurred())
				for _, r := range found.Spec.Http {
					if r.Name == "dubbo-http-route-canary" {
						Expect(r.Route).NotTo(ContainElement(want))
					}
				}
			}, timeout)
		})
		Context("when normal group is empty", func() {
			It("and not in actural subsets, success", func() {
				r := Reconciler{
					Client:     getFakeClient(getTestMeshConfig(), sc, rerouteCsNoBlue, vs),
					Scheme:     getFakeScheme(),
					Log:        nil,
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				err := r.reconcileVirtualService(context.Background(), sc)
				Expect(err).NotTo(HaveOccurred())

				By("check updated virtualservice")
				found := &networkingv1beta1.VirtualService{}
				err = r.Get(context.Background(),
					types.NamespacedName{Namespace: "mesh-test", Name: "normal.reconcile.virtualservice"},
					found)

				By("reroute blue subset weight to 0, check")
				want := &v1beta1.HTTPRouteDestination{
					Destination: &v1beta1.Destination{
						Host:   "normal.reconcile.virtualservice",
						Subset: "blue",
					},
					Weight: 0,
				}
				Expect(err).NotTo(HaveOccurred())
				for _, r := range found.Spec.Http {
					if r.Name == "dubbo-http-route-blue" {
						for _, route := range r.Route {
							if route.Destination.Subset == "blue" {
								Expect(route).To(Equal(want))
							}
						}
					}
				}
			}, timeout)
		})
	})

	Describe("without mock client", func() {
		Context("reconcile virtualservice", func() {
			It("create virtualservice success", func() {
				r := Reconciler{
					Client:     getFakeClient(getTestMeshConfig(), sc, cs),
					Scheme:     getFakeScheme(),
					Log:        nil,
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				err := r.reconcileVirtualService(context.Background(), sc)
				Expect(err).NotTo(HaveOccurred())

				By("check created virtualservice")
				found := &networkingv1beta1.VirtualService{}
				err = r.Get(
					context.Background(),
					types.NamespacedName{Namespace: "mesh-test", Name: "normal.reconcile.virtualservice"},
					found)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.Name).To(Equal("normal.reconcile.virtualservice"))
			}, timeout)

			It("update virtualservice success", func() {
				r := Reconciler{
					Client:     getFakeClient(getTestMeshConfig(), sc, cs, vs, todelete),
					Scheme:     getFakeScheme(),
					Log:        nil,
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				err := r.reconcileVirtualService(context.Background(), sc)
				Expect(err).NotTo(HaveOccurred())

				By("check updated virtualservice")
				found := &networkingv1beta1.VirtualService{}
				err = r.Get(
					context.Background(),
					types.NamespacedName{Namespace: "mesh-test", Name: "normal.reconcile.virtualservice"},
					found)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.Name).To(Equal("normal.reconcile.virtualservice"))
				Expect(found.Spec.Hosts).To(ContainElement("normal.reconcile.virtualservice"))

				By("check deleted unused virtualservice")
				err = r.Get(
					context.Background(),
					types.NamespacedName{Namespace: "mesh-test", Name: "todelete.reconcile.virtualservice"},
					found)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(Equal(true))
			})

		})

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
