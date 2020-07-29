package configuredservice

import (
	"context"

	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *Reconciler) updateStatus(ctx context.Context, req reconcile.Request, cr *meshv1alpha1.ConfiguredService) error {
	status := r.buildStatus(cr)
	if !equality.Semantic.DeepEqual(status, cr.Status) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			status.DeepCopyInto(&cr.Status)
			t := metav1.Now()
			cr.Status.LastUpdateTime = &t

			updateErr := r.Status().Update(ctx, cr)
			if updateErr == nil {
				klog.V(4).Infof("%s/%s update status[%s] successfully",
					req.Namespace, req.Name, cr.Status.Phase)
				return nil
			}

			getErr := r.Get(ctx, req.NamespacedName, cr)
			if getErr != nil {
				return getErr
			}
			return updateErr
		})
		return err
	}
	return nil
}

func (r *Reconciler) buildStatus(cr *meshv1alpha1.ConfiguredService) *meshv1alpha1.ConfiguredServiceStatus {
	ctx := context.TODO()
	serviceEntry := r.getServiceEntryStatus(ctx, cr)
	workloadEntry := r.getWorkloadEntryStatus(ctx, cr)
	virtualService := r.getVirtualServiceStatus(ctx, cr)
	destinationRule := r.getDestinationRuleStatus(ctx, cr)

	status := &meshv1alpha1.ConfiguredServiceStatus{
		Status: &meshv1alpha1.Status{
			ServiceEntry:    serviceEntry,
			WorkloadEntry:   workloadEntry,
			VirtualService:  virtualService,
			DestinationRule: destinationRule,
		},
	}
	status.Phase = calcPhase(status.Status)
	return status
}

func (r *Reconciler) getServiceEntryStatus(ctx context.Context, cr *meshv1alpha1.ConfiguredService) *meshv1alpha1.SubStatus {
	svcCount := 1
	list := &networkingv1beta1.ServiceEntryList{}
	count := r.count(ctx, cr, list)
	status := &meshv1alpha1.SubStatus{Desired: svcCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = svcCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *Reconciler) getWorkloadEntryStatus(ctx context.Context, cr *meshv1alpha1.ConfiguredService) *meshv1alpha1.SubStatus {
	insCount := len(cr.Spec.Instances)
	list := &networkingv1beta1.WorkloadEntryList{}
	count := r.count(ctx, cr, list)
	status := &meshv1alpha1.SubStatus{Desired: insCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = insCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *Reconciler) getVirtualServiceStatus(ctx context.Context, cr *meshv1alpha1.ConfiguredService) *meshv1alpha1.SubStatus {
	// Skip if the service's subset is none
	svcCount := 1
	if len(cr.Spec.Subsets) == 0 {
		svcCount = 0
		return &meshv1alpha1.SubStatus{
			Desired:       svcCount,
			Distributed:   &svcCount,
			Undistributed: &svcCount,
		}
	}

	list := &networkingv1beta1.VirtualServiceList{}
	count := r.count(ctx, cr, list)
	status := &meshv1alpha1.SubStatus{Desired: svcCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = svcCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *Reconciler) getDestinationRuleStatus(ctx context.Context, cr *meshv1alpha1.ConfiguredService) *meshv1alpha1.SubStatus {
	svcCount := 1

	// Skip if the service's subset is none
	if len(cr.Spec.Subsets) == 0 {
		svcCount = 0
		return &meshv1alpha1.SubStatus{
			Desired:       svcCount,
			Distributed:   &svcCount,
			Undistributed: &svcCount,
		}
	}

	list := &networkingv1beta1.DestinationRuleList{}
	count := r.count(ctx, cr, list)
	status := &meshv1alpha1.SubStatus{Desired: svcCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = svcCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *Reconciler) count(ctx context.Context, cr *meshv1alpha1.ConfiguredService, list runtime.Object) *int {
	var c int
	labels := &client.MatchingLabels{"service": cr.Spec.OriginalName}
	opts := &client.ListOptions{Namespace: cr.Namespace}
	labels.ApplyToList(opts)

	err := r.List(ctx, list, opts)
	if err != nil {
		klog.Errorf("%s/%s/%s collecting the substatus error: %v", cr.Namespace, cr.Name, list, err)
		return nil
	}

	switch v := list.(type) {
	case *networkingv1beta1.VirtualServiceList:
		c = len(v.Items)
	case *networkingv1beta1.ServiceEntryList:
		c = len(v.Items)
	case *networkingv1beta1.WorkloadEntryList:
		c = len(v.Items)
	case *networkingv1beta1.DestinationRuleList:
		c = len(v.Items)
	default:
		klog.Errorf("invalid list type: %v", list)
	}
	return &c
}

func calcPhase(status *meshv1alpha1.Status) meshv1alpha1.ConfigPhase {
	// return Unknown if any Distributed is nil
	if status.ServiceEntry.Distributed == nil ||
		status.WorkloadEntry.Distributed == nil ||
		status.VirtualService.Distributed == nil ||
		status.DestinationRule.Distributed == nil {
		return meshv1alpha1.ConfigStatusUnknown
	}

	// return Undistributed if the sum of all Distributed is zero
	if *status.ServiceEntry.Distributed+
		*status.WorkloadEntry.Distributed+
		*status.VirtualService.Distributed+
		*status.DestinationRule.Distributed == 0 {
		return meshv1alpha1.ConfigStatusUndistributed
	}

	// return Distributed if the sum of all Undistributed is zero
	if *status.ServiceEntry.Undistributed+
		*status.WorkloadEntry.Undistributed+
		*status.VirtualService.Undistributed+
		*status.DestinationRule.Undistributed == 0 {
		return meshv1alpha1.ConfigStatusDistributed
	}

	// otherwize return Distributing
	return meshv1alpha1.ConfigStatusDistributing
}
