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

package nodeportproxy

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"testing"
	"time"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/test"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	// serviceName is the service that is exposed and then queried during
	// the tests N times until all expected endpoints have responded at
	// least once.
	serviceName = "test-service"

	// replicas must be > 1 to ensure the tests check that the proxy
	// proxies not just to the first endpoint of a service.
	replicas = 2
)

type testcase struct {
	name              string
	exposeType        nodeportproxy.ExposeType
	extraAnnotations  map[string]string
	serviceType       corev1.ServiceType
	servicePort       corev1.ServicePort // should never have .NodePort set, so it gets allocated dynamically
	dialConfigCreator func(svc *corev1.Service, lbSvc *corev1.Service) DialConfig
}

var (
	versions   = kubermatic.GetVersions()
	logOptions = e2eutils.DefaultLogOptions
	testcases  = []testcase{
		{
			name:        "type NodePort, having the NodePort expose annotation",
			exposeType:  nodeportproxy.NodePortType,
			serviceType: corev1.ServiceTypeNodePort,
			servicePort: corev1.ServicePort{Name: "http", Port: 80, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP},
			dialConfigCreator: func(svc *corev1.Service, lbSvc *corev1.Service) DialConfig {
				targetNp := findExposingNodePort(lbSvc, svc.Spec.Ports[0].NodePort)

				return DialConfig{
					TargetIP:   "127.0.0.1",
					TargetPort: int(targetNp),
					HTTPS:      false,
				}
			},
		},
		{
			name:             "type ClusterIP, having the SNI expose annotation",
			exposeType:       nodeportproxy.SNIType,
			serviceType:      corev1.ServiceTypeClusterIP,
			extraAnnotations: map[string]string{nodeportproxy.PortHostMappingAnnotationKey: fmt.Sprintf(`{"https":"%s.example.com"}`, serviceName)},
			servicePort:      corev1.ServicePort{Name: "https", Port: 6443, TargetPort: intstr.FromInt(6443), Protocol: corev1.ProtocolTCP},
			dialConfigCreator: func(svc *corev1.Service, lbSvc *corev1.Service) DialConfig {
				targetNp := findExposingNodePort(lbSvc, 6443)

				return DialConfig{
					TargetIP:   fmt.Sprintf("%s.example.com", svc.Name),
					TargetPort: int(targetNp),
					HTTPS:      true,
					ExtraCurlArguments: []string{
						"--resolve",
						fmt.Sprintf("%s.example.com:%d:127.0.0.1", svc.Name, targetNp),
					},
				}
			},
		},
		{
			name:        "type ClusterIP, having the Tunneling expose annotation",
			exposeType:  nodeportproxy.TunnelingType,
			serviceType: corev1.ServiceTypeClusterIP,
			servicePort: corev1.ServicePort{Name: "https", Port: 8080, TargetPort: intstr.FromInt(8088), Protocol: corev1.ProtocolTCP},
			dialConfigCreator: func(svc *corev1.Service, lbSvc *corev1.Service) DialConfig {
				targetNp := findExposingNodePort(lbSvc, 8088)

				return DialConfig{
					TargetIP:   fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace),
					TargetPort: 8080,
					HTTPS:      true,
					ExtraCurlArguments: []string{
						"--proxy",
						fmt.Sprintf("127.0.0.1:%d", targetNp),
					},
				}
			},
		},
	}
)

func init() {
	flag.StringVar(&versions.KubermaticContainerTag, "kubermatic-tag", "latest", "Kubermatic image tag to be used for the tests")
	logOptions.AddFlags(flag.CommandLine)
}

func TestNodeportProxy(t *testing.T) {
	ctx := signals.SetupSignalHandler()
	logger := log.NewFromOptions(logOptions).Sugar()

	k8scli, config, err := e2eutils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// setup the nodeport-proxy
	npp := &NodeportProxy{
		Log:      logger,
		Client:   k8scli,
		Versions: versions,
	}

	logger.Info("Setting up nodeport-proxy…")

	if err := npp.Setup(ctx); err != nil {
		t.Fatalf("Failed to setup nodeport-proxy: %v", err)
	}
	defer func() {
		// use a fresh context to run the cleanup when the root context was cancelled (e.g. via Ctrl-C)
		if err := npp.Cleanup(context.Background()); err != nil {
			t.Fatalf("Failed to cleanup nodeport-proxy: %v", err)
		}
	}()

	// prepare the test pod
	networkingTest := &networkingTestConfig{
		TestPodConfig: e2eutils.TestPodConfig{
			Log:           logger,
			Client:        k8scli,
			Namespace:     npp.Namespace,
			Config:        config,
			CreatePodFunc: newAgnhostPod,
		},
	}

	logger.Info("Setting up test pod…")

	if err := networkingTest.DeployTestPod(ctx, logger); err != nil {
		t.Fatalf("Failed to setup test pod: %v", err)
	}
	defer func() {
		// use a fresh context to run the cleanup when the root context was cancelled (e.g. via Ctrl-C)
		if err := networkingTest.CleanUp(context.Background()); err != nil {
			t.Fatalf("Failed to cleanup test pod: %v", err)
		}
	}()

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			svcJig := &ServiceJig{
				Log:    logger,
				Client: k8scli,
			}

			// create service with multiple endpoints
			svc, endpoints, err := createTestServiceWithPods(ctx, svcJig, testcase)
			if err != nil {
				t.Fatalf("Failed to create service with pods: %v", err)
			}
			defer func() {
				if err := svcJig.Cleanup(context.Background()); err != nil {
					t.Fatalf("Failed to cleanup: %v", err)
				}
			}()

			// tests for NodePort Services need to wait until all ports are exposed
			// in the LoadBalancer
			if testcase.serviceType == corev1.ServiceTypeNodePort {
				logger.Info("Waiting for ports to be exposed…")
				portsToBeExposed := extractNodePorts(svc)

				err = wait.PollLog(ctx, logger, 2*time.Second, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
					lbSvc, err := npp.GetLoadBalancer(ctx)
					if err != nil {
						return err, nil
					}

					if remaining := portsToBeExposed.Difference(extractPorts(lbSvc)); remaining.Len() > 0 {
						return fmt.Errorf("ports %v are not yet exposed", sets.List(remaining)), nil
					}

					return nil, nil
				})
				if err != nil {
					t.Fatalf("Failed to expose ports: %v", err)
				}
			}

			// wait until we have reached every endpoint at least once
			unverifiedEndpoints := sets.New(endpoints...)

			logger.Info("Waiting until we have reached every endpoint at least once…")
			err = wait.PollImmediateLog(ctx, logger, 2*time.Second, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
				// get the current state of the nodeport-proxy's LoadBalancer service
				lbSvc, err := npp.GetLoadBalancer(ctx)
				if err != nil {
					return err, nil
				}

				// let the test decide how we try to reach the service
				dialConfig := testcase.dialConfigCreator(svc, lbSvc)

				// try to reach the service
				endpoint, err := networkingTest.Dial(ctx, dialConfig)
				if err != nil {
					return err, nil
				}

				unverifiedEndpoints.Delete(endpoint)
				if unverifiedEndpoints.Len() > 0 {
					return fmt.Errorf("not all endpoints reached yet: %v", sets.List(unverifiedEndpoints)), nil
				}

				return nil, nil
			})
			if err != nil {
				t.Fatalf("Connectivity check failed: %v", err)
			}

			logger.Info("All endpoints reachable, tests passed successfully.")
		})
	}

	// one extra test that has a different check, put here to keep the loop above easier to read

	t.Run("service without the proper annotation should not be exposed", func(t *testing.T) {
		svcJig := &ServiceJig{
			Log:    logger,
			Client: k8scli,
		}

		svc, _, err := createTestServiceWithPods(ctx, svcJig, testcase{
			exposeType:  -1, // this skips the expose annotation
			serviceType: corev1.ServiceTypeNodePort,
			servicePort: corev1.ServicePort{Name: "http", Port: 80, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP},
		})
		if err != nil {
			t.Fatalf("Failed to create service with pods: %v", err)
		}

		// none of the ports of our service should be exposed in the load balancer
		logger.Info("Ensuring ports are not exposed…")
		portsNotToBeExposed := extractNodePorts(svc)

		err = wait.Poll(ctx, 1*time.Second, 30*time.Second, func(ctx context.Context) (transient error, terminal error) {
			lbSvc, err := npp.GetLoadBalancer(ctx)
			if err != nil {
				return err, nil
			}

			if exposed := portsNotToBeExposed.Intersection(extractPorts(lbSvc)); exposed.Len() > 0 {
				// if a port appears, it's not a transient error that might go away, having
				// the port exposed once is already a terminal issue
				return nil, fmt.Errorf("ports %v have been exposed when they should not have", sets.List(exposed))
			}

			return errors.New("nothing exposed, all good"), nil
		})
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Should not have exposed a service, but %v", err)
		}

		logger.Info("Endpoints not accidentally exposed, tests passed successfully.")
	})
}

func createTestServiceWithPods(ctx context.Context, svcJig *ServiceJig, tc testcase) (*corev1.Service, []string, error) {
	svcBuilder := test.NewServiceBuilder(test.NamespacedName{Name: serviceName}).
		WithServiceType(tc.serviceType).
		WithServicePorts(tc.servicePort).
		WithSelector(map[string]string{"apps": "test"})

	// only expose if a valid type is given, some tests rely on creating unexposed services
	if tc.exposeType >= 0 {
		svcBuilder.WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, tc.exposeType.String())
	}

	for k, v := range tc.extraAnnotations {
		svcBuilder.WithAnnotation(k, v)
	}

	return svcJig.CreateServiceWithPods(ctx, svcBuilder.Build(), int32(replicas), tc.servicePort.Name == "https")
}
