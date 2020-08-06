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

package serviceconfig

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

func (r *Reconciler) reconcileVirtualService(ctx context.Context, sc *meshv1alpha1.ServiceConfig) error {
	foundMap, err := r.getVirtualServicesMap(ctx, sc)
	if err != nil {
		klog.Errorf("%s/%s get VirtualService error: %+v", sc.Namespace, sc.Name, err)
		return err
	}

	subsets, err := r.getSubset(ctx, sc)
	if err != nil {
		klog.Errorf("Get subsets[%s/%s] error: %+v", sc.Namespace, sc.Name, err)
		return err
	}

	// Skip if the service's subset is none
	if len(subsets) != 0 {
		vs := r.buildVirtualService(sc, subsets)
		// Set ServiceConfig instance as the owner and controller
		if err := controllerutil.SetControllerReference(sc, vs, r.Scheme); err != nil {
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

func (r *Reconciler) buildVirtualService(svc *meshv1alpha1.ServiceConfig, actualSubsets []*meshv1alpha1.Subset) *networkingv1beta1.VirtualService {
	httpRoute := []*v1beta1.HTTPRoute{}
	for _, subset := range r.MeshConfig.Spec.GlobalSubsets {
		http := r.buildHTTPRoute(svc, subset, actualSubsets)
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

func (r *Reconciler) buildHTTPRoute(svc *meshv1alpha1.ServiceConfig, subset *meshv1alpha1.Subset, actualSubsets []*meshv1alpha1.Subset) *v1beta1.HTTPRoute {
	var buildRoutes []*v1beta1.HTTPRouteDestination
	reroute := len(actualSubsets) < len(r.MeshConfig.Spec.GlobalSubsets)
	switch {
	case svc.Spec.Route != nil && len(svc.Spec.Route) > 0 && !subset.IsCanary && !reroute:
		dynamicRoute(svc.Name, buildRoutes, svc.Spec.Route)
	case reroute:
		rerouteSubset(svc.Name, buildRoutes, subset, actualSubsets)
	default:
		defaultDestination(buildRoutes, svc.Name, subset.Name)
	}

	// set SourceLabels in match
	s := make(map[string]string)
	for _, key := range r.MeshConfig.Spec.MatchSourceLabelKeys {
		s[key] = subset.Labels[key]
	}
	match := &v1beta1.HTTPMatchRequest{SourceLabels: s}

	return &v1beta1.HTTPRoute{
		Name:    httpRouteName + "-" + subset.Name,
		Match:   []*v1beta1.HTTPMatchRequest{match},
		Route:   buildRoutes,
		Timeout: utils.StringToDuration(svc.Spec.Policy.Timeout, int64(svc.Spec.Policy.MaxRetries)),
		Retries: &v1beta1.HTTPRetry{
			Attempts:      svc.Spec.Policy.MaxRetries,
			PerTryTimeout: utils.StringToDuration(svc.Spec.Policy.Timeout, 1),
			RetryOn:       r.Opt.ProxyRetryOn,
		},
	}
}

func (r *Reconciler) buildDefaultRoute(svc *meshv1alpha1.ServiceConfig) *v1beta1.HTTPRoute {
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

// A specific routing strategy is preferred if svc.Spec.Route exists.
// NOTE: The canary group don't care specific routing strategy.
func dynamicRoute(host string, buildRoutes []*v1beta1.HTTPRouteDestination, dynamicRoutes []*meshv1alpha1.Destination) []*v1beta1.HTTPRouteDestination {
	for _, destination := range dynamicRoutes {
		route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: host}}
		if destination.Subset != "" {
			route.Destination.Subset = destination.Subset
		}
		if destination.Weight != 0 {
			route.Weight = destination.Weight
		}
		buildRoutes = append(buildRoutes, route)
	}
	return buildRoutes
}

func rerouteSubset(host string, buildRoutes []*v1beta1.HTTPRouteDestination, subset *meshv1alpha1.Subset, actualSubsets []*meshv1alpha1.Subset) []*v1beta1.HTTPRouteDestination {
	// The consumer of canary subset can only call the provider of the canary subset. When the
	// Provider instance of a canary subset is empty, the traffic of the canary consumer is
	// evenly distributed to other normal subsets.
	switch {
	case subset.IsCanary && !subset.In(actualSubsets):
		for i, actual := range actualSubsets {
			route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: host}}
			route.Destination.Subset = actual.Name
			route.Weight = int32(100 / len(actualSubsets))
			if i == len(actualSubsets)-1 && len(actualSubsets) > 1 {
				route.Weight = 100 - int32(100/len(actualSubsets))
			}
			buildRoutes = append(buildRoutes, route)
		}
	case !subset.IsCanary && !subset.In(actualSubsets):
		route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: host}}
		route.Destination.Subset = subset.Name
		route.Weight = 0
		buildRoutes = append(buildRoutes, route)
		// Calculate the available subsets length, where you need to exclude the Canary first
		l := len(actualSubsets)
		for _, actual := range actualSubsets {
			if actual.IsCanary {
				l--
				continue
			}
		}
		// Distribute traffic evenly among the remaining available subsets
		for i, actual := range actualSubsets {
			if actual.IsCanary {
				continue
			}
			route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: host}}
			route.Destination.Subset = actual.Name
			route.Weight = int32(100 / l)
			if i == len(actualSubsets)-1 && len(actualSubsets) > 1 {
				route.Weight = 100 - int32(100/l)
			}
			buildRoutes = append(buildRoutes, route)
		}
	// case subset.IsCanary && subset.In(actualSubsets):
	// 	defaultDestination(buildRoutes, host, subset.Name)
	// case !subset.IsCanary && subset.In(actualSubsets):
	// 	defaultDestination(buildRoutes, host, subset.Name)
	default:
		defaultDestination(buildRoutes, host, subset.Name)
	}
	return buildRoutes
}

func defaultDestination(buildRoutes []*v1beta1.HTTPRouteDestination, host, subset string) []*v1beta1.HTTPRouteDestination {
	route := &v1beta1.HTTPRouteDestination{
		Destination: &v1beta1.Destination{Host: host, Subset: subset},
		Weight:      100,
	}
	buildRoutes = append(buildRoutes, route)
	return buildRoutes
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

func (r *Reconciler) getVirtualServicesMap(ctx context.Context, sc *meshv1alpha1.ServiceConfig) (map[string]*networkingv1beta1.VirtualService, error) {
	list := &networkingv1beta1.VirtualServiceList{}
	labels := &client.MatchingLabels{r.Opt.SelectLabel: truncated(sc.Spec.OriginalName)}
	opts := &client.ListOptions{Namespace: sc.Namespace}
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
