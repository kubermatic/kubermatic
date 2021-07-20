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
	"fmt"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	types2 "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	userClusterPollInterval = 5 * time.Second
	customTestTimeout       = 10 * time.Minute
)

var _ = ginkgo.Describe("CCM migration", func() {
	var (
		userClient ctrlruntimeclient.Client
		clusterJig *ClusterJig
	)

	ginkgo.Context("supported provider", func() {
		ginkgo.BeforeEach(func() {
			seedClient, _, _ := e2eutils.GetClientsOrDie()
			clusterClientProvider, err := clusterclient.NewExternal(seedClient)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			clusterJig = &ClusterJig{
				Log:            e2eutils.DefaultLogger,
				SeedClient:     seedClient,
				Name:           rand.String(10),
				DatacenterName: options.datacenter,
				Version:        options.kubernetesVersion,
			}

			gomega.Expect(clusterJig.SetUp(kubermaticv1.CloudSpec{
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					CredentialsReference: &types2.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Name:      fmt.Sprintf("credential-openstack-%s", clusterJig.Name),
							Namespace: resources.KubermaticNamespace,
						},
					},
				},
				DatacenterName: options.datacenter,
			}, options.osCredentials)).NotTo(gomega.HaveOccurred(), "user cluster should deploy successfully")
			clusterJig.Log.Debugw("Cluster set up", "name", clusterJig.Cluster.Name)

			time.Sleep(1000*time.Second)

			userClient, err = clusterClientProvider.GetClient(context.TODO(), clusterJig.Cluster)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("migrating cluster to external CCM", func() {
			ginkgo.By("enabling externalCloudProvider feature")
			newCluster := clusterJig.Cluster.DeepCopy()
			newCluster.Spec.Features = map[string]bool{
				kubermaticv1.ClusterFeatureExternalCloudProvider: true,
			}
			gomega.Expect(clusterJig.SeedClient.Patch(context.TODO(), newCluster, ctrlruntimeclient.MergeFrom(clusterJig.Cluster))).NotTo(gomega.HaveOccurred())

			ginkgo.By("getting the patched cluster")
			annotatedCluster := &kubermaticv1.Cluster{}
			gomega.Expect(clusterJig.SeedClient.Get(context.TODO(), types.NamespacedName{Name: clusterJig.Cluster.Name}, annotatedCluster)).NotTo(gomega.HaveOccurred())

			ginkgo.By("asserting the annotations not existence in the cluster")
			_, ccmOk := annotatedCluster.Annotations[kubermaticv1.CCMMigrationNeededAnnotation]
			_, csiOk := annotatedCluster.Annotations[kubermaticv1.CSIMigrationNeededAnnotation]
			gomega.Expect(ccmOk && csiOk).To(gomega.BeTrue())

			ginkgo.By("checking the -node-external-cloud-provider flag in the machineController webhook pod")
			gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
				machineControllerWebhookPods := &corev1.PodList{}
				if err := clusterJig.SeedClient.List(context.TODO(), machineControllerWebhookPods, ctrlruntimeclient.InNamespace(clusterJig.Cluster.Status.NamespaceName), ctrlruntimeclient.MatchingLabels{
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

			ginkgo.By("rolling out all the machines")
			gomega.Expect(userClient.DeleteAllOf(context.TODO(), &v1alpha1.Machine{})).NotTo(gomega.HaveOccurred())

			ginkgo.By("waiting for the complete cluster migration")
			gomega.Expect(wait.Poll(userClusterPollInterval, customTestTimeout, func() (done bool, err error) {
				migratingCluster := &kubermaticv1.Cluster{}
				if err := clusterJig.SeedClient.Get(context.TODO(), types.NamespacedName{Name: clusterJig.Cluster.Name}, migratingCluster); err != nil {
					return false, err
				}
				if helper.ClusterConditionHasStatus(migratingCluster, kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted, corev1.ConditionTrue) {
					return true, nil
				}
				return false, nil
			})).NotTo(gomega.HaveOccurred())
		})
	})

	/*ginkgo.Context("unsupported provider", func() {
		ginkgo.BeforeEach(func() {
			k8scli, _, _ := e2eutils.GetClientsOrDie()
			clusterJig = &ClusterJig{
				Log:            e2eutils.DefaultLogger,
				SeedClient:         k8scli,
				Name:           rand.String(10),
				DatacenterName: options.datacenter,
				Version:        options.kubernetesVersion,
			}
			gomega.Expect(clusterJig.SetUp(kubermaticv1.CloudSpec{
				Fake:   &kubermaticv1.FakeCloudSpec{},
				DatacenterName: options.datacenter,
			})).NotTo(gomega.HaveOccurred(), "user cluster should deploy successfully")
		})

		ginkgo.It("migrating cluster to external CCM", func() {
			ginkgo.By("enabling externalCloudProvider feature")
			newCluster := clusterJig.Cluster.DeepCopy()
			newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] = true
			gomega.Expect(clusterJig.SeedClient.Patch(context.TODO(), newCluster, ctrlruntimeclient.MergeFrom(clusterJig.Cluster))).To(nil)

			ginkgo.By("getting the patched cluster")
			annotatedCluster := &kubermaticv1.Cluster{}
			gomega.Expect(clusterJig.SeedClient.Get(context.TODO(), types.NamespacedName{Name: clusterJig.Cluster.Name}, annotatedCluster)).To(nil)

			ginkgo.By("asserting the annotations not existence in the cluster")
			_, ccmOk := annotatedCluster.Annotations[kubermaticv1.CCMMigrationNeededAnnotation]
			_, csiOk := annotatedCluster.Annotations[kubermaticv1.CSIMigrationNeededAnnotation]
			gomega.Expect(ccmOk && csiOk).To(gomega.BeFalse())
		})
	})*/
})
