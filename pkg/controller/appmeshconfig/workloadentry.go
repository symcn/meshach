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
	"strings"

	meshv1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	"github.com/mesh-operator/pkg/utils"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileAppMeshConfig) reconcileWorkloadEntry(ctx context.Context, cr *meshv1.AppMeshConfig, svc *meshv1.Service) error {
	// Get all workloadEntry of this AppMeshConfig
	foundMap, err := r.getWorkloadEntriesMap(ctx, cr)
	if err != nil {
		klog.Errorf("%s/%s get WorkloadEntries error: %+v", cr.Namespace, cr.Spec.AppName, err)
		return err
	}

	for _, ins := range svc.Instances {
		we := buildWorkloadEntry(cr.Spec.AppName, cr.Namespace, svc, ins)

		// Set AppMeshConfig instance as the owner and controller
		if err := controllerutil.SetControllerReference(cr, we, r.scheme); err != nil {
			klog.Errorf("SetControllerReference error: %v", err)
			return err
		}

		// Check if this WorkloadEntry already exists
		found, ok := foundMap[we.Name]
		if !ok {
			klog.Infof("Creating a new WorkloadEntry, Namespace: %s, Name: %s", we.Namespace, we.Name)
			err = r.client.Create(ctx, we)
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

				updateErr := r.client.Update(ctx, found)
				if updateErr == nil {
					klog.V(4).Infof("%s/%s update WorkloadEntry successfully",
						we.Namespace, we.Name)
					return nil
				}
				return updateErr
			})

			if err != nil {
				klog.Warningf("Update WorkloadEntry [%s] spec failed, err: %+v",
					we.Name, err)
				return err
			}
		}
	}

	// Delete old WorkloadEntries
	for name, we := range foundMap {
		klog.V(4).Infof("Delete unused WorkloadEntry: %s", name)
		err := r.client.Delete(ctx, we)
		if err != nil {
			klog.Errorf("Delete unused WorkloadEntry error: %+v", err)
			return err
		}
	}
	return nil
}

func buildWorkloadEntry(appName, namespace string, svc *meshv1.Service, ins *meshv1.Instance) *networkingv1beta1.WorkloadEntry {
	name := utils.FormatToDNS1123(strings.Join([]string{
		svc.Name, ins.Host, string(ins.Port.Number)}, "."))
	return &networkingv1beta1.WorkloadEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": appName},
		},
		Spec: v1beta1.WorkloadEntry{
			Address: ins.Host,
			Ports: map[string]uint32{
				ins.Port.Name: ins.Port.Number,
			},
			Labels: map[string]string{
				"service":   svc.Name,
				"sym-group": ins.Group,
				"sym-zone":  ins.Zone,
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

func (r *ReconcileAppMeshConfig) getWorkloadEntriesMap(ctx context.Context, cr *meshv1.AppMeshConfig) (map[string]*networkingv1beta1.WorkloadEntry, error) {
	list := &networkingv1beta1.WorkloadEntryList{}
	labels := &client.MatchingLabels{"app": cr.Spec.AppName}
	opts := &client.ListOptions{Namespace: cr.Namespace}
	labels.ApplyToList(opts)

	err := r.client.List(ctx, list, opts)
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
