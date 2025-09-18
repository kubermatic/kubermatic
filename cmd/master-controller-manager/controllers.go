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
	"go.uber.org/zap"

	applicationdefinitionsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/application-definition-synchronizer"
	applicationsecretsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/application-secret-synchronizer"
	clustertemplatesynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/cluster-template-synchronizer"
	encryptionsecretsynchonizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/encryption-secret-synchronizer"
	externalcluster "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/external-cluster"
	kcstatuscontroller "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/kc-status-controller"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/kubeone"
	masterconstraintsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/master-constraint-controller"
	masterconstrainttemplatecontroller "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/master-constraint-template-controller"
	policytemplatesynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/policy-template-synchronizer"
	presetsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/preset-synchronizer"
	projectlabelsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/project-label-synchronizer"
	projectsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/project-synchronizer"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	seedproxy "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/seed-proxy"
	seedstatuscontroller "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/seed-status-controller"
	seedsync "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/seed-sync"
	serviceaccount "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/serviceaccount-projectbinding-controller"
	userprojectbinding "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/user-project-binding"
	userprojectbindingsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/user-project-binding-synchronizer"
	usersynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/user-synchronizer"
	usersshkeyprojectownershipcontroller "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/usersshkey-project-ownership"
	usersshkeysynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/usersshkey-synchronizer"
	seedcontrollerlifecycle "k8c.io/kubermatic/v2/pkg/controller/shared/seed-controller-lifecycle"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func createAllControllers(ctrlCtx *controllerContext) error {
	rbacControllerFactory := rbacControllerFactoryCreator(
		ctrlCtx.mgr.GetConfig(),
		ctrlCtx.log,
		ctrlCtx.seedsGetter,
		ctrlCtx.seedKubeconfigGetter,
		ctrlCtx.workerCount,
		ctrlCtx.labelSelectorFunc,
		ctrlCtx.workerNamePredicate,
	)

	controllerFactories := []seedcontrollerlifecycle.ControllerFactory{
		rbacControllerFactory,
		projectLabelSynchronizerFactoryCreator(ctrlCtx),
		masterConstraintSynchronizerFactoryCreator(ctrlCtx),
		masterConstraintTemplateSynchronizerFactoryCreator(ctrlCtx),
		userSynchronizerFactoryCreator(ctrlCtx),
		clusterTemplateSynchronizerFactoryCreator(ctrlCtx),
		userProjectBindingSynchronizerFactoryCreator(ctrlCtx),
		projectSynchronizerFactoryCreator(ctrlCtx),
		applicationDefinitionSynchronizerFactoryCreator(ctrlCtx),
		applicationSecretSynchronizerFactoryCreator(ctrlCtx),
		presetSynchronizerFactoryCreator(ctrlCtx),
		resourceQuotaSynchronizerFactoryCreator(ctrlCtx),
		resourceQuotaControllerFactoryCreator(ctrlCtx),
		policyTemplateSynchronizerFactoryCreator(ctrlCtx),
		encryptionSecretSynchronizerFactoryCreator(ctrlCtx),
	}

	// Create the user-ssh-key-synchronizer controller even DisableUserSSHKey feature is set
	// to allow it to handle finalizers during Cluster deletion.
	controllerFactories = append(controllerFactories, userSSHKeySynchronizerFactoryCreator(ctrlCtx))
	controllerFactories = append(controllerFactories, setupLifecycleControllerCreators(ctrlCtx)...)

	if err := seedcontrollerlifecycle.Add(ctrlCtx.ctx,
		ctrlCtx.log,
		ctrlCtx.mgr,
		ctrlCtx.namespace,
		ctrlCtx.seedsGetter,
		ctrlCtx.seedKubeconfigGetter,
		controllerFactories...,
	); err != nil {
		//TODO: Find a better name
		return fmt.Errorf("failed to create seedcontrollerlifecycle: %w", err)
	}
	if err := userprojectbinding.Add(ctrlCtx.ctx, ctrlCtx.mgr, ctrlCtx.log); err != nil {
		return fmt.Errorf("failed to create user-project-binding controller: %w", err)
	}

	if !ctrlCtx.featureGates[features.DisableUserSSHKey] {
		if err := usersshkeyprojectownershipcontroller.Add(ctrlCtx.mgr, ctrlCtx.log); err != nil {
			return fmt.Errorf("failed to create usersshkey-project-ownership controller: %w", err)
		}
	}

	if err := serviceaccount.Add(ctrlCtx.mgr, ctrlCtx.log); err != nil {
		return fmt.Errorf("failed to create serviceaccount controller: %w", err)
	}
	if err := seedstatuscontroller.Add(ctrlCtx.ctx, ctrlCtx.mgr, 1, ctrlCtx.log, ctrlCtx.namespace, ctrlCtx.seedKubeconfigGetter, ctrlCtx.versions); err != nil {
		return fmt.Errorf("failed to create seed status controller: %w", err)
	}
	if err := seedsync.Add(ctrlCtx.mgr, 1, ctrlCtx.log, ctrlCtx.namespace, ctrlCtx.seedKubeconfigGetter, ctrlCtx.seedsGetter); err != nil {
		return fmt.Errorf("failed to create seedsync controller: %w", err)
	}
	if err := seedproxy.Add(ctrlCtx.mgr, 1, ctrlCtx.log, ctrlCtx.namespace, ctrlCtx.seedsGetter, ctrlCtx.seedKubeconfigGetter, ctrlCtx.configGetter); err != nil {
		return fmt.Errorf("failed to create seedproxy controller: %w", err)
	}
	if err := externalcluster.Add(ctrlCtx.ctx, ctrlCtx.mgr, ctrlCtx.log); err != nil {
		return fmt.Errorf("failed to create external cluster controller: %w", err)
	}
	if err := kubeone.Add(ctrlCtx.ctx, ctrlCtx.mgr, ctrlCtx.log, ctrlCtx.overwriteRegistry); err != nil {
		return fmt.Errorf("failed to create kubeone controller: %w", err)
	}
	if err := kcstatuscontroller.Add(ctrlCtx.ctx, ctrlCtx.mgr, 1, ctrlCtx.log, ctrlCtx.namespace, ctrlCtx.versions); err != nil {
		return fmt.Errorf("failed to create kubermatic configuration controller: %w", err)
	}

	// init CE/EE-only controllers
	if err := setupControllers(ctrlCtx); err != nil {
		return err
	}

	return nil
}

func rbacControllerFactoryCreator(
	mastercfg *rest.Config,
	log *zap.SugaredLogger,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	workerCount int,
	selectorOps func(*metav1.ListOptions),
	workerNamePredicate predicate.Predicate,
) seedcontrollerlifecycle.ControllerFactory {
	rbacMetrics := rbac.NewMetrics()
	prometheus.MustRegister(rbacMetrics.Workers)

	return func(ctx context.Context, mgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		_, err := rbac.New(ctx, rbacMetrics, mgr, seedManagerMap, log, selectorOps, workerNamePredicate, workerCount)
		if err != nil {
			return "", fmt.Errorf("failed to create rbac controller: %w", err)
		}
		return "rbac-controller", nil
	}
}

func projectLabelSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return projectlabelsynchronizer.ControllerName, projectlabelsynchronizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerCount,
			ctrlCtx.workerName,
		)
	}
}

func userSSHKeySynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return usersshkeysynchronizer.ControllerName, usersshkeysynchronizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerName,
			ctrlCtx.workerCount,
			ctrlCtx.featureGates.Enabled(features.DisableUserSSHKey),
		)
	}
}

func masterConstraintSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return masterconstraintsynchronizer.ControllerName, masterconstraintsynchronizer.Add(
			masterMgr,
			ctrlCtx.namespace,
			seedManagerMap,
			ctrlCtx.log,
		)
	}
}

func masterConstraintTemplateSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return masterconstrainttemplatecontroller.ControllerName, masterconstrainttemplatecontroller.Add(
			masterMgr,
			ctrlCtx.log,
			1,
			ctrlCtx.namespace,
			seedManagerMap,
		)
	}
}

func userSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return usersynchronizer.ControllerName, usersynchronizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerCount,
		)
	}
}

func clusterTemplateSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return clustertemplatesynchronizer.ControllerName, clustertemplatesynchronizer.Add(
			masterMgr,
			ctrlCtx.seedsGetter,
			seedManagerMap,
			ctrlCtx.log,
		)
	}
}

func userProjectBindingSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return userprojectbindingsynchronizer.ControllerName, userprojectbindingsynchronizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerCount,
		)
	}
}

func projectSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return projectsynchronizer.ControllerName, projectsynchronizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerCount,
		)
	}
}

func applicationDefinitionSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return applicationdefinitionsynchronizer.ControllerName, applicationdefinitionsynchronizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerCount,
		)
	}
}

func applicationSecretSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return applicationsecretsynchronizer.ControllerName, applicationsecretsynchronizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.namespace,
			ctrlCtx.log,
			ctrlCtx.workerCount,
		)
	}
}

func presetSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return presetsynchronizer.ControllerName, presetsynchronizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.log,
		)
	}
}

func policyTemplateSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return policytemplatesynchronizer.ControllerName, policytemplatesynchronizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerCount,
		)
	}
}

func encryptionSecretSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(ctx context.Context, masterMgr manager.Manager, seedManagerMap map[string]manager.Manager) (string, error) {
		return encryptionsecretsynchonizer.ControllerName, encryptionsecretsynchonizer.Add(
			masterMgr,
			seedManagerMap,
			ctrlCtx.namespace,
			ctrlCtx.log,
			ctrlCtx.workerName,
			ctrlCtx.workerCount,
		)
	}
}
