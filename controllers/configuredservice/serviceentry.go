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

	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/utils"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileServiceEntry(ctx context.Context, cr *meshv1alpha1.ConfiguredService) error {
	foundMap, err := r.getServiceEntriesMap(ctx, cr)
	if err != nil {
		klog.Errorf("%s/%s get ServiceEntries error: %+v", cr.Namespace, cr.Name, err)
		return err
	}

	se := r.buildServiceEntry(cr)
	// Set ConfiguredService instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, se, r.Scheme); err != nil {
		klog.Errorf("SetControllerReference error: %v", err)
		return err
	}

	// Check if this ServiceEntry already exists
	found, ok := foundMap[se.Name]
	if !ok {
		klog.Infof("Creating a new ServiceEntry, Namespace: %s, Name: %s", se.Namespace, se.Name)
		err = r.Create(ctx, se)
		if err != nil {
			klog.Errorf("Create ServiceEntry error: %+v", err)
			return err
		}
	} else {
		// Update ServiceEntry
		if compareServiceEntry(se, found) {
			klog.Infof("Update ServiceEntry, Namespace: %s, Name: %s", found.Namespace, found.Name)
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				se.Spec.DeepCopyInto(&found.Spec)
				found.Finalizers = se.Finalizers
				found.Labels = se.ObjectMeta.Labels

				updateErr := r.Update(ctx, found)
				if updateErr == nil {
					klog.V(4).Infof("%s/%s update ServiceEntry successfully", se.Namespace, se.Name)
					return nil
				}
				return updateErr
			})

			if err != nil {
				klog.Warningf("Update ServiceEntry [%s] spec failed, err: %+v", se.Name, err)
				return err
			}
		}
		delete(foundMap, se.Name)
	}
	// Delete old ServiceEntry
	for name, se := range foundMap {
		klog.Infof("Delete unused ServiceEntry: %s", name)
		err := r.Delete(ctx, se)
		if err != nil {
			klog.Errorf("Delete unused ServiceEntry error: %+v", err)
			return err
		}
	}

	return nil
}

func (r *Reconciler) buildServiceEntry(svc *meshv1alpha1.ConfiguredService) *networkingv1beta1.ServiceEntry {
	var ports []*v1beta1.Port
	for _, port := range svc.Spec.Ports {
		ports = append(ports, &v1beta1.Port{
			Number:   port.Number,
			Protocol: port.Protocol,
			Name:     port.Name,
		})
	}

	return &networkingv1beta1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.FormatToDNS1123(svc.Name),
			Namespace: svc.Namespace,
			Labels:    map[string]string{r.Opt.SelectLabel: truncated(svc.Spec.OriginalName)},
		},
		Spec: v1beta1.ServiceEntry{
			Hosts:      []string{svc.Name},
			Ports:      ports,
			Location:   v1beta1.ServiceEntry_MESH_INTERNAL,
			Resolution: v1beta1.ServiceEntry_STATIC,
			WorkloadSelector: &v1beta1.WorkloadSelector{
				Labels: map[string]string{r.Opt.SelectLabel: svc.Name},
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

func (r *Reconciler) getServiceEntriesMap(ctx context.Context, cr *meshv1alpha1.ConfiguredService) (map[string]*networkingv1beta1.ServiceEntry, error) {
	list := &networkingv1beta1.ServiceEntryList{}
	labels := &client.MatchingLabels{r.Opt.SelectLabel: truncated(cr.Spec.OriginalName)}
	opts := &client.ListOptions{Namespace: cr.Namespace}
	labels.ApplyToList(opts)

	err := r.List(ctx, list, opts)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*networkingv1beta1.ServiceEntry)
	for i := range list.Items {
		item := list.Items[i]
		m[item.Name] = &item
	}
	return m, nil
}
