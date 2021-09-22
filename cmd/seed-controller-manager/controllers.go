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
	"fmt"
	"io/ioutil"
	"time"

	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/addon"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/addoninstaller"
	backupcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/backup"
	cloudcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/cloud"
	clustertemplatecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/cluster-template-controller"
	seedconstraintsynchronizer "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/constraint-controller"
	constrainttemplatecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/constraint-template-controller"
	etcdbackupcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/etcdbackup"
	etcdrestorecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/etcdrestore"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/initialmachinedeployment"
	kubernetescontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/kubernetes"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/monitoring"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/pvwatcher"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/rancher"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/seedresourcesuptodatecondition"
	updatecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/update"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"

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
	etcdbackupcontroller.ControllerName:           createEtcdBackupController,
	backupcontroller.ControllerName:               createBackupController,
	etcdrestorecontroller.ControllerName:          createEtcdRestoreController,
	monitoring.ControllerName:                     createMonitoringController,
	cloudcontroller.ControllerName:                createCloudController,
	seedresourcesuptodatecondition.ControllerName: createSeedConditionUpToDateController,
	rancher.ControllerName:                        createRancherController,
	pvwatcher.ControllerName:                      createPvWatcherController,
	seedconstraintsynchronizer.ControllerName:     createConstraintController,
	constrainttemplatecontroller.ControllerName:   createConstraintTemplateController,
	initialmachinedeployment.ControllerName:       createInitialMachineDeploymentController,
	mla.ControllerName:                            createMLAController,
	clustertemplatecontroller.ControllerName:      createClusterTemplateController,
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
		ctrlCtx.versions,
	)
}

func defaultComponentSettings(runOptions controllerRunOptions, seed *kubermaticv1.Seed) (kubermaticv1.ComponentSettings, error) {
	// Copy default settings.
	settings := seed.Spec.DefaultComponentSettings

	if replicas := runOptions.apiServerDefaultReplicas; replicas != 0 {
		if settings.Apiserver.Replicas != nil && replicas != int(*settings.Apiserver.Replicas) {
			return settings, fmt.Errorf(
				"conflicting settings, cli option api-server-default-replicas (%v) and field Seed.spec.defaultComponentSettings.apiserver.replicas (%v) do not match",
				replicas,
				*settings.Apiserver.Replicas,
			)
		}

		settings.Apiserver.Replicas = utilpointer.Int32Ptr(int32(replicas))
	}

	if reconcilingDisabled := runOptions.apiServerEndpointReconcilingDisabled; reconcilingDisabled {
		settings.Apiserver.EndpointReconcilingDisabled = &reconcilingDisabled
	}

	if nodePortRange := runOptions.nodePortRange; nodePortRange != "" {
		if settings.Apiserver.NodePortRange != "" && settings.Apiserver.NodePortRange != nodePortRange {
			return settings, fmt.Errorf(
				"conflicting settings, cli option nodeport-range (%v) and field Seed.spec.defaultComponentSettings.apiserver.nodePortRange (%v) do not match",
				nodePortRange,
				settings.Apiserver.NodePortRange,
			)
		}

		settings.Apiserver.NodePortRange = nodePortRange
	}

	if replicas := runOptions.controllerManagerDefaultReplicas; replicas != 0 {
		if settings.ControllerManager.Replicas != nil && replicas != int(*settings.ControllerManager.Replicas) {
			return settings, fmt.Errorf(
				"conflicting settings, cli option controller-manager-default-replicas (%v) and field Seed.spec.defaultComponentSettings.controllerManager.replicas (%v) do not match",
				replicas,
				*settings.ControllerManager.Replicas,
			)
		}

		settings.ControllerManager.Replicas = utilpointer.Int32Ptr(int32(replicas))
	}

	if replicas := runOptions.schedulerDefaultReplicas; replicas != 0 {
		if settings.Scheduler.Replicas != nil && replicas != int(*settings.Scheduler.Replicas) {
			return settings, fmt.Errorf(
				"conflicting settings, cli option schedular-default-replicas (%v) and field Seed.spec.defaultComponentSettings.schedular.replicas (%v) do not match",
				replicas,
				*settings.Scheduler.Replicas,
			)
		}

		settings.Scheduler.Replicas = utilpointer.Int32Ptr(int32(replicas))
	}

	return settings, nil
}

func createCloudController(ctrlCtx *controllerContext) error {
	if err := cloudcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.seedGetter,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.versions,
		ctrlCtx.runOptions.caBundle.CertPool(),
	); err != nil {
		return fmt.Errorf("failed to add cloud controller to mgr: %v", err)
	}
	return nil
}

func createKubernetesController(ctrlCtx *controllerContext) error {
	backupInterval, err := time.ParseDuration(ctrlCtx.runOptions.backupInterval)
	if err != nil {
		return fmt.Errorf("failed to parse %s as duration: %v", ctrlCtx.runOptions.backupInterval, err)
	}

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
		userClusterMLAEnabled(ctrlCtx),
		ctrlCtx.dockerPullConfigJSON,
		ctrlCtx.runOptions.concurrentClusterUpdate,
		ctrlCtx.runOptions.enableEtcdBackupRestoreController,
		backupInterval,
		ctrlCtx.runOptions.oidcIssuerURL,
		ctrlCtx.runOptions.oidcIssuerClientID,
		ctrlCtx.runOptions.kubermaticImage,
		ctrlCtx.runOptions.etcdLauncherImage,
		ctrlCtx.runOptions.dnatControllerImage,
		ctrlCtx.runOptions.machineControllerImageTag,
		ctrlCtx.runOptions.machineControllerImageRepository,
		ctrlCtx.runOptions.tunnelingAgentIP.String(),
		ctrlCtx.runOptions.caBundle,
		kubernetescontroller.Features{
			VPA:                          ctrlCtx.runOptions.featureGates.Enabled(features.VerticalPodAutoscaler),
			EtcdDataCorruptionChecks:     ctrlCtx.runOptions.featureGates.Enabled(features.EtcdDataCorruptionChecks),
			KubernetesOIDCAuthentication: ctrlCtx.runOptions.featureGates.Enabled(features.OpenIDAuthPlugin),
			EtcdLauncher:                 ctrlCtx.runOptions.featureGates.Enabled(features.EtcdLauncher),
			Konnectivity:                 ctrlCtx.runOptions.featureGates.Enabled(features.KonnectivityService),
		},
		ctrlCtx.versions,
	)
}

func createEtcdBackupController(ctrlCtx *controllerContext) error {
	if !ctrlCtx.runOptions.enableEtcdBackupRestoreController {
		return nil
	}
	storeContainer, err := getContainerFromFile(ctrlCtx.runOptions.backupContainerFile)
	if err != nil {
		return err
	}
	var deleteContainer *corev1.Container
	if ctrlCtx.runOptions.backupDeleteContainerFile != "" {
		deleteContainer, err = getContainerFromFile(ctrlCtx.runOptions.backupDeleteContainerFile)
		if err != nil {
			return err
		}
	}
	var cleanupContainer *corev1.Container
	if ctrlCtx.runOptions.cleanupContainerFile != "" {
		cleanupContainer, err = getContainerFromFile(ctrlCtx.runOptions.cleanupContainerFile)
		if err != nil {
			return err
		}
	}
	return etcdbackupcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		storeContainer,
		deleteContainer,
		cleanupContainer,
		ctrlCtx.runOptions.backupContainerImage,
		ctrlCtx.versions,
		ctrlCtx.runOptions.caBundle,
	)
}

func createBackupController(ctrlCtx *controllerContext) error {
	storeContainer, err := getContainerFromFile(ctrlCtx.runOptions.backupContainerFile)
	if err != nil {
		return err
	}
	var cleanupContainer *corev1.Container
	if !ctrlCtx.runOptions.enableEtcdBackupRestoreController {
		cleanupContainer, err = getContainerFromFile(ctrlCtx.runOptions.cleanupContainerFile)
	} else {
		// new backup controller is enabled, this one will only be run to delete any existing backup cronjob.
		// cleanupContainer not needed, just pass an empty dummy
		cleanupContainer = &corev1.Container{}
	}
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
		ctrlCtx.versions,
		ctrlCtx.runOptions.enableEtcdBackupRestoreController,
		ctrlCtx.runOptions.caBundle,
	)
}

func createEtcdRestoreController(ctrlCtx *controllerContext) error {
	if !ctrlCtx.runOptions.enableEtcdBackupRestoreController {
		return nil
	}
	return etcdrestorecontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.versions,
	)
}

func createMonitoringController(ctrlCtx *controllerContext) error {
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
		ctrlCtx.dockerPullConfigJSON,
		ctrlCtx.runOptions.concurrentClusterUpdate,
		monitoring.Features{
			VPA: ctrlCtx.runOptions.featureGates.Enabled(features.VerticalPodAutoscaler),
		},
		ctrlCtx.versions,
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
	return updatecontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.configGetter,
		ctrlCtx.clientProvider,
		ctrlCtx.log,
		ctrlCtx.versions,
	)
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
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.clientProvider,
		ctrlCtx.versions,
	)
}

func createAddonInstallerController(ctrlCtx *controllerContext) error {
	return addoninstaller.Add(
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.kubernetesAddons,
		ctrlCtx.versions,
	)
}

func createRancherController(ctrlCtx *controllerContext) error {
	return rancher.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.clientProvider,
		ctrlCtx.versions,
	)
}

func createPvWatcherController(ctrlCtx *controllerContext) error {
	return pvwatcher.Add(
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
	)
}

func createConstraintTemplateController(ctrlCtx *controllerContext) error {
	return constrainttemplatecontroller.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.clientProvider,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.workerCount,
	)
}

func createInitialMachineDeploymentController(ctrlCtx *controllerContext) error {
	return initialmachinedeployment.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.seedGetter,
		ctrlCtx.clientProvider,
		ctrlCtx.log,
		ctrlCtx.versions,
	)
}

func createMLAController(ctrlCtx *controllerContext) error {
	if !ctrlCtx.runOptions.featureGates.Enabled(features.UserClusterMLA) {
		return nil
	}
	return mla.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.versions,
		ctrlCtx.runOptions.mlaNamespace,
		ctrlCtx.runOptions.grafanaURL,
		ctrlCtx.runOptions.grafanaHeaderName,
		ctrlCtx.runOptions.grafanaSecret,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.cortexAlertmanagerURL,
		ctrlCtx.runOptions.cortexRulerURL,
		ctrlCtx.runOptions.lokiRulerURL,
		ctrlCtx.runOptions.enableUserClusterMLA,
	)
}

func userClusterMLAEnabled(ctrlCtx *controllerContext) bool {
	return ctrlCtx.runOptions.featureGates.Enabled(features.UserClusterMLA) && ctrlCtx.runOptions.enableUserClusterMLA
}

func createConstraintController(ctrlCtx *controllerContext) error {
	return seedconstraintsynchronizer.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.namespace,
		ctrlCtx.runOptions.workerCount,
	)
}

func createClusterTemplateController(ctrlCtx *controllerContext) error {
	return clustertemplatecontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.namespace,
		ctrlCtx.runOptions.workerCount,
	)
}
