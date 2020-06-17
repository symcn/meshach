package appmeshconfig

import (
	"context"
	"reflect"
	"testing"

	meshv1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	"github.com/mesh-operator/pkg/option"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileAppMeshConfig_updateStatus(t *testing.T) {
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		ctx context.Context
		req reconcile.Request
		cr  *meshv1.AppMeshConfig
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileAppMeshConfig{
				client:     tt.fields.client,
				scheme:     tt.fields.scheme,
				opt:        tt.fields.opt,
				meshConfig: tt.fields.meshConfig,
			}
			if err := r.updateStatus(tt.args.ctx, tt.args.req, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileAppMeshConfig.updateStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconcileAppMeshConfig_buildStatus(t *testing.T) {
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		cr *meshv1.AppMeshConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *meshv1.AppMeshConfigStatus
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileAppMeshConfig{
				client:     tt.fields.client,
				scheme:     tt.fields.scheme,
				opt:        tt.fields.opt,
				meshConfig: tt.fields.meshConfig,
			}
			if got := r.buildStatus(tt.args.cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileAppMeshConfig.buildStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileAppMeshConfig_getServiceEntryStatus(t *testing.T) {
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		ctx context.Context
		cr  *meshv1.AppMeshConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *meshv1.SubStatus
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileAppMeshConfig{
				client:     tt.fields.client,
				scheme:     tt.fields.scheme,
				opt:        tt.fields.opt,
				meshConfig: tt.fields.meshConfig,
			}
			if got := r.getServiceEntryStatus(tt.args.ctx, tt.args.cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileAppMeshConfig.getServiceEntryStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileAppMeshConfig_getWorkloadEntryStatus(t *testing.T) {
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		ctx context.Context
		cr  *meshv1.AppMeshConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *meshv1.SubStatus
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileAppMeshConfig{
				client:     tt.fields.client,
				scheme:     tt.fields.scheme,
				opt:        tt.fields.opt,
				meshConfig: tt.fields.meshConfig,
			}
			if got := r.getWorkloadEntryStatus(tt.args.ctx, tt.args.cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileAppMeshConfig.getWorkloadEntryStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileAppMeshConfig_getVirtualServiceStatus(t *testing.T) {
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		ctx context.Context
		cr  *meshv1.AppMeshConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *meshv1.SubStatus
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileAppMeshConfig{
				client:     tt.fields.client,
				scheme:     tt.fields.scheme,
				opt:        tt.fields.opt,
				meshConfig: tt.fields.meshConfig,
			}
			if got := r.getVirtualServiceStatus(tt.args.ctx, tt.args.cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileAppMeshConfig.getVirtualServiceStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileAppMeshConfig_getDestinationRuleStatus(t *testing.T) {
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		ctx context.Context
		cr  *meshv1.AppMeshConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *meshv1.SubStatus
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileAppMeshConfig{
				client:     tt.fields.client,
				scheme:     tt.fields.scheme,
				opt:        tt.fields.opt,
				meshConfig: tt.fields.meshConfig,
			}
			if got := r.getDestinationRuleStatus(tt.args.ctx, tt.args.cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileAppMeshConfig.getDestinationRuleStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileAppMeshConfig_count(t *testing.T) {
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		ctx  context.Context
		cr   *meshv1.AppMeshConfig
		list runtime.Object
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileAppMeshConfig{
				client:     tt.fields.client,
				scheme:     tt.fields.scheme,
				opt:        tt.fields.opt,
				meshConfig: tt.fields.meshConfig,
			}
			if got := r.count(tt.args.ctx, tt.args.cr, tt.args.list); got != tt.want {
				t.Errorf("ReconcileAppMeshConfig.count() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_calcPhase(t *testing.T) {
	type args struct {
		status *meshv1.Status
	}
	tests := []struct {
		name string
		args args
		want meshv1.ConfigPhase
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcPhase(tt.args.status); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calcPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}
