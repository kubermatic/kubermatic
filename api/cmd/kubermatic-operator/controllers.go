package main

import (
	"context"
	"fmt"

	"github.com/oklog/run"
	"go.uber.org/zap"

	operatormaster "github.com/kubermatic/kubermatic/api/pkg/controller/operator-master"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type controllerCreator func(*operatorContext, *zap.SugaredLogger) (runnerFn, error)
type runnerFn func(workerCount int, stopCh <-chan struct{}) error

func noop(workerCount int, stopCh <-chan struct{}) error { <-stopCh; return nil }

func controllerLog(log *zap.SugaredLogger, controller string) *zap.SugaredLogger {
	return log.With("controller", controller)
}

// allControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller
var allControllers = map[string]controllerCreator{
	operatormaster.ControllerName: createMasterController,
}

func createAllControllers(opCtx *operatorContext, log *zap.SugaredLogger) (map[string]runnerFn, error) {
	controllers := map[string]runnerFn{}
	for name, create := range allControllers {
		logger := controllerLog(log, name)
		logger.Info("Creating controller")

		controller, err := create(opCtx, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create '%s' controller: %v", name, err)
		}

		// The controllers managed by the mgr don't have a dedicated runner
		if controller != nil {
			controllers[name] = controller
		}
	}
	return controllers, nil
}

func runAllControllersAndCtrlManager(workerCnt int,
	done <-chan struct{},
	cancel context.CancelFunc,
	mgr manager.Runnable,
	controllers map[string]runnerFn,
	log *zap.SugaredLogger) error {
	var g run.Group

	wrapController := func(workerCnt int, done <-chan struct{}, cancel context.CancelFunc, name string, controller runnerFn) (func() error, func(err error)) {
		startControllerWrapped := func() error {
			logger := controllerLog(log, name)
			logger.Info("Starting controller...")
			err := controller(workerCnt, done)
			if err != nil {
				logger.Errorw("Controller died", "error", err)
			} else {
				logger.Infow("Controller finished")
			}
			return err
		}

		cancelControllerFunc := func(err error) {
			logger := controllerLog(log, name)
			logger.Infow("Killing controller as group member finished/died", "error", err)
			cancel()
		}
		return startControllerWrapped, cancelControllerFunc
	}

	// run controller manager
	g.Add(func() error { return mgr.Start(done) }, func(_ error) { cancel() })

	// run controllers
	for name, startController := range controllers {
		startControllerWrapped, cancelControllerFunc := wrapController(workerCnt, done, cancel, name, startController)
		g.Add(startControllerWrapped, cancelControllerFunc)
	}

	return g.Run()
}

func createMasterController(opCtx *operatorContext, log *zap.SugaredLogger) (runnerFn, error) {
	if err := operatormaster.Add(opCtx.mgr, 1, opCtx.clientConfig, log); err != nil {
		return nil, err
	}
	return noop, nil
}
