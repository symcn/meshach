package configuredservice

import (
	"context"

	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	v1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) reconcileSubset(ctx context.Context, cr *meshv1alpha1.ConfiguredService) error {
	for _, subset := range cr.Spec.Subsets {
		labels := client.MatchingLabels(subset.Labels)
		opts := &client.ListOptions{Namespace: cr.Namespace}
		labels.ApplyToList(opts)
		l := &networkingv1beta1.WorkloadEntryList{}
		err := r.List(ctx, l, opts)
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

func (r *Reconciler) rerouteSubset(ctx context.Context, subset *meshv1alpha1.Subset, cr *meshv1alpha1.ConfiguredService) error {
	if cr.Spec.CanaryRerouteOption == nil ||
		cr.Spec.RerouteOption == nil {
		klog.Warningf("%s/%s not set reroute option.", cr.Namespace, cr.Name)
		return nil
	}

	var err error
	if subset.IsCanary {
		klog.Infof("%s/%s start to rerouting canary subset[%s], policy: %s",
			cr.Namespace, cr.Name, subset.Name, cr.Spec.CanaryRerouteOption.ReroutePolicy)
		err = r.reroute(ctx, subset, cr, cr.Spec.CanaryRerouteOption)
	} else {
		klog.Infof("%s/%s start to rerouting subset[%s], policy: %s",
			cr.Namespace, cr.Name, subset.Name, cr.Spec.RerouteOption.ReroutePolicy)
		err = r.reroute(ctx, subset, cr, cr.Spec.RerouteOption)
	}

	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) reroute(ctx context.Context, subset *meshv1alpha1.Subset, cr *meshv1alpha1.ConfiguredService, rerouteOption *meshv1alpha1.RerouteOption) error {
	switch rerouteOption.ReroutePolicy {
	case meshv1alpha1.Specific:
		cr = rerouteWithSpecificMap(cr, rerouteOption)
	case meshv1alpha1.RoundRobin:
		cr = rerouteWithRoundRobin(cr, subset)
	case meshv1alpha1.Random:
		cr = rerouteWithRandom(cr, subset)
	case meshv1alpha1.LeastConn:
		cr = rerouteWithLeastConn(cr, subset)
	case meshv1alpha1.Passthrough:
		cr = rerouteWithPassthrough(cr, subset)
	case meshv1alpha1.Unchangeable:
		return nil
	default:
		klog.Warningf("Unsupported ReroutePolicy: %s", rerouteOption.ReroutePolicy)
		return nil
	}

	klog.Infof("%s/%s start to update...", cr.Namespace, cr.Name)
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updateErr := r.Update(ctx, cr)
		if updateErr == nil {
			klog.Infof("%s/%s reroute successfully", cr.Namespace, cr.Name)
		}

		getErr := r.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: cr.Name}, cr)
		if getErr != nil {
			klog.Errorf("Get ConfiguredService error: %+v", getErr)
		}
		return updateErr
	})

	if err != nil {
		klog.Errorf("%s/%s reroute error: %+v", cr.Namespace, cr.Name, err)
		return err
	}

	return nil
}

func rerouteWithSpecificMap(cr *meshv1alpha1.ConfiguredService, option *meshv1alpha1.RerouteOption) *meshv1alpha1.ConfiguredService {
	for sourceLabel, route := range option.SpecificRoute {
		for _, s := range cr.Spec.Policy.SourceLabels {
			if s.Name == sourceLabel {
				s.Route = route
			}
		}
	}
	return cr
}

func rerouteWithRoundRobin(cr *meshv1alpha1.ConfiguredService, subset *meshv1alpha1.Subset) *meshv1alpha1.ConfiguredService {
	cr = setDefaultRoute(cr, subset)
	cr.Spec.Policy.LoadBalancer = map[string]string{
		loadBalanceSimple: v1beta1.LoadBalancerSettings_ROUND_ROBIN.String(),
	}
	return cr
}

func rerouteWithRandom(cr *meshv1alpha1.ConfiguredService, subset *meshv1alpha1.Subset) *meshv1alpha1.ConfiguredService {
	cr = setDefaultRoute(cr, subset)
	cr.Spec.Policy.LoadBalancer = map[string]string{
		loadBalanceSimple: v1beta1.LoadBalancerSettings_RANDOM.String(),
	}
	return cr
}

func rerouteWithLeastConn(cr *meshv1alpha1.ConfiguredService, subset *meshv1alpha1.Subset) *meshv1alpha1.ConfiguredService {
	cr = setDefaultRoute(cr, subset)
	cr.Spec.Policy.LoadBalancer = map[string]string{
		loadBalanceSimple: v1beta1.LoadBalancerSettings_LEAST_CONN.String(),
	}
	return cr
}

func rerouteWithPassthrough(cr *meshv1alpha1.ConfiguredService, subset *meshv1alpha1.Subset) *meshv1alpha1.ConfiguredService {
	cr = setDefaultRoute(cr, subset)
	cr.Spec.Policy.LoadBalancer = map[string]string{
		loadBalanceSimple: v1beta1.LoadBalancerSettings_PASSTHROUGH.String(),
	}
	return cr
}

func setDefaultRoute(cr *meshv1alpha1.ConfiguredService, subset *meshv1alpha1.Subset) *meshv1alpha1.ConfiguredService {
	for _, s := range cr.Spec.Policy.SourceLabels {
		if s.Name == subset.Name {
			s.Route = []*meshv1alpha1.Destination{{Subset: "", Weight: 0}}
		}
	}
	return cr
}
