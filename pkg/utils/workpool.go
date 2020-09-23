package utils

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"k8s.io/klog"
)

// WorkerPool workerPool interface
type WorkerPool interface {
	Schedule(task func())

	ScheduleAlways(task func())

	ScheduleAuto(task func())
}

type workerPool struct {
	work chan func()
	sem  chan struct{}
}

// NewWorkerPool build workpool object
func NewWorkerPool(size int) WorkerPool {
	return &workerPool{
		work: make(chan func()),
		sem:  make(chan struct{}, size),
	}
}

func (p *workerPool) Schedule(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.spawnWorker(task)
	}
}

func (p *workerPool) ScheduleAlways(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.spawnWorker(task)
	default:
		klog.Infof("[syncpool] workerpool new goroutine")
		GoWithRecover(func() {
			task()
		}, nil)
	}
}

func (p *workerPool) ScheduleAuto(task func()) {
	select {
	case p.work <- task:
		return
	default:
	}

	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.spawnWorker(task)
	default:
		// klog.V().Infof("[syncpool] workerpool new goroutine")
		GoWithRecover(func() {
			task()
		}, nil)
	}
}

func (p *workerPool) spawnWorker(task func()) {
	defer func() {
		if r := recover(); r != nil {
			klog.Warningf("syncpool panic %v\n%s", r, string(debug.Stack()))
		}
		<-p.sem
	}()

	for {
		task()
		task = <-p.work
	}
}

// GoWithRecover go task with goroutine and recover
func GoWithRecover(handler func(), recoverHandler func(r interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "%s goroutine panic:%v\n%s\n", time.Now().Format("2006-01-02 15:04:05"), r, string(debug.Stack()))

				if recoverHandler != nil {
					go func() {
						defer func() {
							if p := recover(); p != nil {
								fmt.Fprintf(os.Stderr, "recover goroutine panic:%v\n%s\n", p, string(debug.Stack()))
							}
						}()

						recoverHandler(r)
					}()
				}
			}
		}()

		handler()
	}()
}
