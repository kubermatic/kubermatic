package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/addon"
	"github.com/kubermatic/kubermatic/api/pkg/controller/addoninstaller"
	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/backup"
	cloudcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/controller/clustercomponentdefaulter"
	"github.com/kubermatic/kubermatic/api/pkg/controller/monitoring"
	openshiftcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/openshift"
	updatecontroller "github.com/kubermatic/kubermatic/api/pkg/controller/update"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	utilpointer "k8s.io/utils/pointer"
)

// allControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller
var allControllers = map[string]controllerCreator{
	cluster.ControllerName:                   createClusterController,
	updatecontroller.ControllerName:          createUpdateController,
	addon.ControllerName:                     createAddonController,
	addoninstaller.ControllerName:            createAddonInstallerController,
	backupcontroller.ControllerName:          createBackupController,
	monitoring.ControllerName:                createMonitoringController,
	cloudcontroller.ControllerName:           createCloudController,
	openshiftcontroller.ControllerName:       createOpenshiftController,
	clustercomponentdefaulter.ControllerName: createClusterComponentDefaulter,
}

type controllerCreator func(*controllerContext) error

func createAllControllers(ctrlCtx *controllerContext) error {
	for name, create := range allControllers {
		if err := create(ctrlCtx); err != nil {
			return fmt.Errorf("failed to create %q controller: %v", name, err)
		}
	}
	return nil
}

func createClusterComponentDefaulter(ctrlCtx *controllerContext) error {
	predicates := workerlabel.Predicates(ctrlCtx.runOptions.workerName)
	defaultCompontentsOverrides := kubermaticv1.ComponentSettings{
		Apiserver: kubermaticv1.DeploymentSettings{
			Replicas: utilpointer.Int32Ptr(int32(ctrlCtx.runOptions.apiServerDefaultReplicas))},
		ControllerManager: kubermaticv1.DeploymentSettings{
			Replicas: utilpointer.Int32Ptr(int32(ctrlCtx.runOptions.controllerManagerDefaultReplicas))},
		Scheduler: kubermaticv1.DeploymentSettings{
			Replicas: utilpointer.Int32Ptr(int32(ctrlCtx.runOptions.schedulerDefaultReplicas))},
	}
	return clustercomponentdefaulter.Add(
		context.Background(), ctrlCtx.log, ctrlCtx.mgr, ctrlCtx.runOptions.workerCount, defaultCompontentsOverrides, predicates)
}

func createCloudController(ctrlCtx *controllerContext) error {
	predicates := workerlabel.Predicates(ctrlCtx.runOptions.workerName)
	if err := cloudcontroller.Add(ctrlCtx.mgr, ctrlCtx.runOptions.workerCount, ctrlCtx.seedGetter, predicates); err != nil {
		return fmt.Errorf("failed to add cloud controller to mgr: %v", err)
	}
	return nil
}

func createOpenshiftController(ctrlCtx *controllerContext) error {
	if err := openshiftcontroller.Add(
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.seedGetter,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.runOptions.etcdDiskSize,
		ctrlCtx.dockerPullConfigJSON,
		ctrlCtx.runOptions.externalURL,
		openshiftcontroller.OIDCConfig{
			CAFile:       ctrlCtx.runOptions.oidcCAFile,
			ClientID:     ctrlCtx.runOptions.oidcIssuerClientID,
			ClientSecret: ctrlCtx.runOptions.oidcIssuerClientSecret,
			IssuerURL:    ctrlCtx.runOptions.oidcIssuerURL,
		},
		ctrlCtx.runOptions.kubermaticImage,
		openshiftcontroller.Features{
			EtcdDataCorruptionChecks: ctrlCtx.runOptions.featureGates.Enabled(EtcdDataCorruptionChecks),
			VPA:                      ctrlCtx.runOptions.featureGates.Enabled(VerticalPodAutoscaler),
		}); err != nil {
		return fmt.Errorf("failed to add openshift controller to mgr: %v", err)
	}
	return nil
}

func createClusterController(ctrlCtx *controllerContext) error {
	return cluster.Add(
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
		strings.Contains(ctrlCtx.runOptions.kubernetesAddonsList, "nodelocal-dns-cache"),
		ctrlCtx.runOptions.oidcCAFile,
		ctrlCtx.runOptions.oidcIssuerURL,
		ctrlCtx.runOptions.oidcIssuerClientID,
		ctrlCtx.runOptions.kubermaticImage,
		cluster.Features{
			VPA:                      ctrlCtx.runOptions.featureGates.Enabled(VerticalPodAutoscaler),
			EtcdDataCorruptionChecks: ctrlCtx.runOptions.featureGates.Enabled(EtcdDataCorruptionChecks),
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
		strings.Contains(ctrlCtx.runOptions.kubernetesAddonsList, "nodelocal-dns-cache"),

		monitoring.Features{
			VPA: ctrlCtx.runOptions.featureGates.Enabled(VerticalPodAutoscaler),
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

	return updatecontroller.Add(ctrlCtx.mgr, ctrlCtx.runOptions.workerCount, ctrlCtx.runOptions.workerName, updateManager)
}

func createAddonController(ctrlCtx *controllerContext) error {
	return addon.Add(
		ctrlCtx.mgr,
		ctrlCtx.log,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		map[string]interface{}{ // addonVariables
			"openvpn": map[string]interface{}{
				"NodeAccessNetwork": ctrlCtx.runOptions.nodeAccessNetwork,
			},
		},
		ctrlCtx.runOptions.kubernetesAddonsPath,
		ctrlCtx.runOptions.openshiftAddonsPath,
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.clientProvider,
	)
}

func createAddonInstallerController(ctrlCtx *controllerContext) error {

	kubernetesAddons := strings.Split(ctrlCtx.runOptions.kubernetesAddonsList, ",")
	for i, a := range kubernetesAddons {
		kubernetesAddons[i] = strings.TrimSpace(a)
	}

	openshiftAddons := strings.Split(ctrlCtx.runOptions.openshiftAddonsList, ",")
	for i, a := range openshiftAddons {
		openshiftAddons[i] = strings.TrimSpace(a)
	}

	return addoninstaller.Add(
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.runOptions.workerCount,
		ctrlCtx.runOptions.workerName,
		kubernetesAddons,
		openshiftAddons)
}
