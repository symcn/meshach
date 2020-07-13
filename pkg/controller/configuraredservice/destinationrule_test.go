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

package configuraredservice

import (
	"context"
	"testing"

	meshv1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	"github.com/symcn/mesh-operator/pkg/option"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconcileConfiguraredService_reconcileDestinationRule(t *testing.T) {
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
			name: "test-reconcile-destination-create-ok",
			fields: fields{
				client:     GetFakeClient(TestMeshConfig),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.Background(),
				cr:  smeTestOK,
			},
			wantErr: false,
		},
		{
			name: "test-reconcile-destination-update-ok",
			fields: fields{
				client:     GetFakeClient(TestMeshConfig, smeTestOK, fakeDestinationRule),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.Background(),
				cr:  smeTestOK,
			},
			wantErr: false,
		},
		{
			name: "test-reconcile-destination-delete-ok",
			fields: fields{
				client:     GetFakeClient(TestMeshConfig, smeTestOK, fakeDestinationRule, fakeDeleteDestinationRule),
				scheme:     fakeScheme,
				opt:        TestOpt,
				meshConfig: TestMeshConfig,
			},
			args: args{
				ctx: context.Background(),
				cr:  smeTestOK,
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
			if err := r.reconcileDestinationRule(tt.args.ctx, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileConfiguraredService.reconcileDestinationRule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
