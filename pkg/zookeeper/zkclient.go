package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"net/url"
	"path"
	"strings"
	"time"
)

type ZkClient struct {
	conn     *zk.Conn
	root     string
	services map[string]*Service
	out      chan ServiceEvent

	scache  *pathCache
	pcaches map[string]*pathCache
}

// NewClient Create a new client for zookeeper
func NewClient(root string, conn *zk.Conn) *ZkClient {
	return &ZkClient{
		conn:     conn,
		root:     root,
		services: make(map[string]*Service),
		out:      make(chan ServiceEvent),
		scache:   nil,
		pcaches:  make(map[string]*pathCache),
	}
}

// Events channel is a stream of Service and instance updates
func (c *ZkClient) Events() <-chan ServiceEvent {
	return c.out
}

func (c *ZkClient) Service(hostname string) *Service {
	return c.services[hostname]
}

func (c *ZkClient) Services() []*Service {
	services := make([]*Service, 0, len(c.services))
	for _, service := range c.services {
		services = append(services, service)
	}
	return services
}

func (c *ZkClient) Instances(hostname string) []*Instance {
	instances := make([]*Instance, 0)
	service, ok := c.services[hostname]
	if !ok {
		return instances
	}

	for _, instance := range service.instances {
		instances = append(instances, instance)
	}
	return instances
}

func (c *ZkClient) InstancesByHost(hosts []string) []*Instance {
	instances := make([]*Instance, 0)
	for _, service := range c.services {
		for _, instance := range service.instances {
			for _, host := range hosts {
				if instance.Host == host {
					instances = append(instances, instance)
				}
			}
		}
	}
	return instances
}

func (c *ZkClient) Stop() {
	c.scache.stop()
	for _, pcache := range c.pcaches {
		pcache.stop()
	}
	close(c.out)
}

func (c *ZkClient) Start() error {
	// create a cache for all services
	scache, err := newPathCache(c.conn, c.root)
	if err != nil {
		return err
	}
	c.scache = scache

	// TODO observe the status of the root path cache.
	go func() {
		tick := time.Tick(10 * time.Second)
		for {
			select {
			case <-tick:
				fmt.Printf("Observing root pach cache :%v\n%v\n%v\n", scache.path, scache.cached, c.services)
				//spew.Dump(scache)
			}
		}
	}()

	go c.eventLoop()
	return nil
}

// eventLoop Creating the caches for every provider
func (c *ZkClient) eventLoop() {
	for event := range c.scache.events() {
		switch event.eventType {
		case pathCacheEventAdded:
			hostname := path.Base(event.path)
			ppath := path.Join(event.path, providersPath)
			pcache, err := newPathCache(c.conn, ppath)
			if err != nil {
				fmt.Printf("Create a provider cache %s has an error:%v\n", ppath, err)
				continue
			}
			c.pcaches[hostname] = pcache
			go func() {
				for event := range pcache.events() {
					switch event.eventType {
					case pathCacheEventAdded:
						c.addInstance(hostname, path.Base(event.path))
					case pathCacheEventDeleted:
						c.deleteInstance(hostname, path.Base(event.path))
					}
				}
			}()
		case pathCacheEventDeleted:
			// In fact, this snippet always won't be executed.
			// At least one empty node of this service exists.
			hostname := path.Base(event.path)
			c.deleteService(hostname)

			// Especially a service node may be deleted by someone manually,
			// we should avoid to maintain a service's path cache as true without the associated service node.
			// It is very essential to clear the cache flag for this path by setting it as false.
			c.scache.cached[event.path] = false
		}
	}
}

func (c *ZkClient) makeInstance(hostname string, rawUrl string) (*Instance, error) {
	cleanUrl, err := url.QueryUnescape(rawUrl)
	if err != nil {
		return nil, err
	}
	ep, err := url.Parse(cleanUrl)
	if err != nil {
		return nil, err
	}

	instance := &Instance{
		Host: ep.Host,
		Port: &Port{
			Protocol: ep.Scheme,
			Port:     ep.Port(),
		},
		Labels: make(map[string]string),
	}

	for key, value := range ep.Query() {
		if value != nil {
			instance.Labels[key] = value[0]
		}
	}
	return instance, nil
}

func (c *ZkClient) deleteInstance(hostname string, rawUrl string) {
	i, err := c.makeInstance(hostname, rawUrl)
	if err != nil {
		return
	}
	h := makeHostname(hostname, i)
	if s, ok := c.services[h]; ok {
		delete(s.instances, rawUrl)
		go c.notify(ServiceEvent{
			EventType: ServiceInstanceDeleted,
			Instance:  i,
		})
		// TODO should we unregister the service when all the instances are offline?
		//if len(s.instances) == 0 {
		//	c.deleteService(i.Service)
		//}
	}
}

func (c *ZkClient) addInstance(hostname string, rawUrl string) {
	i, err := c.makeInstance(hostname, rawUrl)
	if err != nil {
		return
	}

	s := c.addService(hostname, i)
	i.Service = s
	s.instances[rawUrl] = i
	go c.notify(ServiceEvent{
		EventType: ServiceInstanceAdded,
		Instance:  i,
	})
}

func (c *ZkClient) addService(hostname string, instance *Instance) *Service {
	h := makeHostname(hostname, instance)
	s, ok := c.services[h]
	if !ok {
		s = &Service{
			name:      h,
			ports:     make([]*Port, 0),
			instances: make(map[string]*Instance),
		}
		c.services[h] = s
		s.AddPort(instance.Port)
		go c.notify(ServiceEvent{
			EventType: ServiceAdded,
			Service:   s,
		})
	}

	return s
}

func (c *ZkClient) deleteService(hostname string) {
	cache, ok := c.pcaches[hostname]
	if ok {
		cache.stop()
	}

	for h, s := range c.services {
		if strings.HasSuffix(h, hostname) {
			delete(c.services, h)
			go c.notify(ServiceEvent{
				EventType: ServiceDeleted,
				Service:   s,
			})
		}
	}
}

func (c *ZkClient) notify(event ServiceEvent) {
	c.out <- event
}

func makeHostname(hostname string, instance *Instance) string {
	return hostname
	// We don't need version for the moment.
	// + ":" + instance.Labels["version"]
}
