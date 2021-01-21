package serviceconfig

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	meshv1alpha1 "github.com/symcn/meshach/api/v1alpha1"
	. "github.com/symcn/meshach/test"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var timeout = 5.0

var _ = Describe("Controller", func() {
	var (
		mockCtrl            *gomock.Controller
		mockClient          *MockClient
		mockStatusWriter    *MockStatusWriter
		errReq              ctrl.Request
		normalReq           ctrl.Request
		normalServiceConfig *meshv1alpha1.ServiceConfig
		normalRoute         []*meshv1alpha1.Destination
	)

	BeforeEach(func() {
		normalReq = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "mesh-test",
				Name:      "normal.reconcile.serviceconfig",
			},
		}
		normalRoute = []*meshv1alpha1.Destination{
			{Subset: "blue", Weight: 40},
			{Subset: "green", Weight: 60},
		}
		normalServiceConfig = &meshv1alpha1.ServiceConfig{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.reconcile.serviceconfig",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ServiceConfigSpec{
				OriginalName: "Normal.Reconcile.ServiceConfig",
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
	}, timeout)

	Describe("test reconcile with out mock client", func() {
		Context("error occured", func() {
			It("cannot get meshconfig", func() {
				r := Reconciler{
					Client:     getFakeClient(normalServiceConfig),
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				result, err := r.Reconcile(normalReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("meshconfigs.mesh.symcn.com \"mc-test-case\" not found"))
			}, timeout)

			It("connot get serviceconfig", func() {
				r := Reconciler{
					Client:     getFakeClient(getTestMeshConfig()),
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				result, err := r.Reconcile(errReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).NotTo(HaveOccurred())
			}, timeout)
		})
		Context("return success", func() {
			It("update workloadentry/virtualservice/destinationrule", func() {
				r := Reconciler{
					Client:     getFakeClient(getTestMeshConfig(), normalServiceConfig),
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				result, err := r.Reconcile(normalReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).NotTo(HaveOccurred())

			}, timeout)
		})

		Context("setup with controller manager", func() {
			It("error not occurred", func() {
				r := Reconciler{
					Client:     getFakeClient(getTestMeshConfig(), normalServiceConfig),
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}

				mgr, err := manager.New(cfg, manager.Options{})
				Expect(err).NotTo(HaveOccurred())

				err = r.SetupWithManager(mgr)
				Expect(err).NotTo(HaveOccurred())
			}, timeout)
		})
	})

	Describe("test reconcile used mock client - Get", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockClient.EXPECT().
				Get(gomock.Any(), gomock.Any(), gomock.Not(&meshv1alpha1.MeshConfig{})).
				Return(errors.New("get serviceconfig error"))
		}, timeout)

		AfterEach(func() {
			mockCtrl.Finish()
		}, timeout)

		It("error occurred when get serviceconfig", func() {
			r := Reconciler{
				Client:     mockClient,
				Log:        nil,
				Scheme:     getFakeScheme(),
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			result, err := r.Reconcile(errReq)
			Expect(result).To(Equal(reconcile.Result{}))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("get serviceconfig error"))
		})
	})

	Describe("test reconcile used mock client - list destinationrules error", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockClient.EXPECT().
				List(gomock.Any(), gomock.Not(&networkingv1beta1.DestinationRuleList{}), gomock.Any()).
				Return(nil).AnyTimes()
			mockClient.EXPECT().
				List(gomock.Any(), gomock.Eq(&networkingv1beta1.DestinationRuleList{}), gomock.Any()).
				Return(errors.New("list destinationrules error")).AnyTimes()
		}, timeout)

		AfterEach(func() {
			mockCtrl.Finish()
		}, timeout)

		It("error occurred when list destinationrules", func() {
			r := Reconciler{
				Client:     mockClient,
				Log:        nil,
				Scheme:     getFakeScheme(),
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			result, err := r.Reconcile(errReq)
			Expect(result).To(Equal(reconcile.Result{}))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("list destinationrules error"))
		})
	})

	Describe("test reconcile used mock client - list virtualservice error", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockClient.EXPECT().
				List(gomock.Any(), gomock.Not(&networkingv1beta1.VirtualServiceList{}), gomock.Any()).
				Return(nil).AnyTimes()
			mockClient.EXPECT().
				List(gomock.Any(), gomock.Eq(&networkingv1beta1.VirtualServiceList{}), gomock.Any()).
				Return(errors.New("list virtualservice error")).AnyTimes()
		}, timeout)

		AfterEach(func() {
			mockCtrl.Finish()
		}, timeout)

		It("error occurred when list virtualservice ", func() {
			r := Reconciler{
				Client:     mockClient,
				Log:        nil,
				Scheme:     getFakeScheme(),
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			result, err := r.Reconcile(errReq)
			Expect(result).To(Equal(reconcile.Result{}))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("list virtualservice error"))
		})
	})

	Describe("test reconcile used mock client - update status error", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockStatusWriter = NewMockStatusWriter(mockCtrl)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockClient.EXPECT().Status().Return(mockStatusWriter).AnyTimes()
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("update status error"))
		}, timeout)

		AfterEach(func() {
			mockCtrl.Finish()
		}, timeout)

		It("error occurred when update status", func() {
			r := Reconciler{
				Client:     mockClient,
				Log:        nil,
				Scheme:     getFakeScheme(),
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			result, err := r.Reconcile(errReq)
			Expect(result).To(Equal(reconcile.Result{}))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("update status error"))
		})
	})
})
