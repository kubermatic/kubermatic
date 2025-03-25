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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	fakeClusterName          = "fake-cluster"
	fakeClusterNameIPv6      = "fake-cluster-ipv6"
	fakeDCName               = "europe-west3-c"
	fakeExternalURL          = "dev.kubermatic.io"
	fakeClusterNamespaceName = "cluster-ns"
	externalIP               = "34.89.181.151"
	loadbBalancerHostName    = "xyz.eu-central-1.cloudprovider.test"
	testDomain               = "dns-test.kubermatic.io"
	testDomainWithNewIPs     = "dns-test-with-new-ips.kubermatic.io"
	ipv6Address              = "2a01:4f8:1c0c:4b1d::1"
)

func testLookupFunction(host string) ([]net.IP, error) {
	switch host {
	case testDomain:
		return []net.IP{net.IPv4(192, 168, 1, 1), net.IPv4(192, 168, 1, 2)}, nil
	case testDomainWithNewIPs:
		return []net.IP{net.IPv4(192, 168, 1, 2), net.IPv4(192, 168, 1, 3)}, nil
	case "fake-cluster.europe-west3-c.dev.kubermatic.io":
		fallthrough
	case "fake-cluster.alias-europe-west3-c.dev.kubermatic.io":
		return []net.IP{net.IPv4(34, 89, 181, 151), net.ParseIP("2a01:4f8:1c0c:4b1d::1")}, nil
	case "fake-cluster-ipv6.europe-west3-c.dev.kubermatic.io":
		return []net.IP{net.ParseIP("2a01:4f8:1c0c:4b1d::1")}, nil
	case loadbBalancerHostName:
		return []net.IP{net.IPv4(34, 89, 181, 151)}, nil
	default:
		return []net.IP{}, nil
	}
}

func TestGetExternalIP(t *testing.T) {
	modifier := NewModifiersBuilder(kubermaticlog.Logger)
	modifier.cluster = &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: fakeClusterName,
		},
		Status: kubermaticv1.ClusterStatus{
			Address: kubermaticv1.ClusterAddress{
				IP: "192.168.1.1",
			},
		},
	}

	ip, err := modifier.
		lookupFunc(testLookupFunction).
		getExternalIP(testDomain)
	if err != nil {
		t.Fatalf("failed to get the external IP address for %s: %v", testDomain, err)
	}

	if ip != "192.168.1.1" {
		t.Fatalf("expected to get 192.168.1.1. Got: %s", ip)
	}
}

func TestGetExternalIPWithNewAddress(t *testing.T) {
	modifier := NewModifiersBuilder(kubermaticlog.Logger)
	modifier.cluster = &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: fakeClusterName,
		},
	}

	ip, err := modifier.
		lookupFunc(testLookupFunction).
		getExternalIP(testDomainWithNewIPs)
	if err != nil {
		t.Fatalf("failed to get the external IP address for %s: %v", testDomainWithNewIPs, err)
	}

	if ip != "192.168.1.2" {
		t.Fatalf("expected to get 192.168.1.2. Got: %s", ip)
	}
}

func TestSyncClusterAddress(t *testing.T) {
	testCases := []struct {
		name                 string
		clusterName          *string
		apiserverService     corev1.Service
		frontproxyService    corev1.Service
		exposeStrategy       kubermaticv1.ExposeStrategy
		seedDNSOverwrite     string
		expectedExternalName string
		expectedIP           string
		expectedPort         int32
		expectedURL          string
		expectedAPIServerURL *string
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
			name: "Verify properties for service type LoadBalancer with IPv6",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{NodePort: int32(443)}},
				},
			},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: ipv6Address}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: ipv6Address,
			expectedIP:           ipv6Address,
			expectedPort:         int32(443),
			expectedURL:          fmt.Sprintf("https://[%s]:443", ipv6Address),
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
						Ingress: []corev1.LoadBalancerIngress{{Hostname: loadbBalancerHostName}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: loadbBalancerHostName,
			expectedIP:           externalIP,
			expectedPort:         int32(443),
			expectedURL:          fmt.Sprintf("https://%s", net.JoinHostPort(loadbBalancerHostName, "443")),
		},
		{
			name: "Verify properties for service type LoadBalancer with private IP only",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{NodePort: int32(443)}},
				},
			},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: "10.10.10.2"}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: "10.10.10.2",
			expectedIP:           "10.10.10.2",
			expectedPort:         int32(443),
			expectedURL:          "https://10.10.10.2:443",
		},
		{
			name: "Verify properties for service type LoadBalancer with private and public IP",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{NodePort: int32(443)}},
				},
			},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: "10.10.10.2"}, {IP: "10.10.10.3"}, {IP: "3.67.176.129"}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: "3.67.176.129",
			expectedIP:           "3.67.176.129",
			expectedPort:         int32(443),
			expectedURL:          "https://3.67.176.129:443",
		},
		{
			name: "Verify properties for service type LoadBalancer with IPv6 in the list",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{NodePort: int32(443)}},
				},
			},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: "2a01::1"}, {IP: "10.10.10.3"}, {IP: "3.67.176.129"}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: "3.67.176.129",
			expectedIP:           "3.67.176.129",
			expectedPort:         int32(443),
			expectedURL:          "https://3.67.176.129:443",
		},
		{
			name: "Verify properties for service type LoadBalancer in default case with no IP in status",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{NodePort: int32(443)}},
				},
			},
			frontproxyService: corev1.Service{
				Spec: corev1.ServiceSpec{
					LoadBalancerIP: "10.10.10.2",
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: "10.10.10.2",
			expectedIP:           "10.10.10.2",
			expectedPort:         int32(443),
			expectedURL:          "https://10.10.10.2:443",
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
			name: "Verify properties for Tunneling expose strategy",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Port:       int32(6443),
							TargetPort: intstr.FromInt(6443),
							NodePort:   32000,
						}},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyTunneling,
			expectedExternalName: fmt.Sprintf("%s.%s.%s", fakeClusterName, fakeDCName, fakeExternalURL),
			expectedIP:           externalIP,
			expectedPort:         int32(6443),
			expectedURL:          fmt.Sprintf("https://%s.%s.%s:6443", fakeClusterName, fakeDCName, fakeExternalURL),
		},
		{
			name:        "Verify properties for Tunneling expose strategy with IPv6",
			clusterName: ptr.To(fakeClusterNameIPv6),
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Port:       int32(6443),
							TargetPort: intstr.FromInt(6443),
							NodePort:   32000,
						}},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyTunneling,
			expectedExternalName: fmt.Sprintf("%s.%s.%s", fakeClusterNameIPv6, fakeDCName, fakeExternalURL),
			expectedIP:           ipv6Address,
			expectedPort:         int32(6443),
			expectedURL:          fmt.Sprintf("https://%s.%s.%s:6443", fakeClusterNameIPv6, fakeDCName, fakeExternalURL),
		},
		{
			name: "Verify error when service has less than one ports",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				}},
			errExpected: true,
		},
		{
			name: "Verify properties API Server service of type LoadBalancer with hostname",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{NodePort: int32(443)},
					},
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{Hostname: loadbBalancerHostName}},
					},
				},
			},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{Hostname: loadbBalancerHostName}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: loadbBalancerHostName,
			expectedIP:           externalIP,
			expectedPort:         int32(443),
			expectedURL:          fmt.Sprintf("https://%s", net.JoinHostPort(loadbBalancerHostName, "443")),
			expectedAPIServerURL: ptr.To(fmt.Sprintf("https://%s", loadbBalancerHostName)),
		},
		{
			name: "Verify properties API Server service of type LoadBalancer with hostname and IP, hostname is preferred",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{NodePort: int32(443)},
					},
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{Hostname: loadbBalancerHostName, IP: externalIP}},
					},
				},
			},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{Hostname: loadbBalancerHostName}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: loadbBalancerHostName,
			expectedIP:           externalIP,
			expectedPort:         int32(443),
			expectedURL:          fmt.Sprintf("https://%s", net.JoinHostPort(loadbBalancerHostName, "443")),
			expectedAPIServerURL: ptr.To(fmt.Sprintf("https://%s", loadbBalancerHostName)),
		},
		{
			name: "Verify properties API Server service of type LoadBalancer with IP",
			apiserverService: corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{NodePort: int32(443)},
					},
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: externalIP}},
					},
				},
			},
			frontproxyService: corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{Hostname: loadbBalancerHostName}},
					},
				},
			},
			exposeStrategy:       kubermaticv1.ExposeStrategyLoadBalancer,
			expectedExternalName: loadbBalancerHostName,
			expectedIP:           externalIP,
			expectedPort:         int32(443),
			expectedURL:          fmt.Sprintf("https://%s", net.JoinHostPort(loadbBalancerHostName, "443")),
			expectedAPIServerURL: ptr.To(fmt.Sprintf("https://%s", externalIP)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clusterName := fakeClusterName
			if tc.clusterName != nil {
				clusterName = *tc.clusterName
			}

			cluster := &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
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
			client := fake.NewClientBuilder().WithObjects(apiserverService, lbService).Build()

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

			if cluster.Status.Address.ExternalName != tc.expectedExternalName {
				t.Errorf("expected external name to be %q but was %q", tc.expectedExternalName, cluster.Status.Address.ExternalName)
			}

			if expectedInternalName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.ApiserverServiceName, fakeClusterNamespaceName); cluster.Status.Address.InternalName != expectedInternalName {
				t.Errorf("Expected internal name to be %q but was %q", expectedInternalName, cluster.Status.Address.InternalName)
			}

			if cluster.Status.Address.IP != tc.expectedIP {
				t.Errorf("Expected IP to be %q but was %q", tc.expectedIP, cluster.Status.Address.IP)
			}

			if cluster.Status.Address.Port != tc.expectedPort {
				t.Errorf("Expected Port to be %d but was %d", tc.expectedPort, cluster.Status.Address.Port)
			}

			if cluster.Status.Address.URL != tc.expectedURL {
				t.Errorf("Expected URL to be %q but was %q", tc.expectedURL, cluster.Status.Address.URL)
			}

			if tc.expectedAPIServerURL != nil {
				if cluster.Status.Address.APIServerExternalAddress != *tc.expectedAPIServerURL {
					t.Errorf("Expected APIServerExternalAddress to be %q but was %q", *tc.expectedAPIServerURL, cluster.Status.Address.APIServerExternalAddress)
				}
			}
		})
	}
}
