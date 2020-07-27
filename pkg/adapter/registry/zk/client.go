package zk

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/symcn/mesh-operator/pkg/adapter/constant"
	"github.com/symcn/mesh-operator/pkg/adapter/types"

	zkClient "github.com/samuel/go-zookeeper/zk"
	"github.com/symcn/mesh-operator/pkg/adapter/component"
	"github.com/symcn/mesh-operator/pkg/adapter/registry"
	"github.com/symcn/mesh-operator/pkg/adapter/zookeeper"
	"github.com/symcn/mesh-operator/pkg/option"
	"k8s.io/klog"
)

func init() {
	registry.Registry("zk", New)
}

// RegistryClient ...
type RegistryClient struct {
	conn                   *zkClient.Conn
	root                   string
	services               map[string]*types.Service
	serviceOut             chan *types.ServiceEvent
	accessors              map[string]*types.Service
	accessorOut            chan *types.ServiceEvent
	scopedAccessorsMapping map[string]map[string]struct{}
	scache                 *zookeeper.PathCache
	pcaches                map[string]*zookeeper.PathCache
	ccaches                map[string]*zookeeper.PathCache
}

// New Create a new client for zookeeper
func New(opt option.Registry) (component.Registry, error) {
	conn, _, err := zkClient.Connect(opt.Address, time.Duration(opt.Timeout)*time.Second)
	if err != nil {
		klog.Errorf("Get zookeeper client has an error: %v", err)
	}
	if err != nil || conn == nil {
		return nil, fmt.Errorf("get zookeeper client fail or client is nil, err:%+v", err)
	}

	return &RegistryClient{
		conn:                   conn,
		services:               make(map[string]*types.Service),
		serviceOut:             make(chan *types.ServiceEvent),
		accessors:              make(map[string]*types.Service),
		accessorOut:            make(chan *types.ServiceEvent),
		scopedAccessorsMapping: make(map[string]map[string]struct{}),
		scache:                 nil,
		pcaches:                make(map[string]*zookeeper.PathCache),
		ccaches:                make(map[string]*zookeeper.PathCache),
	}, nil
}

// Start ...
func (c *RegistryClient) Start() error {
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
func (c *RegistryClient) eventLoop() {
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
				klog.Errorf("Create a provider cache %s has an error: %v", ppath, err)
				continue
			}
			c.pcaches[hostname] = pcache
			go func() {
				for event := range pcache.Events() {
					switch event.EventType {
					case zookeeper.PathCacheEventAdded:
						c.addServiceInstance(hostname, path.Base(event.Path))
					case zookeeper.PathCacheEventChildrenReplaced:
						var rawURLs []string
						for _, p := range event.Paths {
							rawURLs = append(rawURLs, path.Base(p))
						}
						c.replaceServiceInstances(hostname, rawURLs)
					case zookeeper.PathCacheEventDeleted:
						c.deleteServiceInstance(hostname, path.Base(event.Path))
					}
				}
			}()
			// creating a path cache for every consumer
			cpath := path.Join(event.Path, zookeeper.ConsumersPath)
			ccache, err := zookeeper.NewPathCache(c.conn, cpath, "REGISTRY", false)
			if err != nil {
				klog.Errorf("Create a consumer cache %s has an error: %v", cpath, err)
			}
			c.ccaches[hostname] = ccache
			go func() {
				for event := range ccache.Events() {
					switch event.EventType {
					case zookeeper.PathCacheEventAdded:
						klog.Warningf("added event has not been support by consumer pathcache yet")
					case zookeeper.PathCacheEventChildrenReplaced:
						var rawURLs []string
						for _, p := range event.Paths {
							rawURLs = append(rawURLs, path.Base(p))
						}
						c.replaceAccessorInstances(hostname, rawURLs)
					case zookeeper.PathCacheEventDeleted:
						klog.Warningf("deleted event has not been support by consumer pathcache yet")
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

// ServiceEvents channel is a stream of Service and instance updates
func (c *RegistryClient) ServiceEvents() <-chan *types.ServiceEvent {
	return c.serviceOut
}

// AccessorEvents this channel will spawn the accessor changing events.
func (c *RegistryClient) AccessorEvents() <-chan *types.ServiceEvent {
	return c.accessorOut
}

// Service ...
func (c *RegistryClient) Service(hostname string) *types.Service {
	return c.services[hostname]
}

// Services ...
func (c *RegistryClient) Services() []*types.Service {
	services := make([]*types.Service, 0, len(c.services))
	for _, service := range c.services {
		services = append(services, service)
	}
	return services
}

// Instances ...
func (c *RegistryClient) Instances(hostname string) []*types.Instance {
	instances := make([]*types.Instance, 0)
	service, ok := c.services[hostname]
	if !ok {
		return instances
	}

	for _, instance := range service.Instances {
		instances = append(instances, instance)
	}
	return instances
}

// InstancesByHost ...
func (c *RegistryClient) InstancesByHost(hosts []string) []*types.Instance {
	instances := make([]*types.Instance, 0)
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

// Stop ...
func (c *RegistryClient) Stop() {
	c.scache.Stop()
	for _, pcache := range c.pcaches {
		pcache.Stop()
	}
	close(c.serviceOut)
	close(c.accessorOut)
}

// GetCachedService ...
func (c *RegistryClient) GetCachedService(serviceName string) *types.Service {
	service, ok := c.services[serviceName]
	if !ok {
		klog.Errorf("Can not find a service with name %s", serviceName)
		return nil
	}
	return service
}

func (c *RegistryClient) makeServiceInstance(hostname string, rawURL string) (*types.Instance, error) {
	cleanURL, err := url.QueryUnescape(rawURL)
	if err != nil {
		return nil, err
	}
	ep, err := url.Parse(cleanURL)
	if err != nil {
		return nil, err
	}

	instance := &types.Instance{
		Host: ep.Host,
		Port: &types.Port{
			// Protocol: ep.Scheme,
			Port:     ep.Port(),
			Protocol: constant.DubboProtocol,
			// Port:     constant.MosnPort,
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
func (c *RegistryClient) deleteServiceInstance(hostname string, rawURL string) {
	i, err := c.makeServiceInstance(hostname, rawURL)
	if err != nil {
		return
	}
	h := makeHostname(hostname, i)
	if s, ok := c.services[h]; ok {
		delete(s.Instances, rawURL)
		go c.notifyServiceEvent(&types.ServiceEvent{
			EventType: types.ServiceInstanceDeleted,
			Instance:  i,
		})
		// TODO should we unregister the service when all the instances are offline?
		//if len(s.instances) == 0 {
		//	c.deleteService(i.Service)
		//}
	}
}

// addInstance
func (c *RegistryClient) addServiceInstance(hostname string, rawURL string) {
	i, err := c.makeServiceInstance(hostname, rawURL)
	if err != nil {
		return
	}

	s := c.addService(hostname, i)
	i.Service = s
	s.Instances[rawURL] = i
	go c.notifyServiceEvent(&types.ServiceEvent{
		EventType: types.ServiceInstanceAdded,
		Instance:  i,
	})
}

// replaceInstances
func (c *RegistryClient) replaceServiceInstances(hostname string, rawURLs []string) {
	instances := make(map[string]*types.Instance)
	var i *types.Instance
	var err error
	for _, ru := range rawURLs {
		i, err = c.makeServiceInstance(hostname, ru)
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
	go c.notifyServiceEvent(&types.ServiceEvent{
		EventType: types.ServiceInstancesReplace,
		Service:   s,
		Instances: instances,
	})
}

// addService
func (c *RegistryClient) addService(hostname string, instance *types.Instance) *types.Service {
	s, ok := c.services[hostname]
	if !ok {
		s = &types.Service{
			Name:      hostname,
			Ports:     make([]*types.Port, 0),
			Instances: make(map[string]*types.Instance),
		}
		c.services[hostname] = s
		if instance != nil {
			s.AddPort(instance.Port)
		}

		go c.notifyServiceEvent(&types.ServiceEvent{
			EventType: types.ServiceAdded,
			Service:   s,
		})
	}

	return s
}

func (c *RegistryClient) deleteService(hostname string) {
	cache, ok := c.pcaches[hostname]
	if ok {
		cache.Stop()
	}

	for h, s := range c.services {
		if strings.HasSuffix(h, hostname) {
			delete(c.services, h)
			go c.notifyServiceEvent(&types.ServiceEvent{
				EventType: types.ServiceDeleted,
				Service:   s,
			})
		}
	}
}

// makeAccessorInstance
func (c *RegistryClient) makeAccessorInstance(hostname string, rawURL string) (*types.Instance, error) {
	cleanURL, err := url.QueryUnescape(rawURL)
	if err != nil {
		return nil, err
	}
	ep, err := url.Parse(cleanURL)
	if err != nil {
		return nil, err
	}

	instance := &types.Instance{
		Host:   ep.Host,
		Labels: make(map[string]string),
	}

	for key, value := range ep.Query() {
		if value != nil {
			instance.Labels[key] = value[0]
		}
	}
	return instance, nil
}

// addAccessor
func (c *RegistryClient) addAccessor(hostname string) *types.Service {
	a, ok := c.accessors[hostname]
	if !ok {
		a = &types.Service{
			Name:      hostname,
			Instances: make(map[string]*types.Instance),
		}
		c.accessors[hostname] = a
	}
	return a
}

// replaceAccessors
func (c *RegistryClient) replaceAccessorInstances(hostname string, rawURLs []string) {
	instances := make(map[string]*types.Instance)
	var i *types.Instance
	var err error
	for _, ru := range rawURLs {
		i, err = c.makeAccessorInstance(hostname, ru)
		if err != nil {
			klog.Errorf("Make a instance has an error: %v", err)
			continue
		}
		instances[ru] = i
	}
	a := c.addAccessor(hostname)

	for k := range instances {
		instances[k].Service = a

		// cache an accessor instance based on a scope key.
		scopedKey := instances[k].Labels["app"]
		_, ok := c.scopedAccessorsMapping[scopedKey]
		if !ok {
			c.scopedAccessorsMapping[scopedKey] = make(map[string]struct{})
		}
		c.scopedAccessorsMapping[scopedKey][a.Name] = struct{}{}
	}

	a.Instances = instances

	go c.notifyAccessorEvent(&types.ServiceEvent{
		EventType: types.ServiceInstancesReplace,
		Service:   a,
		Instances: instances,
	})
}

// notifyServiceEvent
func (c *RegistryClient) notifyServiceEvent(event *types.ServiceEvent) {
	c.serviceOut <- event
}

// notifyAccessorEvent
func (c *RegistryClient) notifyAccessorEvent(event *types.ServiceEvent) {
	c.accessorOut <- event
}

func makeHostname(hostname string, instance *types.Instance) string {
	return hostname
	// We don't need version for the moment.
	// + ":" + instance.Labels["version"]
}

// GetCachedScopedMapping ...
func (c *RegistryClient) GetCachedScopedMapping(scopedKey string) map[string]struct{} {
	return c.scopedAccessorsMapping[scopedKey]
}
