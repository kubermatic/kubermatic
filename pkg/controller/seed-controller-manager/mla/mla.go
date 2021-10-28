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
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type cleaner interface {
	cleanUp(context.Context) error
}

const (
	ControllerName     = "kubermatic_mla_controller"
	mlaFinalizer       = "kubermatic.io/mla"
	defaultOrgID       = 1
	GrafanaUserKey     = "admin-user"
	GrafanaPasswordKey = "admin-password"
)

var (
	// groupToRole map kubermatic groups to grafana roles
	groupToRole = map[string]grafanasdk.RoleType{
		rbac.OwnerGroupNamePrefix:  grafanasdk.ROLE_EDITOR, // we assign the editor (not admin) role to project owners, to make sure they cannot edit datasources in Grafana
		rbac.EditorGroupNamePrefix: grafanasdk.ROLE_EDITOR,
		rbac.ViewerGroupNamePrefix: grafanasdk.ROLE_VIEWER,
	}
)

// Add creates a new MLA controller that is responsible for
// managing Monitoring, Logging and Alerting for user clusters.
// * org grafana controller - create/update/delete Grafana organizations based on Kubermatic Projects
// * org user grafana controller - create/update/delete Grafana Users to organizations based on Kubermatic UserProjectBindings
// * user grafana controller - create/update/delete Grafana Global Users based on Kubermatic User
// * datasource grafana controller - create/update/delete Grafana Datasources to organizations based on Kubermatic Clusters
// * alertmanager configuration controller - manage alertmanager configuration based on Kubermatic Clusters
// * rule group controller - manager rule groups that will be used to generate alerts.
// * dashboard grafana controller - create/delete Grafana dashboards based on configmaps with prefix `grafana-dashboards`
// * ratelimit cortex controller - updates Cortex runtime configuration with rate limits based on kubermatic MLAAdminSetting
// * cleanup controller - this controller runs when mla disabled and clean objects that left from other MLA controller
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

	split := strings.Split(grafanaSecret, "/")
	if n := len(split); n != 2 {
		return fmt.Errorf("splitting value of %q didn't yield two but %d results",
			grafanaSecret, n)
	}
	secret := corev1.Secret{}
	client, err := ctrlruntimeclient.New(mgr.GetConfig(), ctrlruntimeclient.Options{})
	if err != nil {
		return err
	}
	if err := client.Get(ctx, types.NamespacedName{Name: split[1], Namespace: split[0]}, &secret); err != nil {
		if !mlaEnabled {
			return nil // do not return an error if MLA is disabled (e.g. if MLA is not installed in Seed)
		}
		return fmt.Errorf("failed to get Grafana Secret: %v", err)
	}
	adminName, ok := secret.Data[GrafanaUserKey]
	if !ok {
		return fmt.Errorf("Grafana Secret %q does not contain %s key", grafanaSecret, GrafanaUserKey)
	}
	adminPass, ok := secret.Data[GrafanaPasswordKey]
	if !ok {
		return fmt.Errorf("Grafana Secret %q does not contain %s key", grafanaSecret, GrafanaPasswordKey)
	}
	grafanaAuth := fmt.Sprintf("%s:%s", adminName, adminPass)
	httpClient := &http.Client{Timeout: 15 * time.Second}
	grafanaClient, err := grafanasdk.NewClient(grafanaURL, grafanaAuth, httpClient)
	if err != nil {
		return fmt.Errorf("unable to initialize grafana client")
	}

	orgUserGrafanaController := newOrgUserGrafanaController(mgr.GetClient(), log, grafanaClient)
	orgGrafanaController := newOrgGrafanaController(mgr.GetClient(), log, mlaNamespace, grafanaClient)
	alertmanagerController := newAlertmanagerController(mgr.GetClient(), log, httpClient, cortexAlertmanagerURL)
	datasourceGrafanaController := newDatasourceGrafanaController(mgr.GetClient(), httpClient, grafanaURL, grafanaAuth, mlaNamespace, log, overwriteRegistry)
	userGrafanaController := newUserGrafanaController(mgr.GetClient(), log, grafanaClient, httpClient, grafanaURL, grafanaHeader)
	ruleGroupController := newRuleGroupController(mgr.GetClient(), log, httpClient, cortexRulerURL, lokiRulerURL)
	dashboardGrafanaController := newDashboardGrafanaController(mgr.GetClient(), log, mlaNamespace, grafanaClient)
	ratelimitCortexController := newRatelimitCortexController(mgr.GetClient(), log, mlaNamespace)
	if mlaEnabled {
		// ratelimit cortex controller update 1 configmap, so we better to have only one worker
		if err := newRatelimitCortexReconciler(mgr, log, 1, workerName, versions, ratelimitCortexController); err != nil {
			return fmt.Errorf("failed to create mla ratelimit cortex controller: %w", err)
		}
		if err := newDashboardGrafanaReconciler(mgr, log, numWorkers, workerName, versions, dashboardGrafanaController); err != nil {
			return fmt.Errorf("failed to create mla dashboard grafana controller: %w", err)
		}
		if err := newOrgGrafanaReconciler(mgr, log, numWorkers, workerName, versions, orgGrafanaController); err != nil {
			return fmt.Errorf("failed to create mla org grafana controller: %w", err)
		}
		if err := newOrgUserGrafanaReconciler(mgr, log, numWorkers, workerName, versions, orgUserGrafanaController); err != nil {
			return fmt.Errorf("failed to create mla org user garafana controller: %w", err)
		}
		if err := newDatasourceGrafanaReconciler(mgr, log, numWorkers, workerName, versions, datasourceGrafanaController); err != nil {
			return fmt.Errorf("failed to create mla datasource grafana controller: %w", err)
		}
		if err := newAlertmanagerReconciler(mgr, log, numWorkers, workerName, versions, alertmanagerController); err != nil {
			return fmt.Errorf("failed to create mla alertmanager configuration controller: %w", err)
		}
		if err := newUserGrafanaReconciler(mgr, log, numWorkers, workerName, versions, userGrafanaController); err != nil {
			return fmt.Errorf("failed to create mla user grafana controller: %w", err)
		}
		if err := newRuleGroupReconciler(mgr, log, numWorkers, workerName, versions, ruleGroupController); err != nil {
			return fmt.Errorf("failed to create rule group controller %w", err)
		}
	} else {
		cleanupController := newCleanupController(
			mgr.GetClient(),
			log,
			datasourceGrafanaController,
			dashboardGrafanaController,
			alertmanagerController,
			orgUserGrafanaController,
			orgGrafanaController,
			userGrafanaController,
			ruleGroupController,
			ratelimitCortexController,
		)
		if err := newCleanupReconciler(mgr, log, numWorkers, workerName, versions, cleanupController); err != nil {
			return fmt.Errorf("failed to create mla cleanup controller: %w", err)
		}
	}
	return nil
}

func getDatasourceUIDForCluster(datasourceType string, cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("%s-%s", datasourceType, cluster.Name)
}

func getLokiDatasourceNameForCluster(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("Loki %s", cluster.Spec.HumanReadableName)
}

func getPrometheusDatasourceNameForCluster(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("Prometheus %s", cluster.Spec.HumanReadableName)
}
