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
	"fmt"
	"strings"

	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	utils "github.com/symcn/mesh-operator/pkg/utils"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileWorkloadEntry(ctx context.Context, cr *meshv1alpha1.ConfiguredService) error {
	// Get all workloadEntry of this ConfiguredService
	foundMap, err := r.getWorkloadEntriesMap(ctx, cr)
	if err != nil {
		klog.Errorf("%s/%s get WorkloadEntries error: %+v", cr.Namespace, cr.Name, err)
		return err
	}

	for _, ins := range cr.Spec.Instances {
		we := r.buildWorkloadEntry(cr, ins)

		// Set ConfiguredService instance as the owner and controller
		if err := controllerutil.SetControllerReference(cr, we, r.Scheme); err != nil {
			klog.Errorf("SetControllerReference error: %v", err)
			return err
		}

		// Check if this WorkloadEntry already exists
		found, ok := foundMap[we.Name]
		if !ok {
			klog.Infof("Creating a new WorkloadEntry, Namespace: %s, Name: %s", we.Namespace, we.Name)
			err = r.Create(ctx, we)
			if err != nil {
				klog.Errorf("Create WorkloadEntry error: %+v", err)
				return err
			}
			continue
		}
		delete(foundMap, we.Name)

		// Update WorkloadEntry
		if compareWorkloadEntry(we, found) {
			klog.Infof("Update WorkloadEntry, Namespace: %s, Name: %s", found.Namespace, found.Name)
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				we.Spec.DeepCopyInto(&found.Spec)
				found.Finalizers = we.Finalizers
				found.Labels = we.ObjectMeta.Labels

				updateErr := r.Update(ctx, found)
				if updateErr == nil {
					klog.V(6).Infof("%s/%s update WorkloadEntry successfully", we.Namespace, we.Name)
					return nil
				}
				return updateErr
			})

			if err != nil {
				klog.Warningf("Update WorkloadEntry [%s] spec failed, err: %+v", we.Name, err)
				return err
			}
		}
	}

	// Delete old WorkloadEntries
	for name, we := range foundMap {
		klog.Infof("Delete unused WorkloadEntry: %s", name)
		err := r.Delete(ctx, we)
		if err != nil {
			klog.Errorf("Delete unused WorkloadEntry error: %+v", err)
			return err
		}
	}

	// Reroute
	err = r.reconcileSubset(ctx, cr)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) buildWorkloadEntry(svc *meshv1alpha1.ConfiguredService, ins *meshv1alpha1.Instance) *networkingv1beta1.WorkloadEntry {
	name := fmt.Sprintf("%s.%s.%d", svc.Name, ins.Host, ins.Port.Number)
	labels := make(map[string]string)
	labels[r.Opt.SelectLabel] = svc.Name
	for _, k := range r.MeshConfig.Spec.WorkloadEntryLabelKeys {
		labels[k] = ins.Labels[r.MeshConfig.Spec.MeshLabelsRemap[k]]
	}

	return &networkingv1beta1.WorkloadEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.FormatToDNS1123(name),
			Namespace: svc.Namespace,
			Labels:    map[string]string{r.Opt.SelectLabel: truncated(svc.Spec.OriginalName)},
		},
		Spec: v1beta1.WorkloadEntry{
			Address: ins.Host,
			Ports:   map[string]uint32{ins.Port.Name: ins.Port.Number},
			Labels:  labels,
			Weight:  ins.Weight,
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

func (r *Reconciler) getWorkloadEntriesMap(ctx context.Context, cr *meshv1alpha1.ConfiguredService) (map[string]*networkingv1beta1.WorkloadEntry, error) {
	list := &networkingv1beta1.WorkloadEntryList{}
	labels := &client.MatchingLabels{r.Opt.SelectLabel: truncated(cr.Spec.OriginalName)}
	opts := &client.ListOptions{Namespace: cr.Namespace}
	labels.ApplyToList(opts)

	err := r.List(ctx, list, opts)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*networkingv1beta1.WorkloadEntry)
	for i := range list.Items {
		item := list.Items[i]
		m[item.Name] = &item
	}
	return m, nil
}

// To match label charactor limit
func truncated(s string) string {
	if len(s) > 62 {
		return strings.Trim(s[len(s)-62:], ".")
	}
	return strings.Trim(s, ".")
}
