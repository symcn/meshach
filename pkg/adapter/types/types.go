/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import "strconv"

// ServiceEventType ...
type ServiceEventType int

// ServiceEvent ...
type ServiceEvent struct {
	EventType ServiceEventType
	Service   *Service
	Instance  *Instance
	Instances map[string]*Instance
}

// Port ...
type Port struct {
	Protocol string
	Port     string
}

// Service ...
type Service struct {
	Name      string
	Ports     []*Port
	Instances map[string]*Instance
}

// Instance is the instance of the service provider
// instance lables includes:
// 		appName         name of the application which host the service itself
// 		language		language the service is build with
// 		rpcVer			version of the sofa rpc framework
// 		dynamic			...
// 		tartTime		time when this instance is started
// 		version			version of this service instance
// 		accepts			...
// 		delay			...
// 		weight			route weight of this instance
// 		timeout			server side timeout
// 		id				id of the service, already deprecated
// 		pid				process id of the service instance
// 		uniqueId		unique id of the service
type Instance struct {
	Service *Service
	Host    string
	Port    *Port
	Labels  map[string]string
}

// Enumeration of ServiceEventType
const (
	ServiceAdded ServiceEventType = iota
	ServiceDeleted
	ServiceInstanceAdded
	ServiceInstanceDeleted
	ServiceInstancesReplace
)

// ConfigEventType ...
type ConfigEventType int

// Enumeration of ConfigEventType
const (
	ConfigEntryAdded ConfigEventType = iota
	ConfigEntryDeleted
	ConfigEntryChanged
)

// Portoi convert port to int
func (p *Port) Portoi() int {
	port, err := strconv.Atoi(p.Port)
	if err != nil {
		return 0
	}
	return port
}

// AddPort ...
func (s *Service) AddPort(port *Port) {
	exist := false
	for _, p := range s.Ports {
		if p.Port == port.Port && p.Protocol == port.Protocol {
			exist = true
			break
		}
	}
	if !exist {
		s.Ports = append(s.Ports, port)
	}
}

// Hostname ...
func (s *Service) Hostname() string {
	return s.Name
}

// GetPorts ...
func (s *Service) GetPorts() []*Port {
	return s.Ports
}

// GetInstances ...
func (s *Service) GetInstances() map[string]*Instance {
	return s.Instances
}

// ConfiguratorConfig ...
type ConfiguratorConfig struct {
	ConfigVersion string       `yaml:"configVersion"`
	Scope         string       `yaml:"scope"`
	Key           string       `yaml:"key"`
	Enabled       bool         `yaml:"enabled"`
	Configs       []ConfigItem `yaml:"configs"`
}

// ConfigItem ...
type ConfigItem struct {
	Type              string            `yaml:"type"`
	Enabled           bool              `yaml:"enabled"`
	Addresses         []string          `yaml:"addresses"`
	ProviderAddresses []string          `yaml:"providerAddresses"`
	Services          []string          `yaml:"services"`
	Applications      []string          `yaml:"applications"`
	Parameters        map[string]string `yaml:"parameters"`
	Side              string            `yaml:"side"`
}

// ConfigEvent ...
type ConfigEvent struct {
	EventType   ConfigEventType
	Path        string
	ConfigEntry *ConfiguratorConfig
}

// FlagConfigParameter just for customized setting
type FlagConfigParameter struct {
	Flags  []*Flag
	Manual bool
}

// Flag ...
type Flag struct {
	Key    string
	Weight int32
}
