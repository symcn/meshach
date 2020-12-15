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
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) reconcileSidecar(ctx context.Context, cs *meshv1alpha1.ConfiguredService) error {
	if !r.Opt.InitDefaultSidecars {
		r.initDefaultSidecarsByDeploy(ctx)
		r.Opt.InitDefaultSidecars = true
	}

	// Get namespace of app pods
	appName := r.getAppName(cs)
	if appName == "" {
		return nil
	}

	namespaces := r.getAppNamespace(appName)
	for _, ns := range namespaces {
		found := &networkingv1beta1.Sidecar{}
		err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: appName}, found)
		// Check if this Sidecar already exists
		if err != nil {
			if apierrors.IsNotFound(err) {
				sidecar := r.buildSidecar(ns, appName)
				klog.Infof("[cs] creating a default Sidecar, Namespace: %s, Name: %s", sidecar.Namespace, sidecar.Name)
				createErr := r.Create(ctx, sidecar)
				if createErr != nil {
					klog.Errorf("[cs] Create Sidecar error: %+v", createErr)
					return createErr
				}
			}
			klog.Errorf("[cs] Get Sidecar error: %+v", err)
			return err
		}
	}
	return nil
}

func (r *Reconciler) initDefaultSidecarsByDeploy(ctx context.Context) {
	opts := &client.ListOptions{}
	deps := &appv1.DeploymentList{}
	err := r.List(ctx, deps, opts)

	if err != nil {
		klog.Errorf("[cs] get Deployments error: %+v", err)
	}

	for _, dep := range deps.Items {
		if appName, ok := dep.GetLabels()[r.MeshConfig.Spec.SidecarSelectLabel]; ok {
			s := r.buildSidecar(dep.Namespace, appName)
			klog.Infof("[cs] creating a default Sidecar, Namespace: %s, Name: %s", s.Namespace, s.Name)
			createErr := r.Create(ctx, s)
			if createErr != nil {
				klog.Errorf("[cs] Create Sidecar error: %+v", createErr)
			}
		}
	}
}

func (r *Reconciler) buildSidecar(ns, appName string) *networkingv1beta1.Sidecar {
	egress := r.buildDefaultEgress()
	return &networkingv1beta1.Sidecar{
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: ns,
		},
		Spec: v1beta1.Sidecar{
			WorkloadSelector: &v1beta1.WorkloadSelector{
				Labels: map[string]string{
					r.MeshConfig.Spec.SidecarSelectLabel: appName,
				},
			},
			Egress: egress,
			OutboundTrafficPolicy: &v1beta1.OutboundTrafficPolicy{
				Mode: v1beta1.OutboundTrafficPolicy_ALLOW_ANY,
			},
		},
	}
}

func (r *Reconciler) buildDefaultEgress() []*v1beta1.IstioEgressListener {
	var hosts []string
	if len(r.MeshConfig.Spec.SidecarDefaultHosts) > 0 {
		hosts = append(hosts, r.MeshConfig.Spec.SidecarDefaultHosts...)
	} else {
		hosts = append(hosts, "istio-system/*")
	}

	return []*v1beta1.IstioEgressListener{{
		Hosts: hosts,
	}}
}

func (r *Reconciler) getAppNamespace(appName string) []string {
	var namespaces []string

	pods := &corev1.PodList{}
	labels := &client.MatchingLabels{r.MeshConfig.Spec.SidecarSelectLabel: appName}
	opts := &client.ListOptions{}
	labels.ApplyToList(opts)

	err := r.List(context.TODO(), pods, opts)
	if err != nil {
		klog.Warningf("Get pods error when create Sidecar[%s]: %+v", appName, err)
	}

	if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			namespaces = append(namespaces, pod.Namespace)
		}
	} else {
		klog.Warningf("No pods founds, skip create Sidecar[%s]", appName)
	}

	return namespaces
}

func (r *Reconciler) getAppName(cs *meshv1alpha1.ConfiguredService) string {
	for _, ep := range cs.Spec.Instances {
		if appName, ok := ep.Labels[r.MeshConfig.Spec.SidecarSelectLabel]; ok {
			return appName
		}
	}
	return ""
}
