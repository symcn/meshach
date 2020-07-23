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

package zookeeper

import (
	"path"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/symcn/mesh-operator/pkg/adapter/metrics"
	"k8s.io/klog"
)

// Some default settings.
var (
	DubboRootPath    = "/dubbo"
	ProvidersPath    = "providers"
	ConsumersPath    = "consumers"
	IgnoredHostNames = []string{"metadata", "config"}
	ConfiguratorPath = DubboRootPath + "/config/dubbo"
)

// PathCacheEventType ...
type PathCacheEventType int

// PathCache ...
type PathCache struct {
	conn           *zk.Conn
	watchCh        chan zk.Event
	notifyCh       chan PathCacheEvent
	stopCh         chan bool
	addChildCh     chan string
	Path           string
	Cached         map[string]bool
	owner          string
	zkEventCounter int64
}

// PathCacheEvent ...
type PathCacheEvent struct {
	EventType PathCacheEventType
	Path      string
	Paths     []string
}

// Enumeration of PathCacheEventType
const (
	PathCacheEventAdded PathCacheEventType = iota
	PathCacheEventDeleted
	PathCacheEventChanged
	PathCacheEventChildrenReplaced
)

// NewPathCache ...
func NewPathCache(conn *zk.Conn, path string, owner string, isSvcPath bool) (*PathCache, error) {
	klog.Infof("Create a cache for path: [%s]", path)

	p := &PathCache{
		conn:           conn,
		Path:           path,
		Cached:         make(map[string]bool),
		watchCh:        make(chan zk.Event),       // The zookeeper component will be forwarded to this channel.
		notifyCh:       make(chan PathCacheEvent), // The notified component will be send into this channel.
		addChildCh:     make(chan string),
		stopCh:         make(chan bool),
		owner:          owner,
		zkEventCounter: 0,
	}

	var err error
	if isSvcPath {
		err = p.watchAndAddChildren()
	} else {
		err = p.watchChildren()
	}

	if err != nil {
		klog.Errorf("Failed to watch zk path %s, %s", path, err)
		return nil, err
	}

	go func() {
		for {
			select {
			case child := <-p.addChildCh:
				p.onChildAdd(child)
			case event := <-p.watchCh:
				p.onEvent(&event, isSvcPath)
			case <-p.stopCh:
				close(p.notifyCh)
				return
			}
		}
	}()

	return p, nil
}

// Events ...
func (p *PathCache) Events() <-chan PathCacheEvent {
	return p.notifyCh
}

// Stop ...
func (p *PathCache) Stop() {
	go func() {
		p.stopCh <- true
	}()
}

// watch watch a specified path to make the node changed event can be handle by path cache.
func (p *PathCache) watch(path string) error {
	klog.Infof("[ ===== WATCHING ACTION ===== ] - GetW : path [%s]", path)

	_, stat, ch, err := p.conn.GetW(path)
	if err != nil {
		klog.Errorf("Getting and watching on path [%s] has an error: %v.", path, err)
		return err
	}
	klog.Infof("GetW path: [%s], stat: [%v]", path, stat)

	go p.forward(ch)
	return nil
}

// watchAndAddChildren
//
// 1.Watching this node's children
// 2.Forwarding the component which has been send by zookeeper
// 3.Watching every child's node via method GetW()
// 4.Caching every child so that we can receive the deletion event of this path later
func (p *PathCache) watchAndAddChildren() error {
	klog.Infof("[ ===== WATCHING ACTION ===== ] - ChildrenW : path [%s]", p.Path)
	children, stat, ch, err := p.conn.ChildrenW(p.Path)
	if err != nil {
		klog.Errorf("Watching on path [%s]'s children has an error: %v", p.Path, err)
		return err
	}
	klog.Infof("The children of the watched path [%s]，stat: [%v] size: %d:\n%v", p.Path, stat, len(children), children)

	// all of component was send from zookeeper will be forwarded into the channel of this path cache.
	go p.forward(ch)

	// caching every child into a map
	for _, child := range children {
		fp := path.Join(p.Path, child)
		if ok := p.Cached[fp]; !ok {
			go p.addChild(fp)
		}
	}

	metrics.PathCacheLengthGauge.With(prometheus.Labels{"path": p.Path}).Set(float64(len(p.Cached)))

	return nil
}

// watchChildren
func (p *PathCache) watchChildren() error {
	klog.Infof("[ ===== WATCHING ACTION ===== ] - ChildrenW : path [%s]", p.Path)
	children, stat, ch, err := p.conn.ChildrenW(p.Path)
	if err != nil {
		klog.Errorf("Watching on path [%s]'s children has an error: %v", p.Path, err)
		return err
	}
	klog.Infof("The children of the watched path [%s]，stat: [%v] size: %d:\n%v", p.Path, stat, len(children),
		children)
	for _, child := range children {
		klog.Infof("[SET CACHE] true pcaches[%s] %s", p.Path, child)
		p.Cached[child] = true
	}

	// all of component was send from zookeeper will be forwarded into the channel of this path cache.
	go p.forward(ch)

	event := PathCacheEvent{
		EventType: PathCacheEventChildrenReplaced,
		Paths:     children,
	}
	go p.notify(event)

	metrics.PathCacheLengthGauge.With(prometheus.Labels{"path": p.Path}).Set(float64(len(p.Cached)))

	return nil
}

// onChildAdd watch this added child, then inform the client immediately.
func (p *PathCache) onChildAdd(child string) {
	err := p.watch(child)
	if err != nil {
		klog.Errorf("Failed to watch child %s, error：%v", child, err)
		return
	}

	klog.Infof("[SET CACHE] true pcaches[%s] %s", p.Path, child)
	p.Cached[child] = true

	event := PathCacheEvent{
		EventType: PathCacheEventAdded,
		Path:      child,
	}
	go p.notify(event)
}

// onEvent Processing event from zookeeper.
func (p *PathCache) onEvent(event *zk.Event, isSvePath bool) {
	klog.Infof("[===== RECEIVED ZK ORIGINAL EVENT =====]: [%s]:[%d]:[%s]:[%v]:[%s]",
		p.owner, p.zkEventCounter, p.Path, event.Type, event.Path)

	switch event.Type {
	case zk.EventNodeDataChanged:
		p.onNodeChanged(event.Path)
	case zk.EventNodeChildrenChanged:
		if isSvePath {
			p.watchAndAddChildren()
		} else {
			p.watchChildren()
		}
	case zk.EventNodeDeleted:
		p.onChildDeleted(event.Path)
	// case zk.EventNotWatching:
	// // TODO reconnection or connect closed
	default:
		klog.Warningf("Event[%v]:[%s] has not been supported yet", event.Type, event.Path)
	}

	p.zkEventCounter++
}

// onChildDeleted
func (p *PathCache) onChildDeleted(child string) {
	klog.Infof("Received a deletion event from zookeeper: %s", child)

	// Remove the cache of this instance so that another instance which has same host name can be added.
	klog.Infof("[SET CACHE] false pcaches[%s] %s", p.Path, child)
	p.Cached[child] = false

	event := PathCacheEvent{
		EventType: PathCacheEventDeleted,
		Path:      child,
	}
	go p.notify(event)
}

// onNodeChanged
func (p *PathCache) onNodeChanged(path string) {
	klog.Infof("Received a node changed event from zookeeper: %s", path)
	event := PathCacheEvent{
		EventType: PathCacheEventChanged,
		Path:      path,
	}

	// We must watch this zNode again if we have received a data changed event through this channel.
	_, _, ch, err := p.conn.GetW(path)
	if err != nil {
		klog.Errorf("GetW path [%s] has an error: %v", path, err)
	} else {
		go p.forward(ch)
	}

	go p.notify(event)

}

func (p *PathCache) addChild(child string) {
	p.addChildCh <- child
}

func (p *PathCache) notify(event PathCacheEvent) {
	p.notifyCh <- event
}

func (p *PathCache) forward(eventCh <-chan zk.Event) {
	event, ok := <-eventCh
	if ok {
		p.watchCh <- event
	}
}

// Ignore ...
func Ignore(path string) bool {
	for _, v := range IgnoredHostNames {
		if strings.EqualFold(v, path) {
			return true
		}
	}
	return false
}
