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
	httpRouteName    = "dubbo-http-route"
	defaultRouteName = "dubbo-default-route"
)

func (r *Reconciler) reconcileVirtualService(ctx context.Context, cs *meshv1alpha1.ConfiguredService) error {
	vs := r.buildVirtualService(cs)
	// Set ConfiguredService instance as the owner and controller
	if err := controllerutil.SetControllerReference(cs, vs, r.Scheme); err != nil {
		klog.Errorf("SetControllerReference error: %v", err)
		return err
	}

	// Check if this VirtualService already exists
	found := &networkingv1beta1.VirtualService{}
	err := r.Get(ctx, types.NamespacedName{Namespace: cs.Namespace, Name: cs.Name}, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("Creating a new VirtualService, Namespace: %s, Name: %s", vs.Namespace, vs.Name)
			err := r.Create(ctx, vs)
			if err != nil {
				klog.Errorf("Create VirtualService error: %+v", err)
				return err
			}
			return nil
		}
		klog.Errorf("[configuredservice] creating  [%s/%s] error: %+v", vs.Namespace, vs.Name, err)
	}

	return nil
}

func (r *Reconciler) buildVirtualService(cs *meshv1alpha1.ConfiguredService) *networkingv1beta1.VirtualService {
	httpRoute := []*v1beta1.HTTPRoute{}
	actualSubsets := r.getSubset(context.Background(), cs)
	if len(actualSubsets) > 0 {
		for _, subset := range r.MeshConfig.Spec.GlobalSubsets {
			http := r.buildHTTPRoute(cs, subset, actualSubsets)
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
	if subset.IsCanary && !subset.In(actualSubsets) {
		for i, actual := range actualSubsets {
			route := &v1beta1.HTTPRouteDestination{Destination: &v1beta1.Destination{Host: cs.Name}}
			route.Destination.Subset = actual.Name
			route.Weight = int32(100 / len(actualSubsets))
			if i == len(actualSubsets)-1 && len(actualSubsets) > 1 {
				route.Weight = 100 - int32(100/len(actualSubsets))*int32((len(actualSubsets)-1))
			}
			buildRoutes = append(buildRoutes, route)
		}
	} else {
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
			if mapContains(ins.Labels, global.Labels, r.MeshConfig.Spec.MeshLabelsRemap) {
				subsets = append(subsets, global)
			}
		}
	}
	return subsets
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
