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
	"strconv"

	ptypes "github.com/gogo/protobuf/types"
	meshv1alpha1 "github.com/symcn/meshach/api/v1alpha1"
	"github.com/symcn/meshach/pkg/utils"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	httpRouteName    = "dubbo-http-route"
	defaultRouteName = "dubbo-default-route"
)

func (r *Reconciler) reconcileVirtualService(ctx context.Context, cs *meshv1alpha1.ConfiguredService) error {
	foundMap, err := r.getVirtualServicesMap(ctx, cs)
	if err != nil {
		klog.Errorf("[cs] %s/%s get VirtualService error: %+v", cs.Namespace, cs.Name, err)
		return err
	}

	subsets := r.getSubset(context.Background(), cs)
	// Skip if the service's subset is none
	klog.V(6).Infof("[cs] virtualservice subsets length: %d", len(subsets))
	if len(subsets) != 0 {
		vs := r.buildVirtualService(cs, subsets)
		// Set ConfiguredService instance as the owner and controller
		if err := controllerutil.SetControllerReference(cs, vs, r.Scheme); err != nil {
			klog.Errorf("[cs] SetControllerReference error: %v", err)
			return err
		}

		// Check if this VirtualService already exists
		found, ok := foundMap[vs.Name]
		if !ok {
			klog.Infof("[cs] creating a new VirtualService, Namespace: %s, Name: %s", vs.Namespace, vs.Name)
			err = r.Create(ctx, vs)
			if err != nil {
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				klog.Errorf("[cs] create VirtualService [%s/%s], error: %+v", vs.Namespace, vs.Name, err)
				return err
			}
		} else {
			// Update VirtualService
			if compareVirtualService(vs, found) {
				klog.Infof("[cs] Update VirtualService, Namespace: %s, Name: %s",
					found.Namespace, found.Name)
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					vs.Spec.DeepCopyInto(&found.Spec)
					found.Finalizers = vs.Finalizers
					found.Labels = vs.ObjectMeta.Labels

					updateErr := r.Update(ctx, found)
					if updateErr == nil {
						klog.Infof("[cs] %s/%s update VirtualService successfully",
							vs.Namespace, vs.Name)
						return nil
					}
					return updateErr
				})

				if err != nil {
					klog.Warningf("[cs] Update VirtualService [%s] spec failed, err: %+v", vs.Name, err)
					return err
				}
			}
			delete(foundMap, vs.Name)
		}
		// Delete old VirtualServices
		for name, vs := range foundMap {
			klog.Infof("[cs] Delete unused VirtualService: %s", name)
			err := r.Delete(ctx, vs)
			if err != nil {
				klog.Errorf("[cs] Delete unused VirtualService error: %+v", err)
				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) buildVirtualService(cs *meshv1alpha1.ConfiguredService, subsets []*meshv1alpha1.Subset) *networkingv1beta1.VirtualService {
	httpRoute := []*v1beta1.HTTPRoute{}
	if len(subsets) > 0 {
		for _, subset := range r.MeshConfig.Spec.GlobalSubsets {
			http := r.buildHTTPRoute(cs, subset, subsets)
			httpRoute = append(httpRoute, http)
		}
	}

	defaultRoute := r.buildDefaultRoute(cs)
	httpRoute = append(httpRoute, defaultRoute)

	return &networkingv1beta1.VirtualService{
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.FormatToDNS1123(cs.Name),
			Namespace: cs.Namespace,
			Labels:    map[string]string{r.Opt.SelectLabel: truncated(cs.Spec.OriginalName)},
		},
		Spec: v1beta1.VirtualService{
			Hosts: []string{cs.Name},
			Http:  httpRoute,
		},
	}
}

func (r *Reconciler) buildHTTPRoute(cs *meshv1alpha1.ConfiguredService, subset *meshv1alpha1.Subset, actualSubsets []*meshv1alpha1.Subset) *v1beta1.HTTPRoute {
	var buildRoutes []*v1beta1.HTTPRouteDestination
	switch {
	case subset.IsCanary && !subset.In(actualSubsets):
		for i, actual := range actualSubsets {
			route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: cs.Name}}
			route.Destination.Subset = actual.Name
			route.Weight = int32(100 / len(actualSubsets))
			if i == len(actualSubsets)-1 && len(actualSubsets) > 1 {
				route.Weight = 100 - int32(100/len(actualSubsets))*int32((len(actualSubsets)-1))
			}
			buildRoutes = append(buildRoutes, route)
		}
	case !subset.IsCanary && !subset.In(actualSubsets):
		route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: cs.Name}}
		route.Destination.Subset = subset.Name
		route.Weight = 0
		buildRoutes = append(buildRoutes, route)

		l := len(actualSubsets)
		for _, actual := range actualSubsets {
			if actual.IsCanary {
				l--
				continue
			}
		}

		for i, actual := range actualSubsets {
			if actual.IsCanary {
				continue
			}
			route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: cs.Name}}
			route.Destination.Subset = actual.Name
			route.Weight = int32(100 / l)
			if i == len(actualSubsets)-1 && len(actualSubsets) > 1 {
				route.Weight = 100 - int32(100/l)
			}
			buildRoutes = append(buildRoutes, route)
		}
	default:
		buildRoutes = defaultDestination(cs.Name, subset.Name)
	}

	// set SourceLabels in match
	s := make(map[string]string)
	header := make(map[string]*v1beta1.StringMatch)
	for _, key := range r.MeshConfig.Spec.MatchSourceLabelKeys {
		s[key] = subset.Labels[key]
		header[key] = &v1beta1.StringMatch{MatchType: &v1beta1.StringMatch_Exact{Exact: subset.Labels[key]}}
	}
	match := &v1beta1.HTTPMatchRequest{SourceLabels: s, Headers: header}

	var timeout string
	var retries int64
	var err error

	for _, ins := range cs.Spec.Instances {
		timeout = ins.Labels["timeout"]
		retriesStr, ok := ins.Labels["retries"]
		if ok {
			retries, err = strconv.ParseInt(retriesStr, 10, 64)
			if err != nil {
				klog.Errorf("[configuredservice] ParseInt %s error: %+v", retriesStr, err)
			}
		}
		break
	}

	return &v1beta1.HTTPRoute{
		Name:    httpRouteName + "-" + subset.Name,
		Match:   []*v1beta1.HTTPMatchRequest{match},
		Route:   buildRoutes,
		Timeout: r.getTimeout(timeout, retries),
		Retries: r.getRetries(timeout, int32(retries)),
	}
}

func (r *Reconciler) buildDefaultRoute(cs *meshv1alpha1.ConfiguredService) *v1beta1.HTTPRoute {
	route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: cs.Name}}
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

func defaultDestination(host, subset string) []*v1beta1.HTTPRouteDestination {
	var buildRoutes []*v1beta1.HTTPRouteDestination
	route := &v1beta1.HTTPRouteDestination{
		Destination: &v1beta1.Destination{Host: host, Subset: subset},
		Weight:      100,
	}
	buildRoutes = append(buildRoutes, route)
	return buildRoutes
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

func (r *Reconciler) getSubset(ctx context.Context, cs *meshv1alpha1.ConfiguredService) []*meshv1alpha1.Subset {
	var subsets []*meshv1alpha1.Subset
	for _, global := range r.MeshConfig.Spec.GlobalSubsets {
		for _, ins := range cs.Spec.Instances {
			if mapContains(ins.Labels, global.Labels, r.MeshConfig.Spec.MeshLabelsRemap) && !global.In(subsets) {
				subsets = append(subsets, global)
			}
		}
	}
	return subsets
}

func (r *Reconciler) getVirtualServicesMap(ctx context.Context, cs *meshv1alpha1.ConfiguredService) (map[string]*networkingv1beta1.VirtualService, error) {
	list := &networkingv1beta1.VirtualServiceList{}
	labels := &client.MatchingLabels{r.Opt.SelectLabel: truncated(cs.Spec.OriginalName)}
	opts := &client.ListOptions{Namespace: cs.Namespace}
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

func mapContains(std, obj, renameMap map[string]string) bool {
	for sk, sv := range std {
		rk, ok := renameMap[sk]
		if !ok {
			rk = sk
		}
		if ov, ok := obj[rk]; ok && ov == sv {
			return true
		}
	}
	return false
}
