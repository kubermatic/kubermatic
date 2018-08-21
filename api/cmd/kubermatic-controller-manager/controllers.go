package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/controller/addon"
	"github.com/kubermatic/kubermatic/api/pkg/controller/addoninstaller"
	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/backup"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	updatecontroller "github.com/kubermatic/kubermatic/api/pkg/controller/update"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	"github.com/oklog/run"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// allControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller
var allControllers = map[string]controllerCreator{
	"Cluster":        createClusterController,
	"Update":         createUpdateController,
	"Addon":          createAddonController,
	"AddonInstaller": createAddonInstallerController,
	"Backup":         createBackupController,
}

type controllerCreator func(*controllerContext) (runner, error)

type runner interface {
	Run(workerCount int, stopCh <-chan struct{})
}

func createAllControllers(ctrlCtx *controllerContext) (map[string]runner, error) {
	controllers := map[string]runner{}
	for name, create := range allControllers {
		controller, err := create(ctrlCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create '%s' controller: %v", name, err)
		}
		controllers[name] = controller
	}
	return controllers, nil
}

func getControllerStarter(workerCnt int, done <-chan struct{}, cancel context.CancelFunc, name string, controller runner) (func() error, func(err error)) {
	execute := func() error {
		glog.V(2).Infof("Starting %s controller...", name)
		controller.Run(workerCnt, done)

		err := fmt.Errorf("%s controller finished/died", name)
		glog.V(2).Info(err)
		return err
	}

	interrupt := func(err error) {
		glog.V(2).Infof("Killing %s controller as group member finished/died: %v", name, err)
		cancel()
	}
	return execute, interrupt
}

func runAllControllers(workerCnt int, done <-chan struct{}, cancel context.CancelFunc, controllers map[string]runner) error {
	var g run.Group

	for name, controller := range controllers {
		execute, interrupt := getControllerStarter(workerCnt, done, cancel, name, controller)
		g.Add(execute, interrupt)
	}

	return g.Run()
}

func createClusterController(ctrlCtx *controllerContext) (runner, error) {
	dcs, err := provider.LoadDatacentersMeta(ctrlCtx.runOptions.dcFile)
	if err != nil {
		return nil, err
	}

	cps := cloud.Providers(dcs)

	return cluster.NewController(
		ctrlCtx.kubeClient,
		ctrlCtx.kubermaticClient,
		ctrlCtx.runOptions.externalURL,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.dc,
		dcs,
		cps,
		client.New(ctrlCtx.kubeInformerFactory.Core().V1().Secrets().Lister()),
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodePortRange,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.runOptions.etcdDiskSize,
		ctrlCtx.runOptions.inClusterPrometheusRulesFile,
		ctrlCtx.runOptions.inClusterPrometheusDisableDefaultRules,

		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Clusters(),
		ctrlCtx.kubeInformerFactory.Core().V1().Namespaces(),
		ctrlCtx.kubeInformerFactory.Core().V1().Secrets(),
		ctrlCtx.kubeInformerFactory.Core().V1().Services(),
		ctrlCtx.kubeInformerFactory.Core().V1().PersistentVolumeClaims(),
		ctrlCtx.kubeInformerFactory.Core().V1().ConfigMaps(),
		ctrlCtx.kubeInformerFactory.Core().V1().ServiceAccounts(),
		ctrlCtx.kubeInformerFactory.Apps().V1().Deployments(),
		ctrlCtx.kubeInformerFactory.Apps().V1().StatefulSets(),
		ctrlCtx.kubeInformerFactory.Extensions().V1beta1().Ingresses(),
		ctrlCtx.kubeInformerFactory.Rbac().V1().Roles(),
		ctrlCtx.kubeInformerFactory.Rbac().V1().RoleBindings(),
		ctrlCtx.kubeInformerFactory.Rbac().V1().ClusterRoleBindings(),
		ctrlCtx.kubeInformerFactory.Policy().V1beta1().PodDisruptionBudgets(),
	)
}

func createBackupController(ctrlCtx *controllerContext) (runner, error) {
	storeContainer, err := getContainerFromFile(ctrlCtx.runOptions.backupContainerFile)
	if err != nil {
		return nil, err
	}
	cleanupContainer, err := getContainerFromFile(ctrlCtx.runOptions.cleanupContainerFile)
	if err != nil {
		return nil, err
	}
	backupInterval, err := time.ParseDuration(ctrlCtx.runOptions.backupInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s as duration: %v", ctrlCtx.runOptions.backupInterval, err)
	}
	return backupcontroller.New(
		*storeContainer,
		*cleanupContainer,
		backupInterval,
		ctrlCtx.runOptions.backupContainerImage,
		ctrlCtx.runOptions.workerName,
		backupcontroller.NewMetrics(),
		ctrlCtx.kubermaticClient,
		ctrlCtx.kubeClient,
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Clusters(),
		ctrlCtx.kubeInformerFactory.Batch().V1beta1().CronJobs(),
		ctrlCtx.kubeInformerFactory.Batch().V1().Jobs(),
		ctrlCtx.kubeInformerFactory.Core().V1().Secrets(),
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

func createUpdateController(ctrlCtx *controllerContext) (runner, error) {
	updateManager, err := version.NewFromFiles(ctrlCtx.runOptions.versionsFile, ctrlCtx.runOptions.updatesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create update manager: %v", err)
	}

	return updatecontroller.New(
		updatecontroller.NewMetrics(),
		updateManager,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.kubermaticClient,
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Clusters(),
	)
}

func createAddonController(ctrlCtx *controllerContext) (runner, error) {
	return addon.New(
		addon.NewMetrics(),
		map[string]interface{}{ // addonVariables
			"openvpn": map[string]interface{}{
				"NodeAccessNetwork": ctrlCtx.runOptions.nodeAccessNetwork,
			},
		},
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.addonsPath,
		ctrlCtx.runOptions.overwriteRegistry,
		client.New(ctrlCtx.kubeInformerFactory.Core().V1().Secrets().Lister()),
		ctrlCtx.kubermaticClient,
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Addons(),
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Clusters(),
	)
}

func createAddonInstallerController(ctrlCtx *controllerContext) (runner, error) {

	defaultAddonsList := strings.Split(ctrlCtx.runOptions.addonsList, ",")
	for i, a := range defaultAddonsList {
		defaultAddonsList[i] = strings.TrimSpace(a)
	}

	return addoninstaller.New(
		addoninstaller.NewMetrics(),
		ctrlCtx.runOptions.workerName,
		defaultAddonsList,
		ctrlCtx.kubermaticClient,
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Addons(),
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Clusters(),
	)
}
