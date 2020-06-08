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

package appmeshconfig

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
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

func (r *ReconcileAppMeshConfig) reconcileVirtualService(ctx context.Context, cr *meshv1.AppMeshConfig, svc *meshv1.Service) error {
	// Skip if the service's subset is none
	if len(svc.Subsets) == 0 {
		return nil
	}

	vs := r.buildVirtualService(cr, svc)
	// Set AppMeshConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, vs, r.scheme); err != nil {
		klog.Errorf("SetControllerReference error: %v", err)
		return err
	}

	// Check if this VirtualService already exists
	found := &networkingv1beta1.VirtualService{}
	err := r.client.Get(ctx, types.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		klog.Infof("Creating a new VirtualService, Namespace: %s, Name: %s", vs.Namespace, vs.Name)
		err = r.client.Create(ctx, vs)
		if err != nil {
			klog.Errorf("Create VirtualService error: %+v", err)
			return err
		}

		// VirtualService created successfully - don't requeue
		return nil
	} else if err != nil {
		klog.Errorf("Get VirtualService error: %+v", err)
		return err
	}

	// Update VirtualService
	if compareVirtualService(vs, found) {
		klog.Infof("Update VirtualService, Namespace: %s, Name: %s", found.Namespace, found.Name)
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			vs.Spec.DeepCopyInto(&found.Spec)
			found.Finalizers = vs.Finalizers
			found.Labels = vs.ObjectMeta.Labels

			updateErr := r.client.Update(ctx, found)
			if updateErr == nil {
				klog.V(4).Infof("%s/%s update VirtualService successfully", vs.Namespace, vs.Name)
				return nil
			}
			return updateErr
		})

		if err != nil {
			klog.Warningf("update VirtualService [%s] spec failed, err: %+v", vs.Name, err)
			return err
		}
	}

	return nil
}

func (r *ReconcileAppMeshConfig) buildVirtualService(cr *meshv1.AppMeshConfig, svc *meshv1.Service) *networkingv1beta1.VirtualService {
	http := r.buildHTTPRoute(svc)
	proxy := r.buildProxyRoute()
	return &networkingv1beta1.VirtualService{
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.FormatToDNS1123(svc.Name),
			Namespace: cr.Namespace,
			Labels:    map[string]string{"app": cr.Spec.AppName},
		},
		Spec: v1beta1.VirtualService{
			Hosts: []string{svc.Name},
			Http:  []*v1beta1.HTTPRoute{http, proxy},
		},
	}
}

func (r *ReconcileAppMeshConfig) buildHTTPRoute(svc *meshv1.Service) *v1beta1.HTTPRoute {
	m := make(map[string]*v1beta1.StringMatch)
	m["zone"] = &v1beta1.StringMatch{
		MatchType: &v1beta1.StringMatch_Exact{
			Exact: r.opt.Zone,
		},
	}
	match := &v1beta1.HTTPMatchRequest{Headers: m}

	var routes []*v1beta1.HTTPRouteDestination
	for _, destination := range svc.Policy.Route {
		route := &v1beta1.HTTPRouteDestination{
			Destination: &v1beta1.Destination{
				Host:   svc.Name,
				Subset: destination.Subset,
			},
			Weight: destination.Weight,
		}
		routes = append(routes, route)
	}

	return &v1beta1.HTTPRoute{
		Name:    "dubbo-http-route",
		Match:   []*v1beta1.HTTPMatchRequest{match},
		Route:   routes,
		Timeout: utils.StringToDuration(svc.Policy.Timeout),
		Retries: &v1beta1.HTTPRetry{
			Attempts: svc.Policy.MaxRetries,
			RetryOn:  r.opt.ProxyRetryOn,
		},
	}
}

func (r *ReconcileAppMeshConfig) buildProxyRoute() *v1beta1.HTTPRoute {
	route := &v1beta1.HTTPRouteDestination{
		Destination: &v1beta1.Destination{
			Host: r.opt.ProxyHost,
		}}
	return &v1beta1.HTTPRoute{
		Name:  "dubbo-proxy-route",
		Route: []*v1beta1.HTTPRouteDestination{route},
		Retries: &v1beta1.HTTPRetry{
			Attempts:      r.opt.ProxyAttempts,
			PerTryTimeout: &ptypes.Duration{Seconds: r.opt.ProxyPerTryTimeout},
			RetryOn:       r.opt.ProxyRetryOn,
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
