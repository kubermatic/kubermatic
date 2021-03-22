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
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"

	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/test"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
)

var _ = ginkgo.Describe("NodeportProxy", func() {
	ginkgo.Describe("all services", func() {
		var svcJig *ServiceJig
		ginkgo.BeforeEach(func() {
			k8scli, _, _ := e2eutils.GetClientsOrDie()
			svcJig = &ServiceJig{
				Log:    e2eutils.DefaultLogger,
				Client: k8scli,
			}
		})
		ginkgo.AfterEach(func() {
			if !skipCleanup {
				gomega.Expect(svcJig.CleanUp()).NotTo(gomega.HaveOccurred())
			}
		})
		ginkgo.Context("of type NodePort, having the NodePort expose annotation", func() {
			ginkgo.BeforeEach(func() {
				// nodePort set to 0 so that it gets allocated dynamically.
				gomega.Expect(svcJig.CreateServiceWithPods(
					test.NewServiceBuilder(test.NamespacedName{Name: "service-a"}).
						WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, nodeportproxy.NodePortType.String()).
						WithSelector(map[string]string{"apps": "app-a"}).
						WithServiceType(corev1.ServiceTypeNodePort).
						WithServicePort("http", 80, 0, intstr.FromInt(8080), corev1.ProtocolTCP).
						Build(), 1, false)).
					NotTo(gomega.BeNil(), "NodePort service creation failed")
				gomega.Expect(svcJig.CreateServiceWithPods(
					test.NewServiceBuilder(test.NamespacedName{Name: "service-b"}).
						WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "true").
						WithSelector(map[string]string{"apps": "app-b"}).
						WithServiceType(corev1.ServiceTypeNodePort).
						WithServicePort("http", 80, 0, intstr.FromInt(8080), corev1.ProtocolTCP).
						Build(), 2, false)).
					NotTo(gomega.BeNil(), "NodePort service creation failed")
			})

			ginkgo.It("should be exposed", func() {
				ginkgo.By("updating the lb service")
				portsToBeExposed := sets.NewInt32()
				e2eutils.DefaultLogger.Debugw("computing ports to be exposed", "services", svcJig.Services)
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
				}, "2m", "1s").Should(gomega.HaveLen(0), "All exposed service ports should be reflected in the lb service")

				ginkgo.By("by load-balancing on available endpoints")
				for _, svc := range svcJig.Services {
					lbSvc := deployer.GetLbService()
					targetNp := FindExposingNodePort(lbSvc, svc.Spec.Ports[0].NodePort)
					e2eutils.DefaultLogger.Debugw("found target nodeport in lb service", "service", svc, "port", targetNp)
					gomega.Expect(networkingTest.DialFromNode("127.0.0.1", int(targetNp), 5, 1, sets.NewString(svcJig.ServicePods[svc.Name]...), false)).Should(gomega.HaveLen(0), "All exposed endpoints should be hit")
				}
			})
		})

		ginkgo.Context("of type ClusterIP, having the SNI expose annotation", func() {
			ginkgo.BeforeEach(func() {
				gomega.Expect(svcJig.CreateServiceWithPods(
					test.NewServiceBuilder(test.NamespacedName{Name: "service-a"}).
						WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, nodeportproxy.SNIType.String()).
						WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"https":"service-a.example.com"}`).
						WithSelector(map[string]string{"apps": "app-a"}).
						WithServiceType(corev1.ServiceTypeClusterIP).
						WithServicePort("https", 6443, 0, intstr.FromInt(6443), corev1.ProtocolTCP).
						Build(), 1, true)).
					NotTo(gomega.BeNil(), "ClusterIP service creation failed")
				gomega.Expect(svcJig.CreateServiceWithPods(
					test.NewServiceBuilder(test.NamespacedName{Name: "service-b"}).
						WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, nodeportproxy.SNIType.String()).
						WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"https":"service-b.example.com"}`).
						WithSelector(map[string]string{"apps": "app-b"}).
						WithServiceType(corev1.ServiceTypeClusterIP).
						WithServicePort("https", 6443, 0, intstr.FromInt(6443), corev1.ProtocolTCP).
						Build(), 2, true)).
					NotTo(gomega.BeNil(), "ClusterIP service creation failed")
			})

			ginkgo.It("should be exposed", func() {
				ginkgo.By("load-balancing on available endpoints")
				for _, svc := range svcJig.Services {
					lbSvc := deployer.GetLbService()
					targetNp := FindExposingNodePort(lbSvc, 6443)
					e2eutils.DefaultLogger.Debugw("found target nodeport in lb service", "service", svc, "port", targetNp)
					gomega.Expect(networkingTest.DialFromNode(fmt.Sprintf("%s.example.com", svc.Name), int(targetNp), 5, 1, sets.NewString(svcJig.ServicePods[svc.Name]...), true, "-k", "--resolve", fmt.Sprintf("%s.example.com:%d:127.0.0.1", svc.Name, targetNp))).Should(gomega.HaveLen(0), "All exposed endpoints should be hit")
				}
			})
		})

		ginkgo.Context("of type ClusterIP, having the Tunneling expose annotation", func() {
			ginkgo.BeforeEach(func() {
				gomega.Expect(svcJig.CreateServiceWithPods(
					test.NewServiceBuilder(test.NamespacedName{Name: "service-a"}).
						WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, nodeportproxy.TunnelingType.String()).
						WithSelector(map[string]string{"apps": "app-a"}).
						WithServiceType(corev1.ServiceTypeClusterIP).
						WithServicePort("https", 8080, 0, intstr.FromInt(8088), corev1.ProtocolTCP).
						Build(), 1, true)).
					NotTo(gomega.BeNil(), "ClusterIP service creation failed")
				gomega.Expect(svcJig.CreateServiceWithPods(
					test.NewServiceBuilder(test.NamespacedName{Name: "service-b"}).
						WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, nodeportproxy.TunnelingType.String()).
						WithSelector(map[string]string{"apps": "app-b"}).
						WithServiceType(corev1.ServiceTypeClusterIP).
						WithServicePort("https", 8080, 0, intstr.FromInt(8088), corev1.ProtocolTCP).
						Build(), 2, true)).
					NotTo(gomega.BeNil(), "ClusterIP service creation failed")
			})

			ginkgo.It("should be exposed", func() {
				ginkgo.By("load-balancing on available endpoints")
				for _, svc := range svcJig.Services {
					lbSvc := deployer.GetLbService()
					targetNp := FindExposingNodePort(lbSvc, 8088)
					e2eutils.DefaultLogger.Debugw("found target nodeport in lb service", "service", svc, "port", targetNp)
					gomega.Expect(networkingTest.DialFromNode(fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace), 8080, 15, 1, sets.NewString(svcJig.ServicePods[svc.Name]...), true, "--proxy", fmt.Sprintf("127.0.0.1:%d", targetNp))).Should(gomega.HaveLen(0), "All exposed endpoints should be hit")
				}
			})
		})

		ginkgo.Context("not having the proper annotation", func() {
			ginkgo.BeforeEach(func() {
				// nodePort set to 0 so that it gets allocated dynamically.
				gomega.Expect(svcJig.CreateServiceWithPods(
					test.NewServiceBuilder(test.NamespacedName{Name: "service-a"}).
						WithSelector(map[string]string{"apps": "app-a"}).
						WithServiceType(corev1.ServiceTypeNodePort).
						WithServicePort("http", 80, 0, intstr.FromInt(8080), corev1.ProtocolTCP).
						Build(), 1, false)).
					NotTo(gomega.BeNil(), "NodePort service creation failed")
				gomega.Expect(svcJig.CreateServiceWithPods(
					test.NewServiceBuilder(test.NamespacedName{Name: "service-b"}).
						WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "false").
						WithSelector(map[string]string{"apps": "app-b"}).
						WithServiceType(corev1.ServiceTypeNodePort).
						WithServicePort("http", 80, 0, intstr.FromInt(8080), corev1.ProtocolTCP).
						Build(), 1, false)).
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
