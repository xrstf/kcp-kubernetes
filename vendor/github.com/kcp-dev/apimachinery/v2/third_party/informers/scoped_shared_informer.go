/*
Copyright 2022 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package informers

import (
	"time"

	"github.com/kcp-dev/logicalcluster/v3"

	"k8s.io/client-go/tools/cache"

	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
)

// scopedSharedIndexInformer ensures that event handlers added to the underlying
// informer are only called with objects matching the given logical cluster
type scopedSharedIndexInformer struct {
	*sharedIndexInformer
	clusterName logicalcluster.Name
}

// AddEventHandler adds an event handler to the shared informer using the shared informer's resync
// period.  Events to a single handler are delivered sequentially, but there is no coordination
// between different handlers.
func (s *scopedSharedIndexInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	return s.AddEventHandlerWithResyncPeriod(handler, s.sharedIndexInformer.defaultEventHandlerResyncPeriod)
}

// AddEventHandlerWithResyncPeriod adds an event handler to the
// shared informer with the requested resync period; zero means
// this handler does not care about resyncs.  The resync operation
// consists of delivering to the handler an update notification
// for every object in the informer's local cache; it does not add
// any interactions with the authoritative storage.  Some
// informers do no resyncs at all, not even for handlers added
// with a non-zero resyncPeriod.  For an informer that does
// resyncs, and for each handler that requests resyncs, that
// informer develops a nominal resync period that is no shorter
// than the requested period but may be longer.  The actual time
// between any two resyncs may be longer than the nominal period
// because the implementation takes time to do work and there may
// be competing load and scheduling noise.
func (s *scopedSharedIndexInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) (cache.ResourceEventHandlerRegistration, error) {
	scopedHandler := &filteringResourceEventHandler{
		handler:     handler,
		clusterName: s.clusterName,
	}

	return s.sharedIndexInformer.AddEventHandlerWithResyncPeriod(scopedHandler, resyncPeriod)
}

type filteringResourceEventHandler struct {
	handler     cache.ResourceEventHandler
	clusterName logicalcluster.Name
}

func (h *filteringResourceEventHandler) objectMatches(obj interface{}) bool {
	key, err := kcpcache.MetaClusterNamespaceKeyFunc(obj)
	if err != nil {
		return false
	}
	cluster, _, _, err := kcpcache.SplitMetaClusterNamespaceKey(key)
	if err != nil {
		return false
	}
	return cluster == h.clusterName
}

func (h *filteringResourceEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	if h.objectMatches(obj) {
		h.handler.OnAdd(obj, isInInitialList)
	}
}

func (h *filteringResourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	if h.objectMatches(newObj) {
		h.handler.OnUpdate(oldObj, newObj)
	}
}

func (h *filteringResourceEventHandler) OnDelete(obj interface{}) {
	if h.objectMatches(obj) {
		h.handler.OnDelete(obj)
	}
}
