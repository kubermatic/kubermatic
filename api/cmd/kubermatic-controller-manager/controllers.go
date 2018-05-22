package main

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	rbaccontroller "github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	updatecontroller "github.com/kubermatic/kubermatic/api/pkg/controller/update"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	kuberinformers "k8s.io/client-go/informers"
)

// allControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller
var allControllers = map[string]func(controllerContext) error{
	"cluster":       startClusterController,
	"RBACGenerator": startRBACGeneratorController,
	"update":        startUpdateController,
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

	metrics := NewClusterControllerMetrics()
	clusterMetrics := cluster.ControllerMetrics{
		Clusters:        metrics.Clusters,
		ClusterPhases:   metrics.ClusterPhases,
		Workers:         metrics.Workers,
		UnhandledErrors: metrics.UnhandledErrors,
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
		clusterMetrics,
		client.New(ctrlCtx.kubeInformerFactory.Core().V1().Secrets().Lister()),
		ctrlCtx.runOptions.overwriteRegistry,
		ctrlCtx.runOptions.nodePortRange,

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

func startRBACGeneratorController(ctrlCtx controllerContext) error {
	metrics := NewRBACGeneratorControllerMetrics()
	rbacMetrics := rbaccontroller.Metrics{
		Workers: metrics.Workers,
	}
	ctrl, err := rbaccontroller.New(
		rbacMetrics,
		ctrlCtx.kubermaticClient,
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Projects(),
		ctrlCtx.kubermaticInformerFactory.Kubermatic().V1().Users(),
		ctrlCtx.kubeClient,
		ctrlCtx.kubeInformerFactory.Rbac().V1().ClusterRoles(),
		ctrlCtx.kubeInformerFactory.Rbac().V1().ClusterRoleBindings())
	if err != nil {
		return err
	}
	go ctrl.Run(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh)
	return nil
}

func startUpdateController(ctrlCtx controllerContext) error {
	updateManager, err := version.NewFromFiles(ctrlCtx.runOptions.versionsFile, ctrlCtx.runOptions.updatesFile)
	if err != nil {
		return fmt.Errorf("failed to create update manager: %v", err)
	}

	metrics := NewUpdateControllerMetrics()
	updateMetrics := updatecontroller.Metrics{
		Workers: metrics.Workers,
	}
	ctrl, err := updatecontroller.New(
		updateMetrics,
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
