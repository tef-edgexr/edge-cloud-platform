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
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	dme "github.com/edgexr/edge-cloud-platform/api/distributed_match_engine"
	"github.com/edgexr/edge-cloud-platform/api/edgeproto"
	"github.com/edgexr/edge-cloud-platform/pkg/cloudcommon"
	"github.com/edgexr/edge-cloud-platform/pkg/log"
	"github.com/edgexr/edge-cloud-platform/pkg/regiondata"
	"github.com/edgexr/edge-cloud-platform/pkg/resspec"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type CloudletInfoApi struct {
	all   *AllApis
	sync  *regiondata.Sync
	store edgeproto.CloudletInfoStore
	cache edgeproto.CloudletInfoCache
}

var cleanupCloudletInfoTimeout = 5 * time.Minute

func NewCloudletInfoApi(sync *regiondata.Sync, all *AllApis) *CloudletInfoApi {
	cloudletInfoApi := CloudletInfoApi{}
	cloudletInfoApi.all = all
	cloudletInfoApi.sync = sync
	cloudletInfoApi.store = edgeproto.NewCloudletInfoStore(sync.GetKVStore())
	edgeproto.InitCloudletInfoCacheWithStore(&cloudletInfoApi.cache, cloudletInfoApi.store)
	sync.RegisterCache(&cloudletInfoApi.cache)
	return &cloudletInfoApi
}

// We put CloudletInfo in etcd with a lease, so in case both controller
// and CRM suddenly go away, etcd will remove the stale CloudletInfo data.

func (s *CloudletInfoApi) InjectCloudletInfo(ctx context.Context, in *edgeproto.CloudletInfo) (*edgeproto.Result, error) {
	return s.store.Put(ctx, in, s.sync.SyncWait)
}

func (s *CloudletInfoApi) EvictCloudletInfo(ctx context.Context, in *edgeproto.CloudletInfo) (*edgeproto.Result, error) {
	return s.store.Delete(ctx, in, s.sync.SyncWait)
}

func (s *CloudletInfoApi) ShowCloudletInfo(in *edgeproto.CloudletInfo, cb edgeproto.CloudletInfoApi_ShowCloudletInfoServer) error {
	err := s.cache.Show(in, func(obj *edgeproto.CloudletInfo) error {
		err := cb.Send(obj)
		return err
	})
	return err
}

func (s *CloudletInfoApi) UpdateRPC(ctx context.Context, in *edgeproto.CloudletInfo) {
	// CCRM handles Cloudlet Create/Delete calls via GRPC,
	// even though CRM is the source of truth of the CloudletInfo
	// for CRM on edge.
	// So make sure to only update the specified fields.
	s.UpdateFields(ctx, in, 0)
}

func (s *CloudletInfoApi) Update(ctx context.Context, in *edgeproto.CloudletInfo, rev int64) {
	// CRM notify is the owner so it updates all fields
	// This path should never get called for CrmOnEdge=false.
	in.Fields = edgeproto.CloudletInfoAllFields
	s.UpdateFields(ctx, in, rev)
}

func (s *CloudletInfoApi) UpdateFields(ctx context.Context, in *edgeproto.CloudletInfo, rev int64) {
	var err error
	log.SpanLog(ctx, log.DebugLevelNotify, "Cloudlet Info Update", "in", in)
	// for now assume all fields have been specified
	if in.StandbyCrm {
		log.SpanLog(ctx, log.DebugLevelNotify, "skipping due to info from standby CRM")
		return
	}
	fmap := edgeproto.MakeFieldMap(in.Fields)

	inCopy := *in // save Status to publish to Redis
	in.Controller = ControllerId
	changedToOnline := false
	info := edgeproto.CloudletInfo{}
	s.sync.ApplySTMWait(ctx, func(stm concurrency.STM) error {
		info = edgeproto.CloudletInfo{
			Key: in.Key,
		}
		oldState := dme.CloudletState_CLOUDLET_STATE_NOT_PRESENT
		if s.store.STMGet(stm, &in.Key, &info) {
			oldState = info.State
		}
		if fmap.Has(edgeproto.CloudletInfoFieldState) {
			if in.State == dme.CloudletState_CLOUDLET_STATE_READY &&
				oldState != dme.CloudletState_CLOUDLET_STATE_READY {
				changedToOnline = true
			}
		}
		// Clear fields that are cached in Redis as they should not be stored in DB
		in.ClearRedisOnlyFields()
		info.ClearRedisOnlyFields()

		// For federated cloudlets, flavors come from MC using
		// InjectCloudletInfo. Don't let FRM overwrite them.
		if fmap.Has(edgeproto.CloudletInfoFieldKeyFederatedOrganization) && in.Key.FederatedOrganization != "" {
			in.Flavors = info.Flavors
		}

		diffFields := info.GetDiffFields(in)
		if diffFields.Count() > 0 {
			info.CopyInFields(in)
			log.SpanLog(ctx, log.DebugLevelApi, "CloudletInfo updating fields", "fields", diffFields.Fields())
			s.store.STMPut(stm, &info)
		}
		return nil
	})

	// info is the updated state
	in = &info

	// publish the received info object on redis
	// must be done after updating etcd, see AppInst UpdateFromInfo comment
	s.all.streamObjApi.UpdateStatus(ctx, inCopy, nil, &in.State, in.Key.StreamKey())

	cloudlet := edgeproto.Cloudlet{}
	if !s.all.cloudletApi.cache.Get(&in.Key, &cloudlet) {
		return
	}
	features, err := s.all.platformFeaturesApi.GetCloudletFeatures(ctx, cloudlet.PlatformType)
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelApi, "unable to get features for cloudlet", "cloudlet", in.Key, "err", err)
	}
	if changedToOnline {
		log.SpanLog(ctx, log.DebugLevelApi, "cloudlet online", "cloudlet", in.Key)
		nodeMgr.Event(ctx, "Cloudlet online", in.Key.Organization, in.Key.GetTags(), nil, "state", in.State.String(), "version", in.ContainerVersion)
		if features != nil {
			if features.SupportsMultiTenantCluster && cloudlet.EnableDefaultServerlessCluster {
				s.all.cloudletApi.defaultMTClustWorkers.NeedsWork(ctx, in.Key)
			}
		}
	}
	if fmap.HasOrHasChild(edgeproto.CloudletInfoFieldNodePools) && features != nil && features.IsSingleKubernetesCluster {
		// copy node pool resource info to single cluster
		s.all.clusterInstApi.updateCloudletSingleClusterResources(ctx, &cloudlet.Key, cloudlet.SingleKubernetesClusterOwner, in.NodePools, in.Properties)
	}

	newState := edgeproto.TrackedState_TRACKED_STATE_UNKNOWN
	switch in.State {
	case dme.CloudletState_CLOUDLET_STATE_INIT:
		newState = edgeproto.TrackedState_CRM_INITOK
		if in.ContainerVersion != cloudlet.ContainerVersion {
			nodeMgr.Event(ctx, "Upgrading cloudlet", in.Key.Organization, in.Key.GetTags(), nil, "from-version", cloudlet.ContainerVersion, "to-version", in.ContainerVersion)
		}
	case dme.CloudletState_CLOUDLET_STATE_READY:
		newState = edgeproto.TrackedState_READY
	case dme.CloudletState_CLOUDLET_STATE_UPGRADE:
		newState = edgeproto.TrackedState_UPDATING
	case dme.CloudletState_CLOUDLET_STATE_ERRORS:
		if cloudlet.State == edgeproto.TrackedState_UPDATE_REQUESTED ||
			cloudlet.State == edgeproto.TrackedState_UPDATING {
			newState = edgeproto.TrackedState_UPDATE_ERROR
		} else if cloudlet.State == edgeproto.TrackedState_CREATE_REQUESTED ||
			cloudlet.State == edgeproto.TrackedState_CREATING {
			newState = edgeproto.TrackedState_CREATE_ERROR
		}
	default:
		log.SpanLog(ctx, log.DebugLevelNotify, "Skip cloudletInfo state handling", "key", in.Key, "state", in.State)
		return
	}

	newCloudlet := edgeproto.Cloudlet{}
	key := &in.Key
	err = s.all.cloudletApi.sync.ApplySTMWait(ctx, func(stm concurrency.STM) error {
		updateObj := false
		if !s.all.cloudletApi.store.STMGet(stm, key, &newCloudlet) {
			return key.NotFoundError()
		}
		if newCloudlet.TrustPolicyState != in.TrustPolicyState && in.TrustPolicyState != edgeproto.TrackedState_TRACKED_STATE_UNKNOWN {
			newCloudlet.TrustPolicyState = in.TrustPolicyState
			updateObj = true
		}
		if newCloudlet.State != newState {
			newCloudlet.State = newState
			if in.Errors != nil {
				newCloudlet.Errors = in.Errors
			}
			if in.State == dme.CloudletState_CLOUDLET_STATE_READY {
				newCloudlet.Errors = nil
			}
			updateObj = true
		}
		if newCloudlet.ContainerVersion != in.ContainerVersion {
			newCloudlet.ContainerVersion = in.ContainerVersion
			updateObj = true
		}
		if newCloudlet.Key.FederatedOrganization == "" && fmap.HasOrHasChild(edgeproto.CloudletInfoFieldFlavors) {
			// Copy CloudletInfo.Flavors to Cloudlet.InfraFlavors so
			// that developers can see and use cloudlet-specific
			// flavors. Don't do this for federated cloudlets
			// because flavors are injected manually during
			// zone registration, and we don't want the FRM to
			// overwrite them.
			// TODO: compare each flavor to see if update is needed
			newCloudlet.InfraFlavors = in.Flavors
			updateObj = true
		}
		if updateObj {
			s.all.cloudletApi.store.STMPut(stm, &newCloudlet)
		}
		return nil
	})
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelNotify, "CloudletInfo state transition error",
			"key", in.Key, "err", err)
	}
	if changedToOnline {
		s.ClearCloudletAndAppInstDownAlerts(ctx, in)
	}
	if in.State == dme.CloudletState_CLOUDLET_STATE_READY {
		// Validate cloudlet resources and generate appropriate warnings,
		// use cache instead of store.
		cacheSTM := edgeproto.NewOptionalSTM(nil)
		resCalc := NewCloudletResCalc(s.all, cacheSTM, key)
		resCalc.deps.cloudletInfo = in
		warnings, err := resCalc.cloudletFitsReqdVals(ctx, resspec.ResValMap{})
		if err != nil {
			log.SpanLog(ctx, log.DebugLevelNotify, "Failed to validate cloudlet resources",
				"key", in.Key, "err", err)
		} else {
			_ = s.sync.ApplySTMWait(ctx, func(stm concurrency.STM) error {
				s.all.clusterInstApi.handleResourceUsageAlerts(ctx, stm, &cloudlet.Key, warnings)
				return nil
			})
		}
	}
}

func (s *CloudletInfoApi) Del(ctx context.Context, key *edgeproto.CloudletKey, wait func(int64)) {
	in := edgeproto.CloudletInfo{Key: *key}
	s.store.Delete(ctx, &in, wait)
}

func cloudletInfoToAlertLabels(in *edgeproto.CloudletInfo) map[string]string {
	labels := make(map[string]string)
	// Set tags that match cloudlet
	labels["alertname"] = cloudcommon.AlertCloudletDown
	labels[cloudcommon.AlertScopeTypeTag] = cloudcommon.AlertScopeCloudlet
	labels[edgeproto.CloudletKeyTagName] = in.Key.Name
	labels[edgeproto.CloudletKeyTagOrganization] = in.Key.Organization
	return labels
}

func cloudletDownAppInstAlertLabels(appInstKey *edgeproto.AppInstKey) map[string]string {
	labels := appInstKey.GetTags()
	labels["alertname"] = cloudcommon.AlertAppInstDown
	labels[cloudcommon.AlertHealthCheckStatus] = dme.HealthCheck_CamelName[int32(dme.HealthCheck_HEALTH_CHECK_CLOUDLET_OFFLINE)]
	labels[cloudcommon.AlertScopeTypeTag] = cloudcommon.AlertScopeApp
	return labels
}

// Raise the alarm when the cloudlet goes down
func (s *CloudletInfoApi) fireCloudletDownAlert(ctx context.Context, in *edgeproto.CloudletInfo) {
	alert := edgeproto.Alert{}
	alert.State = "firing"
	alert.ActiveAt = dme.Timestamp{}
	ts := time.Now()
	alert.ActiveAt.Seconds = ts.Unix()
	alert.ActiveAt.Nanos = int32(ts.Nanosecond())
	alert.Labels = cloudletInfoToAlertLabels(in)
	alert.Annotations = make(map[string]string)
	alert.Annotations[cloudcommon.AlertAnnotationTitle] = cloudcommon.AlertCloudletDown
	alert.Annotations[cloudcommon.AlertAnnotationDescription] = cloudcommon.AlertCloudletDownDescription
	// send an update
	s.all.alertApi.Update(ctx, &alert, 0)
}

func (s *CloudletInfoApi) FireCloudletAndAppInstDownAlerts(ctx context.Context, in *edgeproto.CloudletInfo) {
	s.fireCloudletDownAlert(ctx, in)
	s.fireCloudletDownAppInstAlerts(ctx, in)
}

func (s *CloudletInfoApi) ClearCloudletAndAppInstDownAlerts(ctx context.Context, in *edgeproto.CloudletInfo) {
	// We ignore the controller and notifyId check when cleaning up the alerts here
	ctx = context.WithValue(ctx, ControllerCreatedAlerts, &ControllerCreatedAlerts)
	s.clearCloudletDownAlert(ctx, in)
	s.clearCloudletDownAppInstAlerts(ctx, in)
}

func (s *CloudletInfoApi) clearCloudletDownAlert(ctx context.Context, in *edgeproto.CloudletInfo) {
	alert := edgeproto.Alert{}
	alert.Labels = cloudletInfoToAlertLabels(in)
	s.all.alertApi.Delete(ctx, &alert, 0)
}

func (s *CloudletInfoApi) clearCloudletDownAppInstAlerts(ctx context.Context, in *edgeproto.CloudletInfo) {
	appInstFilter := edgeproto.AppInst{
		CloudletKey: in.Key,
	}
	appInstKeys := make([]edgeproto.AppInstKey, 0)
	s.all.appInstApi.cache.Show(&appInstFilter, func(obj *edgeproto.AppInst) error {
		appInstKeys = append(appInstKeys, obj.Key)
		return nil
	})
	for _, k := range appInstKeys {
		alert := edgeproto.Alert{}
		alert.Labels = cloudletDownAppInstAlertLabels(&k)
		s.all.alertApi.Delete(ctx, &alert, 0)
	}
}

func (s *CloudletInfoApi) fireCloudletDownAppInstAlerts(ctx context.Context, in *edgeproto.CloudletInfo) {
	// exclude SideCar apps which are auto-deployed as part of the cluster
	excludedAppFilter := cloudcommon.GetSideCarAppFilter()
	excludedAppKeys := make(map[edgeproto.AppKey]bool, 0)
	s.all.appApi.cache.Show(excludedAppFilter, func(obj *edgeproto.App) error {
		excludedAppKeys[obj.Key] = true
		return nil
	})

	appInstFilter := edgeproto.AppInst{
		CloudletKey: in.Key,
	}
	appInstKeys := make([]edgeproto.AppInstKey, 0)
	s.all.appInstApi.cache.Show(&appInstFilter, func(obj *edgeproto.AppInst) error {
		if excluded := excludedAppKeys[obj.AppKey]; excluded {
			return nil
		}
		appInstKeys = append(appInstKeys, obj.Key)
		return nil
	})
	for _, k := range appInstKeys {
		alert := edgeproto.Alert{}
		alert.State = "firing"
		alert.ActiveAt = dme.Timestamp{}
		ts := time.Now()
		alert.ActiveAt.Seconds = ts.Unix()
		alert.ActiveAt.Nanos = int32(ts.Nanosecond())
		alert.Labels = cloudletDownAppInstAlertLabels(&k)
		alert.Annotations = make(map[string]string)
		alert.Annotations[cloudcommon.AlertAnnotationTitle] = cloudcommon.AlertAppInstDown
		alert.Annotations[cloudcommon.AlertAnnotationDescription] = "AppInst down due to cloudlet offline"
		s.all.alertApi.Update(ctx, &alert, 0)
	}
}

// Delete from notify just marks the cloudlet offline
func (s *CloudletInfoApi) Delete(ctx context.Context, in *edgeproto.CloudletInfo, rev int64) {
	buf := edgeproto.CloudletInfo{}
	err := s.sync.ApplySTMWait(ctx, func(stm concurrency.STM) error {
		if !s.store.STMGet(stm, &in.Key, &buf) {
			return nil
		}
		if buf.NotifyId != in.NotifyId || buf.Controller != ControllerId {
			// updated by another thread or controller
			return nil
		}
		buf.State = dme.CloudletState_CLOUDLET_STATE_OFFLINE
		buf.Fields = []string{edgeproto.CloudletInfoFieldState}
		s.store.STMPut(stm, &buf)
		return nil
	})
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelNotify, "notify delete CloudletInfo",
			"key", in.Key, "err", err)
	} else {
		nodeMgr.Event(ctx, "Cloudlet offline", in.Key.Organization, in.Key.GetTags(), nil, "reason", "notify disconnect")
	}
}

func (s *CloudletInfoApi) Flush(ctx context.Context, notifyId int64) {
	// mark all cloudlets from the client as offline
	matches := make([]edgeproto.CloudletKey, 0)
	s.cache.Mux.Lock()
	for _, data := range s.cache.Objs {
		val := data.Obj
		if val.NotifyId != notifyId || val.Controller != ControllerId {
			continue
		}
		matches = append(matches, val.Key)
	}
	s.cache.Mux.Unlock()

	// this creates a new span if there was none - which can happen if this is a cancelled context
	span := log.SpanFromContext(ctx)
	ectx := log.ContextWithSpan(context.Background(), span)

	info := edgeproto.CloudletInfo{}
	for ii, _ := range matches {
		info.Key = matches[ii]
		cloudletReady := false
		err := s.sync.ApplySTMWait(ctx, func(stm concurrency.STM) error {
			if s.store.STMGet(stm, &info.Key, &info) {
				if info.NotifyId != notifyId || info.Controller != ControllerId {
					// Updated by another thread or controller
					return nil
				}
			}
			cloudlet := edgeproto.Cloudlet{}
			if s.all.cloudletApi.store.STMGet(stm, &info.Key, &cloudlet) {
				cloudletReady = (cloudlet.State == edgeproto.TrackedState_READY)
			}
			info.State = dme.CloudletState_CLOUDLET_STATE_OFFLINE
			log.SpanLog(ctx, log.DebugLevelNotify, "mark cloudlet offline", "key", matches[ii], "notifyid", notifyId)
			s.store.STMPut(stm, &info)
			return nil
		})
		if err != nil {
			log.SpanLog(ctx, log.DebugLevelNotify, "mark cloudlet offline", "key", matches[ii], "err", err)
		} else {
			// Send a cloudlet down alert if a cloudlet was ready
			if cloudletReady {
				nodeMgr.Event(ectx, "Cloudlet offline", info.Key.Organization, info.Key.GetTags(), nil, "reason", "notify disconnect")
				s.FireCloudletAndAppInstDownAlerts(ctx, &info)
			}
		}
	}
}

func (s *CloudletInfoApi) Prune(ctx context.Context, keys map[edgeproto.CloudletKey]struct{}) {}

func (s *CloudletInfoApi) getCloudletState(key *edgeproto.CloudletKey) dme.CloudletState {
	s.cache.Mux.Lock()
	defer s.cache.Mux.Unlock()
	for _, data := range s.cache.Objs {
		obj := data.Obj
		if key.Matches(&obj.Key) {
			return obj.State
		}
	}
	return dme.CloudletState_CLOUDLET_STATE_NOT_PRESENT
}

func (s *CloudletInfoApi) checkCloudletReadySTM(cctx *CallContext, stm concurrency.STM, key *edgeproto.CloudletKey, action cloudcommon.Action) error {
	cloudlet := edgeproto.Cloudlet{}
	if !s.all.cloudletApi.store.STMGet(stm, key, &cloudlet) {
		return key.NotFoundError()
	}
	info := edgeproto.CloudletInfo{}
	if !s.all.cloudletInfoApi.store.STMGet(stm, key, &info) {
		return fmt.Errorf("CloudletInfo not found for Cloudlet %s", key.GetKeyString())
	}
	return s.checkCloudletReady(cctx, &cloudlet, &info, action)
}

func (s *CloudletInfoApi) checkCloudletReady(cctx *CallContext, cloudlet *edgeproto.Cloudlet, info *edgeproto.CloudletInfo, action cloudcommon.Action) error {
	if cctx != nil && (ignoreCRM(cctx) || cctx.SkipCloudletReadyCheck) {
		return nil
	}
	// Get tracked state, it could be that cloudlet has initiated
	// upgrade but cloudletInfo is yet to transition to it
	if action == cloudcommon.Delete && (cloudlet.DeletePrepare || cloudlet.State == edgeproto.TrackedState_DELETE_PREPARE) {
		return cloudlet.Key.BeingDeletedError()
	}
	if cloudlet.State == edgeproto.TrackedState_UPDATE_REQUESTED ||
		cloudlet.State == edgeproto.TrackedState_UPDATING {
		return fmt.Errorf("cloudlet %s is upgrading", cloudlet.Key.GetKeyString())
	}
	if cloudlet.MaintenanceState != dme.MaintenanceState_NORMAL_OPERATION {
		return errors.New("cloudlet under maintenance, please try again later")
	}

	if cloudlet.State == edgeproto.TrackedState_UPDATE_ERROR {
		return fmt.Errorf("cloudlet %s is in failed upgrade state, please upgrade it manually", cloudlet.Key.GetKeyString())
	}
	if info.State == dme.CloudletState_CLOUDLET_STATE_READY {
		return nil
	}
	return fmt.Errorf("cloudlet %s not ready, state is %s", cloudlet.Key.GetKeyString(),
		dme.CloudletState_name[int32(info.State)])
}

// Clean up CloudletInfo after Cloudlet delete.
// Only delete if state is Offline.
func (s *CloudletInfoApi) cleanupCloudletInfo(ctx context.Context, in *edgeproto.Cloudlet) {
	var delErr error
	info := edgeproto.CloudletInfo{}
	if !in.CrmOnEdge || in.InfraApiAccess == edgeproto.InfraApiAccess_RESTRICTED_ACCESS || in.Key.FederatedOrganization != "" {
		// no way for the controller to shutdown the crm,
		// so just clean up the CloudletInfo
		delErr = s.sync.ApplySTMWait(ctx, func(stm concurrency.STM) error {
			if !s.store.STMGet(stm, &in.Key, &info) {
				return nil
			}
			s.store.STMDel(stm, &in.Key)
			return nil
		})
	} else {
		done := make(chan bool, 1)
		checkState := func() {
			if !s.cache.Get(&in.Key, &info) {
				done <- true
				return
			}
			if info.State == dme.CloudletState_CLOUDLET_STATE_OFFLINE {
				done <- true
			}
		}
		cancel := s.cache.WatchKey(&in.Key, func(ctx context.Context) {
			checkState()
		})
		defer cancel()
		// after setting up watch, check current state,
		// as it may have already changed to target state
		checkState()

		select {
		case <-done:
		case <-time.After(cleanupCloudletInfoTimeout):
			log.SpanLog(ctx, log.DebugLevelApi, "timed out waiting for CloudletInfo to go Offline", "key", &in.Key)
		}
		delErr = s.sync.ApplySTMWait(ctx, func(stm concurrency.STM) error {
			info := edgeproto.CloudletInfo{}
			if !s.store.STMGet(stm, &in.Key, &info) {
				return nil
			}
			if info.State != dme.CloudletState_CLOUDLET_STATE_OFFLINE {
				return fmt.Errorf("could not delete CloudletInfo, state is %s instead of offline", info.State.String())
			}
			s.store.STMDel(stm, &in.Key)
			return nil
		})
	}
	if delErr != nil {
		log.SpanLog(ctx, log.DebugLevelApi, "cleanup CloudletInfo failed", "err", delErr)
	}
	// clean up any associated alerts with this cloudlet
	s.ClearCloudletAndAppInstDownAlerts(ctx, &info)
}

func (s *CloudletInfoApi) waitForMaintenanceState(ctx context.Context, key *edgeproto.CloudletKey, targetState, errorState dme.MaintenanceState, timeout time.Duration, result *edgeproto.CloudletInfo) error {
	done := make(chan bool, 1)
	check := func(ctx context.Context) {
		if !s.cache.Get(key, result) {
			log.SpanLog(ctx, log.DebugLevelApi, "wait for CloudletInfo state info not found", "key", key)
			return
		}
		if result.MaintenanceState == targetState || result.MaintenanceState == errorState {
			done <- true
		}
	}

	log.SpanLog(ctx, log.DebugLevelApi, "wait for CloudletInfo state", "target", targetState)

	cancel := s.cache.WatchKey(key, check)

	// after setting up watch, check current state,
	// as it may have already changed to the target state
	check(ctx)

	var err error
	select {
	case <-done:
	case <-time.After(timeout):
		err = fmt.Errorf("timed out waiting for CloudletInfo maintenance state")
	}
	cancel()

	return err
}

func getCloudletPropertyBool(info *edgeproto.CloudletInfo, prop string, def bool) bool {
	if info.Properties == nil {
		return def
	}
	str, found := info.Properties[prop]
	if !found {
		return def
	}
	val, err := strconv.ParseBool(str)
	if err != nil {
		return def
	}
	return val
}
