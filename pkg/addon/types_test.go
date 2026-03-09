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

package addon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// There are unit tests in pkg/install/images/ that effectively render
// all addons against a wide variety of cluster combinations, so there
// is little use in having another set of tests (and more importantly,
// testdata or testdata generators) in this package as well.

func TestNewTemplateData(t *testing.T) {
	version := defaulting.DefaultKubernetesVersioning.Default
	feature := "myfeature"
	cluster := kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				IPVS: &kubermaticv1.IPVSConfiguration{
					StrictArp: ptr.To(true),
				},
			},
			CNIPlugin: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCanal,
				Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
			},
			Version: *version,
			Features: map[string]bool{
				feature: true,
			},
		},
		Status: kubermaticv1.ClusterStatus{
			Versions: kubermaticv1.ClusterVersionsStatus{
				ControlPlane: *version,
			},
		},
	}
	ipamAllocationList := kubermaticv1.IPAMAllocationList{
		Items: []kubermaticv1.IPAMAllocation{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ipam-pool-1",
				},
				Spec: kubermaticv1.IPAMAllocationSpec{
					Type: "prefix",
					CIDR: "192.168.0.1/28",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ipam-pool-2",
				},
				Spec: kubermaticv1.IPAMAllocationSpec{
					Type:      "range",
					Addresses: []string{"192.168.0.1-192.168.0.8", "192.168.0.10-192.168.0.17"},
				},
			},
		},
	}

	credentials := resources.Credentials{}

	templateData, err := NewTemplateData(&cluster, credentials, "", "", "", &ipamAllocationList, nil)
	if err != nil {
		t.Fatalf("Failed to create template data: %v", err)
	}

	if !templateData.Cluster.Features.Has(feature) {
		t.Fatalf("Expected cluster features to contain %q, but does not.", feature)
	}

	assert.Equal(t, map[string]IPAMAllocation{
		"ipam-pool-1": {
			Type: "prefix",
			CIDR: "192.168.0.1/28",
		},
		"ipam-pool-2": {
			Type:      "range",
			Addresses: []string{"192.168.0.1-192.168.0.8", "192.168.0.10-192.168.0.17"},
		},
	}, templateData.Cluster.Network.IPAMAllocations)
}

func TestClusterNetworkConfigHash(t *testing.T) {
	tests := []struct {
		name       string
		a          ClusterNetwork
		b          ClusterNetwork
		expectSame bool
	}{
		{
			name: "identical structs produce the same hash",
			a: ClusterNetwork{
				ProxyMode:     "iptables",
				PodCIDRBlocks: []string{"10.244.0.0/16"},
			},
			b: ClusterNetwork{
				ProxyMode:     "iptables",
				PodCIDRBlocks: []string{"10.244.0.0/16"},
			},
			expectSame: true,
		},
		{
			name: "different proxy mode produces different hash",
			a: ClusterNetwork{
				ProxyMode:     "iptables",
				PodCIDRBlocks: []string{"10.244.0.0/16"},
			},
			b: ClusterNetwork{
				ProxyMode:     "ipvs",
				PodCIDRBlocks: []string{"10.244.0.0/16"},
				StrictArp:     ptr.To(true),
			},
			expectSame: false,
		},
		{
			name: "different PodCIDRBlocks order produces the same hash",
			a: ClusterNetwork{
				ProxyMode:     "iptables",
				PodCIDRBlocks: []string{"10.244.0.0/16", "fd00::/48"},
			},
			b: ClusterNetwork{
				ProxyMode:     "iptables",
				PodCIDRBlocks: []string{"fd00::/48", "10.244.0.0/16"},
			},
			expectSame: true,
		},
		{
			name: "different ServiceCIDRBlocks order produces the same hash",
			a: ClusterNetwork{
				ProxyMode:         "iptables",
				ServiceCIDRBlocks: []string{"10.96.0.0/12", "fd01::/108"},
			},
			b: ClusterNetwork{
				ProxyMode:         "iptables",
				ServiceCIDRBlocks: []string{"fd01::/108", "10.96.0.0/12"},
			},
			expectSame: true,
		},
		{
			name: "different StrictArp values produce different hash",
			a: ClusterNetwork{
				ProxyMode: "ipvs",
				StrictArp: ptr.To(true),
			},
			b: ClusterNetwork{
				ProxyMode: "ipvs",
				StrictArp: ptr.To(false),
			},
			expectSame: false,
		},
		{
			name: "nil StrictArp vs non-nil StrictArp produce different hash",
			a: ClusterNetwork{
				ProxyMode: "iptables",
				StrictArp: nil,
			},
			b: ClusterNetwork{
				ProxyMode: "iptables",
				StrictArp: ptr.To(true),
			},
			expectSame: false,
		},
		{
			name:       "empty structs produce the same hash",
			a:          ClusterNetwork{},
			b:          ClusterNetwork{},
			expectSame: true,
		},
		{
			name: "different PodCIDRBlocks values produce different hash",
			a: ClusterNetwork{
				ProxyMode:     "iptables",
				PodCIDRBlocks: []string{"10.244.0.0/16"},
			},
			b: ClusterNetwork{
				ProxyMode:     "iptables",
				PodCIDRBlocks: []string{"10.244.0.0/16", "fd00::/48"},
			},
			expectSame: false,
		},
		{
			name: "different NodePortRange produces different hash",
			a: ClusterNetwork{
				ProxyMode:     "iptables",
				NodePortRange: "30000-32767",
			},
			b: ClusterNetwork{
				ProxyMode:     "iptables",
				NodePortRange: "30000-32000",
			},
			expectSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashA := tt.a.ConfigHash()
			hashB := tt.b.ConfigHash()

			if hashA == "" {
				t.Fatal("ConfigHash returned empty string for a")
			}
			if hashB == "" {
				t.Fatal("ConfigHash returned empty string for b")
			}

			if tt.expectSame {
				assert.Equal(t, hashA, hashB, "expected same hash")
			} else {
				assert.NotEqual(t, hashA, hashB, "expected different hash")
			}
		})
	}
}

func TestClusterNetworkConfigHashDoesNotMutate(t *testing.T) {
	original := []string{"fd00::/48", "10.244.0.0/16"}
	n := ClusterNetwork{
		ProxyMode:     "iptables",
		PodCIDRBlocks: original,
	}

	n.ConfigHash()

	assert.Equal(t, []string{"fd00::/48", "10.244.0.0/16"}, n.PodCIDRBlocks,
		"ConfigHash must not mutate the original PodCIDRBlocks slice")
}

func TestClusterNetworkConfigHashNilStrictArp(t *testing.T) {
	n := ClusterNetwork{
		ProxyMode:     "iptables",
		PodCIDRBlocks: []string{"10.244.0.0/16"},
		StrictArp:     nil,
	}

	hash := n.ConfigHash()
	assert.NotEmpty(t, hash, "ConfigHash must return a valid hash even when StrictArp is nil")

	// Calling again must produce the same result.
	assert.Equal(t, hash, n.ConfigHash(), "ConfigHash must be deterministic with nil StrictArp")
}
