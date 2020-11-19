package nodeport_proxy

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/sets"

	"k8c.io/kubermatic/v2/pkg/controller/nodeport-proxy/envoymanager"
)

var _ = Describe("NodeportProxy", func() {
	Describe("services of type node port", func() {
		var svcJig *ServiceJig
		BeforeEach(func() {
			k8scli, _, _ := GetClientsOrDie()
			svcJig = &ServiceJig{
				Log:    logger,
				Client: k8scli,
			}
		})
		AfterEach(func() {
			if !skipCleanup {
				Expect(svcJig.CleanUp()).NotTo(HaveOccurred())
			}
		})
		Context("with the proper annotation", func() {
			BeforeEach(func() {
				// nodePort set to 0 so that it gets allocated dynamically.
				Expect(svcJig.CreateNodePortService("service-a", 0, 1, map[string]string{envoymanager.DefaultExposeAnnotationKey: "true"})).
					NotTo(BeNil(), "NodePort service creation failed")
				// nodePort set to 0 so that it gets allocated dynamically.
				Expect(svcJig.CreateNodePortService("service-b", 0, 2, map[string]string{envoymanager.DefaultExposeAnnotationKey: "true"})).
					NotTo(BeNil(), "NodePort service creation failed")
			})

			It("should be exposed", func() {
				By("updating the lb service")
				portsToBeExposed := sets.NewInt32()
				for _, svc := range svcJig.Services {
					portsToBeExposed = ExtractNodePorts(svc).Union(portsToBeExposed)
				}
				Eventually(func() sets.Int32 {
					// When the difference between the node ports of the
					// service to be exposed and the ports of the lb server is
					// empty, it means that all ports was exposed.
					lbSvc := deployer.GetLbService()
					Expect(lbSvc).ShouldNot(BeNil())
					return portsToBeExposed.Difference(ExtractPorts(lbSvc))
				}, "10m", "1s").Should(HaveLen(0), "All exposed service ports should be reflected in the lb service")

				By("load-balancing on available endpoints")
				for _, svc := range svcJig.Services {
					lbSvc := deployer.GetLbService()
					Expect(networkingTest.DialFromNode("127.0.0.1", int(FindExposingNodePort(lbSvc, svc.Spec.Ports[0].NodePort)), 5, 1, sets.NewString(svcJig.ServicePods[svc.Name]...))).Should(HaveLen(0), "All exposed endpoints should be hit")
				}
			})
		})

		Context("without the proper annotation", func() {
			BeforeEach(func() {
				// nodePort set to 0 so that it gets allocated dynamically.
				Expect(svcJig.CreateNodePortService("service-a", 0, 1, map[string]string{})).
					NotTo(BeNil(), "NodePort service creation failed")
				Expect(svcJig.CreateNodePortService("service-b", 0, 1, map[string]string{envoymanager.DefaultExposeAnnotationKey: "false"})).
					NotTo(BeNil(), "NodePort service creation failed")
			})

			It("should not be exposed", func() {
				portsNotToBeExposed := sets.NewInt32()
				for _, svc := range svcJig.Services {
					portsNotToBeExposed = ExtractNodePorts(svc).Union(portsNotToBeExposed)
				}
				Consistently(func() sets.Int32 {
					// When the difference between the node ports of the
					// service to be exposed and the ports of the lb server is
					// empty, it means that all ports was exposed.
					lbSvc := deployer.GetLbService()
					Expect(lbSvc).ShouldNot(BeNil())
					return portsNotToBeExposed.Intersection(ExtractPorts(lbSvc))
				}, "10s", "1s").Should(HaveLen(0), "None of the ports should be reflected in the lb service")
			})
		})

	})

})
