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

package exposestrategy

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/rand"

	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
)

var _ = ginkgo.Describe("The Tunneling strategy", func() {
	var (
		clusterJig  *ClusterJig
		agentConfig *AgentConfig
		client      *clientJig
	)
	ginkgo.BeforeEach(func() {
		k8scli, restCli, restConf := e2eutils.GetClientsOrDie()
		clusterJig = &ClusterJig{
			Log:            e2eutils.DefaultLogger,
			Client:         k8scli,
			Name:           rand.String(10),
			DatacenterName: options.datacenter,
			Version:        options.kubernetesVersion,
		}
		gomega.Expect(clusterJig.SetUp()).NotTo(gomega.HaveOccurred(), "user cluster should deploy successfully")
		agentConfig = &AgentConfig{
			Log:       e2eutils.DefaultLogger,
			Client:    k8scli,
			Namespace: clusterJig.Cluster.Status.NamespaceName,
			Versions:  kubermatic.NewDefaultVersions(),
		}
		client = &clientJig{e2eutils.TestPodConfig{
			Log:           e2eutils.DefaultLogger,
			Namespace:     clusterJig.Cluster.Status.NamespaceName,
			Client:        k8scli,
			PodRestClient: restCli,
			Config:        restConf,
			CreatePodFunc: newClientPod,
		}}
		gomega.Expect(agentConfig.DeployAgentPod()).NotTo(gomega.HaveOccurred(), "agent should deploy successfully")
		gomega.Expect(client.DeployTestPod()).NotTo(gomega.HaveOccurred(), "client pod should deploy successfully")
	})
	ginkgo.AfterEach(func() {
		if !options.skipCleanup {
			gomega.Expect(clusterJig.CleanUp()).NotTo(gomega.HaveOccurred())
			if agentConfig != nil {
				gomega.Expect(agentConfig.CleanUp()).NotTo(gomega.HaveOccurred())
			}
			if client != nil {
				gomega.Expect(client.CleanUp()).NotTo(gomega.HaveOccurred())
			}
		}
	})
	ginkgo.Context("uses nodeport-proxy to expose control plane components", func() {
		ginkgo.It("exposes the KAS", func() {
			ginkgo.By("relying on SNI when Kubeconfig is used e.g. Kubelet")
			gomega.Expect(client.QueryApiserverVersion("", false, options.kubernetesVersion, 5, 4)).To(gomega.BeTrue(), "Apiserver should be reachable passing from the SNI entrypoint in nodeport proxy")
			ginkgo.By("tunneling requests using HTTP/2 CONNECT when no SNI is present e.g. pods relying on kubernetes service in default namespace")
			// TODO(irozzo): For sake of simplicity we are deploying an agent in the
			// seed cluster. It would be better to create workers in the future for
			// better coverage.
			gomega.Expect(client.QueryApiserverVersion(agentConfig.GetKASHostPort(), true, options.kubernetesVersion, 5, 4)).To(gomega.BeTrue(), "Apiserver should be reachable passing from the SNI entrypoint in nodeport proxy")
		})
	})
})
