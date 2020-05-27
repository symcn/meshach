// Package controller ...
package controller

import (
	"github.com/mesh-operator/pkg/controller/appmeshconfig"
	"github.com/mesh-operator/pkg/controller/istioconfig"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	AddToManagerFuncs = append(AddToManagerFuncs, appmeshconfig.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, istioconfig.Add)

	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}

	return nil
}
