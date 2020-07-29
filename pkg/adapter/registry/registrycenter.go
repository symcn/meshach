package registry

import (
	"fmt"

	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/option"
	"k8s.io/klog"
)

type constructor func(regOpt option.Registry) (component.Registry, error)

var (
	registryInstance = make(map[string]constructor)
)

// Registry ...
func Registry(typ string, f constructor) {
	if _, ok := registryInstance[typ]; ok {
		klog.Fatalf("repeat registry [registry center instance]: %s", typ)
	}
	registryInstance[typ] = f
}

// GetRegistry ...
func GetRegistry(opt option.Registry) (component.Registry, error) {
	if f, ok := registryInstance[opt.Type]; ok {
		return f(opt)
	}
	return nil, fmt.Errorf("registry center {%s} was not implemented", opt.Type)
}
