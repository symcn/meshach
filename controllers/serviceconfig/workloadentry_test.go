package serviceconfig

import (
	"context"
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	. "github.com/symcn/mesh-operator/test"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("test workloadentry update", func() {
	var (
		mockCtrl    *gomock.Controller
		mockClient  *MockClient
		sc          *meshv1alpha1.ServiceConfig
		normalRoute []*meshv1alpha1.Destination
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
})
