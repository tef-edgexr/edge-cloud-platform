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

package k8smgmt

import (
	"context"
	"fmt"
	"strconv"

	"github.com/edgexr/edge-cloud-platform/api/edgeproto"
	"github.com/edgexr/edge-cloud-platform/pkg/cloudcommon"
	"github.com/edgexr/edge-cloud-platform/pkg/log"
	"github.com/edgexr/edge-cloud-platform/pkg/platform"
	"github.com/edgexr/edge-cloud-platform/pkg/platform/pc"
	ssh "github.com/edgexr/golang-ssh"
)

var crmConfigVersionFile = "crmconfigversion.txt"
var crmConfigVersion = 1

// This function is called after CRM starts and has received all the
// cache data from the controller.
func UpgradeConfig(ctx context.Context, caches *platform.Caches, sharedRootLBClient ssh.Client, getClient func(context.Context, *edgeproto.ClusterInst, string) (ssh.Client, error)) error {
	log.SpanLog(ctx, log.DebugLevelInfra, "upgrade k8smgmt config")
	// Config version file tracks which version the k8s manifest config
	// files are at. We had a few different ways of managing the manifest files.
	version := 0
	out, err := sharedRootLBClient.Output("cat " + crmConfigVersionFile)
	if err == nil {
		version, err = strconv.Atoi(string(out))
		if err != nil {
			return fmt.Errorf("Unable to convert crm config version '%s' to integer: %s", string(out), err)
		}
	}

	log.SpanLog(ctx, log.DebugLevelInfra, "upgrade k8smgmt config", "version", version)
	if version < 1 {
		err = upgradeVersionSingleClusterConfigDir(ctx, caches, getClient)
		if err != nil {
			return err
		}
	}

	// upgrades complete
	log.SpanLog(ctx, log.DebugLevelInfra, "upgrades complete")
	err = pc.WriteFile(sharedRootLBClient, crmConfigVersionFile, fmt.Sprintf("%d", crmConfigVersion), "crm config version", pc.NoSudo)
	return err
}

func getConfigDirNameOld(names *KubeNames) (string, string) {
	return names.ClusterName + "_" + names.AppName + names.AppOrg + names.AppVersion, "manifest.yaml"
}

// Moves all AppInst manifests for a cluster into a single directory that can
// use declarative configuration management with the "prune" option.
func upgradeVersionSingleClusterConfigDir(ctx context.Context, caches *platform.Caches, getClient func(context.Context, *edgeproto.ClusterInst, string) (ssh.Client, error)) error {
	log.SpanLog(ctx, log.DebugLevelInfra, "upgrade k8smgmt single cluster config dir")
	appInsts := make([]*edgeproto.AppInst, 0)

	caches.AppInstCache.Mux.Lock()
	for _, data := range caches.AppInstCache.Objs {
		inst := edgeproto.AppInst{}
		inst.DeepCopyIn(data.Obj)
		appInsts = append(appInsts, &inst)
	}
	caches.AppInstCache.Mux.Unlock()

	for _, appInst := range appInsts {
		log.SpanLog(ctx, log.DebugLevelInfra, "upgrade version single cluster config dir", "AppInst", appInst.Key)
		app := edgeproto.App{}
		if !caches.AppCache.Get(&appInst.AppKey, &app) {
			log.SpanLog(ctx, log.DebugLevelInfra, "upgrade version single cluster config dir, App not found", "AppInst", appInst.Key)
			continue
		}
		if app.Deployment != cloudcommon.DeploymentTypeKubernetes {
			continue
		}
		cinst := edgeproto.ClusterInst{}
		if !caches.ClusterInstCache.Get(appInst.GetClusterKey(), &cinst) {
			log.SpanLog(ctx, log.DebugLevelInfra, "upgrade version single cluster config dir, ClusterInst not found", "AppInst", appInst.Key)
			continue
		}

		names, err := GetKubeNames(&cinst, &app, appInst)
		if err != nil {
			log.SpanLog(ctx, log.DebugLevelInfra, "upgrade version single cluster config dir, names failed", "AppInst", appInst.Key, "err", err)
			continue
		}
		client, err := getClient(ctx, &cinst, cloudcommon.ClientTypeRootLB)
		if err != nil {
			log.SpanLog(ctx, log.DebugLevelInfra, "upgrade version single cluster config dir, get client failed", "AppInst", appInst.Key, "err", err)
			continue
		}

		// make new config dir if necessary (may already have been created
		// if multiple AppInsts in ClusterInst)
		configDir := GetConfigDirName(names)
		configName := getConfigFileName(names, appInst, DeploymentManifestSuffix)
		err = pc.CreateDir(ctx, client, configDir, pc.NoOverwrite, pc.NoSudo)
		if err != nil {
			log.SpanLog(ctx, log.DebugLevelInfra, "upgrade version single cluster config dir, create dir failed", "AppInst", appInst.Key, "err", err)
			continue
		}

		targetFile := configDir + "/" + configName
		// if manifest was local file, move it into the dir
		oldFile := names.AppName + names.AppInstRevision + ".yaml"
		_, err = client.Output("ls " + oldFile)
		if err != nil {
			// no manifest file, check for AppInst-specific dir file
			configDirOld, configNameOld := getConfigDirNameOld(names)
			oldFile = configDirOld + "/" + configNameOld
			_, err = client.Output("ls " + oldFile)
		}
		// if old manifest exists, move it into cluster manifest config dir
		if err == nil {
			out, err := client.Output("cp " + oldFile + " " + targetFile)
			log.SpanLog(ctx, log.DebugLevelInfra, "upgrade version single cluster config dir, move file", "AppInst", appInst.Key, "oldFile", oldFile, "targetFile", targetFile, "err", err, "out", string(out))
		} else {
			// no old manifest found, that's odd, log it
			log.SpanLog(ctx, log.DebugLevelInfra, "upgrade version single cluster config dir, old manifest not found", "AppInst", appInst.Key)
		}
	}
	return nil
}
