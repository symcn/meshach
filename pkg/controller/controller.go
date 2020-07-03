// Package controller ...
package controller

import (
	"github.com/symcn/mesh-operator/pkg/controller/istioconfig"
	"github.com/symcn/mesh-operator/pkg/controller/meshconfig"
	"github.com/symcn/mesh-operator/pkg/controller/servicemeshentry"
	"github.com/symcn/mesh-operator/pkg/option"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, *option.ControllerOption) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, opt *option.ControllerOption) error {
	AddToManagerFuncs = append(AddToManagerFuncs, servicemeshentry.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, istioconfig.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, meshconfig.Add)

	for _, f := range AddToManagerFuncs {
		if err := f(m, opt); err != nil {
			return err
		}
	}

	return nil
}
