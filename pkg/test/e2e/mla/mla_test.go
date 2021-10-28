//go:build mla

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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	grafanasdk "github.com/kubermatic/grafanasdk"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/scheme"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	datacenter = "kubermatic"
	location   = "do-fra1"
	version    = utils.KubernetesVersion()
	credential = "e2e-digitalocean"
)

func TestMLAIntegration(t *testing.T) {
	ctx := context.Background()

	if err := v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("failed to register operator scheme: %v", err)
	}

	seedClient, _, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// login
	masterToken, err := utils.RetrieveMasterToken(ctx)
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}
	masterClient := utils.NewTestClient(masterToken, t)

	// create dummy project
	t.Log("creating project...")
	project, err := masterClient.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("failed to create project: %v", getErrorResponse(err))
	}
	defer masterClient.CleanupProject(t, project.ID)

	t.Log("creating cluster...")
	apiCluster, err := masterClient.CreateDOCluster(project.ID, datacenter, rand.String(10), credential, version, location, 1)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", getErrorResponse(err))
	}

	// wait for the cluster to become healthy
	if err := masterClient.WaitForClusterHealthy(project.ID, datacenter, apiCluster.ID); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	if err := masterClient.WaitForClusterNodeDeploymentsToByReady(project.ID, datacenter, apiCluster.ID, 1); err != nil {
		t.Fatalf("cluster nodes not ready: %v", err)
	}

	// get the cluster object (the CRD, not the API's representation)
	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: apiCluster.ID}, cluster); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}

	seed := &kubermaticv1.Seed{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: "kubermatic", Namespace: "kubermatic"}, seed); err != nil {
		t.Fatalf("failed to get seed: %v", err)
	}
	seed.Spec.MLA = &kubermaticv1.SeedMLASettings{
		UserClusterMLAEnabled: true,
	}
	if err := seedClient.Update(ctx, seed); err != nil {
		t.Fatalf("failed to update seed: %v", err)
	}

	// enable MLA
	t.Log("enabling MLA...")
	if err := setMLAIntegration(ctx, seedClient, cluster, true); err != nil {
		t.Fatalf("failed to set MLA integration to true: %v", err)
	}

	t.Log("waiting for project to get grafana org annotation")
	p := &kubermaticv1.Project{}
	timeout := 300 * time.Second
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		if err := seedClient.Get(ctx, types.NamespacedName{Name: project.ID}, p); err != nil {
			t.Fatalf("failed to get project: %v", err)
		}

		_, ok := p.GetAnnotations()[mla.GrafanaOrgAnnotationKey]
		return ok
	}) {
		t.Fatalf("waiting for project annotation %+v", p)
	}

	t.Log("creating client for user cluster...")
	grafanaSecret := "mla/grafana"

	split := strings.Split(grafanaSecret, "/")
	if n := len(split); n != 2 {
		t.Fatalf("splitting value of %q didn't yield two but %d results",
			grafanaSecret, n)
	}

	secret := corev1.Secret{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: split[1], Namespace: split[0]}, &secret); err != nil {
		t.Fatalf("failed to get Grafana Secret: %v", err)
	}
	adminName, ok := secret.Data[mla.GrafanaUserKey]
	if !ok {
		t.Fatalf("Grafana Secret %q does not contain %s key", grafanaSecret, mla.GrafanaUserKey)
	}
	adminPass, ok := secret.Data[mla.GrafanaPasswordKey]
	if !ok {
		t.Fatalf("Grafana Secret %q does not contain %s key", grafanaSecret, mla.GrafanaPasswordKey)
	}
	grafanaAuth := fmt.Sprintf("%s:%s", adminName, adminPass)
	httpClient := &http.Client{Timeout: 15 * time.Second}

	grafanaURL := "http://localhost:3000"
	grafanaClient, err := grafanasdk.NewClient(grafanaURL, grafanaAuth, httpClient)
	if err != nil {
		t.Fatalf("unable to initialize grafana client")
	}
	orgID, ok := p.GetAnnotations()[mla.GrafanaOrgAnnotationKey]
	if !ok {
		t.Fatal("project should have grafana org annotation set")
	}
	id, err := strconv.ParseUint(orgID, 10, 32)
	if err != nil {
		t.Fatalf("unable to parse uint from %s", orgID)
	}
	org, err := grafanaClient.GetOrgById(ctx, uint(id))
	if err != nil {
		t.Fatalf("error while getting grafana org:  %s", err)
	}
	t.Log("org added to Grafana")

	if err := seedClient.Get(ctx, types.NamespacedName{Name: apiCluster.ID}, cluster); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}

	grafanaClient.SetOrgIDHeader(org.ID)
	t.Log("waiting for datasource added to grafana")
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		_, err := grafanaClient.GetDatasourceByUID(ctx, fmt.Sprintf("%s-%s", mla.PrometheusType, cluster.Name))
		return err == nil
	}) {
		t.Fatalf("waiting for grafana datasource %s-%s", mla.PrometheusType, cluster.Name)
	}

	// Disable MLA Integration
	t.Log("disabling MLA...")
	if err := setMLAIntegration(ctx, seedClient, cluster, false); err != nil {
		t.Fatalf("failed to set MLA integration to false: %v", err)
	}

	seed.Spec.MLA = &kubermaticv1.SeedMLASettings{
		UserClusterMLAEnabled: false,
	}
	if err := seedClient.Update(ctx, seed); err != nil {
		t.Fatalf("failed to update seed: %v", err)
	}

	// Check that cluster is healthy
	t.Log("waiting for cluster to healthy after disabling MLA...")
	if err := masterClient.WaitForClusterHealthy(project.ID, datacenter, apiCluster.ID); err != nil {
		t.Fatalf("cluster not healthy: %v", err)
	}

	t.Log("waiting for project to get rid of grafana org annotation")
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		if err := seedClient.Get(ctx, types.NamespacedName{Name: project.ID}, p); err != nil {
			t.Fatalf("failed to get project: %v", err)
		}

		_, ok := p.GetAnnotations()[mla.GrafanaOrgAnnotationKey]
		return !ok
	}) {
		t.Fatalf("waiting for project annotation removed %+v", p)
	}

	// Test that cluster deletes cleanly
	masterClient.CleanupCluster(t, project.ID, datacenter, apiCluster.ID)
}

// getErrorResponse converts the client error response to string
func getErrorResponse(err error) string {
	rawData, newErr := json.Marshal(err)
	if newErr != nil {
		return err.Error()
	}
	return string(rawData)
}

func setMLAIntegration(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, enabled bool) error {
	oldCluster := cluster.DeepCopy()
	cluster.Spec.MLA = &kubermaticv1.MLASettings{
		MonitoringEnabled: enabled,
		LoggingEnabled:    enabled,
	}

	return client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}
