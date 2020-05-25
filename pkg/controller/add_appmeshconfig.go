package controller

import (
	"github.com/mesh-operator/pkg/controller/appmeshconfig"
)

func init() {

	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, appmeshconfig.Add)
}
