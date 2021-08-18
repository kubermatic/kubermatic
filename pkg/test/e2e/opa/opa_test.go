// +build e2e

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

package opa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	constrainttemplatev1beta1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	datacenter             = "kubermatic"
	location               = "do-fra1"
	version                = utils.KubernetesVersion()
	credential             = "e2e-digitalocean"
	ctKind                 = "RequiredLabels"
	masterNamespace        = "kubermatic"
	defaultConstraintName  = "testconstraint"
	constraintTemplateName = "requiredlabels"
)

func TestOPAIntegration(t *testing.T) {
	ctx := context.Background()

	if err := constrainttemplatev1beta1.AddToSchemes.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("failed to register gatekeeper scheme: %v", err)
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

	masterAdminToken, err := utils.RetrieveAdminMasterToken(ctx)
	if err != nil {
		t.Fatalf("failed to get master admin token: %v", err)
	}
	masterAdminClient := utils.NewTestClient(masterAdminToken, t)

	// create dummy project
	t.Log("creating project...")
	project, err := masterClient.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	defer cleanupProject(t, project.ID)

	t.Log("creating cluster...")
	apiCluster, err := masterClient.CreateDOCluster(project.ID, datacenter, rand.String(10), credential, version, location, 1)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
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

	// enable OPA
	t.Log("enabling OPA...")
	if err := setOPAIntegration(ctx, seedClient, cluster, true); err != nil {
		t.Fatalf("failed to set OPA integration to true: %v", err)
	}

	t.Log("waiting for cluster to healthy after enabling OPA...")
	if err := masterClient.WaitForOPAEnabledClusterHealthy(project.ID, datacenter, apiCluster.ID); err != nil {
		t.Fatalf("cluster not ready: %v", err)
	}

	// Create CT
	t.Log("creating Constraint Template...")
	ct, err := masterAdminClient.CreateCT(constraintTemplateName, ctKind)
	if err != nil {
		t.Fatalf("error creating Constraint Template: %v", err)
	}

	t.Log("creating client for user cluster...")
	userClient, err := masterClient.GetUserClusterClient(datacenter, project.ID, apiCluster.ID)
	if err != nil {
		t.Fatalf("error creating user cluster client: %v", err)
	}

	if err := waitForCTSync(ctx, userClient, ct.Name, false); err != nil {
		t.Fatal(err)
	}

	t.Log("constraint template synced to user cluster...")

	// Create Default Constraint
	t.Log("creating Default Constraint...")
	defaultConstraint, err := masterAdminClient.CreateConstraint(defaultConstraintName, ctKind)
	if err != nil {
		t.Fatalf("error creating Default Constraint: %v", getErrorResponse(err))
	}

	t.Log("waiting for Default Constraint sync...")
	if err := waitForConstraintSync(ctx, seedClient, defaultConstraint.Name, masterNamespace, false); err != nil {
		t.Fatal(err)
	}
	t.Log("synced to Default Constraint kubermatic namespace...")

	if err := waitForConstraintSync(ctx, seedClient, defaultConstraint.Name, cluster.Status.NamespaceName, false); err != nil {
		t.Fatal(err)
	}
	t.Log("synced to Default Constraint user cluster namespace...")

	// Test if constraint works
	t.Log("testing if Constraint works by creating policy-breaking configmap...")
	if err := testConstraintForConfigMap(ctx, userClient); err != nil {
		t.Fatal(err)
	}

	t.Log("testing if Constraint lets through policy-aligned namespace...")
	cm := genTestConfigMap()
	cm.Labels = map[string]string{"gatekeeper": "true"}
	if err := userClient.Create(ctx, cm); err != nil {
		t.Fatalf("error creating policy-aligned configmap on user cluster: %v", err)
	}

	// Delete constraint
	t.Log("Deleting Constraint...")
	if err := masterAdminClient.DeleteConstraint(defaultConstraint.Name); err != nil {
		t.Fatalf("error deleting Constraint: %v", err)
	}
	t.Log("waiting for Constraint sync delete...")

	if err := waitForConstraintSync(ctx, seedClient, defaultConstraint.Name, masterNamespace, true); err != nil {
		t.Fatal(err)
	}
	t.Log("synced to Default Constraint kubermatic namespace...")

	if err := waitForConstraintSync(ctx, seedClient, defaultConstraint.Name, cluster.Status.NamespaceName, true); err != nil {
		t.Fatal(err)
	}
	t.Log("synced to Default Constraint user cluster namespace...")

	// Check that constraint does not work
	t.Log("testing if policy breaking configmap can be created...")
	cmBreaking := genTestConfigMap()
	if err := userClient.Create(ctx, cmBreaking); err != nil {
		t.Fatalf("error creating policy-breaking configmap on user cluster after deleting constraint: %v", err)
	}

	// Delete CT
	t.Log("deleting Constraint Template...")
	if err := masterAdminClient.DeleteConstraintTemplate(ct.Name); err != nil {
		t.Fatalf("error deleting Constraint Template: %v", err)
	}

	// Check that CT is removed
	t.Log("waiting for Constraint Template delete sync...")
	if err := waitForCTSync(ctx, userClient, ct.Name, true); err != nil {
		t.Fatal(err)
	}

	// Disable OPA Integration
	t.Log("disabling OPA...")
	if err := setOPAIntegration(ctx, seedClient, cluster, false); err != nil {
		t.Fatalf("failed to set OPA integration to false: %v", err)
	}

	// Check that cluster is healthy
	t.Log("waiting for cluster to healthy after disabling OPA...")
	if err := masterClient.WaitForClusterHealthy(project.ID, datacenter, apiCluster.ID); err != nil {
		t.Fatalf("cluster not healthy: %v", err)
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

func setOPAIntegration(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, enabled bool) error {
	oldCluster := cluster.DeepCopy()
	cluster.Spec.OPAIntegration = &kubermaticv1.OPAIntegrationSettings{
		Enabled: enabled,
	}

	return client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func testConstraintForConfigMap(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	if !utils.WaitFor(1*time.Second, 2*time.Minute, func() bool {
		cm := genTestConfigMap()
		err := userClient.Create(ctx, cm)
		return err != nil && strings.Contains(err.Error(), "you must provide labels")
	}) {
		return fmt.Errorf("timeout waiting for Constraint policy to be enforced")
	}
	return nil
}

func waitForCTSync(ctx context.Context, userClient ctrlruntimeclient.Client, ctName string, deleted bool) error {
	if !utils.WaitFor(1*time.Second, 1*time.Minute, func() bool {
		gatekeeperCT := &constrainttemplatev1beta1.ConstraintTemplate{}
		err := userClient.Get(ctx, types.NamespacedName{Name: ctName}, gatekeeperCT)

		if deleted {
			return kerrors.IsNotFound(err)
		}
		return err == nil
	}) {
		return fmt.Errorf("timeout waiting for Constraint Template to be synced to user cluster")
	}
	return nil
}

func waitForConstraintSync(ctx context.Context, client ctrlruntimeclient.Client, cName, namespace string, deleted bool) error {
	if !utils.WaitFor(1*time.Second, 1*time.Minute, func() bool {
		constraint := &kubermaticv1.Constraint{}
		err := client.Get(ctx, types.NamespacedName{Name: cName, Namespace: namespace}, constraint)
		if deleted {
			return kerrors.IsNotFound(err)
		}
		return err == nil
	}) {
		return fmt.Errorf("timeout waiting for Constraint to be synced")
	}
	return nil
}

func genTestConfigMap() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	cm.Namespace = corev1.NamespaceDefault
	cm.Name = fmt.Sprintf("test-cm-%d", rand.Int())
	return cm
}

func cleanupProject(t *testing.T, id string) {
	t.Log("cleaning up project and cluster...")

	// use a dedicated context so that cleanups always run, even
	// if the context inside a test was already cancelled
	token, err := utils.RetrieveAdminMasterToken(context.Background())
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}

	utils.NewTestClient(token, t).CleanupProject(t, id)
}
