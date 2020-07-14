package configuraredservice

import (
	"context"
	"testing"

	meshv1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	"github.com/symcn/mesh-operator/pkg/option"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconcileConfiguraredService_reconcileSubset(t *testing.T) {
	fakeScheme := GetFakeScheme()
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		ctx context.Context
		cr  *meshv1.ConfiguraredService
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test-reconcile-subset-ok",
			fields: fields{
				client: GetFakeClient(
					csTestReconcileSubsetOK,
					TestMeshConfig,
				),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.Background(),
				cr:  csTestReconcileSubsetOK,
			},
			wantErr: false,
		},
		{
			name: "test-reconcile-subset-nooption-ok",
			fields: fields{
				client: GetFakeClient(
					csTestReconcileSubsetNoOption,
					TestMeshConfig,
				),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.Background(),
				cr:  csTestReconcileSubsetNoOption,
			},
			wantErr: false,
		},
		{
			name: "test-reconcile-subset-error",
			fields: fields{
				client:     GetFakeClient(TestMeshConfig),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.Background(),
				cr:  csTestReconcileSubsetOK,
			},
			wantErr: true,
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
			if err := r.reconcileSubset(tt.args.ctx, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileConfiguraredService.reconcileSubset() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

var (
	subsetTestOK = []*meshv1.Subset{
		{
			Name:     "blue",
			Labels:   map[string]string{"sym-group": "blue"},
			IsCanary: false,
		},
		{
			Name:     "green",
			Labels:   map[string]string{"sym-group": "green"},
			IsCanary: false,
		},
		{
			Name:     "gray",
			Labels:   map[string]string{"sym-group": "gray"},
			IsCanary: true,
		},
	}
	sourceLabelsTestOK = []*meshv1.SourceLabels{
		{
			Name:   "blue",
			Labels: map[string]string{"sym-group": "blue"},
			Route:  []*meshv1.Destination{{Subset: "blue", Weight: 100}},
		},
		{
			Name:   "green",
			Labels: map[string]string{"sym-group": "green"},
			Route:  []*meshv1.Destination{{Subset: "green", Weight: 100}},
		},
		{
			Name:   "gray",
			Labels: map[string]string{"sym-group": "gray"},
			Route:  []*meshv1.Destination{{Subset: "gray", Weight: 100}},
		},
	}
	csTestReconcileSubsetOK = &meshv1.ConfiguraredService{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.recocile.subset.ok",
			Namespace: "sym-test",
			Labels: map[string]string{
				"app": "test-app",
			},
		},
		Spec: meshv1.ConfiguraredServiceSpec{
			OriginalName: "test.Reconcile.Subset.OK",
			Ports:        nil,
			Instances:    nil,
			Policy: &meshv1.Policy{
				LoadBalancer: map[string]string{
					"simple": "RANDOM",
				},
				MaxConnections: 100,
				Timeout:        "10",
				MaxRetries:     3,
				SourceLabels:   sourceLabelsTestOK,
			},
			Subsets:              subsetTestOK,
			MeshConfigGeneration: 0,
			RerouteOption: &meshv1.RerouteOption{
				ReroutePolicy: meshv1.Specific,
				SpecificRoute: map[string][]*meshv1.Destination{
					"green": {{Subset: "blue", Weight: 100}},
					"blue":  {{Subset: "green", Weight: 100}},
				},
			},
			CanaryRerouteOption: &meshv1.RerouteOption{
				ReroutePolicy: meshv1.Random,
			},
		},
	}
	csTestReconcileSubsetNoOption = &meshv1.ConfiguraredService{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test.recocile.subset.nooption.ok",
			Namespace: "sym-test",
			Labels: map[string]string{
				"app": "test-app",
			},
		},
		Spec: meshv1.ConfiguraredServiceSpec{
			OriginalName: "test.Reconcile.Subset.noOption.OK",
			Ports:        nil,
			Instances:    nil,
			Policy: &meshv1.Policy{
				LoadBalancer: map[string]string{
					"simple": "RANDOM",
				},
				MaxConnections: 100,
				Timeout:        "10",
				MaxRetries:     3,
				SourceLabels:   nil,
			},
			Subsets:              subsetTestOK,
			MeshConfigGeneration: 0,
		},
	}
)
