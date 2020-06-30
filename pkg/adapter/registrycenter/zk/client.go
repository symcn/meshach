package zk

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/mesh-operator/pkg/adapter/constant"
	"github.com/mesh-operator/pkg/adapter/events"
	"github.com/mesh-operator/pkg/adapter/options"
	"github.com/mesh-operator/pkg/adapter/registrycenter"
	"github.com/mesh-operator/pkg/adapter/zookeeper"
	zkClient "github.com/samuel/go-zookeeper/zk"
	"k8s.io/klog"
)

func init() {
	registrycenter.Registry("zk", New)
}

type ZkRegistryClient struct {
	conn     *zkClient.Conn
	root     string
	services map[string]*events.Service
	out      chan *events.ServiceEvent
	scache   *zookeeper.PathCache
	pcaches  map[string]*zookeeper.PathCache
}

// NewClient Create a new client for zookeeper
func New(opt options.Registry) (events.Registry, error) {
	conn, err := zookeeper.GetClient(opt.Address, opt.Timeout)
	if err != nil || conn == nil {
		return nil, fmt.Errorf("get zookeeper client fail or client is nil, err:%+v", err)
	}

	return &ZkRegistryClient{
		conn:     conn,
		services: make(map[string]*events.Service),
		out:      make(chan *events.ServiceEvent),
		scache:   nil,
		pcaches:  make(map[string]*zookeeper.PathCache),
	}, nil
}

// Start ...
func (c *ZkRegistryClient) Start() error {
	// create a cache for every service
	scache, err := zookeeper.NewPathCache(c.conn, zookeeper.DubboRootPath, "REGISTRY", true)
	if err != nil {
		return err
	}
	c.scache = scache
	go c.eventLoop()

	// // FIXME just for debug: observe the status of the root path cache.
	var enablePrint = false
	if enablePrint {
		go func() {
			tick := time.Tick(10 * time.Second)
			for {
				select {
				case <-tick:
					klog.Infof("Observing cache of root path:%v\n  caches: %v\n  services: %v",
						scache.Path, scache.Cached, c.services)
					//spew.Dump(scache)
				}
			}
		}()
	}

	return nil
}

// eventLoop Creating the caches for every provider
func (c *ZkRegistryClient) eventLoop() {
	for event := range c.scache.Events() {
		hostname := path.Base(event.Path)
		if zookeeper.Ignore(hostname) {
			klog.Infof("Path should be ignored by registry client: %s", event.Path)
			continue
		}

		switch event.EventType {
		case zookeeper.PathCacheEventAdded:
			ppath := path.Join(event.Path, zookeeper.ProvidersPath)
			pcache, err := zookeeper.NewPathCache(c.conn, ppath, "REGISTRY", false)
			if err != nil {
				fmt.Printf("Create a provider cache %s has an error:%v\n", ppath, err)
				continue
			}
			c.pcaches[hostname] = pcache
			go func() {
				for event := range pcache.Events() {
					switch event.EventType {
					case zookeeper.PathCacheEventAdded:
						c.addInstance(hostname, path.Base(event.Path))
					case zookeeper.PathCacheEventChildrenReplaced:
						var rawUrls []string
						for _, p := range event.Paths {
							rawUrls = append(rawUrls, path.Base(p))
						}
						c.addInstances(hostname, rawUrls)
					case zookeeper.PathCacheEventDeleted:
						c.deleteInstance(hostname, path.Base(event.Path))
					}
				}
			}()
		case zookeeper.PathCacheEventDeleted:
			// In fact, this snippet always won't be executed.
			// At least one empty node of this service exists.
			// hostname := path.Base(event.path)
			c.deleteService(hostname)

			// Especially a service node may be deleted by someone manually,
			// we should avoid to maintain a service's path cache as true without the associated service node.
			// It is very essential to clear the cache flag for this path by setting it as false.
			c.scache.Cached[event.Path] = false
		}
	}
}

// Events channel is a stream of Service and instance updates
func (c *ZkRegistryClient) Events() <-chan *events.ServiceEvent {
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
	c.scache.Stop()
	for _, pcache := range c.pcaches {
		pcache.Stop()
	}
	close(c.out)
}

// GetCachedService
func (c *ZkRegistryClient) GetCachedService(serviceName string) *events.Service {
	service, ok := c.services[serviceName]
	if !ok {
		klog.Errorf("Can not find a service with name %s", serviceName)
		return nil
	}
	return service
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
			//Protocol: ep.Scheme,
			//Port:     ep.Port(),
			Protocol: constant.DubboProtocol,
			Port:     constant.MosnPort,
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

// deleteInstance
func (c *ZkRegistryClient) deleteInstance(hostname string, rawUrl string) {
	i, err := c.makeInstance(hostname, rawUrl)
	if err != nil {
		return
	}
	h := makeHostname(hostname, i)
	if s, ok := c.services[h]; ok {
		delete(s.Instances, rawUrl)
		go c.notify(&events.ServiceEvent{
			EventType: events.ServiceInstanceDeleted,
			Instance:  i,
		})
		// TODO should we unregister the service when all the instances are offline?
		//if len(s.instances) == 0 {
		//	c.deleteService(i.Service)
		//}
	}
}

// addInstance
func (c *ZkRegistryClient) addInstance(hostname string, rawUrl string) {
	i, err := c.makeInstance(hostname, rawUrl)
	if err != nil {
		return
	}

	s := c.addService(hostname, i)
	i.Service = s
	s.Instances[rawUrl] = i
	go c.notify(&events.ServiceEvent{
		EventType: events.ServiceInstanceAdded,
		Instance:  i,
	})
}

// addInstances
func (c *ZkRegistryClient) addInstances(hostname string, rawUrls []string) {
	instances := make(map[string]*events.Instance)
	var i *events.Instance
	var err error
	for _, ru := range rawUrls {
		i, err = c.makeInstance(hostname, ru)
		if err != nil {
			klog.Errorf("Make a instance has an error: %v", err)
			continue
		}
		instances[ru] = i
	}
	s := c.addService(hostname, i)

	for k := range instances {
		instances[k].Service = s
	}

	s.Instances = instances
	go c.notify(&events.ServiceEvent{
		EventType: events.ServiceInstancesReplace,
		Service:   s,
		Instances: instances,
	})
}

// addService
func (c *ZkRegistryClient) addService(hostname string, instance *events.Instance) *events.Service {
	s, ok := c.services[hostname]
	if !ok {
		s = &events.Service{
			Name:      hostname,
			Ports:     make([]*events.Port, 0),
			Instances: make(map[string]*events.Instance),
		}
		c.services[hostname] = s
		if instance != nil {
			s.AddPort(instance.Port)
		}

		go c.notify(&events.ServiceEvent{
			EventType: events.ServiceAdded,
			Service:   s,
		})
	}

	return s
}

func (c *ZkRegistryClient) deleteService(hostname string) {
	cache, ok := c.pcaches[hostname]
	if ok {
		cache.Stop()
	}

	for h, s := range c.services {
		if strings.HasSuffix(h, hostname) {
			delete(c.services, h)
			go c.notify(&events.ServiceEvent{
				EventType: events.ServiceDeleted,
				Service:   s,
			})
		}
	}
}

// notify
func (c *ZkRegistryClient) notify(event *events.ServiceEvent) {
	c.out <- event
}

func makeHostname(hostname string, instance *events.Instance) string {
	return hostname
	// We don't need version for the moment.
	// + ":" + instance.Labels["version"]
}
