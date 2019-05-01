package address

import (
	"context"
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

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
	// This must be the A record for *.europe-west3c.dev.kubermatic.io and
	// *.alias-europe-west3-c.dev.kubermatic.io
	externalIP = "35.198.93.90"
	// Henrik created 2 dns entries for dns-test @ OVH.com. dns-test.kubermatic.io points to:
	// - 192.168.1.1
	// - 192.168.1.2
	// - 2001:16B8:6844:D700:A1B9:D94B:FDC3:1C33
	testDomain = "dns-test.kubermatic.io"
)

func TestGetExternalIPv4(t *testing.T) {
	ip, err := getExternalIPv4(testDomain)
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
		apiserverService     corev1.ServiceSpec
		seedDNSOverwrite     string
		expectedExternalName string
		expectedIP           string
		expectedPort         int32
		expectedURL          string
		errExpected          bool
	}{
		{
			name: "Verify properties for service type LoadBalancer",
			apiserverService: corev1.ServiceSpec{
				Type:           corev1.ServiceTypeLoadBalancer,
				LoadBalancerIP: "1.2.3.4",
				Ports: []corev1.ServicePort{
					{Port: int32(443)},
				},
			},
			expectedExternalName: "1.2.3.4",
			expectedIP:           "1.2.3.4",
			expectedPort:         int32(443),
			expectedURL:          "https://1.2.3.4:443",
		},
		{
			name: "Verify properties for service type LoadBalancer dont change when seedDNSOverwrite is set",
			apiserverService: corev1.ServiceSpec{
				Type:           corev1.ServiceTypeLoadBalancer,
				LoadBalancerIP: "1.2.3.4",
				Ports: []corev1.ServicePort{
					{Port: int32(443)},
				},
			},
			seedDNSOverwrite:     "alias-europe-west3-c",
			expectedExternalName: "1.2.3.4",
			expectedIP:           "1.2.3.4",
			expectedPort:         int32(443),
			expectedURL:          "https://1.2.3.4:443",
		},
		{
			name: "Verify properties for service type NodePort",
			apiserverService: corev1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
				Ports: []corev1.ServicePort{
					{
						Port:       int32(32000),
						TargetPort: intstr.FromInt(32000),
					},
				},
			},
			expectedExternalName: fmt.Sprintf("%s.%s.%s", fakeClusterName, fakeDCName, fakeExternalURL),
			expectedIP:           externalIP,
			expectedPort:         int32(32000),
			expectedURL:          fmt.Sprintf("https://%s.%s.%s:32000", fakeClusterName, fakeDCName, fakeExternalURL),
		},
		{
			name: "Verify properties for service type NodePort with seedDNSOverwrite",
			apiserverService: corev1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
				Ports: []corev1.ServicePort{
					{
						Port:       int32(32000),
						TargetPort: intstr.FromInt(32000),
					},
				},
			},
			seedDNSOverwrite:     "alias-europe-west3-c",
			expectedExternalName: fmt.Sprintf("%s.alias-europe-west3-c.%s", fakeClusterName, fakeExternalURL),
			expectedIP:           externalIP,
			expectedPort:         int32(32000),
			expectedURL:          fmt.Sprintf("https://%s.alias-europe-west3-c.%s:32000", fakeClusterName, fakeExternalURL),
		},
		{
			name: "Verify error when service is not of type NodePort or LoadBalancer",
			apiserverService: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Port:       int32(32000),
						TargetPort: intstr.FromInt(32000),
					},
				},
			},
			errExpected: true,
		},
		{
			name: "Verify error when service has less than one ports",
			apiserverService: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
			},
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
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: fakeClusterNamespaceName,
				},
			}

			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resources.ApiserverExternalServiceName,
					Namespace: fakeClusterNamespaceName,
				},
				Spec: tc.apiserverService,
			}
			client := fakectrlruntimeclient.NewFakeClient(service)

			nodeDCs := map[string]provider.DatacenterMeta{
				fakeDCName: {
					SeedDNSOverwrite: &tc.seedDNSOverwrite,
				},
			}

			modifiers, err := SyncClusterAddress(context.Background(),
				cluster, client, fakeExternalURL, fakeDCName, nodeDCs)
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

			if expectedInternalName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.ApiserverExternalServiceName, fakeClusterNamespaceName); cluster.Address.InternalName != expectedInternalName {
				t.Errorf("Expected internal name to be %q but was %q", expectedInternalName, cluster.Address.InternalName)
			}

			if cluster.Address.IP != tc.expectedIP {
				t.Errorf("Expected IP to be %q but was %q", tc.expectedIP, cluster.Address.IP)
			}

			if cluster.Address.Port != tc.expectedPort {
				t.Errorf("Expected Port to be %q but was %q", tc.expectedPort, cluster.Address.Port)
			}

			if cluster.Address.URL != tc.expectedURL {
				t.Errorf("Expected URL to be %q but was %q", tc.expectedURL, cluster.Address.URL)
			}
		})
	}

}
