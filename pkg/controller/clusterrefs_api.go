// Copyright 2022 MobiledgeX, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"github.com/edgexr/edge-cloud-platform/api/edgeproto"
	"github.com/edgexr/edge-cloud-platform/pkg/regiondata"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type ClusterRefsApi struct {
	all   *AllApis
	sync  *regiondata.Sync
	store edgeproto.ClusterRefsStore
	cache edgeproto.ClusterRefsCache
}

func NewClusterRefsApi(sync *regiondata.Sync, all *AllApis) *ClusterRefsApi {
	clusterRefsApi := ClusterRefsApi{}
	clusterRefsApi.all = all
	clusterRefsApi.sync = sync
	clusterRefsApi.store = edgeproto.NewClusterRefsStore(sync.GetKVStore())
	edgeproto.InitClusterRefsCacheWithStore(&clusterRefsApi.cache, clusterRefsApi.store)
	sync.RegisterCache(&clusterRefsApi.cache)
	return &clusterRefsApi
}

func (s *ClusterRefsApi) ShowClusterRefs(in *edgeproto.ClusterRefs, cb edgeproto.ClusterRefsApi_ShowClusterRefsServer) error {
	err := s.cache.Show(in, func(obj *edgeproto.ClusterRefs) error {
		err := cb.Send(obj)
		return err
	})
	return err
}

func (s *ClusterRefsApi) deleteRef(stm concurrency.STM, key *edgeproto.ClusterKey) {
	s.store.STMDel(stm, key)
}

func (s *ClusterRefsApi) addRef(stm concurrency.STM, appInst *edgeproto.AppInst) {
	key := appInst.GetClusterKey()
	refs := edgeproto.ClusterRefs{}
	if !s.store.STMGet(stm, key, &refs) {
		refs.Key = *key
	}
	// if creating again to override create error, may already
	// exist
	for _, aikey := range refs.Apps {
		if aikey.Matches(&appInst.Key) {
			return
		}
	}
	refs.Apps = append(refs.Apps, appInst.Key)
	s.store.STMPut(stm, &refs)
}

func (s *ClusterRefsApi) removeRef(stm concurrency.STM, appInst *edgeproto.AppInst) {
	key := appInst.GetClusterKey()
	refs := edgeproto.ClusterRefs{}
	if !s.store.STMGet(stm, key, &refs) {
		return
	}
	changed := false
	refKey := appInst.Key
	for ii := range refs.Apps {
		if refKey.Matches(&refs.Apps[ii]) {
			refs.Apps = append(refs.Apps[:ii], refs.Apps[ii+1:]...)
			changed = true
			break
		}
	}
	if !changed {
		return
	}
	if len(refs.Apps) == 0 {
		s.store.STMDel(stm, key)
	} else {
		s.store.STMPut(stm, &refs)
	}
}

func (s *ClusterRefsApi) canReleaseReservation(stm concurrency.STM, appInst *edgeproto.AppInst) bool {
	// For a reservable clusterinst, we check that if the specified
	// appInst is removed, if there are any other AppInsts owned
	// by the same Organization in the cluster. If not, the cluster
	// can free the reservation, otherwise we need to keep the
	// reservation.
	key := appInst.GetClusterKey()
	refs := edgeproto.ClusterRefs{}
	if !s.store.STMGet(stm, key, &refs) {
		return true
	}
	for _, aikey := range refs.Apps {
		if appInst.Key.Matches(&aikey) {
			continue
		}
		if aikey.Organization == appInst.Key.Organization {
			return false
		}
	}
	return true
}
