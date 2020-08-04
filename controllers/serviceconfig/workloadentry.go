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

package serviceconfig

import (
	"context"
	"fmt"

	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

func (r *Reconciler) reconcileWorkloadEntry(ctx context.Context, sc *meshv1alpha1.ServiceConfig) error {
	for _, ins := range sc.Spec.Instances {
		name := fmt.Sprintf("%s.%s.%d", sc.Name, ins.Host, ins.Port.Number)
		found := &networkingv1beta1.WorkloadEntry{}
		err := r.Get(ctx, types.NamespacedName{Namespace: sc.Namespace, Name: name}, found)
		if err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("Can't found WorkloadEntry[%s/%s], skip", sc.Namespace, name)
				continue
			}
			return err
		}

		// Update WorkloadEntry
		klog.Infof("Update WorkloadEntry, Namespace: %s, Name: %s", found.Namespace, found.Name)
		found.Spec.Weight = ins.Weight
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			updateErr := r.Update(ctx, found)
			if updateErr == nil {
				klog.V(6).Infof("%s/%s update WorkloadEntry successfully", found.Namespace, found.Name)
				return nil
			}
			return updateErr
		})

		if err != nil {
			klog.Warningf("Update WorkloadEntry [%s] spec failed, err: %+v", name, err)
			return err
		}
	}

	return nil
}
