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
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/providers"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	userClusterPollInterval = 5 * time.Second
	customTestTimeout       = 10 * time.Minute
)

var _ = ginkgo.Describe("CCM migration", func() {
	ctx := context.Background()

	var (
		seedClient            ctrlruntimeclient.Client
		userClient            ctrlruntimeclient.Client
		clusterClientProvider *clusterclient.Provider
		clusterJig            providers.ClusterJigInterface

		err error
	)

	ginkgo.BeforeEach(func() {
		seedClient, _, _ = e2eutils.GetClientsOrDie()
		clusterClientProvider, err = clusterclient.NewExternal(seedClient)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("vSphere provider", func() {
		var (
			vsphereCluster *kubermaticv1.Cluster
		)

		ginkgo.BeforeEach(func() {
			vsphereCluster = &kubermaticv1.Cluster{}
			clusterJig = providers.NewClusterJigVsphere(seedClient, options.kubernetesVersion, options.vsphereSeedDatacenter, options.vSphereCredentials)
			userClient = setupAndGetUserClient(clusterJig, vsphereCluster, clusterClientProvider)
		})

		ginkgo.AfterEach(func() {
			//gomega.Expect(clusterJig.Cleanup(userClient)).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("migrating cluster to external CCM", func() {
			testBody(clusterJig, vsphereCluster, userClient)
		})
	})

	ginkgo.Context("OpenStack provider", func() {
		var (
			openstackCluster *kubermaticv1.Cluster
		)

		ginkgo.BeforeEach(func() {
			openstackCluster = &kubermaticv1.Cluster{}
			clusterJig = providers.NewClusterJigOpenstack(seedClient, options.kubernetesVersion, options.osSeedDatacenter, options.osCredentials)
			userClient = setupAndGetUserClient(clusterJig, openstackCluster, clusterClientProvider)
		})

		ginkgo.AfterEach(func() {
			//gomega.Expect(clusterJig.Cleanup(userClient)).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("migrating cluster to external CCM", func() {
			testBody(clusterJig, openstackCluster, userClient)
		})
	})
})

func setupAndGetUserClient(clusterJig providers.ClusterJigInterface, cluster *kubermaticv1.Cluster, clusterClientProvider *clusterclient.Provider) ctrlruntimeclient.Client {
	ginkgo.By("setting up cluster")
	gomega.Expect(clusterJig.Setup()).NotTo(gomega.HaveOccurred(), "user cluster should deploy successfully")
	clusterJig.Log().Debugw("Cluster set up", "name", clusterJig.Name())
	gomega.Expect(clusterJig.Seed().Get(context.TODO(), types.NamespacedName{Name: clusterJig.Name()}, cluster)).NotTo(gomega.HaveOccurred())

	gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
		if err := clusterJig.ExposeAPIServer(); err != nil {
			clusterJig.Log().Debug("nodeport creation failed, retrying...")
			return false, nil
		}
		return true, nil
	})).NotTo(gomega.HaveOccurred())

	clusterJig.Log().Debugw("User cluster exposed through NodePort", clusterJig.Name())

	var userClient ctrlruntimeclient.Client
	gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
		gomega.Expect(clusterJig.Seed().Get(context.TODO(), types.NamespacedName{Name: clusterJig.Name()}, cluster)).NotTo(gomega.HaveOccurred())
		userClient, err = clusterClientProvider.GetClient(context.TODO(), cluster)
		if err != nil {
			clusterJig.Log().Debug("user cluster client get failed, retrying...")
			return false, nil
		}
		return true, nil
	})).NotTo(gomega.HaveOccurred())
	gomega.Expect(clusterv1alpha1.AddToScheme(userClient.Scheme())).NotTo(gomega.HaveOccurred())

	gomega.Expect(clusterJig.CreateMachineDeployment(userClient)).NotTo(gomega.HaveOccurred())
	clusterJig.Log().Debug("MachineDeployment created")

	return userClient
}

func testBody(clusterJig providers.ClusterJigInterface, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) {
	ginkgo.By("enabling externalCloudProvider feature")
	gomega.Expect(clusterJig.Seed().Get(context.TODO(), types.NamespacedName{Name: clusterJig.Name()}, cluster)).NotTo(gomega.HaveOccurred())
	newCluster := cluster.DeepCopy()
	newCluster.Spec.Features = map[string]bool{
		kubermaticv1.ClusterFeatureExternalCloudProvider: true,
	}
	gomega.Expect(clusterJig.Seed().Patch(context.TODO(), newCluster, ctrlruntimeclient.MergeFrom(cluster))).NotTo(gomega.HaveOccurred())

	ginkgo.By("getting the patched cluster")
	annotatedCluster := &kubermaticv1.Cluster{}
	gomega.Expect(clusterJig.Seed().Get(context.TODO(), types.NamespacedName{Name: cluster.Name}, annotatedCluster)).NotTo(gomega.HaveOccurred())

	ginkgo.By("asserting the annotations existence in the cluster")
	_, ccmOk := annotatedCluster.Annotations[kubermaticv1.CCMMigrationNeededAnnotation]
	_, csiOk := annotatedCluster.Annotations[kubermaticv1.CSIMigrationNeededAnnotation]
	gomega.Expect(ccmOk && csiOk).To(gomega.BeTrue())

	ginkgo.By("checking the -node-external-cloud-provider flag in the machineController webhook pod")
	gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
		machineControllerWebhookPods := &corev1.PodList{}
		if err := clusterJig.Seed().List(context.TODO(), machineControllerWebhookPods, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName), ctrlruntimeclient.MatchingLabels{
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
	gomega.Expect(userClient.List(context.TODO(), machines)).NotTo(gomega.HaveOccurred())
	for _, m := range machines.Items {
		gomega.Expect(userClient.Delete(context.TODO(), &m)).NotTo(gomega.HaveOccurred())
	}

	ginkgo.By("waiting for the complete cluster migration")
	gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
		migratingCluster := &kubermaticv1.Cluster{}
		if err := clusterJig.Seed().Get(context.TODO(), types.NamespacedName{Name: clusterJig.Name()}, migratingCluster); err != nil {
			return false, err
		}
		if helper.ClusterConditionHasStatus(migratingCluster, kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted, corev1.ConditionTrue) {
			return true, nil
		}
		return false, nil
	})).NotTo(gomega.HaveOccurred())

	ginkgo.By("checking that all the needed components are up and running")
	gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
		return clusterJig.CheckComponents(userClient)
	})).NotTo(gomega.HaveOccurred())
}
