// Package controller ...
package controller

import (
	"github.com/mesh-operator/pkg/controller/appmeshconfig"
	"github.com/mesh-operator/pkg/controller/istioconfig"
	"github.com/mesh-operator/pkg/option"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, *option.ControllerOption) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, opt *option.ControllerOption) error {
	AddToManagerFuncs = append(AddToManagerFuncs, appmeshconfig.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, istioconfig.Add)

	for _, f := range AddToManagerFuncs {
		if err := f(m, opt); err != nil {
			return err
		}
	}

	return nil
}
