// Package controller ...
package controller

import (
	"github.com/symcn/mesh-operator/pkg/controller/appmeshconfig"
	"github.com/symcn/mesh-operator/pkg/controller/configuraredservice"
	"github.com/symcn/mesh-operator/pkg/controller/istioconfig"
	"github.com/symcn/mesh-operator/pkg/controller/meshconfig"
	"github.com/symcn/mesh-operator/pkg/option"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, *option.ControllerOption) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, opt *option.ControllerOption) error {
	AddToManagerFuncs = append(AddToManagerFuncs, istioconfig.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, meshconfig.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, appmeshconfig.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, configuraredservice.Add)

	for _, f := range AddToManagerFuncs {
		if err := f(m, opt); err != nil {
			return err
		}
	}

	return nil
}
