package handler

import (
	"github.com/mesh-operator/pkg/adapter/events"
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
)

// Init handler initial
func Init(cmg *k8smanager.ClusterManager) ([]events.EventHandler, error) {

	evts := make([]events.EventHandler, 0, 2)

	// add kube v2 handler
	kubev2h, err := NewKubeV2EventHander(cmg)
	if err != nil {
		return nil, err
	}
	evts = append(evts, kubev2h)

	// // add log handler
	// logh, err := NewLogEventHandler()
	// if err != nil {
	// 	return nil, err
	// }
	// evts = append(evts, logh)

	return evts, nil
}
