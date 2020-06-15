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
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"path"
)

var (
	DubboRootPath    = "/dubbo"
	ProvidersPath    = "providers"
	ConfiguratorPath = DubboRootPath + "/config/dubbo"
)

type pathCacheEventType int

type pathCache struct {
	conn       *zk.Conn
	watchCh    chan zk.Event
	notifyCh   chan pathCacheEvent
	stopCh     chan bool
	addChildCh chan string
	path       string
	cached     map[string]bool
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

func newPathCache(conn *zk.Conn, path string) (*pathCache, error) {
	fmt.Printf("Create a cache for path: %s\n", path)

	p := &pathCache{
		conn:       conn,
		path:       path,
		cached:     make(map[string]bool),
		watchCh:    make(chan zk.Event),       // The zookeeper events will be forwarded to this channel.
		notifyCh:   make(chan pathCacheEvent), // The notified events will be send into this channel.
		addChildCh: make(chan string),
		stopCh:     make(chan bool),
	}

	err := p.watchChildren()
	if err != nil {
		fmt.Printf("Failed to watch zk path %s, %s\n", path, err)
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
	fmt.Printf("Watch path[%s]\n", path)

	_, _, ch, err := p.conn.GetW(path)
	if err != nil {
		fmt.Printf("Watching node[%s] has an error: %v.\n", path, err)
		return err
	}
	go p.forward(ch)
	return nil
}

// watchChildren
// 1.watch the node's children
// 2.forward the events which has been send by zookeeper
// 3.cache every child
func (p *pathCache) watchChildren() error {
	fmt.Printf("Watch the children for path[%s]\n", p.path)

	children, _, ch, err := p.conn.ChildrenW(p.path)
	if err != nil {
		fmt.Printf("Watching node[%s]'s children has an error: %v.\n", p.path, err)
		return err
	}
	fmt.Printf("The children of the watched path[%s]:\n%v\n", p.path, children)

	// all of events was send from zookeeper will be forwarded into the channel of this path cache.
	go p.forward(ch)

	// cache every child into a map
	for _, child := range children {
		fp := path.Join(p.path, child)
		if ok := p.cached[fp]; !ok {
			go p.addChild(fp)
		}
	}
	return nil
}

// onChildAdd watch this added child, and inform the client immediately.
func (p *pathCache) onChildAdd(child string) {
	err := p.watch(child)
	if err != nil {
		fmt.Printf("Failed to watch child %s, error：%s\n", child, err)
		return
	}

	p.cached[child] = true

	event := pathCacheEvent{
		eventType: pathCacheEventAdded,
		path:      child,
	}
	go p.notify(event)
}

// onEvent
func (p *pathCache) onEvent(event *zk.Event) {
	switch event.Type {
	case zk.EventNodeChildrenChanged:
		p.watchChildren()
	case zk.EventNodeDeleted:
		p.onChildDeleted(event.Path)
	case zk.EventNodeDataChanged:
		p.onNodeChanged(event.Path)
	default:
		fmt.Printf("Event[%v] has not been supported yet", event)
	}
}

// onChildDeleted
func (p *pathCache) onChildDeleted(child string) {
	fmt.Printf("Received a deletion event by zookeeper: %v\n", child)

	vent := pathCacheEvent{
		eventType: pathCacheEventDeleted,
		path:      child,
	}
	go p.notify(vent)
}

// onNodeChanged
func (p *pathCache) onNodeChanged(path string) {
	fmt.Printf("Received a node changed event by zookeeper: %v\n", path)
	vent := pathCacheEvent{
		eventType: pathCacheEventChanged,
		path:      path,
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
