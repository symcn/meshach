package appmeshconfig

import (
	"context"

	meshv1 "github.com/mesh-operator/pkg/apis/mesh/v1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileAppMeshConfig) buildStatus(cr *meshv1.AppMeshConfig) *meshv1.AppMeshConfigStatus {
	ctx := context.TODO()
	serviceEntry := r.getServiceEntryStatus(ctx, cr)
	workloadEntry := r.getWorkloadEntryStatus(ctx, cr)
	virtualService := r.getVirtualServiceStatus(ctx, cr)
	destinationRule := r.getDestinationRuleStatus(ctx, cr)

	status := &meshv1.AppMeshConfigStatus{
		Status: &meshv1.Status{
			ServiceEntry:    serviceEntry,
			WorkloadEntry:   workloadEntry,
			VirtualService:  virtualService,
			DestinationRule: destinationRule,
		},
	}
	status.Phase = calcPhase(status.Status)
	return status
}

func (r *ReconcileAppMeshConfig) getServiceEntryStatus(ctx context.Context, cr *meshv1.AppMeshConfig) *meshv1.SubStatus {
	svcCount := len(cr.Spec.Services)
	list := &networkingv1beta1.ServiceEntryList{}
	count := r.count(ctx, cr, list)
	status := &meshv1.SubStatus{Desired: svcCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = svcCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *ReconcileAppMeshConfig) getWorkloadEntryStatus(ctx context.Context, cr *meshv1.AppMeshConfig) *meshv1.SubStatus {
	var insCount int
	for _, svc := range cr.Spec.Services {
		insCount += len(svc.Instances)
	}
	list := &networkingv1beta1.WorkloadEntryList{}
	count := r.count(ctx, cr, list)
	status := &meshv1.SubStatus{Desired: insCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = insCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *ReconcileAppMeshConfig) getVirtualServiceStatus(ctx context.Context, cr *meshv1.AppMeshConfig) *meshv1.SubStatus {
	svcCount := len(cr.Spec.Services)
	list := &networkingv1beta1.VirtualServiceList{}
	count := r.count(ctx, cr, list)
	status := &meshv1.SubStatus{Desired: svcCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = svcCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *ReconcileAppMeshConfig) getDestinationRuleStatus(ctx context.Context, cr *meshv1.AppMeshConfig) *meshv1.SubStatus {
	svcCount := len(cr.Spec.Services)
	list := &networkingv1beta1.DestinationRuleList{}
	count := r.count(ctx, cr, list)
	status := &meshv1.SubStatus{Desired: svcCount, Distributed: count}

	var undistributed int
	if count != nil {
		undistributed = svcCount - *count
		status.Undistributed = &undistributed
	}
	return status
}

func (r *ReconcileAppMeshConfig) count(ctx context.Context, cr *meshv1.AppMeshConfig, list runtime.Object) *int {
	var c int
	labels := &client.MatchingLabels{"app": cr.Name}
	opts := &client.ListOptions{Namespace: cr.Namespace}
	labels.ApplyToList(opts)

	err := r.client.List(ctx, list, opts)
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

func calcPhase(status *meshv1.Status) meshv1.ConfigPhase {
	// return Unknown if any Distributed is nil
	if status.ServiceEntry.Distributed == nil ||
		status.WorkloadEntry.Distributed == nil ||
		status.VirtualService.Distributed == nil ||
		status.DestinationRule.Distributed == nil {
		return meshv1.ConfigStatusUnknown
	}

	// return Undistributed if the sum of all Distributed is zero
	if *status.ServiceEntry.Distributed+
		*status.WorkloadEntry.Distributed+
		*status.VirtualService.Distributed+
		*status.DestinationRule.Distributed == 0 {
		return meshv1.ConfigStatusUndistributed
	}

	// return Distributed if the sum of all Undistributed is zero
	if *status.ServiceEntry.Undistributed+
		*status.WorkloadEntry.Undistributed+
		*status.VirtualService.Undistributed+
		*status.DestinationRule.Undistributed == 0 {
		return meshv1.ConfigStatusDistributed
	}

	// otherwize return Distributing
	return meshv1.ConfigStatusDistributing
}
