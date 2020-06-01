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

package appmeshconfig

import (
	"context"

	meshv1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileAppMeshConfig) reconcileWorkloadEntry(cr *meshv1.AppMeshConfig, svc *meshv1.Service) error {
	for _, ins := range svc.Instances {
		we := buildWorkloadEntry(cr.Namespace, svc, ins)

		// Set AppMeshConfig instance as the owner and controller
		if err := controllerutil.SetControllerReference(cr, we, r.scheme); err != nil {
			return err
		}
		// Check if this WorkloadEntry already exists
		found := &networkingv1beta1.WorkloadEntry{}
		err := r.client.Get(context.TODO(), types.NamespacedName{
			Name:      we.Name,
			Namespace: we.Namespace},
			found)
		if err != nil && errors.IsNotFound(err) {
			klog.Info("Creating a new WorkloadEntry", "Namespace",
				we.Namespace, "Name", we.Name)
			err = r.client.Create(context.TODO(), we)
			if err != nil {
				return err
			}

			// WorkloadEntry created successfully - don't requeue
			return nil
		} else if err != nil {
			return err
		}

		// Update WorkloadEntry
		klog.Info("Update WorkloadEntry", "Namespace", found.Namespace, "Name", found.Name)
		if compareWorkloadEntry(we, found) {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				we.Spec.DeepCopyInto(&found.Spec)
				found.Finalizers = we.Finalizers
				found.Labels = we.ObjectMeta.Labels

				updateErr := r.client.Update(context.TODO(), found)
				if updateErr == nil {
					klog.V(4).Infof("%s/%s update WorkloadEntry successfully",
						we.Namespace, we.Name)
					return nil
				}
				return updateErr
			})

			if err != nil {
				klog.Warningf("update WorkloadEntry [%s] spec failed, err: %+v",
					we.Name, err)
				return err
			}
		}
	}
	return nil
}

func buildWorkloadEntry(namespace string, svc *meshv1.Service, ins *meshv1.Instance) *networkingv1beta1.WorkloadEntry {
	var ports []*v1beta1.Port
	for _, port := range svc.Ports {
		ports = append(ports, &v1beta1.Port{
			Number:   port.Number,
			Protocol: port.Protocol,
			Name:     port.Name,
		})
	}

	return &networkingv1beta1.WorkloadEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name + "-we-" + ins.Host,
			Namespace: namespace,
		},
		Spec: v1beta1.WorkloadEntry{
			Address: ins.Host,
			Ports: map[string]uint32{
				ins.Port.Name: ins.Port.Number,
			},
			Labels: map[string]string{
				"app":     svc.AppName,
				"service": svc.Name + ".workload",
				"group":   ins.Group,
				"zone":    ins.Zone,
			},
			// Network:        ins.Zone,
			// Locality:       "",
			Weight: ins.Weight,
		},
	}
}

func compareWorkloadEntry(new, old *networkingv1beta1.WorkloadEntry) bool {
	if !equality.Semantic.DeepEqual(new.ObjectMeta.Finalizers, old.ObjectMeta.Finalizers) {
		return true
	}

	if !equality.Semantic.DeepEqual(new.ObjectMeta.Labels, old.ObjectMeta.Labels) {
		return true
	}

	if !equality.Semantic.DeepEqual(new.Spec, old.Spec) {
		return true
	}
	return false
}
