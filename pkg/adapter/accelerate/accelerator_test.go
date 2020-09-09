package accelerate

import (
	. "github.com/onsi/ginkgo"
	"github.com/symcn/mesh-operator/pkg/adapter/handler"
	"github.com/symcn/mesh-operator/pkg/adapter/types"
	"k8s.io/klog"
)

var ac *Accelerator
var h *handler.LogEventHandler
var done chan struct{}

var _ = Describe("Test cases for accelerator.", func() {
	BeforeEach(func() {
		klog.Infof("before processing for each Describe")
		done = make(chan struct{})
		h = &handler.LogEventHandler{}
		ac = NewAccelerator(2, done)
	})

	Describe("Tests for ConfigurationService", func() {
		var se *types.ServiceEvent
		Context("Start an accelerator", func() {
			BeforeEach(func() {
				se = &types.ServiceEvent{
					Service: &types.Service{
						Name:      "foo",
						Ports:     nil,
						Instances: nil,
					}}
			})

			It("A normal accelerating process", func() {
				klog.Info("start to accelerator in a normal way")
				fn := func() { h.ReplaceInstances(se) }
				ac.Accelerate(fn, se.Service.Name)
			})
		})

	})

	AfterEach(func() {
		klog.Infof("after processing for each context")
		close(done)
	})

})
