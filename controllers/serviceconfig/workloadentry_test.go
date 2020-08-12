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
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("test workloadentry update", func() {
	var (
		mockCtrl    *gomock.Controller
		mockClient  *MockClient
		sc          *meshv1alpha1.ServiceConfig
		normalRoute []*meshv1alpha1.Destination
		we          *networkingv1beta1.WorkloadEntry
	)

	BeforeEach(func() {
		normalRoute = []*meshv1alpha1.Destination{
			{Subset: "blue", Weight: 40},
			{Subset: "green", Weight: 60},
		}
		sc = &meshv1alpha1.ServiceConfig{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test.workloadentry.com",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ServiceConfigSpec{
				OriginalName: "Test.WorkloadEntry.Com",
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
						Host:   "1.1.1.1",
						Weight: 80,
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
					},
					{
						Host:   "1.1.1.2",
						Weight: 20,
						Port:   &meshv1alpha1.Port{Name: "test-port", Number: 12345, Protocol: "HTTP"},
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
		we = &networkingv1beta1.WorkloadEntry{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test.workloadentry.com.1.1.1.1.12345",
				Namespace: "mesh-test",
			},
			Spec: v1beta1.WorkloadEntry{
				Address: "1.1.1.1",
				Weight:  20,
				Ports: map[string]uint32{
					"test-port": 12345,
				},
			},
		}
	}, timeout)

	Describe("occured get error", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(errors.New("get workloadentry error")).AnyTimes()
		}, timeout)

		AfterEach(func() {
			mockCtrl.Finish()
		}, timeout)

		It("get workloadentry error", func() {
			r := Reconciler{
				Client:     mockClient,
				Scheme:     getFakeScheme(),
				Log:        nil,
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}

			err := r.reconcileWorkloadEntry(context.Background(), sc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("get workloadentry error"))
		}, timeout)
	})

	Describe("occured update error", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockClient.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(errors.New("update workloadentry error")).AnyTimes()
		}, timeout)

		AfterEach(func() {
			mockCtrl.Finish()
		}, timeout)

		It("update workloadentry error", func() {
			r := Reconciler{
				Client:     mockClient,
				Scheme:     getFakeScheme(),
				Log:        nil,
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}

			err := r.reconcileWorkloadEntry(context.Background(), sc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("update workloadentry error"))
		}, timeout)
	})

	Describe("update success", func() {
		It("return nil", func() {
			r := Reconciler{
				Client:     getFakeClient(getTestMeshConfig(), sc, we),
				Scheme:     getFakeScheme(),
				Log:        nil,
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}

			err := r.reconcileWorkloadEntry(context.Background(), sc)
			Expect(err).NotTo(HaveOccurred())

			found := &networkingv1beta1.WorkloadEntry{}
			name := "test.workloadentry.com.1.1.1.1.12345"
			err = r.Get(context.Background(), types.NamespacedName{Namespace: "mesh-test", Name: name}, found)
			Expect(err).NotTo(HaveOccurred())
			Expect(found.Spec.Weight).To(Equal(uint32(80)))
		}, timeout)
	})
})
