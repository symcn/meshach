/*
Copyright 2020 The Symcn Authors.
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

	ptypes "github.com/gogo/protobuf/types"
	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	"github.com/symcn/mesh-operator/pkg/utils"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileVirtualService(ctx context.Context, cr *meshv1alpha1.ConfiguredService) error {
	foundMap, err := r.getVirtualServicesMap(ctx, cr)
	if err != nil {
		klog.Errorf("%s/%s get VirtualService error: %+v", cr.Namespace, cr.Name, err)
		return err
	}

	// Skip if the service's subset is none
	if len(cr.Spec.Subsets) != 0 {
		vs := r.buildVirtualService(cr)
		// Set ConfiguredService instance as the owner and controller
		if err := controllerutil.SetControllerReference(cr, vs, r.Scheme); err != nil {
			klog.Errorf("SetControllerReference error: %v", err)
			return err
		}

		// Check if this VirtualService already exists
		found, ok := foundMap[vs.Name]
		if !ok {
			klog.Infof("Creating a new VirtualService, Namespace: %s, Name: %s", vs.Namespace, vs.Name)
			err = r.Create(ctx, vs)
			if err != nil {
				klog.Errorf("Create VirtualService error: %+v", err)
				return err
			}
		} else {
			// Update VirtualService
			if compareVirtualService(vs, found) {
				klog.Infof("Update VirtualService, Namespace: %s, Name: %s",
					found.Namespace, found.Name)
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					vs.Spec.DeepCopyInto(&found.Spec)
					found.Finalizers = vs.Finalizers
					found.Labels = vs.ObjectMeta.Labels

					updateErr := r.Update(ctx, found)
					if updateErr == nil {
						klog.V(4).Infof("%s/%s update VirtualService successfully",
							vs.Namespace, vs.Name)
						return nil
					}
					return updateErr
				})

				if err != nil {
					klog.Warningf("Update VirtualService [%s] spec failed, err: %+v", vs.Name, err)
					return err
				}
			}
			delete(foundMap, vs.Name)
		}
	}

	// Delete old VirtualServices
	for name, vs := range foundMap {
		klog.Infof("Delete unused VirtualService: %s", name)
		err := r.Delete(ctx, vs)
		if err != nil {
			klog.Errorf("Delete unused VirtualService error: %+v", err)
			return err
		}
	}
	return nil
}

func (r *Reconciler) buildVirtualService(svc *meshv1alpha1.ConfiguredService) *networkingv1beta1.VirtualService {
	httpRoute := []*v1beta1.HTTPRoute{}
	for _, sourceLabels := range svc.Spec.Policy.SourceLabels {
		http := r.buildHTTPRoute(svc, sourceLabels)
		httpRoute = append(httpRoute, http)
	}

	if svc.Spec.RerouteOption == nil || svc.Spec.RerouteOption.ReroutePolicy != meshv1alpha1.Unchangeable {
		defaultRoute := r.buildDefaultRoute(svc)
		httpRoute = append(httpRoute, defaultRoute)
	}

	return &networkingv1beta1.VirtualService{
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.FormatToDNS1123(svc.Name),
			Namespace: svc.Namespace,
			Labels:    map[string]string{r.Opt.SelectLabel: truncated(svc.Spec.OriginalName)},
		},
		Spec: v1beta1.VirtualService{
			Hosts: []string{svc.Name},
			Http:  httpRoute,
		},
	}
}

func (r *Reconciler) buildHTTPRoute(svc *meshv1alpha1.ConfiguredService, sourceLabels *meshv1alpha1.SourceLabels) *v1beta1.HTTPRoute {
	// m := make(map[string]*v1beta1.StringMatch)
	// for key, matchType := range r.MeshConfig.Spec.MatchHeaderLabelKeys {
	// m[key] = getMatchType(matchType, sourceLabels.Headers[key])
	// }
	// klog.V(4).Infof("match header map: %+v", m)
	s := make(map[string]string)
	for _, key := range r.MeshConfig.Spec.MatchSourceLabelKeys {
		s[key] = sourceLabels.Labels[key]
	}
	// klog.V(4).Infof("match sourceLabels map: %+v", m)

	match := &v1beta1.HTTPMatchRequest{SourceLabels: s}

	var routes []*v1beta1.HTTPRouteDestination
	for _, destination := range sourceLabels.Route {
		route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: svc.Name}}
		if destination.Subset != "" {
			route.Destination.Subset = destination.Subset
		}
		if destination.Weight != 0 {
			route.Weight = destination.Weight
		}
		routes = append(routes, route)
	}

	return &v1beta1.HTTPRoute{
		Name:    httpRouteName + "-" + sourceLabels.Name,
		Match:   []*v1beta1.HTTPMatchRequest{match},
		Route:   routes,
		Timeout: utils.StringToDuration(svc.Spec.Policy.Timeout, int64(svc.Spec.Policy.MaxRetries)),
		Retries: &v1beta1.HTTPRetry{
			Attempts:      svc.Spec.Policy.MaxRetries,
			PerTryTimeout: utils.StringToDuration(svc.Spec.Policy.Timeout, 1),
			RetryOn:       r.Opt.ProxyRetryOn,
		},
	}
}

func (r *Reconciler) buildDefaultRoute(svc *meshv1alpha1.ConfiguredService) *v1beta1.HTTPRoute {
	route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: svc.Name}}
	return &v1beta1.HTTPRoute{
		Name:  defaultRouteName,
		Route: []*v1beta1.HTTPRouteDestination{route},
		Retries: &v1beta1.HTTPRetry{
			Attempts:      r.Opt.ProxyAttempts,
			PerTryTimeout: &ptypes.Duration{Seconds: r.Opt.ProxyPerTryTimeout},
			RetryOn:       r.Opt.ProxyRetryOn,
		},
	}
}

func compareVirtualService(new, old *networkingv1beta1.VirtualService) bool {
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

func getMatchType(matchType meshv1alpha1.StringMatchType, value string) *v1beta1.StringMatch {
	s := &v1beta1.StringMatch{}
	switch matchType {
	case meshv1alpha1.Prefix:
		s.MatchType = &v1beta1.StringMatch_Prefix{Prefix: value}
	case meshv1alpha1.Regex:
		s.MatchType = &v1beta1.StringMatch_Regex{Regex: value}
	default:
		s.MatchType = &v1beta1.StringMatch_Exact{Exact: value}
	}
	return s
}

func (r *Reconciler) getVirtualServicesMap(ctx context.Context, cr *meshv1alpha1.ConfiguredService) (map[string]*networkingv1beta1.VirtualService, error) {
	list := &networkingv1beta1.VirtualServiceList{}
	labels := &client.MatchingLabels{r.Opt.SelectLabel: truncated(cr.Spec.OriginalName)}
	opts := &client.ListOptions{Namespace: cr.Namespace}
	labels.ApplyToList(opts)

	err := r.List(ctx, list, opts)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*networkingv1beta1.VirtualService)
	for i := range list.Items {
		item := list.Items[i]
		m[item.Name] = &item
	}
	return m, nil
}
