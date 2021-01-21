package accelerate

import (
	. "github.com/onsi/ginkgo"
	"github.com/symcn/meshach/pkg/adapter/handler"
	"github.com/symcn/meshach/pkg/adapter/types"
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

	})

	Describe("Tests for ConfigurationService", func() {
		var se *types.ServiceEvent
		Context("Start an accelerator", func() {
			BeforeEach(func() {
				ac = NewAccelerator(2, done)
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

		Context("Start an accelerator", func() {
			BeforeEach(func() {
				ac = NewAccelerator(0, done)
				se = &types.ServiceEvent{
					Service: &types.Service{
						Name:      "foo",
						Ports:     nil,
						Instances: nil,
					}}
			})

			It("channels' size is less than 0", func() {
				klog.Info("start to accelerator with a zero channels' size")
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
