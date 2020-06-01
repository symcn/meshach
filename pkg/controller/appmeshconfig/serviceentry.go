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

func (r *ReconcileAppMeshConfig) reconcileServiceEntry(cr *meshv1.AppMeshConfig, svc *meshv1.Service) error {
	se := buildServiceEntry(cr, svc)
	// Set AppMeshConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, se, r.scheme); err != nil {
		return err
	}

	// Check if this ServiceEntry already exists
	found := &networkingv1beta1.ServiceEntry{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: se.Name, Namespace: se.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		klog.Info("Creating a new ServiceEntry", "Namespace", se.Namespace, "Name", se.Name)
		err = r.client.Create(context.TODO(), se)
		if err != nil {
			return err
		}

		// ServiceEntry created successfully - don't requeue
		return nil
	} else if err != nil {
		return err
	}

	// Update ServiceEntry
	klog.Info("Update ServiceEntry", "Namespace", found.Namespace, "Name", found.Name)
	if compareServiceEntry(se, found) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			se.Spec.DeepCopyInto(&found.Spec)
			found.Finalizers = se.Finalizers
			found.Labels = se.ObjectMeta.Labels

			updateErr := r.client.Update(context.TODO(), found)
			if updateErr == nil {
				klog.V(4).Infof("%s/%s update ServiceEntry successfully", se.Namespace, se.Name)
				return nil
			}
			return updateErr
		})

		if err != nil {
			klog.Warningf("update ServiceEntry [%s] spec failed, err: %+v", se.Name, err)
			return err
		}
	}

	return nil
}

func buildServiceEntry(cr *meshv1.AppMeshConfig, svc *meshv1.Service) *networkingv1beta1.ServiceEntry {
	var ports []*v1beta1.Port
	for _, port := range svc.Ports {
		ports = append(ports, &v1beta1.Port{
			Number:   port.Number,
			Protocol: port.Protocol,
			Name:     port.Name,
		})
	}

	return &networkingv1beta1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name + "-se",
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": svc.AppName,
			},
			// Finalizers: nil,
		},
		Spec: v1beta1.ServiceEntry{
			Hosts:      []string{svc.Name},
			Ports:      ports,
			Location:   v1beta1.ServiceEntry_MESH_INTERNAL,
			Resolution: v1beta1.ServiceEntry_STATIC,
			WorkloadSelector: &v1beta1.WorkloadSelector{
				Labels: map[string]string{
					"app":     svc.AppName,
					"service": svc.Name + ".workload",
				},
			},
		},
	}
}

func compareServiceEntry(new, old *networkingv1beta1.ServiceEntry) bool {
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
