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
	"github.com/mesh-operator/pkg/utils"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var lbMap = map[string]v1beta1.LoadBalancerSettings_SimpleLB{
	"ROUND_ROBIN": v1beta1.LoadBalancerSettings_ROUND_ROBIN,
	"LEAST_CONN":  v1beta1.LoadBalancerSettings_LEAST_CONN,
	"RANDOM":      v1beta1.LoadBalancerSettings_RANDOM,
	"PASSTHROUGH": v1beta1.LoadBalancerSettings_PASSTHROUGH,
}

func (r *ReconcileAppMeshConfig) reconcileDestinationRule(ctx context.Context, cr *meshv1.AppMeshConfig, svc *meshv1.Service) error {
	// Skip if the service's subset is none
	if len(svc.Subsets) == 0 {
		return nil
	}

	dr := buildDestinationRule(cr, svc)
	// Set AppMeshConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, dr, r.scheme); err != nil {
		return err
	}

	// Check if this DestinationRule already exists
	found := &networkingv1beta1.DestinationRule{}
	err := r.client.Get(ctx, types.NamespacedName{Name: dr.Name, Namespace: dr.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		klog.Info("Creating a new DestinationRule", "Namespace", dr.Namespace, "Name", dr.Name)
		err = r.client.Create(ctx, dr)
		if err != nil {
			return err
		}

		// DestinationRule created successfully - don't requeue
		return nil
	} else if err != nil {
		return err
	}

	// Update DestinationRule
	if compareDestinationRule(dr, found) {
		klog.Info("Update DestinationRule", "Namespace", found.Namespace, "Name", found.Name)
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			dr.Spec.DeepCopyInto(&found.Spec)
			found.Finalizers = dr.Finalizers
			found.Labels = dr.ObjectMeta.Labels

			updateErr := r.client.Update(ctx, found)
			if updateErr == nil {
				klog.V(4).Infof("%s/%s update DestinationRule successfully", dr.Namespace, dr.Name)
				return nil
			}
			return updateErr
		})

		if err != nil {
			klog.Warningf("update DestinationRule [%s] spec failed, err: %+v", dr.Name, err)
			return err
		}
	}

	return nil
}

func buildDestinationRule(cr *meshv1.AppMeshConfig, svc *meshv1.Service) *networkingv1beta1.DestinationRule {
	var subsets []*v1beta1.Subset
	for _, sub := range svc.Subsets {
		subset := &v1beta1.Subset{Name: sub.Name, Labels: sub.Labels}
		if sub.Policy != nil {
			subset.TrafficPolicy = &v1beta1.TrafficPolicy{
				LoadBalancer: &v1beta1.LoadBalancerSettings{
					LbPolicy: &v1beta1.LoadBalancerSettings_Simple{
						Simple: getlb(sub.Policy.LoadBalancer["simple"]),
					},
				},
			}
		}
		subsets = append(subsets, subset)
	}
	return &networkingv1beta1.DestinationRule{
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.FormatToDNS1123(svc.Name),
			Namespace: cr.Namespace,
		},
		Spec: v1beta1.DestinationRule{
			Host: svc.Name,
			TrafficPolicy: &v1beta1.TrafficPolicy{
				LoadBalancer: &v1beta1.LoadBalancerSettings{
					LbPolicy: &v1beta1.LoadBalancerSettings_Simple{
						Simple: getlb(svc.Policy.LoadBalancer["simple"]),
					},
				},
			},
			Subsets: subsets,
		},
	}
}

func compareDestinationRule(new, old *networkingv1beta1.DestinationRule) bool {
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

func getlb(s string) v1beta1.LoadBalancerSettings_SimpleLB {
	lb, ok := lbMap[s]
	if !ok {
		lb = v1beta1.LoadBalancerSettings_RANDOM
	}
	return lb
}
