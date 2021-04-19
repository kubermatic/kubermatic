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

const (
	ControllerName = "kubermatic_mla_controller"
	mlaFinalizer   = "kubermatic.io/mla"
	defaultOrgID   = 1
)

var (
	// groupToRole map kubermatic groups to grafana roles
	groupToRole = map[string]models.RoleType{
		rbac.OwnerGroupNamePrefix:  models.ROLE_ADMIN,
		rbac.EditorGroupNamePrefix: models.ROLE_EDITOR,
		rbac.ViewerGroupNamePrefix: models.ROLE_VIEWER,
	}
)

// Add creates a new MLA controller that is responsible for
// managing Monitoring, Logging and Alerting for user clusters.
// * org grafana controller - create/update/delete Grafana organizations based on Kubermatic Projects
// * user grafana controller - create/update/delete Grafana Users to organizations based on Kubermatic UserProjectBindings
// * datasource grafana controller - create/update/delete Grafana Datasources to organizations based on Kubermatic Clusters
// * alertmanager configuartion controller - manage alertmanager configuration based on Kubermatic Clusters
func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	grafanaURL string,
	grafanaHeader string,
	grafanaSecret string,
	overwriteRegistry string,
) error {

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
	auth, ok := secret.Data["auth"]
	if !ok {
		return fmt.Errorf("Grafana Secret %q does not contain auth", grafanaSecret)
	}
	httpClient := &http.Client{Timeout: 15 * time.Second}
	grafanaClient := grafanasdk.NewClient(grafanaURL, string(auth), httpClient)
	if err := newOrgGrafanaReconciler(mgr, log, numWorkers, workerName, versions, grafanaClient); err != nil {
		return fmt.Errorf("failed to create mla project controller: %v", err)
	}
	if err := newUserGrafanaReconciler(mgr, log, numWorkers, workerName, versions, grafanaClient, httpClient, grafanaURL, grafanaHeader); err != nil {
		return fmt.Errorf("failed to create mla userprojectbinding controller: %v", err)
	}
	if err := newDatasourceGrafanaReconciler(mgr, log, numWorkers, workerName, versions, grafanaClient, overwriteRegistry); err != nil {
		return fmt.Errorf("failed to create mla cluster controller: %v", err)
	}
	if err := newAlertmanagerReconciler(mgr, log, numWorkers, workerName, versions, httpClient); err != nil {
		return fmt.Errorf("failed to create mla alertmanager configuration controller: %v", err)
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
