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

	"github.com/grafana/grafana/pkg/models"
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
	grafanaUserKey     = "admin-user"
	grafanaPasswordKey = "admin-password"
)

var (
	// groupToRole map kubermatic groups to grafana roles
	groupToRole = map[string]models.RoleType{
		rbac.OwnerGroupNamePrefix:  models.ROLE_EDITOR, // we assign the editor (not admin) role to project owners, to make sure they cannot edit datasources in Grafana
		rbac.EditorGroupNamePrefix: models.ROLE_EDITOR,
		rbac.ViewerGroupNamePrefix: models.ROLE_VIEWER,
	}
)

// Add creates a new MLA controller that is responsible for
// managing Monitoring, Logging and Alerting for user clusters.
// * org grafana controller - create/update/delete Grafana organizations based on Kubermatic Projects
// * org user grafana controller - create/update/delete Grafana Users to organizations based on Kubermatic UserProjectBindings
// * user grafana controller - create/update/delete Grafana Global Users based on Kubermatic User
// * datasource grafana controller - create/update/delete Grafana Datasources to organizations based on Kubermatic Clusters
// * alertmanager configuration controller - manage alertmanager configuration based on Kubermatic Clusters
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
		return fmt.Errorf("failed to get Grafana Secret: %v", err)
	}
	adminName, ok := secret.Data[grafanaUserKey]
	if !ok {
		return fmt.Errorf("Grafana Secret %q does not contain %s key", grafanaSecret, grafanaUserKey)
	}
	adminPass, ok := secret.Data[grafanaPasswordKey]
	if !ok {
		return fmt.Errorf("Grafana Secret %q does not contain %s key", grafanaSecret, grafanaPasswordKey)
	}
	grafanaAuth := fmt.Sprintf("%s:%s", adminName, adminPass)
	httpClient := &http.Client{Timeout: 15 * time.Second}
	grafanaClient := grafanasdk.NewClient(grafanaURL, grafanaAuth, httpClient)

	orgGrafanaController := newOrgGrafanaController(mgr.GetClient(), log, grafanaClient)
	alertmanagerController := newAlertmanagerController(mgr.GetClient(), log, httpClient, cortexAlertmanagerURL)
	orgUserGrafanaController := newOrgUserGrafanaController(mgr.GetClient(), log, grafanaClient)
	datasourceGrafanaController := newDatasourceGrafanaController(mgr.GetClient(), httpClient, grafanaURL, grafanaAuth, mlaNamespace, log, overwriteRegistry)
	userGrafanaController := newUserGrafanaController(mgr.GetClient(), log, grafanaClient, httpClient, grafanaURL, grafanaHeader)
	if mlaEnabled {
		if err := newOrgGrafanaReconciler(mgr, log, numWorkers, workerName, versions, orgGrafanaController); err != nil {
			return fmt.Errorf("failed to create mla project controller: %w", err)
		}
		if err := newOrgUserGrafanaReconciler(mgr, log, numWorkers, workerName, versions, orgUserGrafanaController); err != nil {
			return fmt.Errorf("failed to create mla userprojectbinding controller: %w", err)
		}
		if err := newDatasourceGrafanaReconciler(mgr, log, numWorkers, workerName, versions, datasourceGrafanaController); err != nil {
			return fmt.Errorf("failed to create mla cluster controller: %w", err)
		}
		if err := newAlertmanagerReconciler(mgr, log, numWorkers, workerName, versions, alertmanagerController); err != nil {
			return fmt.Errorf("failed to create mla alertmanager configuration controller: %w", err)
		}
		if err := newUserGrafanaReconciler(mgr, log, numWorkers, workerName, versions, userGrafanaController); err != nil {
			return fmt.Errorf("failed to create mla user controller: %w", err)
		}
	} else {
		cleanupController := newCleanupController(
			mgr.GetClient(),
			log,
			datasourceGrafanaController,
			alertmanagerController,
			orgUserGrafanaController,
			orgGrafanaController,
			userGrafanaController,
		)
		if err := newCleanupReconciler(mgr, log, numWorkers, workerName, versions, cleanupController); err != nil {
			return fmt.Errorf("failed to create mla cleanup controller: %w", err)
		}
	}
	if err := newRuleGroupReconciler(mgr, log, numWorkers, workerName, versions, httpClient, cortexRulerURL, mlaEnabled); err != nil {
		return fmt.Errorf("failed to create mla rule group controller: %w", err)
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
