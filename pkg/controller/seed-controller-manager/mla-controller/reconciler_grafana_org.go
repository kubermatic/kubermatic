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
	"fmt"
	"strings"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"
	"k8c.io/kubermatic/v3/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	GrafanaOrganization = "kkp-user-clusters"
)

// grafanaOrgReconciler ensures that each cluster is marked to use a Grafana organization.
type grafanaOrgReconciler struct {
	seedClient     ctrlruntimeclient.Client
	log            *zap.SugaredLogger
	recorder       record.EventRecorder
	clientProvider grafana.ClientProvider
	workerName     string
	mlaNamespace   string
}

func newGrafanaOrgReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	clientProvider grafana.ClientProvider,
	mlaNamespace string,
) *grafanaOrgReconciler {
	return &grafanaOrgReconciler{
		seedClient:     mgr.GetClient(),
		log:            log.Named("grafana-org"),
		recorder:       mgr.GetEventRecorderFor(ControllerName),
		workerName:     workerName,
		clientProvider: clientProvider,
		mlaNamespace:   mlaNamespace,
	}
}

func (r *grafanaOrgReconciler) Start(ctx context.Context, mgr manager.Manager, workers int) error {
	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: workers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, workerlabel.Predicates(r.workerName)); err != nil {
		return fmt.Errorf("failed to watch Clusters: %w", err)
	}
	return err
}

func (r *grafanaOrgReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.NamespacedName)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !cluster.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	grafanaClient, err := r.clientProvider(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	if grafanaClient == nil {
		return reconcile.Result{}, nil
	}

	orgName := GrafanaOrganization
	org := grafanasdk.Org{Name: orgName}
	log = log.With("org", orgName)

	orgID, err := r.ensureOrganization(ctx, log, cluster, org, grafanaClient)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana organization: %w", err)
	}

	if err := r.ensureDashboards(ctx, log, orgID, grafanaClient); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana dashboards: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *grafanaOrgReconciler) ensureOrganization(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, expected grafanasdk.Org, gClient grafana.Client) (uint, error) {
	existingOrg, err := gClient.GetOrgByOrgName(ctx, expected.Name)
	if err == nil {
		return existingOrg.ID, nil
	}

	log.Info("Creating Grafana organization")
	status, err := gClient.CreateOrg(ctx, expected)
	if err != nil {
		return 0, fmt.Errorf("failed to create organization: %w", err)
	}

	return *status.OrgID, nil
}

func (r *grafanaOrgReconciler) ensureDashboards(ctx context.Context, log *zap.SugaredLogger, orgID uint, gClient grafana.Client) error {
	configMapList := &corev1.ConfigMapList{}
	if err := r.seedClient.List(ctx, configMapList, ctrlruntimeclient.InNamespace(r.mlaNamespace)); err != nil {
		return fmt.Errorf("failed to list configmaps: %w", err)
	}

	for _, configMap := range configMapList.Items {
		if !strings.HasPrefix(configMap.GetName(), grafanaDashboardsConfigmapNamePrefix) {
			continue
		}

		if err := addDashboards(ctx, log, gClient.WithOrgIDHeader(orgID), &configMap); err != nil {
			return err
		}
	}

	return nil
}
