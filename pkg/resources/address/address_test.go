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

package address

import (
	"context"
	"fmt"
	"net"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	fakeClusterName          = "fake-cluster"
	fakeDCName               = "europe-west3-c"
	fakeExternalURL          = "dev.kubermatic.io"
	fakeClusterNamespaceName = "cluster-ns"
	externalIP               = "34.89.181.151"
	loadbBalancerHostName    = "xyz.eu-central-1.cloudprovider.test"
	testDomain               = "dns-test.kubermatic.io"
)

func testLookupFunction(host string) ([]net.IP, error) {
	switch host {
	case testDomain:
		return []net.IP{net.IPv4(192, 168, 1, 1), net.IPv4(192, 168, 1, 2)}, nil
	case "fake-cluster.europe-west3-c.dev.kubermatic.io":
		fallthrough
	case "fake-cluster.alias-europe-west3-c.dev.kubermatic.io":
		return []net.IP{net.IPv4(34, 89, 181, 151)}, nil
	case loadbBalancerHostName:
		return []net.IP{net.IPv4(34, 89, 181, 151)}, nil
	default:
		return []net.IP{}, nil
	}
}

func TestGetExternalIPv4(t *testing.T) {
	ip, err := NewModifiersBuilder(kubermaticlog.Logger).
		lookupFunc(testLookupFunction).
		getExternalIPv4(testDomain)
	if err != nil {
		t.Fatalf("failed to get the external IPv4 address for %s: %v", testDomain, err)
	}

	if ip != "192.168.1.1" {
		t.Fatalf("expected to get 192.168.1.1. Got: %s", ip)
	}
}

func TestSyncClusterAddress(t *testing.T) {
	testCases := []struct {
		name                 string
		apiserverService     corev1.Service
		frontproxyService    corev1.Service
		exposeStrategy       kubermaticv1.ExposeStrategy
		seedDNSOverwrite     string
		expectedExternalName string
		expectedIP           string
		expectedPort         int32
		expectedURL          string
		errExpected          bool
	}{
		{
			name: "Verify properties for service type LoadBalancer",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{NodePort: int32(443)}},
				},
			},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: "1.2.3.4"}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: "1.2.3.4",
			expectedIP:           "1.2.3.4",
			expectedPort:         int32(443),
			expectedURL:          "https://1.2.3.4:443",
		},
		{
			name: "Verify properties for service type LoadBalancer dont change when seedDNSOverwrite is set",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{NodePort: int32(443)},
					},
				}},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: "1.2.3.4"}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			seedDNSOverwrite:     "alias-europe-west3-c",
			expectedExternalName: "1.2.3.4",
			expectedIP:           "1.2.3.4",
			expectedPort:         int32(443),
			expectedURL:          "https://1.2.3.4:443",
		},
		{
			name: "Verify properties for service type LoadBalancer with LB hostname instead of IP",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{NodePort: int32(443)},
					},
				}},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{Hostname: "xyz.eu-central-1.cloudprovider.test"}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: loadbBalancerHostName,
			expectedIP:           externalIP,
			expectedPort:         int32(443),
			expectedURL:          fmt.Sprintf("https://%s:443", loadbBalancerHostName),
		},
		{
			name: "Verify properties for service type NodePort",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Port:       int32(32000),
							TargetPort: intstr.FromInt(32000),
							NodePort:   32000,
						}},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyNodePort,
			expectedExternalName: fmt.Sprintf("%s.%s.%s", fakeClusterName, fakeDCName, fakeExternalURL),
			expectedIP:           externalIP,
			expectedPort:         int32(32000),
			expectedURL:          fmt.Sprintf("https://%s.%s.%s:32000", fakeClusterName, fakeDCName, fakeExternalURL),
		},
		{
			name: "Verify properties for service type NodePort with seedDNSOverwrite",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Port:       int32(32000),
							TargetPort: intstr.FromInt(32000),
							NodePort:   32000,
						},
					},
				}},
			exposeStrategy:       kubermaticv1.ExposeStrategyNodePort,
			seedDNSOverwrite:     "alias-europe-west3-c",
			expectedExternalName: fmt.Sprintf("%s.alias-europe-west3-c.%s", fakeClusterName, fakeExternalURL),
			expectedIP:           externalIP,
			expectedPort:         int32(32000),
			expectedURL:          fmt.Sprintf("https://%s.alias-europe-west3-c.%s:32000", fakeClusterName, fakeExternalURL),
		},
		{
			name: "Verify error when service has less than one ports",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				}},
			errExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: fakeClusterName,
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						DatacenterName: fakeDCName,
					},
					ExposeStrategy: tc.exposeStrategy,
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: fakeClusterNamespaceName,
				},
			}

			apiserverService := &tc.apiserverService
			apiserverService.Name = resources.ApiserverServiceName
			apiserverService.Namespace = fakeClusterNamespaceName
			lbService := &tc.frontproxyService
			lbService.Name = resources.FrontLoadBalancerServiceName
			lbService.Namespace = fakeClusterNamespaceName
			client := fakectrlruntimeclient.NewClientBuilder().WithObjects(apiserverService, lbService).Build()

			seed := &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: fakeDCName,
				},
				Spec: kubermaticv1.SeedSpec{
					SeedDNSOverwrite: tc.seedDNSOverwrite,
				},
			}

			modifiers, err := NewModifiersBuilder(kubermaticlog.Logger).
				Client(client).
				Cluster(cluster).
				Seed(seed).
				ExternalURL(fakeExternalURL).
				lookupFunc(testLookupFunction).
				Build(context.Background())
			if err != nil {
				if tc.errExpected {
					return
				}
				t.Fatalf("got unexpected error %v", err)
			}

			for _, modifier := range modifiers {
				modifier(cluster)
			}

			if cluster.Address.ExternalName != tc.expectedExternalName {
				t.Errorf("expected external name to be %q but was %q", tc.expectedExternalName, cluster.Address.ExternalName)
			}

			if expectedInternalName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.ApiserverServiceName, fakeClusterNamespaceName); cluster.Address.InternalName != expectedInternalName {
				t.Errorf("Expected internal name to be %q but was %q", expectedInternalName, cluster.Address.InternalName)
			}

			if cluster.Address.IP != tc.expectedIP {
				t.Errorf("Expected IP to be %q but was %q", tc.expectedIP, cluster.Address.IP)
			}

			if cluster.Address.Port != tc.expectedPort {
				t.Errorf("Expected Port to be %d but was %d", tc.expectedPort, cluster.Address.Port)
			}

			if cluster.Address.URL != tc.expectedURL {
				t.Errorf("Expected URL to be %q but was %q", tc.expectedURL, cluster.Address.URL)
			}
		})
	}

}
