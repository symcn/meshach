package configuraredservice

import (
	"context"
	"reflect"
	"testing"

	meshv1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	"github.com/symcn/mesh-operator/pkg/option"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileConfiguraredService_updateStatus(t *testing.T) {
	fakeScheme := GetFakeScheme()
	type fields struct {
		client     client.Client
		scheme     *runtime.Scheme
		opt        *option.ControllerOption
		meshConfig *meshv1.MeshConfig
	}
	type args struct {
		ctx context.Context
		req reconcile.Request
		cr  *meshv1.ConfiguraredService
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test-status-update-ok",
			fields: fields{
				client: GetFakeClient(
					csTestOK,
					fakeServiceEntry,
					fakeVirtualService,
					fakeDestinationRule,
					fakeWorkloadEntry,
				),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.TODO(),
				req: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "cs-test-case",
					Namespace: "sym-test",
				}},
				cr: csTestOK,
			},
			wantErr: false,
		},
		{
			name: "test-status-update-error",
			fields: fields{
				client:     GetFakeClient(),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.TODO(),
				req: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "cs-test-case",
					Namespace: "sym-test",
				}},
				cr: csTestOK,
			},
			wantErr: true,
		},
		{
			name: "test-status-update-distributing",
			fields: fields{
				client:     GetFakeClient(csTestOK, fakeServiceEntry, fakeVirtualService),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.TODO(),
				req: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "cs-test-case",
					Namespace: "sym-test",
				}},
				cr: csTestOK,
			},
			wantErr: false,
		},
		{
			name: "test-status-update-undistributed",
			fields: fields{
				client:     GetFakeClient(csTestOK),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.TODO(),
				req: reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      "cs-test-case",
					Namespace: "sym-test",
				}},
				cr: csTestOK,
			},
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
			if err := r.updateStatus(tt.args.ctx, tt.args.req, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileConfiguraredService.updateStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			klog.Infof("[serviceentry] desired: %#v, distributed: %#v, undistributed: %#v",
				tt.args.cr.Status.Status.ServiceEntry.Desired,
				*tt.args.cr.Status.Status.ServiceEntry.Distributed,
				*tt.args.cr.Status.Status.ServiceEntry.Undistributed,
			)
			klog.Infof("[workloadentry] desired: %#v, distributed: %#v, undistributed: %#v",
				tt.args.cr.Status.Status.WorkloadEntry.Desired,
				*tt.args.cr.Status.Status.WorkloadEntry.Distributed,
				*tt.args.cr.Status.Status.WorkloadEntry.Undistributed,
			)
			klog.Infof("[virtualservice] desired: %#v, distributed: %#v, undistributed: %#v",
				tt.args.cr.Status.Status.VirtualService.Desired,
				*tt.args.cr.Status.Status.VirtualService.Distributed,
				*tt.args.cr.Status.Status.VirtualService.Undistributed,
			)
			klog.Infof("[destinationrule] desired: %#v, distributed: %#v, undistributed: %#v",
				tt.args.cr.Status.Status.DestinationRule.Desired,
				*tt.args.cr.Status.Status.DestinationRule.Distributed,
				*tt.args.cr.Status.Status.DestinationRule.Undistributed,
			)
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
		{
			name: "test-calc-phase-status-unknown-ok",
			args: args{
				status: &meshv1.Status{
					ServiceEntry: &meshv1.SubStatus{
						Desired:       1,
						Distributed:   nil,
						Undistributed: nil,
					},
					WorkloadEntry: &meshv1.SubStatus{
						Desired:       1,
						Distributed:   nil,
						Undistributed: nil,
					},
					VirtualService: &meshv1.SubStatus{
						Desired:       1,
						Distributed:   nil,
						Undistributed: nil,
					},
					DestinationRule: &meshv1.SubStatus{
						Desired:       1,
						Distributed:   nil,
						Undistributed: nil,
					},
				},
			},
			want: meshv1.ConfigStatusUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcPhase(tt.args.status); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calcPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}
