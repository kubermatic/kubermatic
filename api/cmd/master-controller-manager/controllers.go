package main

import (
	"context"
	"fmt"

	projectlabelsynchronizer "github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/project-label-synchronizer"
	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac"
	seedproxy "github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/seed-proxy"
	seedsync "github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/seed-sync"
	serviceaccount "github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/service-account"
	userprojectbinding "github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/user-project-binding"
	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/usersshkeyssynchronizer"
	seedcontrollerlifecycle "github.com/kubermatic/kubermatic/api/pkg/controller/shared/seed-controller-lifecycle"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/prometheus/client_golang/prometheus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func createAllControllers(ctrlCtx *controllerContext) error {
	rbacControllerFactory := rbacControllerFactoryCreator(
		ctrlCtx.mgr.GetConfig(),
		ctrlCtx.seedsGetter,
		ctrlCtx.seedKubeconfigGetter,
		ctrlCtx.workerCount,
		ctrlCtx.labelSelectorFunc,
		ctrlCtx.workerNamePredicate,
	)
	projectLabelSynchronizerFactory := projectLabelSynchronizerFactoryCreator(ctrlCtx)
	userSSHKeysSynchronizerFactory := userSSHKeysSynchronizerFactoryCreator(ctrlCtx)

	if err := seedcontrollerlifecycle.Add(ctrlCtx.ctx,
		kubermaticlog.Logger,
		ctrlCtx.mgr,
		ctrlCtx.namespace,
		ctrlCtx.seedsGetter,
		ctrlCtx.seedKubeconfigGetter,
		rbacControllerFactory,
		projectLabelSynchronizerFactory,
		userSSHKeysSynchronizerFactory); err != nil {
		//TODO: Find a better name
		return fmt.Errorf("failed to create seedcontrollerlifecycle: %v", err)
	}
	if err := userprojectbinding.Add(ctrlCtx.mgr); err != nil {
		return fmt.Errorf("failed to create userprojectbinding controller: %v", err)
	}
	if err := serviceaccount.Add(ctrlCtx.mgr); err != nil {
		return fmt.Errorf("failed to create serviceaccount controller: %v", err)
	}
	if err := seedsync.Add(ctrlCtx.ctx, ctrlCtx.mgr, 1, ctrlCtx.log, ctrlCtx.namespace, ctrlCtx.seedKubeconfigGetter); err != nil {
		return fmt.Errorf("failed to create seedsync controller: %v", err)
	}
	if err := seedproxy.Add(ctrlCtx.ctx, ctrlCtx.mgr, 1, ctrlCtx.log, ctrlCtx.namespace, ctrlCtx.seedsGetter, ctrlCtx.seedKubeconfigGetter); err != nil {
		return fmt.Errorf("failed to create seedproxy controller: %v", err)
	}
	return nil
}

func rbacControllerFactoryCreator(
	mastercfg *rest.Config,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	workerCount int,
	selectorOps func(*metav1.ListOptions),
	workerNamePredicate predicate.Predicate,
) seedcontrollerlifecycle.ControllerFactory {
	rbacMetrics := rbac.NewMetrics()
	prometheus.MustRegister(rbacMetrics.Workers)

	return func(ctx context.Context, mgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		ctrl, err := rbac.New(rbacMetrics, mgr, seedManagerMap, selectorOps, workerNamePredicate, workerCount)
		if err != nil {
			return "", fmt.Errorf("failed to create rbac controller: %v", err)
		}
		return "rbac-controller", mgr.Add(ctrl)
	}
}

func projectLabelSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return projectlabelsynchronizer.ControllerName, projectlabelsynchronizer.Add(
			ctx,
			masterMgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerCount,
			ctrlCtx.workerName,
		)
	}
}

func userSSHKeysSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, mgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return usersshkeyssynchronizer.ControllerName, usersshkeyssynchronizer.Add(
			ctx,
			mgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerName,
			ctrlCtx.workerCount,
		)
	}
}
