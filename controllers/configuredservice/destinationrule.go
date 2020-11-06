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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
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
	dr := r.buildDestinationRule(cs)
	// Set ServiceConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(cs, dr, r.Scheme); err != nil {
		klog.Errorf("[configuredservice] SetControllerReference error: %v", err)
		return err
	}

	// Check if this DestinationRule already exists
	found := &networkingv1beta1.DestinationRule{}
	err := r.Get(ctx, types.NamespacedName{Namespace: cs.Namespace, Name: cs.Name}, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("[configuredservice] creating a default DestinationRule, Namespace: %s, Name: %s", dr.Namespace, dr.Name)
			err = r.Create(ctx, dr)
			if err != nil {
				klog.Errorf("[configuredservice] create DestinationRule error: %+v", err)
				return err
			}
			return nil
		}
		klog.Errorf("[configuredservice] Get DestinationRule error: %+v", err)
	}
	return nil
}

func (r *Reconciler) buildDestinationRule(cs *meshv1alpha1.ConfiguredService) *networkingv1beta1.DestinationRule {
	var subsets []*v1beta1.Subset
	actualSubsets := r.getSubset(context.Background(), cs)
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

func getlb(s string) v1beta1.LoadBalancerSettings_SimpleLB {
	lb, ok := lbMap[s]
	if !ok {
		lb = v1beta1.LoadBalancerSettings_RANDOM
	}
	return lb
}
