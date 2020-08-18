package serviceconfig

import (
	"context"

	meshv1alpha1 "github.com/symcn/mesh-operator/api/v1alpha1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *Reconciler) updateStatus(ctx context.Context, req reconcile.Request, sc *meshv1alpha1.ServiceConfig) error {
	status := r.buildStatus(sc)
	if !equality.Semantic.DeepEqual(status, sc.Status) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			status.DeepCopyInto(&sc.Status)
			t := metav1.Now()
			sc.Status.LastUpdateTime = &t

			updateErr := r.Status().Update(ctx, sc)
			if updateErr == nil {
				klog.V(4).Infof("%s/%s update status[%s] successfully",
					req.Namespace, req.Name, sc.Status.Phase)
				return nil
			}

			getErr := r.Get(ctx, req.NamespacedName, sc)
			if getErr != nil {
				return getErr
			}
			return updateErr
		})
		return err
	}
	return nil
}

func (r *Reconciler) buildStatus(sc *meshv1alpha1.ServiceConfig) *meshv1alpha1.ServiceConfigStatus {
	ctx := context.TODO()
	serviceEntry := r.getServiceEntryStatus(ctx, sc)
	workloadEntry := r.getWorkloadEntryStatus(ctx, sc)
	virtualService := r.getVirtualServiceStatus(ctx, sc)
	destinationRule := r.getDestinationRuleStatus(ctx, sc)

	status := &meshv1alpha1.ServiceConfigStatus{
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

func (r *Reconciler) getServiceEntryStatus(ctx context.Context, sc *meshv1alpha1.ServiceConfig) *meshv1alpha1.SubStatus {
	svcCount := 1
	list := &networkingv1beta1.ServiceEntryList{}
	count := r.count(ctx, sc, list)
	status := &meshv1alpha1.SubStatus{Desired: svcCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = svcCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *Reconciler) getWorkloadEntryStatus(ctx context.Context, sc *meshv1alpha1.ServiceConfig) *meshv1alpha1.SubStatus {
	insCount := 0
	zero := &meshv1alpha1.SubStatus{
		Desired:       insCount,
		Distributed:   &insCount,
		Undistributed: &insCount,
	}

	cs := &meshv1alpha1.ConfiguredService{}
	err := r.Get(ctx, types.NamespacedName{Namespace: sc.Namespace, Name: sc.Name}, cs)
	if err != nil {
		klog.Infof("The configuredservice[%s/%s] in update status get error: %+v", sc.Namespace, sc.Name, err)
		return zero
	}

	insCount = len(cs.Spec.Instances)
	if insCount == 0 {
		return zero
	}

	list := &networkingv1beta1.WorkloadEntryList{}
	count := r.count(ctx, sc, list)
	status := &meshv1alpha1.SubStatus{Desired: insCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = insCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *Reconciler) getVirtualServiceStatus(ctx context.Context, sc *meshv1alpha1.ServiceConfig) *meshv1alpha1.SubStatus {
	// Skip if the service's subset is none
	svcCount := 1
	subsets, err := r.getSubset(ctx, sc)
	if err != nil {
		klog.Errorf("Get subset error: %+v", err)
	}

	klog.Infof("the length of subsets is %d", len(subsets))
	if len(subsets) == 0 {
		svcCount = 0
		return &meshv1alpha1.SubStatus{
			Desired:       svcCount,
			Distributed:   &svcCount,
			Undistributed: &svcCount,
		}
	}

	list := &networkingv1beta1.VirtualServiceList{}
	count := r.count(ctx, sc, list)
	status := &meshv1alpha1.SubStatus{Desired: svcCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = svcCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *Reconciler) getDestinationRuleStatus(ctx context.Context, sc *meshv1alpha1.ServiceConfig) *meshv1alpha1.SubStatus {
	svcCount := 1
	subsets, err := r.getSubset(ctx, sc)
	if err != nil {
		klog.Errorf("Get subset error: %+v", err)
	}

	// Skip if the service's subset is none
	if len(subsets) == 0 {
		svcCount = 0
		return &meshv1alpha1.SubStatus{
			Desired:       svcCount,
			Distributed:   &svcCount,
			Undistributed: &svcCount,
		}
	}

	list := &networkingv1beta1.DestinationRuleList{}
	count := r.count(ctx, sc, list)
	status := &meshv1alpha1.SubStatus{Desired: svcCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = svcCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *Reconciler) count(ctx context.Context, sc *meshv1alpha1.ServiceConfig, list runtime.Object) *int {
	var c int
	labels := &client.MatchingLabels{r.Opt.SelectLabel: truncated(sc.Spec.OriginalName)}
	opts := &client.ListOptions{Namespace: sc.Namespace}
	labels.ApplyToList(opts)

	err := r.List(ctx, list, opts)
	if err != nil {
		klog.Errorf("%s/%s/%s collecting the substatus error: %v", sc.Namespace, sc.Name, list, err)
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
