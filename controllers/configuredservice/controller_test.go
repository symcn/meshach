/*


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
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	. "github.com/symcn/mesh-operator/test"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var timeout = 5.0

var _ = Describe("Controller", func() {
	var (
		mockCtrl            *gomock.Controller
		mockClient          *MockClient
		errReq              ctrl.Request
		onlyServiceEntryReq ctrl.Request
		normalReq           ctrl.Request
		wrongHostReq        ctrl.Request
		onlyServiceEntryCs  *meshv1alpha1.ConfiguredService
		wrongHostCs         *meshv1alpha1.ConfiguredService
		normalCs            *meshv1alpha1.ConfiguredService
		existWorkloadEntry  *networkingv1beta1.WorkloadEntry
		reconciler          Reconciler
	)

	BeforeEach(func() {
		reconciler = Reconciler{
			Client:     k8sClient,
			Log:        ctrl.Log.WithName("controllers").WithName("configuredservice"),
			Scheme:     scheme.Scheme,
			Opt:        testOpt,
			MeshConfig: getTestMeshConfig(),
		}
		errReq = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "mesh-test",
			},
		}
		onlyServiceEntryReq = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "mesh-test",
				Name:      "only.serviceentry.com",
			},
		}
		wrongHostReq = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "mesh-test",
				Name:      "wrong.host.configuredservice.com",
			},
		}
		normalReq = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "mesh-test",
				Name:      "normal.configuredservice.com",
			},
		}
		onlyServiceEntryCs = &meshv1alpha1.ConfiguredService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "only.serviceentry.com",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName:         "Only.ServiceEntry.com",
				Instances:            nil,
				MeshConfigGeneration: 0,
			},
		}
		port := &meshv1alpha1.Port{Protocol: "HTTP", Number: 1111, Name: "test-port"}
		wrongHostCs = &meshv1alpha1.ConfiguredService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "wrong.host.configuredservice.com",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName: "Wrong.Host.ConfiguredService.com",
				Instances: []*meshv1alpha1.Instance{
					{
						Host:   "1.1.1.1",
						Labels: map[string]string{"flag": "blue"},
						Port:   port,
						Weight: 100,
					},
				},
				MeshConfigGeneration: 0,
			},
		}
		existWorkloadEntry = &networkingv1beta1.WorkloadEntry{
			ObjectMeta: v1.ObjectMeta{
				Name:      "wrong.host.configuredservice.com.1.1.1.1.1111",
				Namespace: "mesh-test",
			},
			Spec: v1beta1.WorkloadEntry{
				Address: "1.1.1.1",
				Ports: map[string]uint32{
					"test-port": 1111,
				},
				Weight: 100,
			},
		}
		normalCs = &meshv1alpha1.ConfiguredService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "normal.configuredservice.com",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName: "Normal.ConfiguredService.com",
				Instances: []*meshv1alpha1.Instance{
					{
						Host:   "10.10.10.10",
						Labels: map[string]string{"flag": "blue"},
						Port:   &meshv1alpha1.Port{Protocol: "HTTP", Number: 1111, Name: "test-port"},
						Weight: 100,
					},
				},
				MeshConfigGeneration: 0,
			},
		}
	})

	Describe("test reconcile serviceentry use mock client Create function", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = NewMockClient(mockCtrl)
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("create se error"))
			mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		}, timeout)
		AfterEach(func() {
			mockCtrl.Finish()
		}, timeout)

		Context("test crate serviceentry error", func() {
			It("return create error from mock client", func() {
				r := Reconciler{
					Client:     mockClient,
					Log:        nil,
					Scheme:     getFakeScheme(),
					Opt:        testOpt,
					MeshConfig: getTestMeshConfig(),
				}
				result, err := r.Reconcile(onlyServiceEntryReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("create se error"))
			})
		})
	})

	Describe("test configuredservice reconcile", func() {
		Context("occurred meshconfig not found error when reconcile configuredservice", func() {
			It("can not found meshconfig object", func() {
				result, err := reconciler.Reconcile(onlyServiceEntryReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`meshconfigs.mesh.symcn.com "mc-test-case" not found`))
			}, timeout)
		})

		Context("occurred error when reconcile configuredservice", func() {
			BeforeEach(func() {
				err := reconciler.Create(context.Background(), getTestMeshConfig())
				Expect(err).NotTo(HaveOccurred())
			})

			It("can not found configuredservice object", func() {
				result, err := reconciler.Reconcile(onlyServiceEntryReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).NotTo(HaveOccurred())
			}, timeout)

			It("resource name may not be empty", func() {
				result, err := reconciler.Reconcile(errReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("resource name may not be empty"))
			}, timeout)

			AfterEach(func() {
				err := reconciler.Delete(context.Background(), getTestMeshConfig())
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("reconcile when only create serviceentry", func() {
			BeforeEach(func() {
				err := reconciler.Create(context.Background(), getTestMeshConfig())
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.Create(context.Background(), onlyServiceEntryCs)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returned success", func() {
				result, err := reconciler.Reconcile(onlyServiceEntryReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).NotTo(HaveOccurred())

				By("get created serviceentry")
				found := &networkingv1beta1.ServiceEntry{}
				err = reconciler.Get(
					context.Background(),
					types.NamespacedName{Namespace: "mesh-test", Name: "only.serviceentry.com"},
					found)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.Name).To(Equal("only.serviceentry.com"))
			}, timeout)

			AfterEach(func() {
				err := reconciler.Delete(context.Background(), getTestMeshConfig())
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.Delete(context.Background(), onlyServiceEntryCs)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("reconcile when create workloadentry", func() {
			BeforeEach(func() {
				err := reconciler.Create(context.Background(), getTestMeshConfig())
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.Create(context.Background(), wrongHostCs)
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.Create(context.Background(), existWorkloadEntry)
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.Create(context.Background(), normalCs)
				Expect(err).NotTo(HaveOccurred())
			})

			It("occured error", func() {
				result, err := reconciler.Reconcile(wrongHostReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).To(HaveOccurred())
			}, timeout)

			It("returned success", func() {
				result, err := reconciler.Reconcile(normalReq)
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).NotTo(HaveOccurred())

				foundServiceEntry := &networkingv1beta1.ServiceEntry{}
				err = reconciler.Get(
					context.Background(),
					types.NamespacedName{Namespace: "mesh-test", Name: "normal.configuredservice.com"},
					foundServiceEntry)
				Expect(err).NotTo(HaveOccurred())
				Expect(foundServiceEntry.Name).To(Equal("normal.configuredservice.com"))

				foundWorkloadEntry := &networkingv1beta1.WorkloadEntry{}
				ins := normalCs.Spec.Instances[0]
				name := fmt.Sprintf("%s.%s.%d", normalCs.Name, ins.Host, ins.Port.Number)
				err = reconciler.Get(context.Background(),
					types.NamespacedName{Namespace: "mesh-test", Name: name},
					foundWorkloadEntry,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(foundWorkloadEntry.Name).To(Equal(name))
			}, timeout)

			AfterEach(func() {
				err := reconciler.Delete(context.Background(), getTestMeshConfig())
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.Delete(context.Background(), wrongHostCs)
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.Delete(context.Background(), normalCs)
				Expect(err).NotTo(HaveOccurred())
				err = reconciler.Delete(context.Background(), existWorkloadEntry)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("setup with controller manager", func() {
			It("error not occurred", func() {
				mgr, err := manager.New(cfg, manager.Options{})
				Expect(err).NotTo(HaveOccurred())

				err = reconciler.SetupWithManager(mgr)
				Expect(err).NotTo(HaveOccurred())
			}, timeout)
		})
	})
})
