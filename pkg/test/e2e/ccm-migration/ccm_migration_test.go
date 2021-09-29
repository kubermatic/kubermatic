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

		openstackClusterJig *providers.OpenstackClusterJig
		openstackCluster    *kubermaticv1.Cluster

		err error
	)

	ginkgo.BeforeEach(func() {
		seedClient, _, _ = e2eutils.GetClientsOrDie()
		clusterClientProvider, err = clusterclient.NewExternal(seedClient)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("OpenStack provider", func() {
		ginkgo.BeforeEach(func() {
			openstackClusterJig = &providers.OpenstackClusterJig{
				providers.CommonClusterJig{
					Name:           options.userClusterName,
					DatacenterName: options.datacenter,
					Version:        options.kubernetesVersion,
					SeedClient:     seedClient,
				}, e2eutils.DefaultLogger, options.osCredentials}

			gomega.Expect(openstackClusterJig.Setup()).NotTo(gomega.HaveOccurred(), "user cluster should deploy successfully")
			openstackClusterJig.Log.Debugw("Cluster set up", "name", openstackClusterJig.Name)
			gomega.Expect(openstackClusterJig.SeedClient.Get(context.TODO(), types.NamespacedName{Name: openstackClusterJig.Name}, openstackCluster)).To(nil)

			gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
				userClient, err = clusterClientProvider.GetClient(context.TODO(), openstackCluster)
				if err != nil {
					openstackClusterJig.Log.Debug("user cluster client get failed, retrying...")
					return false, nil
				}
				return true, nil
			})).NotTo(gomega.HaveOccurred())
			gomega.Expect(clusterv1alpha1.AddToScheme(userClient.Scheme())).NotTo(gomega.HaveOccurred())

			gomega.Expect(openstackClusterJig.CreateMachineDeployment(userClient))
			openstackClusterJig.Log.Debug("MachineDeployment created")
		})

		ginkgo.AfterEach(func() {
			gomega.Expect(openstackClusterJig.CleanUp(userClient)).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("migrating cluster to external CCM", func() {
			ginkgo.By("enabling externalCloudProvider feature")
			gomega.Expect(openstackClusterJig.SeedClient.Get(context.TODO(), types.NamespacedName{Name: openstackClusterJig.Name}, openstackCluster)).To(nil)
			newCluster := openstackCluster.DeepCopy()
			newCluster.Spec.Features = map[string]bool{
				kubermaticv1.ClusterFeatureExternalCloudProvider: true,
			}
			gomega.Expect(openstackClusterJig.SeedClient.Patch(context.TODO(), newCluster, ctrlruntimeclient.MergeFrom(openstackCluster))).NotTo(gomega.HaveOccurred())

			ginkgo.By("getting the patched cluster")
			annotatedCluster := &kubermaticv1.Cluster{}
			gomega.Expect(openstackClusterJig.SeedClient.Get(context.TODO(), types.NamespacedName{Name: openstackCluster.Name}, annotatedCluster)).NotTo(gomega.HaveOccurred())

			ginkgo.By("asserting the annotations existence in the cluster")
			_, ccmOk := annotatedCluster.Annotations[kubermaticv1.CCMMigrationNeededAnnotation]
			_, csiOk := annotatedCluster.Annotations[kubermaticv1.CSIMigrationNeededAnnotation]
			gomega.Expect(ccmOk && csiOk).To(gomega.BeTrue())

			ginkgo.By("checking the -node-external-cloud-provider flag in the machineController webhook pod")
			gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
				machineControllerWebhookPods := &corev1.PodList{}
				if err := openstackClusterJig.SeedClient.List(context.TODO(), machineControllerWebhookPods, ctrlruntimeclient.InNamespace(openstackCluster.Status.NamespaceName), ctrlruntimeclient.MatchingLabels{
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
			gomega.Expect(userClient.List(context.Background(), machines)).NotTo(gomega.HaveOccurred())
			for _, m := range machines.Items {
				gomega.Expect(userClient.Delete(context.Background(), &m)).NotTo(gomega.HaveOccurred())
			}

			ginkgo.By("waiting for the complete cluster migration")
			gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
				migratingCluster := &kubermaticv1.Cluster{}
				if err := openstackClusterJig.SeedClient.Get(context.TODO(), types.NamespacedName{Name: openstackClusterJig.Name}, migratingCluster); err != nil {
					return false, err
				}

				return migratingCluster.Status.Conditions[kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted].Status == corev1.ConditionTrue, nil
			})).NotTo(gomega.HaveOccurred())

			ginkgo.By("checking that all the needed components are up and running")
			gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
				return openstackClusterJig.CheckComponents()
			})).NotTo(gomega.HaveOccurred())
		})
	})
})
