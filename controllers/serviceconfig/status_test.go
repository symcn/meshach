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
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("test status update", func() {
	var (
		mockCtrl         *gomock.Controller
		mockClient       *MockClient
		req              ctrl.Request
		sc               *meshv1alpha1.ServiceConfig
		cs               *meshv1alpha1.ConfiguredService
		vs               *networkingv1beta1.VirtualService
		normalRoute      []*meshv1alpha1.Destination
		mockStatusWriter *MockStatusWriter
	)

	BeforeEach(func() {
		req = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "mesh-test",
				Name:      "test.update.status",
			},
		}

		normalRoute = []*meshv1alpha1.Destination{
			{Subset: "blue", Weight: 40},
			{Subset: "green", Weight: 60},
		}

		sc = &meshv1alpha1.ServiceConfig{
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
				Name:      "normal.reconcile.serviceconfig",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName: "Normal.Reconcile.ServiceConfig",
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

		vs = &networkingv1beta1.VirtualService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.reconcile.serviceconfig",
				Namespace: "mesh-test",
				Labels: map[string]string{
					"service": "Normal.Reconcile.ServiceConfig",
				},
			},
			Spec: v1beta1.VirtualService{
				Hosts: []string{"normal.reconcile.serviceconfig"},
				Http:  nil,
			},
		}
	}, timeout)

	Describe("test get error when after update status", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockStatusWriter = NewMockStatusWriter(mockCtrl)
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("update status error")).AnyTimes()
			mockClient.EXPECT().Status().Return(mockStatusWriter).AnyTimes()
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Eq(sc)).Return(errors.New("get status error"))
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(sc)).Return(nil).AnyTimes()
			mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		}, timeout)

		It("return get status error", func() {
			r := Reconciler{
				Client:     mockClient,
				Scheme:     getFakeScheme(),
				Log:        nil,
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}

			err := r.updateStatus(context.Background(), req, sc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("get status error"))
		}, timeout)
	})

	Describe("test get virtualservice status", func() {
		It("occurred error when get subset, return default status", func() {
			r := Reconciler{
				Client:     getFakeClient(getTestMeshConfig()),
				Scheme:     &runtime.Scheme{},
				Log:        nil,
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			result := r.getVirtualServiceStatus(context.Background(), sc)
			svcCount := 0
			Expect(result).To(Equal(&meshv1alpha1.SubStatus{
				Desired:       svcCount,
				Distributed:   &svcCount,
				Undistributed: &svcCount,
			}))
		})

		It("return right virtualservice status", func() {
			r := Reconciler{
				Client:     getFakeClient(getTestMeshConfig(), sc, cs, vs),
				Scheme:     getFakeScheme(),
				Log:        nil,
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			result := r.getVirtualServiceStatus(context.Background(), sc)
			svcCount := 1
			undistributed := 0
			Expect(result).To(Equal(&meshv1alpha1.SubStatus{
				Desired:       svcCount,
				Distributed:   &svcCount,
				Undistributed: &undistributed,
			}))
		})
	})
})
