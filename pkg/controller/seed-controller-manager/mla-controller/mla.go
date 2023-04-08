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
	"net/http"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/cortex"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	ControllerName = "kkp-mla-controller"
	mlaFinalizer   = "kubermatic.k8c.io/mla"
)

// Add creates a new MLA controller that is responsible for managing Monitoring, Logging and Alerting for user clusters.
func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	mlaNamespace string,
	grafanaURL string,
	grafanaHeader string,
	grafanaSecret string,
	overwriteRegistry string,
	cortexAlertmanagerURL string,
	cortexRulerURL string,
	lokiRulerURL string,
	mlaEnabled bool,
) error {
	log = log.Named(ControllerName)

	httpClient := &http.Client{Timeout: 15 * time.Second}
	clientProvider, err := grafana.NewClientProvider(mgr.GetClient(), httpClient, grafanaSecret, grafanaURL, mlaEnabled)
	if err != nil {
		return fmt.Errorf("failed to prepare Grafana client provider: %w", err)
	}

	cClient := cortex.NewClient(httpClient, cortexAlertmanagerURL, cortexRulerURL, lokiRulerURL)
	cortexClientProvider := func() cortex.Client {
		return cClient
	}

	grafanaOrgReconciler := newGrafanaOrgReconciler(mgr, log, workerName, clientProvider, mlaNamespace)
	grafanaUserReconciler := newGrafanaUserReconciler(ctx, mgr, log, workerName, clientProvider)
	grafanaDatasourceReconciler := newGrafanaDatasourceReconciler(mgr, log, workerName, versions, clientProvider, overwriteRegistry, mlaNamespace)
	grafanaDashboardReconciler := newGrafanaDashboardReconciler(mgr, log, workerName, clientProvider, mlaNamespace)
	alertmanagerReconciler := newAlertmanagerReconciler(mgr, log, workerName, versions, cortexClientProvider)
	cortexRatelimitReconciler := newCortexRatelimitReconciler(mgr, log, workerName, mlaNamespace)
	ruleGroupReconciler := newRuleGroupReconciler(mgr, log, workerName, versions, mlaNamespace, cortexClientProvider)
	ruleGroupSyncReconciler := newRuleGroupSyncReconciler(mgr, log, workerName, versions, mlaNamespace)

	// if MLA is enabled, start the controllers the usual way
	if mlaEnabled {
		if err := grafanaOrgReconciler.Start(ctx, mgr, numWorkers); err != nil {
			return fmt.Errorf("failed to create Grafana organization controller: %w", err)
		}
		if err := grafanaUserReconciler.Start(ctx, mgr, numWorkers); err != nil {
			return fmt.Errorf("failed to create Grafana user controller: %w", err)
		}
		if err := grafanaDatasourceReconciler.Start(ctx, mgr, numWorkers); err != nil {
			return fmt.Errorf("failed to create Grafana datasource controller: %w", err)
		}
		if err := grafanaDashboardReconciler.Start(ctx, mgr, numWorkers); err != nil {
			return fmt.Errorf("failed to create Grafana dashboard controller: %w", err)
		}
		if err := alertmanagerReconciler.Start(ctx, mgr, numWorkers); err != nil {
			return fmt.Errorf("failed to create Alertmanager controller: %w", err)
		}
		// ratelimit cortex controller update 1 configmap, so we better to have only one worker
		if err := cortexRatelimitReconciler.Start(ctx, mgr, 1); err != nil {
			return fmt.Errorf("failed to create Cortex rate limiting controller: %w", err)
		}
		if err := ruleGroupReconciler.Start(ctx, mgr, numWorkers); err != nil {
			return fmt.Errorf("failed to create rule group controller: %w", err)
		}
		if err := ruleGroupSyncReconciler.Start(ctx, mgr, numWorkers); err != nil {
			return fmt.Errorf("failed to create rule group sync controller: %w", err)
		}
	} else {
		// ... if MLA is disabled however, re/mis-use the reconcilers to perform a one-time cleanup.
		cleanupController := newCleanupReconciler(
			mgr,
			log,
			// no need to cleanup the Grafana organization itself
			grafanaUserReconciler,
			grafanaDatasourceReconciler,
			grafanaDashboardReconciler,
			alertmanagerReconciler,
			ruleGroupReconciler,
			ruleGroupSyncReconciler,
			cortexRatelimitReconciler,
		)
		if err := cleanupController.Start(ctx, mgr, 1); err != nil {
			return fmt.Errorf("failed to create cleanup controller: %w", err)
		}
	}

	return nil
}

func getDatasourceUIDForCluster(datasourceType string, cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("%s-%s", datasourceType, cluster.Name)
}

func getAlertmanagerDatasourceNameForCluster(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("Alertmanager %s", cluster.Spec.HumanReadableName)
}

func getLokiDatasourceNameForCluster(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("Loki %s", cluster.Spec.HumanReadableName)
}

func getPrometheusDatasourceNameForCluster(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("Prometheus %s", cluster.Spec.HumanReadableName)
}
