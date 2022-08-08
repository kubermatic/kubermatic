/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	grafanaDashboardsConfigmapNamePrefix = "grafana-dashboards"
)

// dashboardGrafanaReconciler stores necessary components that are required to manage MLA(Monitoring, Logging, and Alerting) setup.
type dashboardGrafanaReconciler struct {
	ctrlruntimeclient.Client

	log                        *zap.SugaredLogger
	workerName                 string
	recorder                   record.EventRecorder
	versions                   kubermatic.Versions
	dashboardGrafanaController *dashboardGrafanaController
}

// Add creates a new MLA controller that is responsible for
// managing Monitoring, Logging and Alerting for user clusters.
func newDashboardGrafanaReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	dashboardGrafanaController *dashboardGrafanaController,
) error {
	client := mgr.GetClient()

	reconciler := &dashboardGrafanaReconciler{
		Client: client,

		log:                        log.Named("grafana-dashboard"),
		workerName:                 workerName,
		recorder:                   mgr.GetEventRecorderFor(ControllerName),
		versions:                   versions,
		dashboardGrafanaController: dashboardGrafanaController,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	enqueueGrafanaConfigMap := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		if !strings.HasPrefix(a.GetName(), grafanaDashboardsConfigmapNamePrefix) {
			return []reconcile.Request{}
		}
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: a.GetName(), Namespace: a.GetNamespace()}}}
	})

	if err := c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, enqueueGrafanaConfigMap, predicateutil.ByNamespace(dashboardGrafanaController.mlaNamespace)); err != nil {
		return fmt.Errorf("failed to watch ConfigMap: %w", err)
	}

	return err
}

func (r *dashboardGrafanaReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, request.NamespacedName, configMap); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	grafanaClient, err := r.dashboardGrafanaController.clientProvider(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	if !configMap.DeletionTimestamp.IsZero() {
		if err := r.dashboardGrafanaController.handleDeletion(ctx, log, configMap, grafanaClient); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if grafanaClient == nil {
		return reconcile.Result{}, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r, configMap, mlaFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := r.dashboardGrafanaController.ensureDashboards(ctx, log, configMap, grafanaClient); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana Dashboards: %w", err)
	}

	return reconcile.Result{}, nil
}

type dashboardGrafanaController struct {
	ctrlruntimeclient.Client
	clientProvider grafanaClientProvider
	mlaNamespace   string

	log *zap.SugaredLogger
}

func newDashboardGrafanaController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	mlaNamespace string,
	clientProvider grafanaClientProvider,
) *dashboardGrafanaController {
	return &dashboardGrafanaController{
		Client:         client,
		clientProvider: clientProvider,
		mlaNamespace:   mlaNamespace,

		log: log,
	}
}

func (r *dashboardGrafanaController) CleanUp(ctx context.Context) error {
	configMapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configMapList, ctrlruntimeclient.InNamespace(r.mlaNamespace)); err != nil {
		return fmt.Errorf("Failed to list configmaps: %w", err)
	}
	grafanaClient, err := r.clientProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}
	for _, configMap := range configMapList.Items {
		if !strings.HasPrefix(configMap.GetName(), grafanaDashboardsConfigmapNamePrefix) {
			continue
		}
		if err := r.handleDeletion(ctx, r.log, &configMap, grafanaClient); err != nil {
			return fmt.Errorf("handling deletion: %w", err)
		}
	}
	return nil
}

func (r *dashboardGrafanaController) handleDeletion(ctx context.Context, log *zap.SugaredLogger, configMap *corev1.ConfigMap, grafanaClient *grafanasdk.Client) error {
	if grafanaClient != nil {
		projectList := &kubermaticv1.ProjectList{}
		if err := r.List(ctx, projectList); err != nil {
			return fmt.Errorf("failed to list Projects: %w", err)
		}
		for _, project := range projectList.Items {
			orgID, ok := project.GetAnnotations()[GrafanaOrgAnnotationKey]
			if !ok {
				// looks like corresponding Grafana Org already remove, so we can skip this project
				log.Debugf("project %+v doesn't have grafana org annotation, skipping", project)
				continue
			}
			id, err := strconv.ParseUint(orgID, 10, 32)
			if err != nil {
				return fmt.Errorf("unable to parse grafana org annotation %s: %w", orgID, err)
			}
			if err := deleteDashboards(ctx, log, grafanaClient.WithOrgIDHeader(uint(id)), configMap); err != nil {
				return err
			}
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r, configMap, mlaFinalizer)
}

func (r *dashboardGrafanaController) ensureDashboards(ctx context.Context, log *zap.SugaredLogger, configMap *corev1.ConfigMap, grafanaClient *grafanasdk.Client) error {
	projectList := &kubermaticv1.ProjectList{}
	if err := r.List(ctx, projectList); err != nil {
		return fmt.Errorf("failed to list Projects: %w", err)
	}
	for _, project := range projectList.Items {
		orgID, ok := project.GetAnnotations()[GrafanaOrgAnnotationKey]
		if !ok {
			// looks like this project doesn't have corresponding Grafana Organization yet,
			// we can skip it for now and it will be reconciled by org_grafana_controller later
			log.Debugf("project %+v doesn't have grafana org annotation, skipping", project)
			continue
		}
		id, err := strconv.ParseUint(orgID, 10, 32)
		if err != nil {
			return fmt.Errorf("unable to parse grafana org annotation %s: %w", orgID, err)
		}
		if err := addDashboards(ctx, log, grafanaClient.WithOrgIDHeader(uint(id)), configMap); err != nil {
			return err
		}
	}
	return nil
}
