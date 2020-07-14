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

// Package configuraredservice ...
package configuraredservice

import (
	"reflect"
	"testing"

	meshv1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	"github.com/symcn/mesh-operator/pkg/option"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileConfiguraredService_Reconcile(t *testing.T) {
	fakeScheme := GetFakeScheme()
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		request reconcile.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    reconcile.Result
		wantErr bool
	}{
		{
			name: "test-cs-reconcile-no-service-ok",
			fields: fields{
				client:     GetFakeClient(csNoService, TestMeshConfig),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				request: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "dubbo.noservice",
					Namespace: "sym-test",
				}},
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test-cs-reconcile-no-meshconfig-error",
			fields: fields{
				client:     GetFakeClient(),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				request: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "dubbo.nomeshconfig",
					Namespace: "sym-test",
				}},
			},
			want:    reconcile.Result{},
			wantErr: true,
		},
		{
			name: "test-cs-reconcile-no-configuraredservice-ok",
			fields: fields{
				client:     GetFakeClient(TestMeshConfig),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				request: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "test.no.configuraredservice",
					Namespace: "sym-test",
				}},
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test-cs-reconcile-only-serviceentry-ok",
			fields: fields{
				client:     GetFakeClient(csTestServiceEntryOK, TestMeshConfig),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				request: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "test.serviceentry.ok",
					Namespace: "sym-test",
				}},
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test-cs-reconcile-only-workloadentry-ok",
			fields: fields{
				client:     GetFakeClient(csTestWorkloadEntryOK, TestMeshConfig),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				request: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "test.workloadentry.ok",
					Namespace: "sym-test",
				}},
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test-cs-reconcile-all-ok",
			fields: fields{
				client:     GetFakeClient(csTestOK, TestMeshConfig),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				request: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "test.all.ok",
					Namespace: "sym-test",
				}},
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileConfiguraredService{
				client:     tt.fields.client,
				scheme:     tt.fields.scheme,
				opt:        tt.fields.opt,
				meshConfig: tt.fields.meshConfig,
			}
			got, err := r.Reconcile(tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileConfiguraredService.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileConfiguraredService.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test case used struct
var (
	TestOpt = &option.ControllerOption{
		HTTPAddress:             ":8080",
		SyncPeriod:              120,
		MetricsEnabled:          true,
		GinLogEnabled:           true,
		GinLogSkipPath:          []string{"/ready", "/live"},
		EnableLeaderElection:    true,
		LeaderElectionID:        "mesh-operator-lock",
		LeaderElectionNamespace: "sym-test",
		PprofEnabled:            true,
		GoroutineThreshold:      1000,
		ProxyHost:               "mosn.io.dubbo.proxy",
		ProxyAttempts:           3,
		ProxyPerTryTimeout:      2,
		ProxyRetryOn:            "gateway-error,connect-failure,refused-stream",
		Zone:                    "gz01",
		MeshConfigName:          "mc-test-case",
		MeshConfigNamespace:     "sym-test",
		SelectLabel:             "service",
	}
	blueSubset = &meshv1.Subset{
		Name:   "blue",
		Labels: map[string]string{"sym-group": "blue"},
	}

	greenSubset = &meshv1.Subset{
		Name:   "green",
		Labels: map[string]string{"sym-group": "green"},
	}

	TestMeshConfig = &meshv1.MeshConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mc-test-case",
			Namespace: "sym-test",
		},
		Spec: meshv1.MeshConfigSpec{
			MatchHeaderLabelKeys: map[string]meshv1.StringMatchType{
				"sym-zone": "exact",
			},
			MatchSourceLabelKeys:   []string{"sym-group"},
			WorkloadEntryLabelKeys: []string{"sym-zone", "sym-group"},
			MeshLabelsRemap: map[string]string{
				"sym-zone":  "zone",
				"sym-group": "flag",
			},
			GlobalSubsets: []*meshv1.Subset{blueSubset, greenSubset},
			GlobalPolicy: &meshv1.Policy{
				LoadBalancer: map[string]string{
					"simple": "ROUND_ROBIN",
				},
				MaxConnections: 100,
				Timeout:        "5s",
				MaxRetries:     3,
			},
		},
	}

	csTestServiceEntryOK = &meshv1.ConfiguraredService{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.serviceentry.ok",
			Namespace: "sym-test",
			Labels: map[string]string{
				"app_code": "test-case-service",
			},
		},
		Spec: meshv1.ConfiguraredServiceSpec{
			OriginalName: "cs.Test.Case",
			Ports: []*meshv1.Port{{
				Name:     "dubbo-http",
				Protocol: "HTTP",
				Number:   20882,
			}},
			Instances: []*meshv1.Instance{},
			Policy: &meshv1.Policy{
				LoadBalancer: map[string]string{
					"simple": "ROUND_ROBIN",
				},
				MaxConnections: 10,
				Timeout:        "20s",
				MaxRetries:     3,
				SourceLabels:   nil,
			},
			Subsets:              nil,
			MeshConfigGeneration: 0,
		},
	}
	csNoService = &meshv1.ConfiguraredService{
		ObjectMeta: v1.ObjectMeta{
			Name:      "dubbo.noservice",
			Namespace: "sym-test",
			Labels: map[string]string{
				"app_code": "test-case-service",
			},
		},
		Spec: meshv1.ConfiguraredServiceSpec{},
	}

	testWorkloadEntryOKInstance = &meshv1.Instance{
		Host: "10.10.10.10",
		Labels: map[string]string{
			"aaa": "bbb",
		},
		Port: &meshv1.Port{
			Name:     "dubbo-http",
			Protocol: "HTTP",
			Number:   20882,
		},
		Weight: 0,
	}
	csTestWorkloadEntryOK = &meshv1.ConfiguraredService{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.workloadentry.ok",
			Namespace: "sym-test",
			Labels: map[string]string{
				"app_code": "test-case-service",
			},
		},
		Spec: meshv1.ConfiguraredServiceSpec{
			OriginalName: "dubbo.TestServiceOK",
			Ports:        nil,
			Instances:    []*meshv1.Instance{testWorkloadEntryOKInstance},
			Policy: &meshv1.Policy{
				LoadBalancer: map[string]string{
					"simple": "ROUND_ROBIN",
				},
				MaxConnections: 10,
				Timeout:        "20s",
				MaxRetries:     3,
				SourceLabels:   nil,
			},
			Subsets: nil,
		},
	}

	testOKSubset = &meshv1.Subset{
		Name: "blue-test",
		Labels: map[string]string{
			"sym-group": "blue",
		},
		Policy: &meshv1.Policy{},
	}
	csTestOK = &meshv1.ConfiguraredService{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.all.ok",
			Namespace: "sym-test",
			Labels: map[string]string{
				"app_code": "test-case-service",
			},
		},
		Spec: meshv1.ConfiguraredServiceSpec{
			OriginalName: "dubbo.TestServiceOK",
			Ports: []*meshv1.Port{{
				Name:     "dubbo-http",
				Protocol: "HTTP",
				Number:   20882,
			}},
			Instances: []*meshv1.Instance{testWorkloadEntryOKInstance},
			Policy: &meshv1.Policy{
				LoadBalancer: map[string]string{
					"simple": "ROUND_ROBIN",
				},
				MaxConnections: 10,
				Timeout:        "20s",
				MaxRetries:     3,
				SourceLabels:   nil,
			},
			Subsets: []*meshv1.Subset{testOKSubset},
		},
	}
	fakeWorkloadEntry = &networkingv1beta1.WorkloadEntry{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.all.ok.10.10.10.10.20882",
			Namespace: "sym-test",
			Labels: map[string]string{
				TestOpt.SelectLabel: "dubbo.TestServiceOK",
			},
		},
		Spec: v1beta1.WorkloadEntry{},
	}
	fakeDeleteWorkloadEntry = &networkingv1beta1.WorkloadEntry{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.all.ok.10.10.10.11.20882",
			Namespace: "sym-test",
			Labels: map[string]string{
				TestOpt.SelectLabel: "dubbo.TestServiceOK",
			},
		},
		Spec: v1beta1.WorkloadEntry{},
	}
	fakeVirtualService = &networkingv1beta1.VirtualService{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.all.ok",
			Namespace: "sym-test",
			Labels: map[string]string{
				TestOpt.SelectLabel: "dubbo.TestServiceOK",
			},
		},
		Spec: v1beta1.VirtualService{
			Hosts: []string{"dubbo.testserviceok"},
		},
	}
	fakeDeleteVirtualService = &networkingv1beta1.VirtualService{
		ObjectMeta: v1.ObjectMeta{
			Name:      "dubbo.testserviceok.delete",
			Namespace: "sym-test",
			Labels: map[string]string{
				TestOpt.SelectLabel: "dubbo.TestServiceOK",
			},
		},
		Spec: v1beta1.VirtualService{
			Hosts: []string{"dubbo.testserviceok"},
		},
	}
	fakeServiceEntry = &networkingv1beta1.ServiceEntry{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.all.ok",
			Namespace: "sym-test",
			Labels: map[string]string{
				TestOpt.SelectLabel: "dubbo.TestServiceOK",
			},
		},
		Spec: v1beta1.ServiceEntry{
			Hosts: []string{"dubbo.testserviceok"},
		},
	}
	fakeDeleteServiceEntry = &networkingv1beta1.ServiceEntry{
		ObjectMeta: v1.ObjectMeta{
			Name:      "dubbo.testserviceok.delete",
			Namespace: "sym-test",
			Labels: map[string]string{
				TestOpt.SelectLabel: "dubbo.TestServiceOK",
			},
		},
		Spec: v1beta1.ServiceEntry{
			Hosts: []string{"dubbo.testserviceok"},
		},
	}
	fakeDestinationRule = &networkingv1beta1.DestinationRule{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.all.ok",
			Namespace: "sym-test",
			Labels: map[string]string{
				TestOpt.SelectLabel: "dubbo.TestServiceOK",
			},
		},
		Spec: v1beta1.DestinationRule{
			Host:          "",
			TrafficPolicy: nil,
		},
	}
	fakeDeleteDestinationRule = &networkingv1beta1.DestinationRule{
		ObjectMeta: v1.ObjectMeta{
			Name:      "dubbo.testservice.delete",
			Namespace: "sym-test",
			Labels: map[string]string{
				TestOpt.SelectLabel: "dubbo.TestServiceOK",
			},
		},
		Spec: v1beta1.DestinationRule{
			Host:          "",
			TrafficPolicy: nil,
		},
	}
)

// GetFakeClient return a fake client to mock API calls.
func GetFakeClient(objs ...runtime.Object) client.Client {
	return fake.NewFakeClient(objs...)
}

// GetFakeScheme register operator types with the runtime scheme.
func GetFakeScheme() *runtime.Scheme {
	s := scheme.Scheme
	s.AddKnownTypes(meshv1.SchemeGroupVersion,
		&meshv1.MeshConfig{},
		&meshv1.MeshConfigList{},
		&meshv1.ConfiguraredService{},
		&meshv1.ConfiguraredServiceList{},
	)
	s.AddKnownTypes(networkingv1beta1.SchemeGroupVersion,
		&networkingv1beta1.DestinationRule{},
		&networkingv1beta1.DestinationRuleList{},
		&networkingv1beta1.Gateway{},
		&networkingv1beta1.GatewayList{},
		&networkingv1beta1.ServiceEntry{},
		&networkingv1beta1.ServiceEntryList{},
		&networkingv1beta1.Sidecar{},
		&networkingv1beta1.SidecarList{},
		&networkingv1beta1.VirtualService{},
		&networkingv1beta1.VirtualServiceList{},
		&networkingv1beta1.WorkloadEntry{},
		&networkingv1beta1.WorkloadEntryList{},
	)
	return s
}
