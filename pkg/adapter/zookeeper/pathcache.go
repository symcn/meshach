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
	"github.com/samuel/go-zookeeper/zk"
	"k8s.io/klog"
	"path"
)

var (
	DubboRootPath    = "/dubbo"
	ProvidersPath    = "providers"
	IgnoredHostNames = []string{"metadata", "config"}
	ConfiguratorPath = DubboRootPath + "/config/dubbo"
)

type pathCacheEventType int

type pathCache struct {
	conn           *zk.Conn
	watchCh        chan zk.Event
	notifyCh       chan pathCacheEvent
	stopCh         chan bool
	addChildCh     chan string
	path           string
	cached         map[string]bool
	owner          string
	zkEventCounter int64
}

type pathCacheEvent struct {
	eventType pathCacheEventType
	path      string
}

const (
	pathCacheEventAdded pathCacheEventType = iota
	pathCacheEventDeleted
	pathCacheEventChanged
)

func newPathCache(conn *zk.Conn, path string, owner string) (*pathCache, error) {
	klog.Infof("=======> Create a cache for path: [%s]", path)

	p := &pathCache{
		conn:           conn,
		path:           path,
		cached:         make(map[string]bool),
		watchCh:        make(chan zk.Event),       // The zookeeper events will be forwarded to this channel.
		notifyCh:       make(chan pathCacheEvent), // The notified events will be send into this channel.
		addChildCh:     make(chan string),
		stopCh:         make(chan bool),
		owner:          owner,
		zkEventCounter: 0,
	}

	err := p.watchAndAddChildren()
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
				p.onEvent(&event)
			case <-p.stopCh:
				close(p.notifyCh)
				return
			}
		}
	}()

	return p, nil
}

func (p *pathCache) events() <-chan pathCacheEvent {
	return p.notifyCh
}

func (p *pathCache) stop() {
	go func() {
		p.stopCh <- true
	}()
}

// watch watch a specified path to make the node changed event can be handle by path cache.
func (p *pathCache) watch(path string) error {
	klog.Infof("[ ===== WATCHING ACTION ===== ] - GetW : path [%s]", path)

	_, _, ch, err := p.conn.GetW(path)
	if err != nil {
		klog.Errorf("Getting and watching on path [%s] has an error: %v.", path, err)
		return err
	}
	go p.forward(ch)
	return nil
}

// watchAndAddChildren
//
// 1.Watching this node's children
// 2.Forwarding the events which has been send by zookeeper
// 3.Watching every child's node via method GetW()
// 4.Caching every child so that we can receive the deletion event of this path later
func (p *pathCache) watchAndAddChildren() error {
	klog.Infof("[ ===== WATCHING ACTION ===== ] - ChildrenW : path [%s]", p.path)
	children, _, ch, err := p.conn.ChildrenW(p.path)
	if err != nil {
		klog.Errorf("Watching on path [%s]'s children has an error: %v", p.path, err)
		return err
	}
	klog.Infof("The children of the watched path [%s]， size: %d:\n%v", p.path, len(children),
		children)

	// all of events was send from zookeeper will be forwarded into the channel of this path cache.
	go p.forward(ch)

	// caching every child into a map
	for _, child := range children {
		fp := path.Join(p.path, child)
		if ok := p.cached[fp]; !ok {
			go p.addChild(fp)
		}
	}
	return nil
}

// onChildAdd watch this added child, then inform the client immediately.
func (p *pathCache) onChildAdd(child string) {
	err := p.watch(child)
	if err != nil {
		klog.Errorf("Failed to watch child %s, error：%v", child, err)
		return
	}

	klog.Infof("[SET CACHE] true pcaches[%s] %s", p.path, child)
	p.cached[child] = true

	event := pathCacheEvent{
		eventType: pathCacheEventAdded,
		path:      child,
	}
	go p.notify(event)
}

// onEvent Processing event from zookeeper.
func (p *pathCache) onEvent(event *zk.Event) {
	klog.Infof("[===== RECEIVED ZK ORIGINAL EVENT =====]: [%s]:[%d]:[%s]:[%v]:[%s]",
		p.owner, p.zkEventCounter, p.path, event.Type, event.Path)

	switch event.Type {
	case zk.EventNodeDataChanged:
		p.onNodeChanged(event.Path)
	case zk.EventNodeChildrenChanged:
		p.watchAndAddChildren()
	case zk.EventNodeDeleted:
		p.onChildDeleted(event.Path)
	default:
		klog.Warningf("Event[%v]:[%s] has not been supported yet", event.Type, event.Path)
	}

	p.zkEventCounter++
}

// onChildDeleted
func (p *pathCache) onChildDeleted(child string) {
	klog.Infof("Received a deletion event from zookeeper: %s", child)

	// Remove the cache of this instance so that another instance which has same host name can be added.
	klog.Infof("[SET CACHE] false pcaches[%s] %s", p.path, child)
	p.cached[child] = false

	vent := pathCacheEvent{
		eventType: pathCacheEventDeleted,
		path:      child,
	}
	go p.notify(vent)
}

// onNodeChanged
func (p *pathCache) onNodeChanged(path string) {
	klog.Infof("Received a node changed event from zookeeper: %s", path)
	vent := pathCacheEvent{
		eventType: pathCacheEventChanged,
		path:      path,
	}

	// We must watch this zNode again if we have received a data changed event through this channel.
	_, _, ch, err := p.conn.GetW(path)
	if err != nil {
		klog.Errorf("GetW path [%s] has an error: %v", path, err)
	} else {
		go p.forward(ch)
	}

	go p.notify(vent)

}

func (p *pathCache) addChild(child string) {
	p.addChildCh <- child
}

func (p *pathCache) notify(event pathCacheEvent) {
	p.notifyCh <- event
}

func (p *pathCache) forward(eventCh <-chan zk.Event) {
	event, ok := <-eventCh
	if ok {
		p.watchCh <- event
	}
}
