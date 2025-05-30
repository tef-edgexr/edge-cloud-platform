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

package gcp

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/edgexr/edge-cloud-platform/api/edgeproto"
	"github.com/edgexr/edge-cloud-platform/pkg/k8smgmt"
	"github.com/edgexr/edge-cloud-platform/pkg/log"
	"github.com/edgexr/edge-cloud-platform/pkg/platform/common/infracommon"
)

// SetProject sets the project in gcloud config
func (g *GCPPlatform) SetProject(ctx context.Context, project string) error {
	log.SpanLog(ctx, log.DebugLevelInfra, "SetProject", "project", project)
	out, err := infracommon.Sh(g.accessVars).Command("gcloud", "config", "set", "project", project).CombinedOutput()
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelInfra, "Error in SetProject", "out", string(out), "err", err)
		return fmt.Errorf("Error in SetProject: %s - %v", string(out), err)
	}
	return nil
}

// SetZone sets the zone in gcloud config
func (g *GCPPlatform) SetZone(ctx context.Context, zone string) error {
	log.SpanLog(ctx, log.DebugLevelInfra, "SetZone", "zone", zone)
	out, err := infracommon.Sh(g.accessVars).Command("gcloud", "config", "set", "compute/zone", zone).CombinedOutput()
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelInfra, "Error in SetZone", "zone", zone, "out", string(out), "err", err)
		return fmt.Errorf("Error in SetZone: %s - %v", string(out), err)
	}
	return nil
}

// CreateClusterPrerequisites currently does nothing
func (a *GCPPlatform) CreateClusterPrerequisites(ctx context.Context, clusterName string) error {
	return nil
}

// RunClusterCreateCommand creates a kubernetes cluster on gcloud
func (g *GCPPlatform) RunClusterCreateCommand(ctx context.Context, clusterName string, clusterInst *edgeproto.ClusterInst) (map[string]string, error) {
	log.SpanLog(ctx, log.DebugLevelInfra, "RunClusterCreateCommand", "clusterName", clusterName)
	pool := clusterInst.NodePools[0]
	flavor := pool.NodeResources.InfraNodeFlavor
	numNodesStr := fmt.Sprintf("%d", pool.NumNodes)
	out, err := infracommon.Sh(g.accessVars).Command("gcloud", "container", "clusters", "create", "--num-nodes="+numNodesStr, "--machine-type="+flavor, clusterName).CombinedOutput()
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelInfra, "Error in cluster create", "out", string(out), "err", err)
		return nil, fmt.Errorf("Error in cluster create: %s - %v", string(out), err)
	}
	return nil, nil
}

func (s *GCPPlatform) RunClusterUpdateCommand(ctx context.Context, clusterName string, clusterInst *edgeproto.ClusterInst) (map[string]string, error) {
	return nil, errors.New("update cluster instance not implemented")
}

// RunClusterDeleteCommand removes kubernetes cluster on gcloud
func (g *GCPPlatform) RunClusterDeleteCommand(ctx context.Context, clusterName string, clusterInst *edgeproto.ClusterInst) error {
	log.SpanLog(ctx, log.DebugLevelInfra, "RunClusterDeleteCommand", "clusterName", clusterName)
	out, err := infracommon.Sh(g.accessVars).Command("gcloud", "container", "clusters", "delete", "--quiet", clusterName).CombinedOutput()
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelInfra, "Error in cluster delete", "out", string(out), "err", err)
		return fmt.Errorf("Error in cluster delete: %s - %v", string(out), err)
	}
	return nil
}

// GetCredentials retrieves kubeconfig credentials from gcloud.
func (g *GCPPlatform) GetCredentials(ctx context.Context, clusterName string, clusterInst *edgeproto.ClusterInst) ([]byte, error) {
	kconf := "/tmp/" + clusterName + ".kubeconfig"
	envVars := make(map[string]string)
	for k, v := range g.accessVars {
		envVars[k] = v
	}
	envVars["KUBECONFIG"] = kconf
	out, err := infracommon.Sh(envVars).Command("gcloud", "container", "clusters", "get-credentials", clusterName).CombinedOutput()
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelInfra, "Error in GetCredentials", "out", string(out), "err", err)
		return nil, fmt.Errorf("get credential failed: %s - %v", string(out), err)
	}
	defer func() {
		os.Remove(kconf)
	}()
	data, err := os.ReadFile(kconf)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s, %s", kconf, err)
	}
	return data, nil
}

func (a *GCPPlatform) GetClusterAddonInfo(ctx context.Context, clusterName string, clusterInst *edgeproto.ClusterInst) (*k8smgmt.ClusterAddonInfo, error) {
	info := k8smgmt.ClusterAddonInfo{}
	return &info, nil
}

func (g *GCPPlatform) GetCloudletInfraResourcesInfo(ctx context.Context) ([]edgeproto.InfraResource, error) {
	return []edgeproto.InfraResource{}, nil
}

// called by controller, make sure it doesn't make any calls to infra API
func (g *GCPPlatform) GetClusterAdditionalResources(ctx context.Context, cloudlet *edgeproto.Cloudlet, vmResources []edgeproto.VMResource) map[string]edgeproto.InfraResource {
	return nil
}

func (g *GCPPlatform) GetClusterAdditionalResourceMetric(ctx context.Context, cloudlet *edgeproto.Cloudlet, resMetric *edgeproto.Metric, resources []edgeproto.VMResource) error {
	return nil
}
