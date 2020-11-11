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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	loadBalanceSimple = "simple"
)

var lbMap = map[string]v1beta1.LoadBalancerSettings_SimpleLB{
	"ROUND_ROBIN": v1beta1.LoadBalancerSettings_ROUND_ROBIN,
	"LEAST_CONN":  v1beta1.LoadBalancerSettings_LEAST_CONN,
	"RANDOM":      v1beta1.LoadBalancerSettings_RANDOM,
	"PASSTHROUGH": v1beta1.LoadBalancerSettings_PASSTHROUGH,
}

func (r *Reconciler) reconcileDestinationRule(ctx context.Context, cs *meshv1alpha1.ConfiguredService) error {
	foundMap, err := r.getDestinationRuleMap(ctx, cs)
	if err != nil {
		klog.Errorf("[cs] %s/%s get DestinationRules error: %+v", cs.Namespace, cs.Name, err)
		return err
	}

	actualSubsets := r.getSubset(context.Background(), cs)
	// Skip if the service's subset is none
	klog.V(6).Infof("[cs] destinationrule subsets length: %d", len(actualSubsets))

	if len(actualSubsets) != 0 {
		dr := r.buildDestinationRule(cs, actualSubsets)
		// Set ServiceConfig instance as the owner and controller
		if err := controllerutil.SetControllerReference(cs, dr, r.Scheme); err != nil {
			klog.Errorf("[cs] SetControllerReference error: %v", err)
			return err
		}

		// Check if this DestinationRule already exists
		found, ok := foundMap[dr.Name]
		if !ok {
			klog.Infof("[cs] creating a default DestinationRule, Namespace: %s, Name: %s", dr.Namespace, dr.Name)
			err = r.Create(ctx, dr)
			if err != nil {
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				klog.Errorf("[cs] Create DestinationRule error: %+v", err)
				return err
			}
		} else {
			// Update DestinationRule
			if compareDestinationRule(dr, found) {
				klog.Infof("[cs] Update DestinationRule, Namespace: %s, Name: %s",
					found.Namespace, found.Name)
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					// TODO get found
					dr.Spec.DeepCopyInto(&found.Spec)
					found.Finalizers = dr.Finalizers
					found.Labels = dr.ObjectMeta.Labels

					updateErr := r.Update(ctx, found)
					if updateErr == nil {
						klog.Infof("[cs] %s/%s update DestinationRule successfully",
							dr.Namespace, dr.Name)
						return nil
					}
					return updateErr
				})

				if err != nil {
					klog.Warningf("[cs] Update DestinationRule [%s] spec failed, err: %+v", dr.Name, err)
					return err
				}
			}
			delete(foundMap, dr.Name)
		}
		// Delete old DestinationRules
		for name, dr := range foundMap {
			klog.Infof("[cs] Delete unused DestinationRule: %s", name)
			err := r.Delete(ctx, dr)
			if err != nil {
				klog.Errorf("[cs] Delete unused DestinationRule error: %+v", err)
				return err
			}
		}
	}
	return nil
}

func (r *Reconciler) buildDestinationRule(cs *meshv1alpha1.ConfiguredService, actualSubsets []*meshv1alpha1.Subset) *networkingv1beta1.DestinationRule {
	var subsets []*v1beta1.Subset
	for _, sub := range actualSubsets {
		subset := &v1beta1.Subset{Name: sub.Name, Labels: sub.Labels}
		if sub.Policy != nil {
			subset.TrafficPolicy = &v1beta1.TrafficPolicy{
				LoadBalancer: &v1beta1.LoadBalancerSettings{
					LbPolicy: &v1beta1.LoadBalancerSettings_Simple{
						Simple: getlb(sub.Policy.LoadBalancer[loadBalanceSimple]),
					},
				},
			}
		}
		subsets = append(subsets, subset)
	}
	return &networkingv1beta1.DestinationRule{
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.FormatToDNS1123(cs.Name),
			Namespace: cs.Namespace,
			Labels:    map[string]string{r.Opt.SelectLabel: truncated(cs.Spec.OriginalName)},
		},
		Spec: v1beta1.DestinationRule{
			Host: cs.Name,
			TrafficPolicy: &v1beta1.TrafficPolicy{
				LoadBalancer: &v1beta1.LoadBalancerSettings{
					LbPolicy: &v1beta1.LoadBalancerSettings_Simple{
						Simple: getlb(r.MeshConfig.Spec.GlobalPolicy.LoadBalancer[loadBalanceSimple]),
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

func (r *Reconciler) getDestinationRuleMap(ctx context.Context, cs *meshv1alpha1.ConfiguredService) (map[string]*networkingv1beta1.DestinationRule, error) {
	list := &networkingv1beta1.DestinationRuleList{}
	labels := &client.MatchingLabels{r.Opt.SelectLabel: truncated(cs.Spec.OriginalName)}
	opts := &client.ListOptions{Namespace: cs.Namespace}
	labels.ApplyToList(opts)

	err := r.List(ctx, list, opts)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*networkingv1beta1.DestinationRule)
	for i := range list.Items {
		item := list.Items[i]
		m[item.Name] = &item
	}
	return m, nil
}
