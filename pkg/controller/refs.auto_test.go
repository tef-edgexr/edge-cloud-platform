// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: refs.proto

package controller

import (
	fmt "fmt"
	"github.com/edgexr/edge-cloud-platform/api/edgeproto"
	"github.com/edgexr/edge-cloud-platform/pkg/objstore"
	_ "github.com/edgexr/edge-cloud-platform/tools/protogen"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	"go.etcd.io/etcd/client/v3/concurrency"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Auto-generated code: DO NOT EDIT

// CloudletRefsStoreTracker wraps around the usual
// store to track the STM used for gets/puts.
type CloudletRefsStoreTracker struct {
	edgeproto.CloudletRefsStore
	getSTM concurrency.STM
	putSTM concurrency.STM
}

// Wrap the Api's store with a tracker store.
// Returns the tracker store, and the unwrap function to defer.
func wrapCloudletRefsTrackerStore(api *CloudletRefsApi) (*CloudletRefsStoreTracker, func()) {
	orig := api.store
	tracker := &CloudletRefsStoreTracker{
		CloudletRefsStore: api.store,
	}
	api.store = tracker
	if api.cache.Store != nil {
		api.cache.Store = tracker
	}
	unwrap := func() {
		api.store = orig
		if api.cache.Store != nil {
			api.cache.Store = orig
		}
	}
	return tracker, unwrap
}

func (s *CloudletRefsStoreTracker) STMGet(stm concurrency.STM, key *edgeproto.CloudletKey, buf *edgeproto.CloudletRefs) bool {
	found := s.CloudletRefsStore.STMGet(stm, key, buf)
	if s.getSTM == nil {
		s.getSTM = stm
	}
	return found
}

func (s *CloudletRefsStoreTracker) STMPut(stm concurrency.STM, obj *edgeproto.CloudletRefs, ops ...objstore.KVOp) {
	s.CloudletRefsStore.STMPut(stm, obj, ops...)
	if s.putSTM == nil {
		s.putSTM = stm
	}
}

// ClusterRefsStoreTracker wraps around the usual
// store to track the STM used for gets/puts.
type ClusterRefsStoreTracker struct {
	edgeproto.ClusterRefsStore
	getSTM concurrency.STM
	putSTM concurrency.STM
}

// Wrap the Api's store with a tracker store.
// Returns the tracker store, and the unwrap function to defer.
func wrapClusterRefsTrackerStore(api *ClusterRefsApi) (*ClusterRefsStoreTracker, func()) {
	orig := api.store
	tracker := &ClusterRefsStoreTracker{
		ClusterRefsStore: api.store,
	}
	api.store = tracker
	if api.cache.Store != nil {
		api.cache.Store = tracker
	}
	unwrap := func() {
		api.store = orig
		if api.cache.Store != nil {
			api.cache.Store = orig
		}
	}
	return tracker, unwrap
}

func (s *ClusterRefsStoreTracker) STMGet(stm concurrency.STM, key *edgeproto.ClusterKey, buf *edgeproto.ClusterRefs) bool {
	found := s.ClusterRefsStore.STMGet(stm, key, buf)
	if s.getSTM == nil {
		s.getSTM = stm
	}
	return found
}

func (s *ClusterRefsStoreTracker) STMPut(stm concurrency.STM, obj *edgeproto.ClusterRefs, ops ...objstore.KVOp) {
	s.ClusterRefsStore.STMPut(stm, obj, ops...)
	if s.putSTM == nil {
		s.putSTM = stm
	}
}

// AppInstRefsStoreTracker wraps around the usual
// store to track the STM used for gets/puts.
type AppInstRefsStoreTracker struct {
	edgeproto.AppInstRefsStore
	getSTM concurrency.STM
	putSTM concurrency.STM
}

// Wrap the Api's store with a tracker store.
// Returns the tracker store, and the unwrap function to defer.
func wrapAppInstRefsTrackerStore(api *AppInstRefsApi) (*AppInstRefsStoreTracker, func()) {
	orig := api.store
	tracker := &AppInstRefsStoreTracker{
		AppInstRefsStore: api.store,
	}
	api.store = tracker
	if api.cache.Store != nil {
		api.cache.Store = tracker
	}
	unwrap := func() {
		api.store = orig
		if api.cache.Store != nil {
			api.cache.Store = orig
		}
	}
	return tracker, unwrap
}

func (s *AppInstRefsStoreTracker) STMGet(stm concurrency.STM, key *edgeproto.AppKey, buf *edgeproto.AppInstRefs) bool {
	found := s.AppInstRefsStore.STMGet(stm, key, buf)
	if s.getSTM == nil {
		s.getSTM = stm
	}
	return found
}

func (s *AppInstRefsStoreTracker) STMPut(stm concurrency.STM, obj *edgeproto.AppInstRefs, ops ...objstore.KVOp) {
	s.AppInstRefsStore.STMPut(stm, obj, ops...)
	if s.putSTM == nil {
		s.putSTM = stm
	}
}
