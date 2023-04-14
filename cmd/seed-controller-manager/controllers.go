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
	"time"

	"github.com/prometheus/client_golang/prometheus"

	addoncontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/addon-controller"
	addoninstallercontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/addon-installer-controller"
	applicationsecretclustercontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/application-secret-cluster-controller"
	autoupdatecontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/auto-update-controller"
	cloudcontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/cloud-controller"
	clustercredentialscontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/cluster-credentials-controller"
	clusterphasecontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/cluster-phase-controller"
	clusterstuckcontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/cluster-stuck-controller"
	clustertemplatecontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/cluster-template-controller"
	clusterupdatecontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/cluster-update-controller"
	clusterusersshkeyscontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/cluster-usersshkeys-controller"
	cniapplicationinstallationcontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/cni-application-installation-controller"
	controlplanecontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/control-plane-controller"
	controlplanestatuscontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/control-plane-status-controller"
	datacenterstatuscontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/datacenter-status-controller"
	initialapplicationinstallationcontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/initial-application-installation-controller"
	initialmachinedeploymentcontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/initial-machinedeployment-controller"
	ipamcontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/ipam-controller"
	kcstatuscontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/kc-status-controller"
	monitoringcontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/monitoring-controller"
	operatingsystemprofilesynchronizer "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/operating-system-profile-synchronizer"
	presetcontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/preset-controller"
	pvwatchercontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/pvwatcher-controller"
	"k8c.io/kubermatic/v3/pkg/features"
)

// AllControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller.
var AllControllers = map[string]controllerCreator{
	addoncontroller.ControllerName:                          createAddonController,
	addoninstallercontroller.ControllerName:                 createAddonInstallerController,
	applicationsecretclustercontroller.ControllerName:       createApplicationSecretClusterController,
	autoupdatecontroller.ControllerName:                     createAutoUpdateController,
	cloudcontroller.ControllerName:                          createCloudController,
	clustercredentialscontroller.ControllerName:             createClusterCredentialsController,
	clusterphasecontroller.ControllerName:                   createClusterPhaseController,
	clusterstuckcontroller.ControllerName:                   createClusterStuckController,
	clustertemplatecontroller.ControllerName:                createClusterTemplateController,
	clusterupdatecontroller.ControllerName:                  createClusterUpdateController,
	clusterusersshkeyscontroller.ControllerName:             createClusterUserSSHKeysController,
	cniapplicationinstallationcontroller.ControllerName:     createCNIApplicationInstallationController,
	controlplanecontroller.ControllerName:                   createControlPlaneController,
	controlplanestatuscontroller.ControllerName:             createControlPlaneStatusController,
	datacenterstatuscontroller.ControllerName:               createDatacenterStatusController,
	initialapplicationinstallationcontroller.ControllerName: createInitialApplicationInstallationController,
	initialmachinedeploymentcontroller.ControllerName:       createInitialMachineDeploymentController,
	ipamcontroller.ControllerName:                           createIPAMController,
	kcstatuscontroller.ControllerName:                       createKcStatusController,
	// mla.ControllerName:                                      createMLAController,
	monitoringcontroller.ControllerName:               createMonitoringController,
	operatingsystemprofilesynchronizer.ControllerName: createOperatingSystemProfileController,
	presetcontroller.ControllerName:                   createPresetController,
	pvwatchercontroller.ControllerName:                createPvWatcherController,
}

type controllerCreator func(*controllerContext) error

func createAllControllers(ctrlCtx *controllerContext) error {
	for name, create := range AllControllers {
		if err := create(ctrlCtx); err != nil {
			return fmt.Errorf("failed to create %q controller: %w", name, err)
		}
	}

	return nil
}

func createControlPlaneStatusController(ctrlCtx *controllerContext) error {
	return controlplanestatuscontroller.Add(
		ctrlCtx.ctx,
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.versions,
	)
}

func createDatacenterStatusController(ctrlCtx *controllerContext) error {
	return datacenterstatuscontroller.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.log,
		ctrlCtx.versions,
	)
}

func createCloudController(ctrlCtx *controllerContext) error {
	cloudcontroller.MustRegisterMetrics(prometheus.DefaultRegisterer)

	if err := cloudcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.datacenterGetter,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.versions,
		ctrlCtx.runOptions.caBundle.CertPool(),
	); err != nil {
		return fmt.Errorf("failed to add cloud controller to mgr: %w", err)
	}
	return nil
}

func createControlPlaneController(ctrlCtx *controllerContext) error {
	backupInterval, err := time.ParseDuration(ctrlCtx.runOptions.backupInterval)
	if err != nil {
		return fmt.Errorf("failed to parse %s as duration: %w", ctrlCtx.runOptions.backupInterval, err)
	}

	return controlplanecontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.externalURL,
		ctrlCtx.datacenterGetter,
		ctrlCtx.configGetter,
		ctrlCtx.clientProvider,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.runOptions.etcdDiskSize,
		userClusterMLAEnabled(ctrlCtx),
		ctrlCtx.dockerPullConfigJSON,
		ctrlCtx.runOptions.concurrentClusterUpdate,
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
		controlplanecontroller.Features{
			EtcdDataCorruptionChecks:     ctrlCtx.runOptions.featureGates.Enabled(features.EtcdDataCorruptionChecks),
			KubernetesOIDCAuthentication: ctrlCtx.runOptions.featureGates.Enabled(features.OpenIDAuthPlugin),
			EtcdLauncher:                 ctrlCtx.runOptions.featureGates.Enabled(features.EtcdLauncher),
		},
		ctrlCtx.versions,
	)
}

func createMonitoringController(ctrlCtx *controllerContext) error {
	return monitoringcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.clientProvider,
		ctrlCtx.configGetter,
		ctrlCtx.datacenterGetter,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.dockerPullConfigJSON,
		ctrlCtx.runOptions.concurrentClusterUpdate,
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

func createClusterUpdateController(ctrlCtx *controllerContext) error {
	return clusterupdatecontroller.Add(
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
	return addoncontroller.Add(
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
		ctrlCtx.runOptions.addonsPath,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.clientProvider,
		ctrlCtx.versions,
	)
}

func createAddonInstallerController(ctrlCtx *controllerContext) error {
	return addoninstallercontroller.Add(
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
	return pvwatchercontroller.Add(
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
	)
}

func createInitialMachineDeploymentController(ctrlCtx *controllerContext) error {
	return initialmachinedeploymentcontroller.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.datacenterGetter,
		ctrlCtx.clientProvider,
		ctrlCtx.log,
		ctrlCtx.versions,
	)
}

// func createMLAController(ctrlCtx *controllerContext) error {
// 	if !ctrlCtx.runOptions.featureGates.Enabled(features.UserClusterMLA) {
// 		return nil
// 	}
// 	return mla.Add(
// 		ctrlCtx.ctx,
// 		ctrlCtx.mgr,
// 		ctrlCtx.log,
// 		ctrlCtx.runOptions.workerCount,
// 		ctrlCtx.runOptions.workerName,
// 		ctrlCtx.versions,
// 		ctrlCtx.runOptions.mlaNamespace,
// 		ctrlCtx.runOptions.grafanaURL,
// 		ctrlCtx.runOptions.grafanaHeaderName,
// 		ctrlCtx.runOptions.grafanaSecret,
// 		ctrlCtx.runOptions.overwriteRegistry,
// 		ctrlCtx.runOptions.cortexAlertmanagerURL,
// 		ctrlCtx.runOptions.cortexRulerURL,
// 		ctrlCtx.runOptions.lokiRulerURL,
// 		ctrlCtx.runOptions.enableUserClusterMLA,
// 	)
// }

func userClusterMLAEnabled(ctrlCtx *controllerContext) bool {
	return ctrlCtx.runOptions.featureGates.Enabled(features.UserClusterMLA) && ctrlCtx.runOptions.enableUserClusterMLA
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

func createIPAMController(ctrlCtx *controllerContext) error {
	return ipamcontroller.Add(
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

func createClusterCredentialsController(ctrlCtx *controllerContext) error {
	return clustercredentialscontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.log,
		ctrlCtx.versions,
	)
}

func createClusterUserSSHKeysController(ctrlCtx *controllerContext) error {
	return clusterusersshkeyscontroller.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.workerCount,
	)
}

func createKcStatusController(ctrlCtx *controllerContext) error {
	return kcstatuscontroller.Add(
		ctrlCtx.ctx,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.log,
		ctrlCtx.runOptions.namespace,
		ctrlCtx.versions,
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
