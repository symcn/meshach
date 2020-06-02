package utils

import (
	"errors"
	"sync"
)

// Runnable allows a component to be started.
type Runnable interface {
	Start(<-chan struct{}) error
}

// Components ...
type Components struct {
	mu         sync.Mutex
	started    bool
	components []Runnable
}

// Add a new Runnable to Components. It panics if the Components is already started.
func (c *Components) Add(r Runnable) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		panic("Components.Add: Components is already started")
	}
	c.components = append(c.components, r)
}

// Start Components.
func (c *Components) Start(stopCh <-chan struct{}) error {
	if c.started {
		return errors.New("process.Host: already started")
	}

	var errChan chan error
	for _, c := range c.components {
		ctrl := c
		go func() {
			errChan <- ctrl.Start(stopCh)
		}()
	}

	c.started = true
	select {
	case <-stopCh:
		// We are done
		return nil
	case err := <-errChan:
		// Error starting a controller
		return err
	}
}
