package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/controller/addon"
	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/backup"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	updatecontroller "github.com/kubermatic/kubermatic/api/pkg/controller/update"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	kuberinformers "k8s.io/client-go/informers"
)

// allControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller
var allControllers = map[string]func(controllerContext) error{
	"cluster": startClusterController,
	"update":  startUpdateController,
	"addon":   startAddonController,
	"backup":  startBackupController,
}

func runAllControllers(ctrlCtx controllerContext) error {

	ctrlCtx.kubermaticInformerFactory = externalversions.NewSharedInformerFactory(ctrlCtx.kubermaticClient, time.Minute*5)
	ctrlCtx.kubeInformerFactory = kuberinformers.NewSharedInformerFactory(ctrlCtx.kubeClient, time.Minute*5)

	for name, startControllerFun := range allControllers {
		glog.Infof("Running %s controller", name)
		err := startControllerFun(ctrlCtx)
		if err != nil {
			return err
		}
	}

	ctrlCtx.kubermaticInformerFactory.Start(ctrlCtx.stopCh)
	ctrlCtx.kubeInformerFactory.Start(ctrlCtx.stopCh)

	<-ctrlCtx.stopCh
	return nil
}

func startClusterController(ctrlCtx controllerContext) error {
	dcs, err := provider.LoadDatacentersMeta(ctrlCtx.runOptions.dcFile)
	if err != nil {
		return err
	}

	cps := cloud.Providers(dcs)

	ctrl, err := cluster.NewController(
		ctrlCtx.kubeClient,
		ctrlCtx.kubermaticClient,
		ctrlCtx.runOptions.externalURL,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.dc,
		dcs,
		cps,
		cluster.NewMetrics(true),
		client.New(ctrlCtx.kubeInformerFactory.Core().V1().Secrets().Lister()),
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodePortRange,
		ctrlCtx.runOptions.nodeAccessNetwork,
		ctrlCtx.runOptions.etcdDiskSize,

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
	)
	if err != nil {
		return err
	}

	go ctrl.Run(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh)
	return nil
}

func startBackupController(ctrlCtx controllerContext) error {
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
		return fmt.Errorf("failed to parse %s as duratation: %v", ctrlCtx.runOptions.backupInterval, err)
	}
	ctrl, err := backupcontroller.New(
		*storeContainer,
		*cleanupContainer,
		backupInterval,
		ctrlCtx.runOptions.backupContainerImage,
		ctrlCtx.runOptions.workerName,
		backupcontroller.NewMetrics(),
		ctrlCtx.kubermaticClient,
		ctrlCtx.kubeClient,
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Clusters(),
		ctrlCtx.kubeInformerFactory.Batch().V1beta1().CronJobs())
	if err != nil {
		return err
	}
	go ctrl.Run(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh)
	return nil
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

func startUpdateController(ctrlCtx controllerContext) error {
	updateManager, err := version.NewFromFiles(ctrlCtx.runOptions.versionsFile, ctrlCtx.runOptions.updatesFile)
	if err != nil {
		return fmt.Errorf("failed to create update manager: %v", err)
	}

	ctrl, err := updatecontroller.New(
		updatecontroller.NewMetrics(),
		updateManager,
		ctrlCtx.runOptions.workerName,
		ctrlCtx.kubermaticClient,
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Clusters())
	if err != nil {
		return err
	}
	go ctrl.Run(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh)
	return nil
}

func startAddonController(ctrlCtx controllerContext) error {
	ctrl, err := addon.New(
		addon.NewMetrics(),
		map[string]interface{}{ // addonVariables
			"openvpn": map[string]interface{}{
				"NodeAccessNetwork": ctrlCtx.runOptions.nodeAccessNetwork,
			},
		},
		ctrlCtx.runOptions.workerName,
		ctrlCtx.runOptions.addons,
		ctrlCtx.runOptions.overwriteRegistry,
		client.New(ctrlCtx.kubeInformerFactory.Core().V1().Secrets().Lister()),
		ctrlCtx.kubermaticClient,
		ctrlCtx.kubeInformerFactory.Core().V1().Secrets(),
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Addons(),
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Clusters())
	if err != nil {
		return err
	}
	go ctrl.Run(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh)
	return nil
}
