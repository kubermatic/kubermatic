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
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/providers"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/utils"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Options holds the e2e test options.
type testOptions struct {
	skipCleanup       bool
	logOptions        log.Options
	kubernetesRelease string

	provider string

	vsphereSeedDatacenter string
	osSeedDatacenter      string
	azureSeedDatacenter   string
	awsSeedDatacenter     string

	osCredentials      providers.OpenstackCredentialsType
	vSphereCredentials providers.VsphereCredentialsType
	azureCredentials   providers.AzureCredentialsType
	awsCredentials     providers.AWSCredentialsType
}

func (o testOptions) KubernetesVersion() semver.Semver {
	return *test.ParseVersionOrRelease(o.kubernetesRelease, nil)
}

var (
	options = testOptions{
		logOptions: e2eutils.DefaultLogOptions,
	}
)

func init() {
	flag.BoolVar(&options.skipCleanup, "skip-cleanup", false, "Skip clean-up of resources")
	flag.StringVar(&options.provider, "provider", "", "Cloud provider to test")
	flag.StringVar(&options.kubernetesRelease, "cluster-version", "", "Kubernetes version or release for the usercluster")

	flag.StringVar(&options.osSeedDatacenter, "openstack-seed-datacenter", "", "openstack datacenter")
	flag.StringVar(&options.vsphereSeedDatacenter, "vsphere-seed-datacenter", "", "vsphere seed datacenter")
	flag.StringVar(&options.azureSeedDatacenter, "azure-seed-datacenter", "", "azure seed datacenter")
	flag.StringVar(&options.awsSeedDatacenter, "aws-seed-datacenter", "", "aws seed datacenter")

	flag.StringVar(&options.osCredentials.Username, "openstack-username", "", "openstack username")
	flag.StringVar(&options.osCredentials.Password, "openstack-password", "", "openstack password")
	flag.StringVar(&options.osCredentials.Tenant, "openstack-tenant", "", "openstack tenant")
	flag.StringVar(&options.osCredentials.Domain, "openstack-domain", "", "openstack domain")
	flag.StringVar(&options.osCredentials.FloatingIPPool, "openstack-floating-ip-pool", "", "openstack floating ip pool")
	flag.StringVar(&options.osCredentials.Network, "openstack-network", "", "openstack network")

	flag.StringVar(&options.vSphereCredentials.Username, "vsphere-username", "", "vsphere username")
	flag.StringVar(&options.vSphereCredentials.Password, "vsphere-password", "", "vsphere password")

	flag.StringVar(&options.azureCredentials.TenantID, "azure-tenant-id", "", "azure tenant id")
	flag.StringVar(&options.azureCredentials.SubscriptionID, "azure-subscription-id", "", "azure subscription id")
	flag.StringVar(&options.azureCredentials.ClientID, "azure-client-id", "", "azure client id")
	flag.StringVar(&options.azureCredentials.ClientSecret, "azure-client-secret", "", "azure client secret")

	flag.StringVar(&options.awsCredentials.AccessKeyID, "aws-access-key-id", "", "AWS access key ID")
	flag.StringVar(&options.awsCredentials.SecretAccessKey, "aws-secret-access-key", "", "AWS secret access key")

	options.logOptions.AddFlags(flag.CommandLine)
}

func TestCCMMigration(t *testing.T) {
	ctx := context.Background()
	seedClient, _, _ := e2eutils.GetClientsOrDie()

	clusterClientProvider, err := clusterclient.NewExternal(seedClient)
	if err != nil {
		t.Fatalf("Failed to create cluster cluster provider: %v", err)
	}

	// prepare cluster
	clusterJig, cluster, userClient, err := setupClusterByProvider(t, ctx, seedClient, clusterClientProvider, options)
	if err != nil {
		t.Fatalf("Failed to setup preconditions: %v", err)
	}

	// run tests
	if err := testBody(t, ctx, clusterJig, cluster, userClient); err != nil {
		t.Error(err)
	}

	// cleanup
	if err := clusterJig.Cleanup(ctx, userClient); err != nil {
		t.Error(err)
	}
}

func setupClusterByProvider(t *testing.T, ctx context.Context, seedClient ctrlruntimeclient.Client, clientProvider *clusterclient.Provider, options testOptions) (providers.ClusterJigInterface, *kubermaticv1.Cluster, ctrlruntimeclient.Client, error) {
	var clusterJig providers.ClusterJigInterface

	logger := log.NewFromOptions(options.logOptions).Sugar().With("provider", options.provider)
	version := options.KubernetesVersion()

	switch kubermaticv1.ProviderType(options.provider) {
	case kubermaticv1.OpenstackCloudProvider:
		clusterJig = providers.NewClusterJigOpenstack(seedClient, logger, version, options.osSeedDatacenter, options.osCredentials)
	case kubermaticv1.VSphereCloudProvider:
		clusterJig = providers.NewClusterJigVsphere(seedClient, logger, version, options.vsphereSeedDatacenter, options.vSphereCredentials)
	case kubermaticv1.AzureCloudProvider:
		clusterJig = providers.NewClusterJigAzure(seedClient, logger, version, options.azureSeedDatacenter, options.azureCredentials)
	case kubermaticv1.AWSCloudProvider:
		clusterJig = providers.NewClusterJigAWS(seedClient, logger, version, options.awsSeedDatacenter, options.awsCredentials)
	default:
		return nil, nil, nil, errors.New("provider not supported for CCM tests")
	}

	cluster := &kubermaticv1.Cluster{}
	userClient, err := setupAndGetUserClient(t, ctx, clusterJig, cluster, clientProvider)

	return clusterJig, cluster, userClient, err
}

func setupAndGetUserClient(t *testing.T, ctx context.Context, clusterJig providers.ClusterJigInterface, cluster *kubermaticv1.Cluster, clusterClientProvider *clusterclient.Provider) (ctrlruntimeclient.Client, error) {
	logger := clusterJig.Log()

	logger.Info("Setting up cluster...")
	if err := clusterJig.Setup(ctx); err != nil {
		return nil, fmt.Errorf("failed to create user cluster: %w", err)
	}

	logger = logger.With("cluster", clusterJig.Name())
	logger.Info("Cluster has been created.")
	if err := clusterJig.Seed().Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, cluster); err != nil {
		return nil, fmt.Errorf("failed to get the cluster we just created: %w", err)
	}

	logger.Info("Giving KKP some time to reconcile...")
	time.Sleep(30 * time.Second)

	logger.Info("Waiting for user cluster to become available...")
	var userClient ctrlruntimeclient.Client
	err := wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		if err := clusterJig.Seed().Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, cluster); err != nil {
			return false, err
		}

		var err error
		userClient, err = clusterClientProvider.GetClient(ctx, cluster)
		if err != nil {
			logger.Debugw("User cluster not ready yet", zap.Error(err))
		}

		return err == nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check cluster readiness: %w", err)
	}
	logger.Info("Cluster is available.")

	if err := clusterv1alpha1.AddToScheme(userClient.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to setup scheme: %w", err)
	}

	logger.Info("Creating MachineDeployment...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		err := clusterJig.CreateMachineDeployment(ctx, userClient)
		if err != nil {
			logger.Errorw("MachineDeployment creation failed", zap.Error(err))
		}

		return err == nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MachineDeployment: %w", err)
	}
	logger.Info("MachineDeployment created.")

	logger.Info("Waiting for node(s) to be ready...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		ready, err := clusterJig.WaitForNodeToBeReady(ctx, userClient)
		if err != nil {
			logger.Errorw("Failed to check node readiness", zap.Error(err))
		}
		return ready, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for Nodes: %w", err)
	}
	logger.Info("Node(s) are ready.")

	return userClient, nil
}

func testBody(t *testing.T, ctx context.Context, clusterJig providers.ClusterJigInterface, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) error {
	seedClient := clusterJig.Seed()
	logger := clusterJig.Log()

	logger.Info("Enabling externalCloudProvider feature...")
	if err := seedClient.Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	newCluster := cluster.DeepCopy()
	newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] = true
	newCluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] = true

	if err := seedClient.Patch(ctx, newCluster, ctrlruntimeclient.MergeFrom(cluster)); err != nil {
		return fmt.Errorf("failed to patch cluster: %w", err)
	}

	logger.Info("Asserting the annotations existence in the cluster...")
	err := wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		annotatedCluster := &kubermaticv1.Cluster{}
		if err := seedClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, annotatedCluster); err != nil {
			return false, fmt.Errorf("failed to get cluster: %w", err)
		}

		_, ccmOk := annotatedCluster.Annotations[kubermaticv1.CCMMigrationNeededAnnotation]
		_, csiOk := annotatedCluster.Annotations[kubermaticv1.CSIMigrationNeededAnnotation]

		return ccmOk && csiOk, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for annotations to appear: %w", err)
	}

	logger.Info("Checking the -node-external-cloud-provider flag in the machine-controller webhook Pod...")
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

	logger.Info("Rotating all the machines...")
	machines := &clusterv1alpha1.MachineList{}
	if err := userClient.List(ctx, machines); err != nil {
		return fmt.Errorf("failed to list machines: %w", err)
	}

	for _, m := range machines.Items {
		if err := userClient.Delete(ctx, &m); err != nil {
			return fmt.Errorf("failed to delete machine: %w", err)
		}
	}

	logger.Info("Waiting for the complete cluster migration...")
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

	logger.Info("Waiting for node(s) to come up again...")
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

	logger.Info("Checking that all the needed components are up and running...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		return clusterJig.CheckComponents(ctx, userClient)
	})
	if err != nil {
		return fmt.Errorf("failed to wait for components: %w", err)
	}

	return nil
}
