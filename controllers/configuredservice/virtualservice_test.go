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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("VirtualService", func() {
	var (
		cs *meshv1alpha1.ConfiguredService
	)

	BeforeEach(func() {
		cs = &meshv1alpha1.ConfiguredService{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test.normal.cs",
				Namespace: "mesh-test",
			},
			Spec: meshv1alpha1.ConfiguredServiceSpec{
				OriginalName: "Test.Normal.Cs",
				Instances: []*meshv1alpha1.Instance{
					{
						Host:   "10.1.1.1",
						Labels: map[string]string{"flag": "blue"},
						Port: &meshv1alpha1.Port{
							Name:     "dubbo-http",
							Protocol: "HTTP",
							Number:   20882,
						},
						Weight: 100,
					},
					{
						Host:   "10.1.2.1",
						Labels: map[string]string{"flag": "green"},
						Port: &meshv1alpha1.Port{
							Name:     "dubbo-http",
							Protocol: "HTTP",
							Number:   20882,
						},
						Weight: 100,
					},
				},
				MeshConfigGeneration: 0,
			},
		}
	})

	Context("test reconcile virtualservice", func() {
		It("create default virtualservice", func() {
			r := Reconciler{
				Client:     getFakeClient(),
				Log:        nil,
				Scheme:     getFakeScheme(),
				Opt:        testOpt,
				MeshConfig: getTestMeshConfig(),
			}
			err := r.reconcileVirtualService(context.Background(), cs)
			Expect(err).NotTo(HaveOccurred())

			found := &networkingv1beta1.VirtualService{}
			err = r.Get(context.Background(), types.NamespacedName{Namespace: "mesh-test", Name: "test.normal.cs"}, found)
			Expect(err).NotTo(HaveOccurred())

			for _, http := range found.Spec.Http {
				if http.Name == "dubbo-http-route-canary" {
					for _, route := range http.Route {
						Expect(route.Weight).To(Equal(int32(50)))
					}
				}
			}
		})
	})
})
