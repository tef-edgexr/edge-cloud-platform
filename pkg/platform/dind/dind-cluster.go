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

package dind

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	sh "github.com/codeskyblue/go-sh"
	"github.com/edgexr/edge-cloud-platform/api/edgeproto"
	"github.com/edgexr/edge-cloud-platform/pkg/cloudcommon"
	"github.com/edgexr/edge-cloud-platform/pkg/k8smgmt"
	"github.com/edgexr/edge-cloud-platform/pkg/log"
)

type DindCluster struct {
	ClusterName string
	ClusterID   int
	MasterAddr  string
	KContext    string
}

func (s *Platform) CreateClusterInst(ctx context.Context, clusterInst *edgeproto.ClusterInst, updateCallback edgeproto.CacheUpdateCallback, timeout time.Duration) (map[string]string, error) {
	var err error

	switch clusterInst.Deployment {
	case cloudcommon.DeploymentTypeDocker:
		updateCallback(edgeproto.UpdateTask, "Create done for Docker Cluster on DIND")
		return nil, nil
	case cloudcommon.DeploymentTypeKubernetes:
		updateCallback(edgeproto.UpdateTask, "Create DIND Cluster")
	default:
		return nil, fmt.Errorf("Only K8s and Docker clusters are supported on DIND")
	}
	// Create K8s cluster
	clusterName := k8smgmt.NormalizeName(clusterInst.Key.Name + clusterInst.Key.Organization)
	log.SpanLog(ctx, log.DebugLevelInfra, "creating local dind cluster", "clusterName", clusterName)

	kconfName := k8smgmt.GetKconfName(clusterInst)
	if err = s.CreateDINDCluster(ctx, clusterName, kconfName); err != nil {
		return nil, err
	}
	log.SpanLog(ctx, log.DebugLevelInfra, "created dind", "name", clusterName)
	return nil, nil
}

func (s *Platform) UpdateClusterInst(ctx context.Context, clusterInst *edgeproto.ClusterInst, updateCallback edgeproto.CacheUpdateCallback) (map[string]string, error) {
	return nil, fmt.Errorf("update cluster not supported for DIND")
}

func (s *Platform) ChangeClusterInstDNS(ctx context.Context, clusterInst *edgeproto.ClusterInst, oldFqdn string, updateCallback edgeproto.CacheUpdateCallback) error {
	return fmt.Errorf("cluster dns change not supported for DIND")
}

func (s *Platform) DeleteClusterInst(ctx context.Context, clusterInst *edgeproto.ClusterInst, updateCallback edgeproto.CacheUpdateCallback) error {
	return s.DeleteDINDCluster(ctx, clusterInst)
}

// CreateDINDCluster creates kubernetes cluster on local mac
func (s *Platform) CreateDINDCluster(ctx context.Context, clusterName, kconfName string) error {
	clusters, err := GetClusters()
	if err != nil {
		return err
	}
	ids := make(map[int]struct{})
	for _, clust := range clusters {
		if clust.ClusterName == clusterName {
			return fmt.Errorf("ERROR - Cluster %s already exists (%v)", clusterName, clust)
		}
		ids[clust.ClusterID] = struct{}{}
	}
	clusterID := 1
	for {
		if _, found := ids[clusterID]; !found {
			break
		}
		clusterID++
	}
	// if KUBECONFIG is set, then the dind-cluster script will write config
	// to that file instead of ~/.kube/config, which is super confusing.
	// For consistency, make sure KUBECONFIG is not set (it may be pointing
	// to the wrong place).
	os.Unsetenv("KUBECONFIG")
	os.Setenv("DIND_LABEL", clusterName)
	os.Setenv("CLUSTER_ID", GetClusterID(clusterID))
	cluster := NewClusterFor(clusterName, clusterID)
	log.SpanLog(ctx, log.DebugLevelInfra, "CreateDINDCluster", "scriptName", cloudcommon.DindScriptName, "name", clusterName, "clusterid", clusterID)

	out, err := sh.Command(cloudcommon.DindScriptName, "up").Command("tee", "/tmp/dind.log").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ERROR creating Dind Cluster: [%s] %v", out, err)
	}
	log.SpanLog(ctx, log.DebugLevelInfra, "Finished CreateDINDCluster", "name", clusterName)
	//race condition exists where the config file is not ready until just after the cluster create is done
	time.Sleep(3 * time.Second)

	//now set the k8s config
	log.SpanLog(ctx, log.DebugLevelInfra, "set config context", "kcontext", cluster.KContext)
	out, err = sh.Command("kubectl", "config", "use-context", cluster.KContext).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ERROR setting kube config context: [%s] %v", string(out), err)
	}
	log.SpanLog(ctx, log.DebugLevelInfra, "set config context output", "out", string(out), "err", err)

	//copy kubeconfig locally
	log.SpanLog(ctx, log.DebugLevelInfra, "locally copying kubeconfig", "kconfName", kconfName)
	home := os.Getenv("HOME")
	out, err = sh.Command("cp", home+"/.kube/config", kconfName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v", out, err)
	}
	// xind is looking for config in /tmp dir
	out, err = sh.Command("cp", home+"/.kube/config", "/tmp/"+kconfName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v", out, err)
	}
	out, err = sh.Command("cat", home+"/.kube/config").CombinedOutput()
	log.SpanLog(ctx, log.DebugLevelInfra, "config file", "home", home, "out", string(out), "err", err)

	return nil
}

// DeleteDINDCluster creates kubernetes cluster on local mac
func (s *Platform) DeleteDINDCluster(ctx context.Context, clusterInst *edgeproto.ClusterInst) error {

	clusterName := k8smgmt.NormalizeName(clusterInst.Key.Name + clusterInst.Key.Organization)
	log.SpanLog(ctx, log.DebugLevelInfra, "DeleteDINDCluster", "clusterName", clusterName)

	if clusterInst.Deployment == cloudcommon.DeploymentTypeDocker {
		log.SpanLog(ctx, log.DebugLevelInfra, "No delete required for DIND docker cluster", "clusterName", clusterName)
		return nil
	}
	cluster, err := FindCluster(clusterName)
	if err != nil {
		return fmt.Errorf("ERROR - Cluster %s not found, %v", clusterName, err)
	}

	os.Setenv("DIND_LABEL", cluster.ClusterName)
	os.Setenv("CLUSTER_ID", GetClusterID(cluster.ClusterID))
	out, err := sh.Command(cloudcommon.DindScriptName, "clean").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v", out, err)
	}
	log.SpanLog(ctx, log.DebugLevelInfra, "Finished dind clean", "scriptName", cloudcommon.DindScriptName, "clusterName", clusterName, "out", out)
	return nil
}

func GetClusterID(id int) string {
	return strconv.Itoa(id)
}

func FindCluster(clusterName string) (*DindCluster, error) {
	clusters, err := GetClusters()
	if err != nil {
		return nil, err
	}
	for ii, _ := range clusters {
		if clusters[ii].ClusterName == clusterName {
			return &clusters[ii], nil
		}
	}
	return nil, fmt.Errorf("dind cluster %s not found", clusterName)
}

func GetClusters() ([]DindCluster, error) {
	out, err := sh.Command("docker", "ps", "--format", "{{.Names}}").CombinedOutput()
	if err != nil {
		return nil, err
	}
	clusters := []DindCluster{}
	r, _ := regexp.Compile("kube-master-(\\S+)-(\\d+)")
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if r.MatchString(line) {
			matches := r.FindStringSubmatch(line)
			cname := matches[1]
			cid, err := strconv.Atoi(matches[2])
			if err != nil {
				return nil, fmt.Errorf("Could not parse kube-master id: %s", line)
			}
			clusters = append(clusters, NewClusterFor(cname, cid))
		}
	}
	return clusters, nil
}

func NewClusterFor(clusterName string, id int) DindCluster {
	return DindCluster{
		ClusterName: clusterName,
		ClusterID:   id,
		KContext:    "dind-" + clusterName + "-" + GetClusterID(id),
		MasterAddr:  "10.192." + GetClusterID(id) + ".2",
	}
}

func GetDockerNetworkName(cluster *DindCluster) string {
	return "kubeadm-dind-net-" + cluster.ClusterName + "-" + GetClusterID(cluster.ClusterID)
}

func (s *Platform) GetMasterIp(ctx context.Context, names *k8smgmt.KubeNames) (string, error) {
	cluster, err := FindCluster(names.ClusterName)
	if err != nil {
		return "", err
	}
	return cluster.MasterAddr, nil
}

func (s *Platform) GetDockerNetworkName(ctx context.Context, names *k8smgmt.KubeNames) (string, error) {
	cluster, err := FindCluster(names.ClusterName)
	if err != nil {
		return "", err
	}
	return GetDockerNetworkName(cluster), nil
}
