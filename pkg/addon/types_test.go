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
