package zookeeper

import (
	"github.com/samuel/go-zookeeper/zk"
)

var zkServersUrl = "10.12.210.70"

type Client struct {
	conn     *zk.Conn
	root     string
	services map[string]*Service
	out      chan ServiceEvent

	scache  *pathCache
	pcaches map[string]*pathCache
}

// NewClient Create a new client for zookeeper
func NewClient(root string, conn *zk.Conn) *Client {
	return &Client{
		conn:     conn,
		root:     root,
		services: make(map[string]*Service),
		out:      make(chan ServiceEvent),
		scache:   nil,
		pcaches:  make(map[string]*pathCache),
	}
}

// Events channel is a stream of Service and instance updates
func (c *Client) Events() <-chan ServiceEvent {
	return c.out
}

func (c *Client) Service(hostname string) *Service {
	return c.services[hostname]
}

func (c *Client) Services() []*Service {
	services := make([]*Service, 0, len(c.services))
	for _, service := range c.services {
		services = append(services, service)
	}
	return services
}

func (c *Client) Instances(hostname string) []*Instance {
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

func (c *Client) InstancesByHost(hosts []string) []*Instance {
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

func (c *Client) Stop() {
	c.scache.stop()
	for _, pcache := range c.pcaches {
		pcache.stop()
	}
	close(c.out)
}

func (c *Client) Start() error {
	cache, err := newPathCache(c.conn, c.root)
	if err != nil {
		return err
	}
	c.scache = cache

}
