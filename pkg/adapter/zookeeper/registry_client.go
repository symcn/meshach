package zookeeper

import (
	"fmt"
	"github.com/mesh-operator/pkg/adapter/constant"
	"github.com/mesh-operator/pkg/adapter/events"
	"github.com/samuel/go-zookeeper/zk"
	"net/url"
	"path"
	"strings"
	"time"
)

type ZkRegistryClient struct {
	conn     *zk.Conn
	root     string
	services map[string]*events.Service
	out      chan events.ServiceEvent
	scache   *pathCache
	pcaches  map[string]*pathCache
}

// NewClient Create a new client for zookeeper
func NewRegistryClient(conn *zk.Conn) *ZkRegistryClient {
	return &ZkRegistryClient{
		conn:     conn,
		services: make(map[string]*events.Service),
		out:      make(chan events.ServiceEvent),
		scache:   nil,
		pcaches:  make(map[string]*pathCache),
	}
}

// Events channel is a stream of Service and instance updates
func (c *ZkRegistryClient) Events() <-chan events.ServiceEvent {
	return c.out
}

func (c *ZkRegistryClient) Service(hostname string) *events.Service {
	return c.services[hostname]
}

func (c *ZkRegistryClient) Services() []*events.Service {
	services := make([]*events.Service, 0, len(c.services))
	for _, service := range c.services {
		services = append(services, service)
	}
	return services
}

func (c *ZkRegistryClient) Instances(hostname string) []*events.Instance {
	instances := make([]*events.Instance, 0)
	service, ok := c.services[hostname]
	if !ok {
		return instances
	}

	for _, instance := range service.Instances {
		instances = append(instances, instance)
	}
	return instances
}

func (c *ZkRegistryClient) InstancesByHost(hosts []string) []*events.Instance {
	instances := make([]*events.Instance, 0)
	for _, service := range c.services {
		for _, instance := range service.Instances {
			for _, host := range hosts {
				if instance.Host == host {
					instances = append(instances, instance)
				}
			}
		}
	}
	return instances
}

func (c *ZkRegistryClient) Stop() {
	c.scache.stop()
	for _, pcache := range c.pcaches {
		pcache.stop()
	}
	close(c.out)
}

func (c *ZkRegistryClient) Start() error {
	// create a cache for all services
	scache, err := newPathCache(c.conn, DubboRootPath)
	if err != nil {
		return err
	}
	c.scache = scache
	go c.eventLoop()

	// // FIXME just for debug: observe the status of the root path cache.
	go func() {
		tick := time.Tick(10 * time.Second)
		for {
			select {
			case <-tick:
				fmt.Printf("Observing cache of root path:%v\n  caches: %v\n  services: %v\n",
					scache.path, scache.cached, c.services)
				//spew.Dump(scache)
			}
		}
	}()

	return nil
}

// FindAppIdentifier
func (c *ZkRegistryClient) FindAppIdentifier(serviceName string) string {
	var appName string
	service, ok := c.services[serviceName]
	if !ok {
		fmt.Printf("Can not find a service with name %s\n", service)
		return appName
	}

	if service.Instances == nil || len(service.Instances) <= 0 {
		fmt.Printf("Can not find any instance from a service %s which has an empty instances list.\n", service)
		return appName
	}
	for _, ins := range service.Instances {
		if ins != nil && ins.Labels != nil {
			return ins.Labels[constant.ApplicationLabel]
		}
	}
	return appName

}

// eventLoop Creating the caches for every provider
func (c *ZkRegistryClient) eventLoop() {
	for event := range c.scache.events() {
		switch event.eventType {
		case pathCacheEventAdded:
			hostname := path.Base(event.path)
			ppath := path.Join(event.path, ProvidersPath)
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

func (c *ZkRegistryClient) makeInstance(hostname string, rawUrl string) (*events.Instance, error) {
	cleanUrl, err := url.QueryUnescape(rawUrl)
	if err != nil {
		return nil, err
	}
	ep, err := url.Parse(cleanUrl)
	if err != nil {
		return nil, err
	}

	instance := &events.Instance{
		Host: ep.Host,
		Port: &events.Port{
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

func (c *ZkRegistryClient) deleteInstance(hostname string, rawUrl string) {
	i, err := c.makeInstance(hostname, rawUrl)
	if err != nil {
		return
	}
	h := makeHostname(hostname, i)
	if s, ok := c.services[h]; ok {
		delete(s.Instances, rawUrl)
		go c.notify(events.ServiceEvent{
			EventType: events.ServiceInstanceDeleted,
			Instance:  i,
		})
		// TODO should we unregister the service when all the instances are offline?
		//if len(s.instances) == 0 {
		//	c.deleteService(i.Service)
		//}
	}
}

func (c *ZkRegistryClient) addInstance(hostname string, rawUrl string) {
	i, err := c.makeInstance(hostname, rawUrl)
	if err != nil {
		return
	}

	s := c.addService(hostname, i)
	i.Service = s
	s.Instances[rawUrl] = i
	go c.notify(events.ServiceEvent{
		EventType: events.ServiceInstanceAdded,
		Instance:  i,
	})
}

func (c *ZkRegistryClient) addService(hostname string, instance *events.Instance) *events.Service {
	h := makeHostname(hostname, instance)
	s, ok := c.services[h]
	if !ok {
		s = &events.Service{
			Name:      h,
			Ports:     make([]*events.Port, 0),
			Instances: make(map[string]*events.Instance),
		}
		c.services[h] = s
		s.AddPort(instance.Port)
		go c.notify(events.ServiceEvent{
			EventType: events.ServiceAdded,
			Service:   s,
		})
	}

	return s
}

func (c *ZkRegistryClient) deleteService(hostname string) {
	cache, ok := c.pcaches[hostname]
	if ok {
		cache.stop()
	}

	for h, s := range c.services {
		if strings.HasSuffix(h, hostname) {
			delete(c.services, h)
			go c.notify(events.ServiceEvent{
				EventType: events.ServiceDeleted,
				Service:   s,
			})
		}
	}
}

// notify
func (c *ZkRegistryClient) notify(event events.ServiceEvent) {
	c.out <- event
}

func makeHostname(hostname string, instance *events.Instance) string {
	return hostname
	// We don't need version for the moment.
	// + ":" + instance.Labels["version"]
}
