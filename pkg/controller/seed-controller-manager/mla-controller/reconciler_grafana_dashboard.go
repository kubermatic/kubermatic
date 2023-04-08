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

package mlacontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"
	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v3/pkg/kubernetes"

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

// Create/delete Grafana dashboards based on configmaps with prefix `grafana-dashboards`.
type grafanaDashboardReconciler struct {
	seedClient     ctrlruntimeclient.Client
	log            *zap.SugaredLogger
	workerName     string
	recorder       record.EventRecorder
	clientProvider grafana.ClientProvider
	mlaNamespace   string
}

var _ cleaner = &grafanaDashboardReconciler{}

func newGrafanaDashboardReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	clientProvider grafana.ClientProvider,
	mlaNamespace string,
) *grafanaDashboardReconciler {
	return &grafanaDashboardReconciler{
		seedClient:     mgr.GetClient(),
		log:            log.Named("grafana-dashboard"),
		workerName:     workerName,
		recorder:       mgr.GetEventRecorderFor(ControllerName),
		clientProvider: clientProvider,
		mlaNamespace:   mlaNamespace,
	}
}

func (r *grafanaDashboardReconciler) Start(ctx context.Context, mgr manager.Manager, workers int) error {
	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: workers,
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

	if err := c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, enqueueGrafanaConfigMap, predicateutil.ByNamespace(r.mlaNamespace)); err != nil {
		return fmt.Errorf("failed to watch ConfigMaps: %w", err)
	}

	return err
}

func (r *grafanaDashboardReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("configmap", request.NamespacedName)
	log.Debug("Processing")

	configMap := &corev1.ConfigMap{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, configMap); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	grafanaClient, err := r.clientProvider(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	if !configMap.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, r.handleDeletion(ctx, log, configMap, grafanaClient)
	}

	if grafanaClient == nil {
		return reconcile.Result{}, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r.seedClient, configMap, mlaFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := r.ensureDashboards(ctx, log, configMap, grafanaClient); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana Dashboards: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *grafanaDashboardReconciler) Cleanup(ctx context.Context) error {
	configMapList := &corev1.ConfigMapList{}
	if err := r.seedClient.List(ctx, configMapList, ctrlruntimeclient.InNamespace(r.mlaNamespace)); err != nil {
		return fmt.Errorf("failed to list ConfigMaps: %w", err)
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
			return err
		}
	}
	return nil
}

func (r *grafanaDashboardReconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, configMap *corev1.ConfigMap, gClient grafana.Client) error {
	if gClient != nil {
		org, err := gClient.GetOrgByOrgName(ctx, GrafanaOrganization)
		if err != nil && !grafana.IsNotFoundErr(err) {
			return fmt.Errorf("failed to get Grafana organization %q: %w", GrafanaOrganization, err)
		}

		if err == nil {
			if err := deleteDashboards(ctx, log, gClient.WithOrgIDHeader(org.ID), configMap); err != nil {
				return err
			}
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r.seedClient, configMap, mlaFinalizer)
}

func (r *grafanaDashboardReconciler) ensureDashboards(ctx context.Context, log *zap.SugaredLogger, configMap *corev1.ConfigMap, gClient grafana.Client) error {
	org, err := gClient.GetOrgByOrgName(ctx, GrafanaOrganization)
	if err != nil {
		return fmt.Errorf("failed to get Grafana organization %q: %w", GrafanaOrganization, err)
	}

	return addDashboards(ctx, log, gClient.WithOrgIDHeader(org.ID), configMap)
}

func addDashboards(ctx context.Context, log *zap.SugaredLogger, gClient grafana.Client, configMap *corev1.ConfigMap) error {
	for _, data := range configMap.Data {
		var board grafanasdk.Board
		if err := json.Unmarshal([]byte(data), &board); err != nil {
			return fmt.Errorf("unable to unmarshal dashboard: %w", err)
		}
		if _, err := gClient.SetDashboard(ctx, board, grafanasdk.SetDashboardParams{Overwrite: true}); err != nil {
			return err
		}
	}

	return nil
}

func deleteDashboards(ctx context.Context, log *zap.SugaredLogger, gClient grafana.Client, configMap *corev1.ConfigMap) error {
	for _, data := range configMap.Data {
		var board grafanasdk.Board
		if err := json.Unmarshal([]byte(data), &board); err != nil {
			return fmt.Errorf("unable to unmarshal dashboard: %w", err)
		}
		if board.UID == "" {
			log.Debugw("dashboard doesn't have UID set, skipping", "title", board.Title)
			continue
		}
		if _, err := gClient.DeleteDashboardByUID(ctx, board.UID); err != nil && !grafana.IsNotFoundErr(err) {
			return err
		}
	}

	return nil
}
