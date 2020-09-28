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

	"github.com/go-zookeeper/zk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/symcn/mesh-operator/pkg/adapter/metrics"
	"github.com/symcn/mesh-operator/pkg/utils"
	"k8s.io/klog"
)

// Constants for ACL permissions
const (
	PermRead = 1 << iota
	PermWrite
	PermCreate
	PermDelete
	PermAdmin
	PermAll = 0x1f
)

// Some default settings.
var (
	DubboRootPath    = "/dubbo"
	ProvidersPath    = "providers"
	ConsumersPath    = "consumers"
	IgnoredHostNames = []string{"metadata", "config"}
	ConfiguratorPath = DubboRootPath + "/config/dubbo"
	workPool         utils.WorkerPool
)

func init() {
	workPool = utils.NewWorkerPool(10000)
}

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
	autoFillNode   bool // You can specify that this path cache fill a node automatically due to the absence of this node.
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

// NewPathCache we need a cache manager for a given path,
// In some scenario it probably creates plenty of PathCaches by a registry client.
func NewPathCache(conn *zk.Conn, path string, owner string, isSvcPath bool, autoFillNode bool) (*PathCache, error) {
	klog.V(6).Infof("Creating a cache for a given path: [%s]", path)

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
		autoFillNode:   autoFillNode,
	}

	// trying to create this given node firstly to avoid to watch a non-exist node.
	if p.autoFillNode {
		create(p.conn, p.Path)
	}

	var err error
	if isSvcPath {
		err = p.watchAndAddChildren()
	} else {
		err = p.watchChildren()
	}

	if err != nil {
		return nil, err
	}

	workPool.ScheduleAuto(func() {
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
	})

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

// watch watching a specified path to make the changed event can be handled by path cache.
func (p *PathCache) watch(path string) error {
	klog.V(6).Infof("[ ===== WATCHING ACTION ===== ] - GetW : path [%s]", path)

	_, _, ch, err := p.conn.GetW(path)
	if err != nil {
		klog.Errorf("Getting and watching on path [%s] has an error: %v.", path, err)
		return err
	}

	workPool.ScheduleAuto(func() { p.forward(ch) })
	return nil
}

// watchAndAddChildren The purposes of this method in fourfold:
// 1.Watching this node's children
// 2.Forwarding the event which has been send by zookeeper
// 3.Watching every child's node via method GetW()
// 4.Caching every child so that we can receive the deletion event of this path later
func (p *PathCache) watchAndAddChildren() error {
	klog.V(6).Infof("[ ===== WATCHING ACTION ===== ] - ChildrenW : path [%s]", p.Path)
	cch, _, ch, err := p.conn.ChildrenW(p.Path)
	if err != nil {
		klog.Errorf("Watching on path [%s]'s children has an error: %v", p.Path, err)
		return err
	}
	children := deepCopySlice(cch)

	// klog.V(6).Infof("The children of the watched path [%s]，stat: [%v] size: %d:\n%v", p.Path, stat, len(children), children)

	// all of event was send by zookeeper will be forwarded into the inner channel of this path cache.
	workPool.ScheduleAuto(func() { p.forward(ch) })

	// caching every child into a map
	for _, child := range children {
		fp := path.Join(p.Path, child)
		if ok := p.Cached[fp]; !ok {
			workPool.ScheduleAuto(func() { p.addChild(fp) })
		}
	}

	metrics.PathCacheLengthGauge.With(prometheus.Labels{"path": p.Path}).Set(float64(len(p.Cached)))

	return nil
}

// watchChildren Watching the children's changing of this path, meanwhile fetching the children of this path
func (p *PathCache) watchChildren() error {
	klog.V(6).Infof("[ ===== WATCHING ACTION ===== ] - ChildrenW : path [%s]", p.Path)
	cch, _, ch, err := p.conn.ChildrenW(p.Path)
	if err != nil {
		klog.Errorf("Watching on path [%s]'s children has an error: %v", p.Path, err)
		return err
	}
	children := deepCopySlice(cch)

	// klog.V(6).Infof("The children of the watched path [%s]，stat: [%v] size: %d", p.Path, stat, len(children))
	cached := make(map[string]bool, len(children))
	for _, child := range children {
		klog.V(6).Infof("[SET CACHE] true pcaches[%s] %s", p.Path, child)
		cached[child] = true
	}
	p.Cached = cached

	// all of component was send from zookeeper will be forwarded into the channel of this path cache.
	workPool.ScheduleAuto(func() { p.forward(ch) })

	event := PathCacheEvent{
		EventType: PathCacheEventChildrenReplaced,
		Paths:     children,
	}
	workPool.ScheduleAuto(func() { p.notify(event) })

	metrics.PathCacheLengthGauge.With(prometheus.Labels{"path": p.Path}).Set(float64(len(p.Cached)))

	return nil
}

// onChildAdd watch this added child, then it will inform the client immediately.
func (p *PathCache) onChildAdd(child string) {
	err := p.watch(child)
	if err != nil {
		klog.Errorf("Failed to watch child %s, error：%v", child, err)
		return
	}

	klog.V(6).Infof("[SET CACHE] true pcaches[%s] %s", p.Path, child)
	p.Cached[child] = true

	event := PathCacheEvent{
		EventType: PathCacheEventAdded,
		Path:      child,
	}
	workPool.ScheduleAuto(func() { p.notify(event) })
}

// onEvent Processing events come from zookeeper.
func (p *PathCache) onEvent(event *zk.Event, isSvePath bool) {
	klog.V(6).Infof("[===== RECEIVED ZK ORIGINAL EVENT =====]: [%s]:[%d]:[%s]:[%v]:[%s]",
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
	klog.V(6).Infof("Received a deletion event from zookeeper: %s", child)

	// Remove the cache of this instance so that another instance which has same host name can be added.
	klog.V(6).Infof("[SET CACHE] false pcaches[%s] %s", p.Path, child)
	p.Cached[child] = false

	event := PathCacheEvent{
		EventType: PathCacheEventDeleted,
		Path:      child,
	}
	workPool.ScheduleAuto(func() { p.notify(event) })
}

// onNodeChanged
func (p *PathCache) onNodeChanged(path string) {
	klog.V(6).Infof("Received a node changed event from zookeeper: %s", path)
	event := PathCacheEvent{
		EventType: PathCacheEventChanged,
		Path:      path,
	}

	// We must watch this zNode again if we have received a data changed event through this channel.
	_, _, ch, err := p.conn.GetW(path)
	if err != nil {
		klog.Errorf("GetW path [%s] has an error: %v", path, err)
	} else {
		workPool.ScheduleAuto(func() { p.forward(ch) })
	}

	workPool.ScheduleAuto(func() { p.notify(event) })

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

func deepCopySlice(src []string) (dst []string) {
	if len(src) < 1 {
		return nil
	}
	dst = make([]string, len(src))

	for i := 0; i < len(src); i++ {
		dst[i] = string(append([]byte(nil), []byte(src[i])...))
	}
	return dst
}

// WorldACL produces an ACL list containing a single ACL which uses the
// provided permissions, with the scheme "world", and ID "anyone", which
// is used by ZooKeeper to represent any user at all.
func worldACL(perms int32) []zk.ACL {
	return []zk.ACL{{perms, "world", "anyone"}}
}

// createRecursivelyIfNecessary ...
func createRecursivelyIfNecessary(conn *zk.Conn, path string) {
	// Strip trailing slashes.
	for len(path) > 0 && path[len(path)-1] == '/' {
		path = path[0 : len(path)-1]
	}

	elements := strings.Split(path, "/")
	if len(elements) == 0 {
		return
	}

	var ops []interface{}
	var pathPrefix string
	for i, e := range elements {
		if i == 0 {
			continue
		}

		ops = append(ops, &zk.CreateRequest{
			Path: pathPrefix + "/" + e,
			Data: nil,
			Acl:  worldACL(PermAll),
		})
		pathPrefix = pathPrefix + "/" + e
	}

	if res, err := conn.Multi(ops...); err != nil {
		klog.Errorf("Create the path recursively has an error: %v", err)
	} else if len(res) != len(elements)-1 {
		klog.Warningf("Just only a part of paths has been created: %s", path)
	}

	klog.V(6).Infof("Created the path %s recursively", path)
}

// create ...
func create(conn *zk.Conn, path string) {
	if _, err := conn.Create(path, nil, 0, worldACL(PermAll)); err != nil {
		klog.Errorf("create path %s has an error:%v", path, err)
	}
	klog.V(6).Infof("Created the path %s", path)
}
