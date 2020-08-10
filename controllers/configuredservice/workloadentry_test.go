/*
Copyright 2020 The Symcn Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package configuredservice

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

var _ = Describe("WorkloadEntry", func() {
	var (
		mockCtrl                 *gomock.Controller
		mockClient               *MockClient
		existedWorkloadEntry     *networkingv1beta1.WorkloadEntry
		updateWorkloadEntry      *networkingv1beta1.WorkloadEntry
		toBeDeletedWorkloadEntry *networkingv1beta1.WorkloadEntry
		testCr                   *meshv1alpha1.ConfiguredService
		testServiceConfig        *meshv1alpha1.ServiceConfig
	)

	BeforeEach(func() {
		testCr = &meshv1alpha1.ConfiguredService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test.workloadentry.cr",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName: "Test.Workload.Entry",
				Instances: []*meshv1alpha1.Instance{
					{
						Host: "3.3.3.3",
						Port: &meshv1alpha1.Port{
							Name:   "test-port",
							Number: 12345,
						},
					},
				},
				MeshConfigGeneration: 0,
			},
		}
		testServiceConfig = &meshv1alpha1.ServiceConfig{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test.workloadentry.cr",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ServiceConfigSpec{
				Instances: []*meshv1alpha1.InstanceConfig{
					{
						Host: "3.3.3.3",
						Port: &meshv1alpha1.Port{
							Name:   "test-port",
							Number: 12345,
						},
						Weight: 20,
					},
				},
			},
		}
		existedWorkloadEntry = &networkingv1beta1.WorkloadEntry{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test.workloadentry.cr.3.3.3.3.12345",
				Namespace: "mesh-test",
				Labels: map[string]string{
					testOpt.SelectLabel: "Test.Workload.Entry",
				},
			},
			Spec: v1beta1.WorkloadEntry{
				Address: "1.1.1.1",
			},
		}
		updateWorkloadEntry = &networkingv1beta1.WorkloadEntry{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test.workloadentry.cr",
				Namespace: "mesh-test",
				Labels: map[string]string{
					testOpt.SelectLabel: "Test.Workload.Entry",
				},
			},
			Spec: v1beta1.WorkloadEntry{
				Address: "2.2.2.2",
				Weight:  20,
			},
		}
		toBeDeletedWorkloadEntry = &networkingv1beta1.WorkloadEntry{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test.to.be.delete",
				Namespace: "mesh-test",
				Labels: map[string]string{
					testOpt.SelectLabel: "Test.Workload.Entry",
				},
			},
			Spec: v1beta1.WorkloadEntry{
				Address: "2.2.2.2",
			},
		}
	})

	Describe("test reconcile workloadentry use mock client Create function", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("create workloadentry error"))
			mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		}, timeout)
		AfterEach(func() {
			mockCtrl.Finish()
		}, timeout)

		Context("test crate workloadentry error", func() {
			It("return create error from mock client", func() {
				r := Reconciler{
					Client:     mockClient,
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				err := r.reconcileWorkloadEntry(context.Background(), testCr)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("create workloadentry error"))
			})
		})
	})

	Describe("test reconcile workloadentry use mock client List function", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("list workloadentry erorr"))
		}, timeout)
		AfterEach(func() {
			mockCtrl.Finish()
		}, timeout)

		Context("get serviceentries map error", func() {
			It("no workloadentry exist", func() {
				r := Reconciler{
					Client:     mockClient,
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				err := r.reconcileWorkloadEntry(context.Background(), testCr)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("list workloadentry erorr"))
			}, timeout)
		})

	})

	Describe("test reconcile workloadentry without mock client", func() {
		Context("test set controller reference error", func() {
			It("no kind is registered in scheme", func() {
				r := Reconciler{
					Client:     getFakeClient(),
					Log:        nil,
					Scheme:     runtime.NewScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				err := r.reconcileWorkloadEntry(context.Background(), testCr)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`no kind is registered for the type v1alpha1.ConfiguredService in scheme "pkg/runtime/scheme.go:101"`))
			}, timeout)
		})

		Context("test update workloadentry success", func() {
			It("no error occurred", func() {
				r := Reconciler{
					Client:     getFakeClient(existedWorkloadEntry, testServiceConfig),
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				err := r.reconcileWorkloadEntry(context.Background(), testCr)
				Expect(err).NotTo(HaveOccurred())

				By("get updated workloadentry")
				found := &networkingv1beta1.WorkloadEntry{}
				name := "test.workloadentry.cr.3.3.3.3.12345"
				err = r.Get(context.Background(), types.NamespacedName{Namespace: "mesh-test", Name: name}, found)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.Name).To(Equal(name))
				Expect(found.Spec.Address).To(Equal("3.3.3.3"))
				Expect(found.Spec.Address).NotTo(Equal("1.1.1.1"))
			}, timeout)

			It("delete unused workloadentry", func() {
				r := Reconciler{
					Client:     getFakeClient(existedWorkloadEntry, toBeDeletedWorkloadEntry, testServiceConfig),
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				err := r.reconcileWorkloadEntry(context.Background(), testCr)
				Expect(err).NotTo(HaveOccurred())

				By("get updated workloadentry")
				found := &networkingv1beta1.WorkloadEntry{}
				err = r.Get(context.Background(), types.NamespacedName{Namespace: "mesh-test", Name: "test.to.be.delete"}, found)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(Equal(true))
			}, timeout)
		})

		Context("test update workloadentry error", func() {
			BeforeEach(func() {
				mockCtrl = gomock.NewController(GinkgoT())
				mockClient = NewMockClient(mockCtrl)
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("update se error"))
			}, timeout)
			AfterEach(func() {
				mockCtrl.Finish()
			}, timeout)

			It("returned update error from mock client", func() {
				r := Reconciler{
					Client:     mockClient,
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				weightMap := map[string]uint32{"test.workloadentry.cr": 100}
				err := r.updateWorkloadEntry(context.Background(), weightMap, updateWorkloadEntry, existedWorkloadEntry)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("update se error"))
			}, timeout)
		})

		Context("test compare workloadentry", func() {
			It("return true while finalizers is not equal", func() {
				new := &networkingv1beta1.WorkloadEntry{ObjectMeta: v1.ObjectMeta{Finalizers: []string{"test.finalizers.a"}}}
				old := &networkingv1beta1.WorkloadEntry{}
				result := compareWorkloadEntry(new, old)
				Expect(result).To(Equal(true))
			}, timeout)

			It("return true while labels is not equal", func() {
				m := map[string]string{"app": "test"}
				new := &networkingv1beta1.WorkloadEntry{ObjectMeta: v1.ObjectMeta{Labels: m}}
				old := &networkingv1beta1.WorkloadEntry{}
				result := compareWorkloadEntry(new, old)
				Expect(result).To(Equal(true))
			}, timeout)

			It("return false", func() {
				se := &networkingv1beta1.WorkloadEntry{}
				result := compareWorkloadEntry(se, se)
				Expect(result).To(Equal(false))
			}, timeout)
		})

		Context("test truncated", func() {
			It("truncate when string length exceeds 62", func() {
				s := make([]byte, 66)
				want := make([]byte, 62)
				for i := range s {
					s[i] = 'a'
				}
				for i := range want {
					want[i] = 'a'
				}
				result := truncated(string(s))
				Expect(result).To(Equal(string(want)))
				Expect(len(result)).To(Equal(62))
			})
		})
	})
})
