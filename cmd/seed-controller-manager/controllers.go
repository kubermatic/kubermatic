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
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	addonutil "k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/addon"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/addoninstaller"
	applicationsecretclustercontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/application-secret-cluster-controller"
	autoupdatecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/auto-update-controller"
	cloudcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/cloud"
	clustercredentialscontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/cluster-credentials-controller"
	clusterphasecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/cluster-phase-controller"
	clusterstuckcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/cluster-stuck-controller"
	clustertemplatecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/cluster-template-controller"
	cniapplicationinstallationcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/cni-application-installation-controller"
	seedconstraintsynchronizer "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/constraint-controller"
	constrainttemplatecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/constraint-template-controller"
	defaultapplicationcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/default-application-controller"
	encryptionatrestcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/encryption-at-rest-controller"
	etcdbackupcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/etcdbackup"
	etcdrestorecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/etcdrestore"
	initialapplicationinstallationcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/initial-application-installation-controller"
	initialmachinedeployment "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/initial-machinedeployment-controller"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/ipam"
	kubernetescontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/kubernetes"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/monitoring"
	operatingsystemmanagermigrator "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/operating-system-manager-migrator"
	operatingsystemprofilesynchronizer "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/operating-system-profile-synchronizer"
	presetcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/preset-controller"
	projectcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/project"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/pvwatcher"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/seedresourcesuptodatecondition"
	updatecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/update-controller"
	"k8c.io/kubermatic/v2/pkg/features"
)

// AllControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller.
var AllControllers = map[string]controllerCreator{
	kubernetescontroller.ControllerName:                     createKubernetesController,
	autoupdatecontroller.ControllerName:                     createAutoUpdateController,
	updatecontroller.ControllerName:                         createUpdateController,
	addon.ControllerName:                                    createAddonController,
	addoninstaller.ControllerName:                           createAddonInstallerController,
	etcdbackupcontroller.ControllerName:                     createEtcdBackupController,
	etcdrestorecontroller.ControllerName:                    createEtcdRestoreController,
	monitoring.ControllerName:                               createMonitoringController,
	cloudcontroller.ControllerName:                          createCloudController,
	seedresourcesuptodatecondition.ControllerName:           createSeedConditionUpToDateController,
	pvwatcher.ControllerName:                                createPvWatcherController,
	seedconstraintsynchronizer.ControllerName:               createConstraintController,
	constrainttemplatecontroller.ControllerName:             createConstraintTemplateController,
	initialmachinedeployment.ControllerName:                 createInitialMachineDeploymentController,
	initialapplicationinstallationcontroller.ControllerName: createInitialApplicationInstallationController,
	cniapplicationinstallationcontroller.ControllerName:     createCNIApplicationInstallationController,
	mla.ControllerName:                                      createMLAController,
	clustertemplatecontroller.ControllerName:                createClusterTemplateController,
	projectcontroller.ControllerName:                        createProjectController,
	clusterphasecontroller.ControllerName:                   createClusterPhaseController,
	presetcontroller.ControllerName:                         createPresetController,
	encryptionatrestcontroller.ControllerName:               createEncryptionAtRestController,
	ipam.ControllerName:                                     createIPAMController,
	clusterstuckcontroller.ControllerName:                   createClusterStuckController,
	operatingsystemprofilesynchronizer.ControllerName:       createOperatingSystemProfileController,
	operatingsystemmanagermigrator.ControllerName:           createOperatingSystemManagerMigratorController,
	defaultapplicationcontroller.ControllerName:             createDefaultApplicationController,
	clustercredentialscontroller.ControllerName:             createClusterCredentialsController,
	applicationsecretclustercontroller.ControllerName:       createApplicationSecretClusterController,
}

type controllerCreator func(*controllerContext) error

func createAllControllers(ctrlCtx *controllerContext) error {
	for name, create := range AllControllers {
		if err := create(ctrlCtx); err != nil {
			return fmt.Errorf("failed to create %q controller: %w", name, err)
		}
	}

	// init CE/EE-only controllers
	if err := setupControllers(ctrlCtx); err != nil {
		return err
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

func createCloudController(ctrlCtx *controllerContext) error {
	cloudcontroller.MustRegisterMetrics(prometheus.DefaultRegisterer)

	if err := cloudcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.seedGetter,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.versions,
		ctrlCtx.runOptions.caBundle.CertPool(),
	); err != nil {
		return fmt.Errorf("failed to add cloud controller to mgr: %w", err)
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
		ctrlCtx.configGetter,
		ctrlCtx.clientProvider,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.runOptions.etcdDiskSize,
		userClusterMLAEnabled(ctrlCtx),
		ctrlCtx.dockerPullConfigJSON,
		ctrlCtx.runOptions.concurrentClusterUpdate,
		ctrlCtx.runOptions.backupInterval,
		ctrlCtx.runOptions.backupCount,
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
			DynamicResourceAllocation:    ctrlCtx.runOptions.featureGates.Enabled(features.DynamicResourceAllocation),
		},
		ctrlCtx.versions,
	)
}

func createProjectController(ctrlCtx *controllerContext) error {
	return projectcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
	)
}

func createEtcdBackupController(ctrlCtx *controllerContext) error {
	return etcdbackupcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,

		ctrlCtx.versions,
		ctrlCtx.runOptions.caBundle,
		ctrlCtx.seedGetter,
		ctrlCtx.configGetter,

		ctrlCtx.runOptions.etcdLauncherImage,
		ctrlCtx.runOptions.overwriteRegistry,
	)
}

func createEtcdRestoreController(ctrlCtx *controllerContext) error {
	return etcdrestorecontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.versions,
		ctrlCtx.seedGetter,
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
		ctrlCtx.configGetter,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.dockerPullConfigJSON,
		ctrlCtx.runOptions.concurrentClusterUpdate,
		monitoring.Features{
			VPA: ctrlCtx.runOptions.featureGates.Enabled(features.VerticalPodAutoscaler),
		},
		ctrlCtx.versions,
	)
}

func createAutoUpdateController(ctrlCtx *controllerContext) error {
	return autoupdatecontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.configGetter,
		ctrlCtx.clientProvider,
		ctrlCtx.log,
		ctrlCtx.versions,
	)
}

func createUpdateController(ctrlCtx *controllerContext) error {
	return updatecontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.configGetter,
		ctrlCtx.log,
		ctrlCtx.versions,
	)
}

func createClusterPhaseController(ctrlCtx *controllerContext) error {
	return clusterphasecontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.log,
		ctrlCtx.versions,
	)
}

func createAddonController(ctrlCtx *controllerContext) error {
	allAddons, err := addonutil.LoadAddonsFromDirectory(ctrlCtx.runOptions.addonsPath)
	if err != nil {
		return fmt.Errorf("failed to load addons: %w", err)
	}

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
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.clientProvider,
		ctrlCtx.versions,
		allAddons,
	)
}

func createAddonInstallerController(ctrlCtx *controllerContext) error {
	return addoninstaller.Add(
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.configGetter,
		ctrlCtx.versions,
	)
}

func createInitialApplicationInstallationController(ctrlCtx *controllerContext) error {
	return initialapplicationinstallationcontroller.Add(
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

func createCNIApplicationInstallationController(ctrlCtx *controllerContext) error {
	return cniapplicationinstallationcontroller.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.systemAppEnforceInterval,
		ctrlCtx.clientProvider,
		ctrlCtx.log,
		ctrlCtx.versions,
		ctrlCtx.runOptions.overwriteRegistry,
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

func createPresetController(ctrlCtx *controllerContext) error {
	return presetcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.workerCount,
	)
}

func createEncryptionAtRestController(ctrlCtx *controllerContext) error {
	return encryptionatrestcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.clientProvider,
		ctrlCtx.seedGetter,
		ctrlCtx.configGetter,
		ctrlCtx.versions,
		ctrlCtx.runOptions.overwriteRegistry,
	)
}

func createIPAMController(ctrlCtx *controllerContext) error {
	return ipam.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.configGetter,
		ctrlCtx.versions,
	)
}

func createClusterStuckController(ctrlCtx *controllerContext) error {
	if !ctrlCtx.runOptions.featureGates.Enabled(features.DevelopmentEnvironment) {
		return nil
	}

	return clusterstuckcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.log,
	)
}

func createOperatingSystemProfileController(ctrlCtx *controllerContext) error {
	return operatingsystemprofilesynchronizer.Add(
		ctrlCtx.mgr,
		ctrlCtx.clientProvider,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.namespace,
		ctrlCtx.runOptions.workerCount,
	)
}

func createOperatingSystemManagerMigratorController(ctrlCtx *controllerContext) error {
	return operatingsystemmanagermigrator.Add(
		ctrlCtx.mgr,
		ctrlCtx.clientProvider,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.versions,
	)
}

func createDefaultApplicationController(ctrlCtx *controllerContext) error {
	return defaultapplicationcontroller.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.seedGetter,
		ctrlCtx.configGetter,
		ctrlCtx.clientProvider,
		ctrlCtx.log,
		ctrlCtx.versions,
	)
}

func createClusterCredentialsController(ctrlCtx *controllerContext) error {
	return clustercredentialscontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.log,
		ctrlCtx.versions,
		ctrlCtx.runOptions.namespace,
	)
}

func createApplicationSecretClusterController(ctrlCtx *controllerContext) error {
	return applicationsecretclustercontroller.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.namespace,
	)
}
