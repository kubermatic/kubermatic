// +build e2e

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

package nodeportproxy

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"k8c.io/kubermatic/v2/pkg/controller/nodeport-proxy/envoymanager"

	"k8s.io/apimachinery/pkg/util/sets"
)

var _ = ginkgo.Describe("NodeportProxy", func() {
	ginkgo.Describe("services of type node port", func() {
		var svcJig *ServiceJig
		ginkgo.BeforeEach(func() {
			k8scli, _, _ := GetClientsOrDie()
			svcJig = &ServiceJig{
				Log:    logger,
				Client: k8scli,
			}
		})
		ginkgo.AfterEach(func() {
			if !skipCleanup {
				gomega.Expect(svcJig.CleanUp()).NotTo(gomega.HaveOccurred())
			}
		})
		ginkgo.Context("with the proper annotation", func() {
			ginkgo.BeforeEach(func() {
				// nodePort set to 0 so that it gets allocated dynamically.
				gomega.Expect(svcJig.CreateNodePortService("service-a", 0, 1, map[string]string{envoymanager.DefaultExposeAnnotationKey: "true"})).
					NotTo(gomega.BeNil(), "NodePort service creation failed")
				// nodePort set to 0 so that it gets allocated dynamically.
				gomega.Expect(svcJig.CreateNodePortService("service-b", 0, 2, map[string]string{envoymanager.DefaultExposeAnnotationKey: "true"})).
					NotTo(gomega.BeNil(), "NodePort service creation failed")
			})

			ginkgo.It("should be exposed", func() {
				ginkgo.By("updating the lb service")
				portsToBeExposed := sets.NewInt32()
				logger.Debugw("computing ports to be exposed", "services", svcJig.Services)
				for _, svc := range svcJig.Services {
					portsToBeExposed = ExtractNodePorts(svc).Union(portsToBeExposed)
				}
				gomega.Eventually(func() sets.Int32 {
					// When the difference between the node ports of the
					// service to be exposed and the ports of the lb server is
					// empty, it means that all ports was exposed.
					lbSvc := deployer.GetLbService()
					gomega.Expect(lbSvc).ShouldNot(gomega.BeNil())
					return portsToBeExposed.Difference(ExtractPorts(lbSvc))
				}, "10m", "1s").Should(gomega.HaveLen(0), "All exposed service ports should be reflected in the lb service")

				ginkgo.By("load-balancing on available endpoints")
				for _, svc := range svcJig.Services {
					lbSvc := deployer.GetLbService()
					targetNp := FindExposingNodePort(lbSvc, svc.Spec.Ports[0].NodePort)
					logger.Debugw("found target nodeport in lb service", "service", svc, "port", targetNp)
					gomega.Expect(networkingTest.DialFromNode("127.0.0.1", int(targetNp), 5, 1, sets.NewString(svcJig.ServicePods[svc.Name]...))).Should(gomega.HaveLen(0), "All exposed endpoints should be hit")
				}
			})
		})

		ginkgo.Context("without the proper annotation", func() {
			ginkgo.BeforeEach(func() {
				// nodePort set to 0 so that it gets allocated dynamically.
				gomega.Expect(svcJig.CreateNodePortService("service-a", 0, 1, map[string]string{})).
					NotTo(gomega.BeNil(), "NodePort service creation failed")
				gomega.Expect(svcJig.CreateNodePortService("service-b", 0, 1, map[string]string{envoymanager.DefaultExposeAnnotationKey: "false"})).
					NotTo(gomega.BeNil(), "NodePort service creation failed")
			})

			ginkgo.It("should not be exposed", func() {
				portsNotToBeExposed := sets.NewInt32()
				for _, svc := range svcJig.Services {
					portsNotToBeExposed = ExtractNodePorts(svc).Union(portsNotToBeExposed)
				}
				gomega.Consistently(func() sets.Int32 {
					// When the difference between the node ports of the
					// service to be exposed and the ports of the lb server is
					// empty, it means that all ports was exposed.
					lbSvc := deployer.GetLbService()
					gomega.Expect(lbSvc).ShouldNot(gomega.BeNil())
					return portsNotToBeExposed.Intersection(ExtractPorts(lbSvc))
				}, "10s", "1s").Should(gomega.HaveLen(0), "None of the ports should be reflected in the lb service")
			})
		})

	})

})
