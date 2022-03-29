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
	"strings"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

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

var _ = ginkgo.Describe("CCM migration", func() {
	var (
		seedClient            ctrlruntimeclient.Client
		userClient            ctrlruntimeclient.Client
		clusterClientProvider *clusterclient.Provider
		clusterJig            providers.ClusterJigInterface
		ctx                   context.Context

		err error
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		seedClient, _, _ = e2eutils.GetClientsOrDie()
		clusterClientProvider, err = clusterclient.NewExternal(seedClient)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("testing provider", func() {
		var cluster *kubermaticv1.Cluster

		ginkgo.BeforeEach(func() {
			clusterJig, cluster, userClient = SetupClusterByProvider(ctx, seedClient, clusterClientProvider, options)
		})

		ginkgo.AfterEach(func() {
			gomega.Expect(clusterJig.Cleanup(ctx, userClient)).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("migrating cluster to external CCM", func() {
			testBody(ctx, clusterJig, cluster, userClient)
		})
	})
})

func SetupClusterByProvider(ctx context.Context, seedClient ctrlruntimeclient.Client, clientProvider *clusterclient.Provider, options testOptions) (providers.ClusterJigInterface, *kubermaticv1.Cluster, ctrlruntimeclient.Client) {
	var (
		clusterJig providers.ClusterJigInterface
		userClient ctrlruntimeclient.Client
		cluster    *kubermaticv1.Cluster
	)
	cluster = &kubermaticv1.Cluster{}

	switch kubermaticv1.ProviderType(options.provider) {
	case kubermaticv1.OpenstackCloudProvider:
		clusterJig = providers.NewClusterJigOpenstack(seedClient, options.kubernetesVersion, options.osSeedDatacenter, options.osCredentials)
	case kubermaticv1.VSphereCloudProvider:
		clusterJig = providers.NewClusterJigVsphere(seedClient, options.kubernetesVersion, options.vsphereSeedDatacenter, options.vSphereCredentials)
	default:
		ginkgo.Fail("provider not supported")
	}

	userClient = setupAndGetUserClient(ctx, clusterJig, cluster, clientProvider)
	return clusterJig, cluster, userClient
}

func setupAndGetUserClient(ctx context.Context, clusterJig providers.ClusterJigInterface, cluster *kubermaticv1.Cluster, clusterClientProvider *clusterclient.Provider) ctrlruntimeclient.Client {
	ginkgo.By("setting up cluster")
	gomega.Expect(clusterJig.Setup(ctx)).NotTo(gomega.HaveOccurred(), "user cluster should deploy successfully")
	clusterJig.Log().Debugw("Cluster set up", "name", clusterJig.Name())
	gomega.Expect(clusterJig.Seed().Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, cluster)).NotTo(gomega.HaveOccurred())

	var userClient ctrlruntimeclient.Client
	gomega.Expect(wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (done bool, err error) {
		gomega.Expect(clusterJig.Seed().Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, cluster)).NotTo(gomega.HaveOccurred())
		userClient, err = clusterClientProvider.GetClient(ctx, cluster)
		if err != nil {
			clusterJig.Log().Debug("user cluster client get failed: %v", err)
			return false, nil
		}
		return true, nil
	})).NotTo(gomega.HaveOccurred())
	clusterJig.Log().Debugw("User cluster client correctly created")

	gomega.Expect(clusterv1alpha1.AddToScheme(userClient.Scheme())).NotTo(gomega.HaveOccurred())

	gomega.Expect(wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (done bool, err error) {
		if err = clusterJig.CreateMachineDeployment(ctx, userClient); err != nil {
			clusterJig.Log().Debug("machine deployment creation failed: %v", err)
			return false, nil
		}
		return true, nil
	})).NotTo(gomega.HaveOccurred())
	clusterJig.Log().Debug("MachineDeployment created")

	gomega.Expect(wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (done bool, err error) {
		var ready bool
		if ready, err = clusterJig.WaitForNodeToBeReady(ctx, userClient); err != nil {
			clusterJig.Log().Debug("node not ready yet, %v", err)
			return false, nil
		}
		return ready, nil
	})).NotTo(gomega.HaveOccurred())
	clusterJig.Log().Debug("Node ready")

	return userClient
}

func testBody(ctx context.Context, clusterJig providers.ClusterJigInterface, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) {
	ginkgo.By("enabling externalCloudProvider feature")
	gomega.Expect(clusterJig.Seed().Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, cluster)).NotTo(gomega.HaveOccurred())
	newCluster := cluster.DeepCopy()
	newCluster.Spec.Features = map[string]bool{
		kubermaticv1.ClusterFeatureExternalCloudProvider: true,
	}
	gomega.Expect(clusterJig.Seed().Patch(ctx, newCluster, ctrlruntimeclient.MergeFrom(cluster))).NotTo(gomega.HaveOccurred())

	clusterJig.Log().Debug("getting the patched cluster")
	annotatedCluster := &kubermaticv1.Cluster{}
	gomega.Expect(clusterJig.Seed().Get(ctx, types.NamespacedName{Name: cluster.Name}, annotatedCluster)).NotTo(gomega.HaveOccurred())

	clusterJig.Log().Debug("asserting the annotations existence in the cluster")
	gomega.Expect(wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (done bool, err error) {
		_, ccmOk := annotatedCluster.Annotations[kubermaticv1.CCMMigrationNeededAnnotation]
		_, csiOk := annotatedCluster.Annotations[kubermaticv1.CSIMigrationNeededAnnotation]
		if ccmOk && csiOk {
			return true, nil
		}
		return false, nil
	})).NotTo(gomega.HaveOccurred())

	clusterJig.Log().Debug("checking the -node-external-cloud-provider flag in the machineController webhook pod")
	gomega.Expect(wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (done bool, err error) {
		machineControllerWebhookPods := &corev1.PodList{}
		if err := clusterJig.Seed().List(ctx, machineControllerWebhookPods, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName), ctrlruntimeclient.MatchingLabels{
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
	})).NotTo(gomega.HaveOccurred())

	machines := &clusterv1alpha1.MachineList{}
	ginkgo.By("rolling out all the machines")
	gomega.Expect(userClient.List(ctx, machines)).NotTo(gomega.HaveOccurred())
	for _, m := range machines.Items {
		gomega.Expect(userClient.Delete(ctx, &m)).NotTo(gomega.HaveOccurred())
	}

	ginkgo.By("waiting for the complete cluster migration")
	gomega.Expect(wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (done bool, err error) {
		migratingCluster := &kubermaticv1.Cluster{}
		if err := clusterJig.Seed().Get(ctx, types.NamespacedName{Name: clusterJig.Name()}, migratingCluster); err != nil {
			return false, err
		}
		return kubermaticv1helper.CCMMigrationCompleted(migratingCluster), nil
	})).NotTo(gomega.HaveOccurred())

	clusterJig.Log().Debug("waiting for node to come up again")
	gomega.Expect(wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (done bool, err error) {
		var ready bool
		if ready, err = clusterJig.WaitForNodeToBeReady(ctx, userClient); err != nil {
			clusterJig.Log().Debug("node not ready yet, %v", err)
			return false, nil
		}
		return ready, nil
	})).NotTo(gomega.HaveOccurred())

	ginkgo.By("checking that all the needed components are up and running")
	gomega.Expect(wait.Poll(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (done bool, err error) {
		return clusterJig.CheckComponents(ctx, userClient)
	})).NotTo(gomega.HaveOccurred())
}
