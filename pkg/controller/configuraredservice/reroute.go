package configuraredservice

import (
	"context"

	meshv1 "github.com/symcn/mesh-operator/pkg/apis/mesh/v1"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileConfiguraredService) reconcileSubset(ctx context.Context, cr *meshv1.ConfiguraredService) error {
	for _, subset := range cr.Spec.Subsets {
		labels := client.MatchingLabels(subset.Labels)
		opts := &client.ListOptions{Namespace: cr.Namespace}
		labels.ApplyToList(opts)
		l := &networkingv1beta1.WorkloadEntryList{}
		err := r.client.List(ctx, l, opts)
		if err != nil {
			klog.Errorf("Get WorkloadEntryList error: %s", err)
			return err
		}

		if len(l.Items) == 0 {
			err := r.rerouteSubset(ctx, subset, cr)
			if err != nil {
				klog.Errorf("%s/%s rerouteSubset error: %+v", cr.Namespace, cr.Name, err)
				return err
			}
		}
	}
	return nil
}

func (r *ReconcileConfiguraredService) rerouteSubset(ctx context.Context, subset *meshv1.Subset, cr *meshv1.ConfiguraredService) error {
	var err error
	if subset.IsCanary {
		err = r.reroute(ctx, subset, cr, cr.Spec.CanaryRerouteOption.ReroutePolicy)
	}

	err = r.reroute(ctx, subset, cr, cr.Spec.RerouteOption.ReroutePolicy)
	if err != nil {
		return err
	}
	return nil
}

func (r *ReconcileConfiguraredService) reroute(ctx context.Context, subset *meshv1.Subset, cr *meshv1.ConfiguraredService, reroutePolicy meshv1.ReroutePolicy) error {
	switch reroutePolicy {
	case meshv1.Specific:
		cr = rerouteWithSpecificMap(cr)
	case meshv1.RoundRobin:
		cr = rerouteWithRoundRobin(cr, subset)
	case meshv1.Random:
		cr = rerouteWithRandom(cr, subset)
	case meshv1.LeastConn:
		cr = rerouteWithLeastConn(cr, subset)
	case meshv1.Passthrough:
		cr = rerouteWithPassthrough(cr, subset)
	case meshv1.Unchangeable:
		return nil
	default:
		klog.Warningf("Unsupported ReroutePolicy: %s", reroutePolicy)
		return nil
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updateErr := r.client.Update(ctx, cr)
		if updateErr == nil {
			klog.V(4).Infof("%s/%s reroute successfully", cr.Namespace, cr.Name)
			return nil
		}

		getErr := r.client.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: cr.Name}, cr)
		if getErr != nil {
			klog.Errorf("Get ConfiguraredService error: %+v", getErr)
		}
		return updateErr
	})

	if err != nil {
		klog.Errorf("%s/%s reroute error: %+v", cr.Namespace, cr.Name, err)
		return err
	}

	return nil
}

func rerouteWithSpecificMap(cr *meshv1.ConfiguraredService) *meshv1.ConfiguraredService {
	for sourceLabel, route := range cr.Spec.CanaryRerouteOption.SpecificRoute {
		for _, s := range cr.Spec.Policy.SourceLabels {
			if s.Name == sourceLabel {
				s.Route = route
			}
		}
	}
	return cr
}

func rerouteWithRoundRobin(cr *meshv1.ConfiguraredService, subset *meshv1.Subset) *meshv1.ConfiguraredService {
	cr = setDefaultRoute(cr, subset)
	cr.Spec.Policy.LoadBalancer = map[string]string{
		loadBalanceSimple: v1beta1.LoadBalancerSettings_ROUND_ROBIN.String(),
	}
	return cr
}

func rerouteWithRandom(cr *meshv1.ConfiguraredService, subset *meshv1.Subset) *meshv1.ConfiguraredService {
	cr = setDefaultRoute(cr, subset)
	cr.Spec.Policy.LoadBalancer = map[string]string{
		loadBalanceSimple: v1beta1.LoadBalancerSettings_RANDOM.String(),
	}
	return cr
}

func rerouteWithLeastConn(cr *meshv1.ConfiguraredService, subset *meshv1.Subset) *meshv1.ConfiguraredService {
	cr = setDefaultRoute(cr, subset)
	cr.Spec.Policy.LoadBalancer = map[string]string{
		loadBalanceSimple: v1beta1.LoadBalancerSettings_LEAST_CONN.String(),
	}
	return cr
}

func rerouteWithPassthrough(cr *meshv1.ConfiguraredService, subset *meshv1.Subset) *meshv1.ConfiguraredService {
	cr = setDefaultRoute(cr, subset)
	cr.Spec.Policy.LoadBalancer = map[string]string{
		loadBalanceSimple: v1beta1.LoadBalancerSettings_PASSTHROUGH.String(),
	}
	return cr
}

func setDefaultRoute(cr *meshv1.ConfiguraredService, subset *meshv1.Subset) *meshv1.ConfiguraredService {
	for _, s := range cr.Spec.Policy.SourceLabels {
		if s.Name == subset.Name {
			s.Route = []*meshv1.Destination{{Subset: "", Weight: 0}}
		}
	}
	return cr
}
