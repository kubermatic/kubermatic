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
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	externalcluster "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/external-cluster"
	masterconstraintsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/master-constraint-controller"
	masterconstrainttemplatecontroller "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/master-constraint-template-controller"
	projectlabelsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/project-label-synchronizer"
	projectsync "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/project-sync"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	seedproxy "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/seed-proxy"
	seedsync "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/seed-sync"
	serviceaccount "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/service-account"
	userprojectbinding "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/user-project-binding"
	userprojectbindingsync "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/user-project-binding-sync"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/usersshkeyssynchronizer"
	seedcontrollerlifecycle "k8c.io/kubermatic/v2/pkg/controller/shared/seed-controller-lifecycle"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"

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
	if err := externalcluster.Add(ctrlCtx.ctx, ctrlCtx.mgr, ctrlCtx.log); err != nil {
		return fmt.Errorf("failed to create external cluster controller: %v", err)
	}
	if err := masterconstrainttemplatecontroller.Add(ctrlCtx.ctx, ctrlCtx.mgr, ctrlCtx.log, 1, ctrlCtx.namespace, ctrlCtx.seedKubeconfigGetter); err != nil {
		return fmt.Errorf("failed to create master constraint template controller: %v", err)
	}
	if err := masterconstraintsynchronizer.Add(ctrlCtx.ctx, ctrlCtx.mgr, ctrlCtx.log, ctrlCtx.namespace, ctrlCtx.seedKubeconfigGetter); err != nil {
		return fmt.Errorf("failed to create master constraints synchronisation controller: %v", err)
	}
	if err := projectsync.Add(ctrlCtx.mgr, ctrlCtx.log, 1, ctrlCtx.seedKubeconfigGetter); err != nil {
		return fmt.Errorf("failed to create projectsync controller: %v", err)
	}
	if err := userprojectbindingsync.Add(ctrlCtx.mgr, ctrlCtx.log, 1, ctrlCtx.seedKubeconfigGetter); err != nil {
		return fmt.Errorf("failed to create userprojectbindingsync controller: %v", err)
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
		_, err := rbac.New(ctx, rbacMetrics, mgr, seedManagerMap, selectorOps, workerNamePredicate, workerCount)
		if err != nil {
			return "", fmt.Errorf("failed to create rbac controller: %v", err)
		}
		return "rbac-controller", nil
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
