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

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/api/v3/pkg/semver"
	clusterhelper "k8c.io/kubermatic/v3/pkg/cluster"
	"k8c.io/kubermatic/v3/pkg/log"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/test"
	"k8c.io/kubermatic/v3/pkg/test/e2e/ccm-migration/providers"
	"k8c.io/kubermatic/v3/pkg/test/e2e/ccm-migration/utils"
	"k8c.io/kubermatic/v3/pkg/test/e2e/jig"
	e2eutils "k8c.io/kubermatic/v3/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Options holds the e2e test options.
type testOptions struct {
	skipCleanup       bool
	logOptions        log.Options
	kubernetesRelease string

	provider string

	osCredentials      jig.OpenStackCredentials
	vsphereCredentials jig.VSphereCredentials
	azureCredentials   jig.AzureCredentials
	awsCredentials     jig.AWSCredentials
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
	flag.StringVar(&options.kubernetesRelease, "kubernetes-release", "", "Kubernetes version or release for the usercluster")

	options.awsCredentials.AddFlags(flag.CommandLine)
	options.azureCredentials.AddFlags(flag.CommandLine)
	options.osCredentials.AddFlags(flag.CommandLine)
	options.vsphereCredentials.AddFlags(flag.CommandLine)
	options.logOptions.AddFlags(flag.CommandLine)

	jig.AddFlags(flag.CommandLine)
}

func TestCCMMigration(t *testing.T) {
	switch kubermaticv1.CloudProvider(options.provider) {
	case kubermaticv1.CloudProviderAWS:
		runtime.Must(options.awsCredentials.Parse())
	case kubermaticv1.CloudProviderAzure:
		runtime.Must(options.azureCredentials.Parse())
	case kubermaticv1.CloudProviderOpenStack:
		runtime.Must(options.osCredentials.Parse())
	case kubermaticv1.CloudProviderVSphere:
		runtime.Must(options.vsphereCredentials.Parse())
	default:
		t.Fatalf("Unknown provider %q", options.provider)
	}

	ctx := context.Background()
	seedClient, _, _ := e2eutils.GetClientsOrDie()
	log := log.NewFromOptions(options.logOptions).Sugar().With("provider", options.provider)

	// prepare cluster
	scenario, cluster, userClient, err := setupClusterByProvider(t, ctx, log, seedClient, options)
	if err != nil {
		t.Fatalf("Failed to setup preconditions: %v", err)
	}

	// run tests
	if err := testBody(t, ctx, log, seedClient, scenario, cluster, userClient); err != nil {
		t.Error(err)
	}

	// cleanup
	if err := scenario.Cleanup(ctx, cluster, userClient); err != nil {
		t.Error(err)
	}
}

func setupClusterByProvider(t *testing.T, ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, options testOptions) (providers.TestScenario, *kubermaticv1.Cluster, ctrlruntimeclient.Client, error) {
	var scenario providers.TestScenario

	version := options.KubernetesVersion()

	switch kubermaticv1.CloudProvider(options.provider) {
	case kubermaticv1.CloudProviderOpenStack:
		scenario = providers.NewOpenStackScenario(log, seedClient, options.osCredentials)
	case kubermaticv1.CloudProviderVSphere:
		scenario = providers.NewVSphereScenario(log, seedClient, options.vsphereCredentials)
	case kubermaticv1.CloudProviderAzure:
		scenario = providers.NewAzureScenario(log, seedClient, options.azureCredentials)
	case kubermaticv1.CloudProviderAWS:
		scenario = providers.NewAWSScenario(log, seedClient, options.awsCredentials)
	default:
		return nil, nil, nil, errors.New("provider not supported for CCM tests")
	}

	scenario.ClusterJig().
		WithName("ccmmig-" + rand.String(5)).
		WithVersion(version.String()).
		WithFeatures(map[string]bool{
			kubermaticv1.ClusterFeatureExternalCloudProvider: false,
		})

	cluster, err := scenario.Setup(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	userClient, err := scenario.ClusterJig().ClusterClient(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return scenario, cluster, userClient, err
}

func testBody(t *testing.T, ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, scenario providers.TestScenario, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) error {
	log.Info("Enabling externalCloudProvider feature...")
	newCluster := cluster.DeepCopy()
	newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] = true
	newCluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] = true

	if err := seedClient.Patch(ctx, newCluster, ctrlruntimeclient.MergeFrom(cluster)); err != nil {
		return fmt.Errorf("failed to patch cluster: %w", err)
	}

	log.Info("Asserting the annotations existence in the cluster...")
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

	log.Info("Checking the -node-external-cloud-provider flag in the machine-controller webhook Pod...")
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

	log.Info("Rotating all the machines...")
	machines := &clusterv1alpha1.MachineList{}
	if err := userClient.List(ctx, machines); err != nil {
		return fmt.Errorf("failed to list machines: %w", err)
	}

	for _, m := range machines.Items {
		if err := userClient.Delete(ctx, &m); err != nil {
			return fmt.Errorf("failed to delete machine: %w", err)
		}
	}

	log.Info("Waiting for the complete cluster migration...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		migratingCluster := &kubermaticv1.Cluster{}
		if err := seedClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, migratingCluster); err != nil {
			return false, err
		}
		return clusterhelper.CCMMigrationCompleted(migratingCluster), nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for migration to finish: %w", err)
	}

	log.Info("Waiting for node(s) to come up again...")
	if err := scenario.MachineJig().WaitForReadyNodes(ctx, userClient); err != nil {
		return fmt.Errorf("failed to wait for nodes: %w", err)
	}

	log.Info("Checking that all the needed components are up and running...")
	err = wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		return scenario.CheckComponents(ctx, cluster, userClient)
	})
	if err != nil {
		return fmt.Errorf("failed to wait for components: %w", err)
	}

	return nil
}
