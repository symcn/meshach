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
		}
		klog.Errorf("[configuredservice] creating  [%s/%s] error: %+v", vs.Namespace, vs.Name, err)
	}

	return nil
}

func (r *Reconciler) buildVirtualService(cs *meshv1alpha1.ConfiguredService) *networkingv1beta1.VirtualService {
	httpRoute := []*v1beta1.HTTPRoute{}
	for _, subset := range r.MeshConfig.Spec.GlobalSubsets {
		http := r.buildHTTPRoute(cs, subset)
		httpRoute = append(httpRoute, http)
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

func (r *Reconciler) buildHTTPRoute(cs *meshv1alpha1.ConfiguredService, subset *meshv1alpha1.Subset) *v1beta1.HTTPRoute {
	var buildRoutes []*v1beta1.HTTPRouteDestination
	buildRoutes = defaultDestination(cs.Name, subset.Name)

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
		Timeout: r.getDefaultTimeout(),
		Retries: r.getDefaultRetries(),
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

func (r *Reconciler) getDefaultRetries() *v1beta1.HTTPRetry {
	timeout := r.MeshConfig.Spec.GlobalPolicy.Timeout
	maxRetries := r.MeshConfig.Spec.GlobalPolicy.MaxRetries
	return &v1beta1.HTTPRetry{
		Attempts:      maxRetries,
		PerTryTimeout: utils.StringToDuration(timeout, 1),
		RetryOn:       r.Opt.ProxyRetryOn,
	}
}

func (r *Reconciler) getDefaultTimeout() *ptypes.Duration {
	timeout := r.MeshConfig.Spec.GlobalPolicy.Timeout
	maxRetries := int64(r.MeshConfig.Spec.GlobalPolicy.MaxRetries)
	return utils.StringToDuration(timeout, maxRetries)
}
