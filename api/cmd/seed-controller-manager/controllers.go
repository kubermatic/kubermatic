/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/addon"
	"github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/addoninstaller"
	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/backup"
	cloudcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/clustercomponentdefaulter"
	kubernetescontroller "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/monitoring"
	openshiftcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/openshift"
	"github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/rancher"
	"github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/seedresourcesuptodatecondition"
	updatecontroller "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/update"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/features"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	utilpointer "k8s.io/utils/pointer"
)

// AllControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller
var AllControllers = map[string]controllerCreator{
	kubernetescontroller.ControllerName:           createKubernetesController,
	updatecontroller.ControllerName:               createUpdateController,
	addon.ControllerName:                          createAddonController,
	addoninstaller.ControllerName:                 createAddonInstallerController,
	backupcontroller.ControllerName:               createBackupController,
	monitoring.ControllerName:                     createMonitoringController,
	cloudcontroller.ControllerName:                createCloudController,
	openshiftcontroller.ControllerName:            createOpenshiftController,
	clustercomponentdefaulter.ControllerName:      createClusterComponentDefaulter,
	seedresourcesuptodatecondition.ControllerName: createSeedConditionUpToDateController,
	rancher.ControllerName:                        createRancherController,
}

type controllerCreator func(*controllerContext) error

func createAllControllers(ctrlCtx *controllerContext) error {
	for name, create := range AllControllers {
		if err := create(ctrlCtx); err != nil {
			return fmt.Errorf("failed to create %q controller: %v", name, err)
		}
	}
	return nil
}

func createSeedConditionUpToDateController(ctrlCtx *controllerContext) error {
	return seedresourcesuptodatecondition.Add(
		ctrlCtx.ctx,
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
	)
}

func createClusterComponentDefaulter(ctrlCtx *controllerContext) error {
	defaultCompontentsOverrides := kubermaticv1.ComponentSettings{
		Apiserver: kubermaticv1.APIServerSettings{
			DeploymentSettings:          kubermaticv1.DeploymentSettings{Replicas: utilpointer.Int32Ptr(int32(ctrlCtx.runOptions.apiServerDefaultReplicas))},
			EndpointReconcilingDisabled: utilpointer.BoolPtr(ctrlCtx.runOptions.apiServerEndpointReconcilingDisabled),
		},
		ControllerManager: kubermaticv1.ControllerSettings{
			DeploymentSettings: kubermaticv1.DeploymentSettings{
				Replicas: utilpointer.Int32Ptr(int32(ctrlCtx.runOptions.controllerManagerDefaultReplicas)),
			},
		},
		Scheduler: kubermaticv1.ControllerSettings{
			DeploymentSettings: kubermaticv1.DeploymentSettings{
				Replicas: utilpointer.Int32Ptr(int32(ctrlCtx.runOptions.schedulerDefaultReplicas)),
			},
		},
	}
	return clustercomponentdefaulter.Add(
		context.Background(),
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		defaultCompontentsOverrides,
		ctrlCtx.runOptions.workerName,
	)
}

func createCloudController(ctrlCtx *controllerContext) error {
	if err := cloudcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.seedGetter,
		ctrlCtx.runOptions.workerName,
	); err != nil {
		return fmt.Errorf("failed to add cloud controller to mgr: %v", err)
	}
	return nil
}

func createOpenshiftController(ctrlCtx *controllerContext) error {
	if err := openshiftcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.seedGetter,
		ctrlCtx.clientProvider,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.runOptions.etcdDiskSize,
		ctrlCtx.dockerPullConfigJSON,
		ctrlCtx.runOptions.externalURL,
		ctrlCtx.runOptions.kubermaticImage,
		ctrlCtx.runOptions.dnatControllerImage,
		openshiftcontroller.Features{
			EtcdDataCorruptionChecks: ctrlCtx.runOptions.featureGates.Enabled(features.EtcdDataCorruptionChecks),
			VPA:                      ctrlCtx.runOptions.featureGates.Enabled(features.VerticalPodAutoscaler),
		},
		ctrlCtx.runOptions.concurrentClusterUpdate); err != nil {
		return fmt.Errorf("failed to add openshift controller to mgr: %v", err)
	}
	return nil
}

func createKubernetesController(ctrlCtx *controllerContext) error {

	return kubernetescontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.externalURL,
		ctrlCtx.seedGetter,
		ctrlCtx.clientProvider,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodePortRange,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.runOptions.etcdDiskSize,
		ctrlCtx.runOptions.monitoringScrapeAnnotationPrefix,
		ctrlCtx.runOptions.inClusterPrometheusRulesFile,
		ctrlCtx.runOptions.inClusterPrometheusDisableDefaultRules,
		ctrlCtx.runOptions.inClusterPrometheusDisableDefaultScrapingConfigs,
		ctrlCtx.runOptions.inClusterPrometheusScrapingConfigsFile,
		ctrlCtx.dockerPullConfigJSON,
		ctrlCtx.runOptions.nodeLocalDNSCacheEnabled(),
		ctrlCtx.runOptions.concurrentClusterUpdate,
		ctrlCtx.runOptions.oidcCAFile,
		ctrlCtx.runOptions.oidcIssuerURL,
		ctrlCtx.runOptions.oidcIssuerClientID,
		ctrlCtx.runOptions.kubermaticImage,
		ctrlCtx.runOptions.dnatControllerImage,
		kubernetescontroller.Features{
			VPA:                          ctrlCtx.runOptions.featureGates.Enabled(features.VerticalPodAutoscaler),
			EtcdDataCorruptionChecks:     ctrlCtx.runOptions.featureGates.Enabled(features.EtcdDataCorruptionChecks),
			KubernetesOIDCAuthentication: ctrlCtx.runOptions.featureGates.Enabled(features.OpenIDAuthPlugin),
		},
	)
}

func createBackupController(ctrlCtx *controllerContext) error {
	storeContainer, err := getContainerFromFile(ctrlCtx.runOptions.backupContainerFile)
	if err != nil {
		return err
	}
	cleanupContainer, err := getContainerFromFile(ctrlCtx.runOptions.cleanupContainerFile)
	if err != nil {
		return err
	}
	backupInterval, err := time.ParseDuration(ctrlCtx.runOptions.backupInterval)
	if err != nil {
		return fmt.Errorf("failed to parse %s as duration: %v", ctrlCtx.runOptions.backupInterval, err)
	}
	return backupcontroller.Add(
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		*storeContainer,
		*cleanupContainer,
		backupInterval,
		ctrlCtx.runOptions.backupContainerImage,
	)
}

func createMonitoringController(ctrlCtx *controllerContext) error {
	dockerPullConfigJSON, err := ioutil.ReadFile(ctrlCtx.runOptions.dockerPullConfigJSONFile)
	if err != nil {
		return fmt.Errorf("failed to load ImagePullSecret from %s: %v", ctrlCtx.runOptions.dockerPullConfigJSONFile, err)
	}

	return monitoring.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.clientProvider,

		ctrlCtx.seedGetter,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodePortRange,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.runOptions.monitoringScrapeAnnotationPrefix,
		ctrlCtx.runOptions.inClusterPrometheusRulesFile,
		ctrlCtx.runOptions.inClusterPrometheusDisableDefaultRules,
		ctrlCtx.runOptions.inClusterPrometheusDisableDefaultScrapingConfigs,
		ctrlCtx.runOptions.inClusterPrometheusScrapingConfigsFile,
		dockerPullConfigJSON,
		ctrlCtx.runOptions.concurrentClusterUpdate,
		monitoring.Features{
			VPA: ctrlCtx.runOptions.featureGates.Enabled(features.VerticalPodAutoscaler),
		},
	)
}

func getContainerFromFile(path string) (*corev1.Container, error) {
	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	container := &corev1.Container{}
	manifestReader := bytes.NewReader(fileContents)
	manifestDecoder := yaml.NewYAMLToJSONDecoder(manifestReader)
	if err := manifestDecoder.Decode(container); err != nil {
		return nil, err
	}

	// Just because its a valid corev1.Container does not mean
	// the APIServer will accept it, thus we do some additional
	// checks
	if container.Name == "" {
		return nil, fmt.Errorf("container must have a name")
	}
	if container.Image == "" {
		return nil, fmt.Errorf("container must have an image")
	}
	return container, nil
}

func createUpdateController(ctrlCtx *controllerContext) error {
	updateManager, err := version.NewFromFiles(ctrlCtx.runOptions.versionsFile, ctrlCtx.runOptions.updatesFile)
	if err != nil {
		return fmt.Errorf("failed to create update manager: %v", err)
	}

	return updatecontroller.Add(ctrlCtx.mgr, ctrlCtx.runOptions.workerCount, ctrlCtx.runOptions.workerName, updateManager,
		ctrlCtx.clientProvider, ctrlCtx.log)
}

func createAddonController(ctrlCtx *controllerContext) error {
	return addon.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.addonEnforceInterval,
		map[string]interface{}{ // addonVariables
			"openvpn": map[string]interface{}{
				"NodeAccessNetwork": ctrlCtx.runOptions.nodeAccessNetwork,
			},
		},
		ctrlCtx.runOptions.kubernetesAddonsPath,
		ctrlCtx.runOptions.openshiftAddonsPath,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodeLocalDNSCacheEnabled(),
		ctrlCtx.clientProvider,
	)
}

func createAddonInstallerController(ctrlCtx *controllerContext) error {
	return addoninstaller.Add(
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.kubernetesAddons,
		ctrlCtx.runOptions.openshiftAddons)
}

func createRancherController(ctrlCtx *controllerContext) error {
	return rancher.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.clientProvider)
}
