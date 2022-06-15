//go:build e2e

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

package ccmmigration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/providers"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/utils"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCCMMigration(t *testing.T) {
	ctx := context.Background()
	seedClient, _, _ := e2eutils.GetClientsOrDie()

	clusterClientProvider, err := clusterclient.NewExternal(seedClient)
	if err != nil {
		t.Fatalf("Failed to create cluster cluster provider: %v", err)
	}

	// prepare cluster
	clusterJig, cluster, userClient, err := SetupClusterByProvider(t, ctx, seedClient, clusterClientProvider, options)
	if err != nil {
		t.Fatalf("Failed to setup preconditions: %v", err)
	}

	// run tests
	err = testBody(t, ctx, clusterJig, cluster, userClient)

	// cleanup
	clusterJig.Cleanup(ctx, userClient)

	// finish
	if err != nil {
		t.Fatal(err)
	}
}

func SetupClusterByProvider(t *testing.T, ctx context.Context, seedClient ctrlruntimeclient.Client, clientProvider *clusterclient.Provider, options testOptions) (providers.ClusterJigInterface, *kubermaticv1.Cluster, ctrlruntimeclient.Client, error) {
	var (
		clusterJig providers.ClusterJigInterface
	)

	switch kubermaticv1.ProviderType(options.provider) {
	case kubermaticv1.OpenstackCloudProvider:
		clusterJig = providers.NewClusterJigOpenstack(seedClient, options.kubernetesVersion, options.osSeedDatacenter, options.osCredentials)
	case kubermaticv1.VSphereCloudProvider:
		clusterJig = providers.NewClusterJigVsphere(seedClient, options.kubernetesVersion, options.vsphereSeedDatacenter, options.vSphereCredentials)
	case kubermaticv1.AzureCloudProvider:
		clusterJig = providers.NewClusterJigAzure(seedClient, options.kubernetesVersion, options.azureSeedDatacenter, options.azureCredentials)
	default:
		return nil, nil, nil, errors.New("provider not supported for CCM tests")
	}

	cluster := &kubermaticv1.Cluster{}
	userClient, err := setupAndGetUserClient(t, ctx, clusterJig, cluster, clientProvider)

	return clusterJig, cluster, userClient, err
}

func setupAndGetUserClient(t *testing.T, ctx context.Context, clusterJig providers.ClusterJigInterface, cluster *kubermaticv1.Cluster, clusterClientProvider *clusterclient.Provider) (ctrlruntimeclient.Client, error) {
	t.Log("Setting up cluster...")
	if err := clusterJig.Setup(ctx); err != nil {
		return nil, fmt.Errorf("failed to create user cluster: %w", err)
	}

	t.Logf("Cluster %s has been created.", clusterJig.Name())
	if err := clusterJig.Seed().Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, cluster); err != nil {
		return nil, fmt.Errorf("failed to get the cluster we just created: %w", err)
	}

	t.Log("Giving KKP some time to reconcile...")
	time.Sleep(30 * time.Second)

	t.Log("Waiting for user cluster to become available...")
	var userClient ctrlruntimeclient.Client
	err := wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool bool, err error) {
		if err := clusterJig.Seed().Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, cluster); err != nil {
			return false, err
		}

		userClient, err = clusterClientProvider.GetClient(ctx, cluster)
		if err != nil {
			t.Logf("User cluster not ready yet: %v", err)
		}

		return err == nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check cluster readiness: %w", err)
	}
	t.Log("Cluster is available.")

	if err := clusterv1alpha1.AddToScheme(userClient.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to setup scheme: %w", err)
	}

	t.Log("Creating MachineDeployment...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		err := clusterJig.CreateMachineDeployment(ctx, userClient)
		if err != nil {
			t.Logf("MachineDeployment creation failed: %v", err)
		}

		return err == nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MachineDeployment: %w", err)
	}
	t.Log("MachineDeployment created.")

	t.Log("Waiting for machine(s) to be ready...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		ready, err := clusterJig.WaitForNodeToBeReady(ctx, userClient)
		if err != nil {
			t.Logf("Node not ready yet: %v", err)
		}
		return ready, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for Nodes: %w", err)
	}
	t.Log("Node(s) are ready.")

	return userClient, nil
}

func testBody(t *testing.T, ctx context.Context, clusterJig providers.ClusterJigInterface, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) error {
	seedClient := clusterJig.Seed()

	t.Log("Enabling externalCloudProvider feature...")
	if err := seedClient.Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	newCluster := cluster.DeepCopy()
	newCluster.Spec.Features = map[string]bool{
		kubermaticv1.ClusterFeatureExternalCloudProvider: true,
	}
	if err := seedClient.Patch(ctx, newCluster, ctrlruntimeclient.MergeFrom(cluster)); err != nil {
		return fmt.Errorf("failed to patch cluster: %w", err)
	}

	t.Log("Getting the patched cluster...")
	annotatedCluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	t.Log("Asserting the annotations existence in the cluster...")
	err := wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		_, ccmOk := annotatedCluster.Annotations[kubermaticv1.CCMMigrationNeededAnnotation]
		_, csiOk := annotatedCluster.Annotations[kubermaticv1.CSIMigrationNeededAnnotation]

		return ccmOk && csiOk, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for annotations to appear: %w", err)
	}

	t.Log("Checking the -node-external-cloud-provider flag in the machine-controller webhook Pod...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		machineControllerWebhookPods := &corev1.PodList{}
		if err := seedClient.List(ctx, machineControllerWebhookPods, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName), ctrlruntimeclient.MatchingLabels{
			resources.AppLabelKey: resources.MachineControllerWebhookDeploymentName,
		}); err != nil {
			return false, err
		}
		if len(machineControllerWebhookPods.Items) != 1 {
			return false, nil
		}
		for _, arg := range machineControllerWebhookPods.Items[0].Spec.Containers[0].Args {
			if strings.Contains(arg, "-node-external-cloud-provider") {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for flag in Pod: %w", err)
	}

	t.Log("Rotating all the machines...")
	machines := &clusterv1alpha1.MachineList{}
	if err := userClient.List(ctx, machines); err != nil {
		return fmt.Errorf("failed to list machines: %w", err)
	}

	for _, m := range machines.Items {
		if err := userClient.Delete(ctx, &m); err != nil {
			return fmt.Errorf("failed to delete machine: %w", err)
		}
	}

	t.Log("Waiting for the complete cluster migration...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		migratingCluster := &kubermaticv1.Cluster{}
		if err := seedClient.Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, migratingCluster); err != nil {
			return false, err
		}
		return kubermaticv1helper.CCMMigrationCompleted(migratingCluster), nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for migration to finish: %w", err)
	}

	t.Log("Waiting for node(s) to come up again...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		ready, err := clusterJig.WaitForNodeToBeReady(ctx, userClient)
		if err != nil {
			return false, nil
		}
		return ready, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for nodes: %w", err)
	}

	t.Log("Checking that all the needed components are up and running...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		return clusterJig.CheckComponents(ctx, userClient)
	})
	if err != nil {
		return fmt.Errorf("failed to wait for components: %w", err)
	}

	return nil
}
