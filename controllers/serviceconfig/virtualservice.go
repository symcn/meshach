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

	subsets, err := r.getSubset(context.Background(), sc)
	if err != nil {
		klog.Errorf("Get subsets[%s/%s] error: %+v", sc.Namespace, sc.Name, err)
		return err
	}

	// Skip if the service's subset is none
	klog.V(6).Infof("virtualservice subsets length: %d", len(subsets))
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
						klog.Infof("%s/%s update VirtualService successfully",
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
	klog.V(6).Infof("%s length of actual subsets: %d, global subsets: %d", svc.Name, len(actualSubsets), len(r.MeshConfig.Spec.GlobalSubsets))
	reroute := len(actualSubsets) < len(r.MeshConfig.Spec.GlobalSubsets)
	for _, subset := range actualSubsets {
		klog.V(6).Infof("%s actual subset name: %s, %+v", svc.Name, subset.Name, subset.Labels)
	}
	switch {
	case len(svc.Spec.Route) > 0 && !subset.IsCanary && onlyMissingCanary(actualSubsets, r.MeshConfig.Spec.GlobalSubsets):
		klog.Infof("dynamic route service: %s, subset：%s", svc.Name, subset.Name)
		for _, r := range svc.Spec.Route {
			klog.Infof("dynamic route: %s, weight: %d", r.Subset, r.Weight)
		}
		buildRoutes = dynamicRoute(svc.Name, svc.Spec.Route)
	case (reroute && !onlyMissingCanary(actualSubsets, r.MeshConfig.Spec.GlobalSubsets)) || reroute && subset.IsCanary:
		klog.Infof("only missing canary %s: %v", svc.Name, onlyMissingCanary(actualSubsets, r.MeshConfig.Spec.GlobalSubsets))
		klog.Infof("reroute service: %s, subset: %s, len of actualSubsets: %d", svc.Name, subset.Name, len(actualSubsets))
		buildRoutes = rerouteSubset(svc.Name, subset, actualSubsets)
	default:
		klog.Infof("default route service: %s, subset: %s, len of actualSubsets: %d", svc.Name, subset.Name, len(actualSubsets))
		buildRoutes = defaultDestination(svc.Name, subset.Name)
	}

	// set SourceLabels in match
	s := make(map[string]string)
	header := make(map[string]*v1beta1.StringMatch)
	for _, key := range r.MeshConfig.Spec.MatchSourceLabelKeys {
		s[key] = subset.Labels[key]
		header[key] = &v1beta1.StringMatch{MatchType: &v1beta1.StringMatch_Exact{Exact: subset.Labels[key]}}
	}
	match := &v1beta1.HTTPMatchRequest{SourceLabels: s, Headers: header}

	return &v1beta1.HTTPRoute{
		Name:    httpRouteName + "-" + subset.Name,
		Match:   []*v1beta1.HTTPMatchRequest{match},
		Route:   buildRoutes,
		Timeout: r.getTimeout(svc.Spec.Policy.Timeout, int64(svc.Spec.Policy.MaxRetries)),
		Retries: r.getRetries(svc.Spec.Policy.Timeout, svc.Spec.Policy.MaxRetries),
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
func dynamicRoute(host string, dynamicRoutes []*meshv1alpha1.Destination) []*v1beta1.HTTPRouteDestination {
	var buildRoutes []*v1beta1.HTTPRouteDestination
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

func rerouteSubset(host string, subset *meshv1alpha1.Subset, actualSubsets []*meshv1alpha1.Subset) []*v1beta1.HTTPRouteDestination {
	var buildRoutes []*v1beta1.HTTPRouteDestination
	// The consumer of canary subset can only call the provider of the canary subset. When the
	// Provider instance of a canary subset is empty, the traffic of the canary consumer is
	// evenly distributed to other normal subsets.
	switch {
	case subset.IsCanary && !subset.In(actualSubsets):
		klog.V(6).Infof("reroute canary subset: %s which is not in actual subsets", subset.Name)
		for i, actual := range actualSubsets {
			route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: host}}
			route.Destination.Subset = actual.Name
			route.Weight = int32(100 / len(actualSubsets))
			if i == len(actualSubsets)-1 && len(actualSubsets) > 1 {
				route.Weight = 100 - int32(100/len(actualSubsets))*int32((len(actualSubsets)-1))
			}
			buildRoutes = append(buildRoutes, route)
		}
	case !subset.IsCanary && !subset.In(actualSubsets):
		klog.V(6).Infof("reroute subset: %s which is not in actual subsets", subset.Name)
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
	case subset.IsCanary && subset.In(actualSubsets):
		klog.V(6).Infof("reroute canary subset: %s which is in actual subsets, use default route policy", subset.Name)
		buildRoutes = defaultDestination(host, subset.Name)
	case !subset.IsCanary && subset.In(actualSubsets):
		klog.V(6).Infof("reroute subset: %s which is in actual subsets, use default route policy", subset.Name)
		buildRoutes = defaultDestination(host, subset.Name)
	default:
		buildRoutes = defaultDestination(host, subset.Name)
	}

	for _, r := range buildRoutes {
		klog.Infof("host: %s, subset: %s, weight: %d", r.Destination.Host, r.Destination.Subset, r.Weight)
	}
	return buildRoutes
}

func defaultDestination(host, subset string) []*v1beta1.HTTPRouteDestination {
	var buildRoutes []*v1beta1.HTTPRouteDestination
	klog.V(6).Infof("use default destination policy, subset: %s", subset)
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

// func getMatchType(matchType meshv1alpha1.StringMatchType, value string) *v1beta1.StringMatch {
// 	s := &v1beta1.StringMatch{}
// 	switch matchType {
// 	case meshv1alpha1.Prefix:
// 		s.MatchType = &v1beta1.StringMatch_Prefix{Prefix: value}
// 	case meshv1alpha1.Regex:
// 		s.MatchType = &v1beta1.StringMatch_Regex{Regex: value}
// 	default:
// 		s.MatchType = &v1beta1.StringMatch_Exact{Exact: value}
// 	}
// 	return s
// }

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

func (r *Reconciler) getRetries(timeout string, maxRetries int32) *v1beta1.HTTPRetry {
	if timeout == "" {
		timeout = r.MeshConfig.Spec.GlobalPolicy.Timeout
	}
	if maxRetries == 0 {
		maxRetries = r.MeshConfig.Spec.GlobalPolicy.MaxRetries
	}
	return &v1beta1.HTTPRetry{
		Attempts:      maxRetries,
		PerTryTimeout: utils.StringToDuration(timeout, 1),
		RetryOn:       r.Opt.ProxyRetryOn,
	}
}

func (r *Reconciler) getTimeout(timeout string, maxRetries int64) *ptypes.Duration {
	if timeout == "" {
		timeout = r.MeshConfig.Spec.GlobalPolicy.Timeout
	}
	if maxRetries == 0 {
		maxRetries = int64(r.MeshConfig.Spec.GlobalPolicy.MaxRetries)
	}
	return utils.StringToDuration(timeout, maxRetries)
}

func onlyMissingCanary(actualSubsets, globalSubsets []*meshv1alpha1.Subset) bool {
	missingSubset := []*meshv1alpha1.Subset{}
	for _, global := range globalSubsets {
		if !global.In(actualSubsets) {
			missingSubset = append(missingSubset, global)
		}
	}

	for _, missing := range missingSubset {
		if !missing.IsCanary {
			return false
		}
	}
	return true
}
