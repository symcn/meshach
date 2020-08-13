package serviceconfig

import (
	"context"
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	. "github.com/symcn/mesh-operator/test"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("test reconcile destinationrule", func() {
	var (
		mockCtrl    *gomock.Controller
		mockClient  *MockClient
		sc          *meshv1alpha1.ServiceConfig
		cs          *meshv1alpha1.ConfiguredService
		dr          *networkingv1beta1.DestinationRule
		todelete    *networkingv1beta1.DestinationRule
		normalRoute []*meshv1alpha1.Destination
	)

	BeforeEach(func() {
		normalRoute = []*meshv1alpha1.Destination{
			{Subset: "blue", Weight: 40},
			{Subset: "green", Weight: 60},
		}
		sc = &meshv1alpha1.ServiceConfig{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.reconcile.destinationrule",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ServiceConfigSpec{
				OriginalName: "Normal.Reconcile.DestinationRule",
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
				Name:      "normal.reconcile.destinationrule",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName: "Normal.Reconcile.DestinationRule",
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
						Host:   "1.1.1.3",
						Labels: map[string]string{"flag": "canary"},
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

		dr = &networkingv1beta1.DestinationRule{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.reconcile.destinationrule",
				Namespace: "mesh-test",
				Labels: map[string]string{
					testOpt.SelectLabel: "Normal.Reconcile.DestinationRule",
				},
			},
			Spec: v1beta1.DestinationRule{
				Host: "test.host",
			},
		}

		todelete = &networkingv1beta1.DestinationRule{
			ObjectMeta: v1.ObjectMeta{
				Name:      "todelete.reconcile.destinationrule",
				Namespace: "mesh-test",
				Labels: map[string]string{
					testOpt.SelectLabel: "Normal.Reconcile.DestinationRule",
				},
			},
			Spec: v1beta1.DestinationRule{
				Host: "test.host",
			},
		}
	}, timeout)

	Describe("occured error when get subset", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("get cs error"))
			mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		}, timeout)

		It("occurred get cs error", func() {
			r := Reconciler{
				Client:     mockClient,
				Scheme:     getFakeScheme(),
				Log:        nil,
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			err := r.reconcileDestinationRule(context.Background(), sc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("get cs error"))
		})
	})

	Describe("test destinationrule without mock client", func() {
		It("create successful", func() {
			r := Reconciler{
				Client:     getFakeClient(getTestMeshConfig(), sc, cs),
				Scheme:     getFakeScheme(),
				Log:        nil,
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			err := r.reconcileDestinationRule(context.Background(), sc)
			Expect(err).NotTo(HaveOccurred())

			found := &networkingv1beta1.DestinationRule{}
			err = r.Get(
				context.Background(),
				types.NamespacedName{
					Namespace: "mesh-test",
					Name:      "normal.reconcile.destinationrule",
				},
				found,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(found.Name).To(Equal("normal.reconcile.destinationrule"))
		}, timeout)

		It("update successful and delete unused destinationrule", func() {
			r := Reconciler{
				Client:     getFakeClient(getTestMeshConfig(), sc, cs, dr, todelete),
				Scheme:     getFakeScheme(),
				Log:        nil,
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			err := r.reconcileDestinationRule(context.Background(), sc)
			Expect(err).NotTo(HaveOccurred())

			By("check updated destinationrule")
			found := &networkingv1beta1.DestinationRule{}
			err = r.Get(
				context.Background(),
				types.NamespacedName{
					Namespace: "mesh-test",
					Name:      "normal.reconcile.destinationrule",
				},
				found,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(found.Name).To(Equal("normal.reconcile.destinationrule"))
			Expect(found.Spec.Host).To(Equal("normal.reconcile.destinationrule"))

			By("check deleted destinationrule")
			err = r.Get(
				context.Background(),
				types.NamespacedName{
					Namespace: "mesh-test",
					Name:      "todelete.reconcile.destinationrule",
				},
				found,
			)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(Equal(true))
		}, timeout)

		It("occurred error when set controller reference", func() {
			r := Reconciler{
				Client:     getFakeClient(getTestMeshConfig(), sc, cs),
				Log:        nil,
				Scheme:     runtime.NewScheme(),
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}

			err := r.reconcileDestinationRule(context.Background(), sc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(`no kind is registered for the type v1alpha1.ServiceConfig in scheme "pkg/runtime/scheme.go:101"`))
		})

		Context("test compare destinationrule", func() {
			It("return true while finalizers is not equal", func() {
				new := &networkingv1beta1.DestinationRule{ObjectMeta: v1.ObjectMeta{Finalizers: []string{"test.finalizers.a"}}}
				old := &networkingv1beta1.DestinationRule{}
				result := compareDestinationRule(new, old)
				Expect(result).To(Equal(true))
			}, timeout)

			It("return true while labels is not equal", func() {
				m := map[string]string{"app": "test"}
				new := &networkingv1beta1.DestinationRule{ObjectMeta: v1.ObjectMeta{Labels: m}}
				old := &networkingv1beta1.DestinationRule{}
				result := compareDestinationRule(new, old)
				Expect(result).To(Equal(true))
			}, timeout)

			It("return false", func() {
				se := &networkingv1beta1.DestinationRule{}
				result := compareDestinationRule(se, se)
				Expect(result).To(Equal(false))
			}, timeout)
		})

		Context("test getlb", func() {
			It("can not found in lbMap", func() {
				result := getlb("test-lb")
				Expect(result).To(Equal(v1beta1.LoadBalancerSettings_RANDOM))
			})
		})
	})
})
