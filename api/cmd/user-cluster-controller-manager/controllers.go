package main

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/oklog/run"

	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster"
)

// allUserClusterControllers stores the list of all controllers to be
// run in the user-cluster.
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller
var allUserClusterControllers = map[string]controllerCreator{
	"UserCluster": createUserClusterController,
}

type controllerCreator func(*controllerContext) (runner, error)

type runner interface {
	Run(workerCount int, stopCh <-chan struct{})
}

func createAllUserClusterControllers(ctrlCtx *controllerContext) (map[string]runner, error) {
	controllers := map[string]runner{}
	for name, create := range allUserClusterControllers {
		controller, err := create(ctrlCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create '%s' user-cluster-controller: %v", name, err)
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

func createUserClusterController(ctrlCtx *controllerContext) (runner, error) {

	return usercluster.NewController(
		ctrlCtx.kubeClient,
		ctrlCtx.kubeInformerFactory.Core().V1().ConfigMaps())
}
