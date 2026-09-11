/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package cluster

import (
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testDatacenter = "hetzner-dc"

func keyConfigurationTestSeed() *kubermaticv1.Seed {
	return &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{Name: "test-seed", Namespace: "kubermatic"},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				testDatacenter: {
					Spec: kubermaticv1.DatacenterSpec{
						Hetzner: &kubermaticv1.DatacenterSpecHetzner{},
					},
				},
			},
		},
	}
}

func keyConfigurationTestCluster() *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: kubermaticv1.ClusterSpec{
			Version: *semver.NewSemverOrDie("1.31.0"),
			Cloud: kubermaticv1.CloudSpec{
				ProviderName:   string(kubermaticv1.HetznerCloudProvider),
				DatacenterName: testDatacenter,
				Hetzner:        &kubermaticv1.HetznerCloudSpec{},
			},
		},
	}
}

func configWithKeyConfiguration(keyConfig *kubermaticv1.KeyConfiguration) *kubermaticv1.KubermaticConfiguration {
	config := &kubermaticv1.KubermaticConfiguration{}
	config.Spec.UserCluster.KeyConfiguration = keyConfig
	return config
}

var ecdsaKeyConfiguration = &kubermaticv1.KeyConfiguration{
	ServiceAccountKey: &kubermaticv1.KeySpec{
		Algorithm:  kubermaticv1.KeyAlgorithmECDSA,
		ECDSACurve: kubermaticv1.ECDSACurveP384,
	},
	Certificates: &kubermaticv1.KeySpec{
		Algorithm: kubermaticv1.KeyAlgorithmECDSA,
	},
}

// TestMutateCreateStampsKeyConfiguration covers the freeze half of the design:
// a new cluster inherits whatever the global configuration says at that moment.
func TestMutateCreateStampsKeyConfiguration(t *testing.T) {
	testCases := []struct {
		name     string
		config   *kubermaticv1.KubermaticConfiguration
		cluster  *kubermaticv1.Cluster
		expected *kubermaticv1.KeyConfiguration
	}{
		{
			name:     "the global default is copied into the cluster",
			config:   configWithKeyConfiguration(ecdsaKeyConfiguration),
			cluster:  keyConfigurationTestCluster(),
			expected: ecdsaKeyConfiguration,
		},
		{
			name:     "no global default leaves the field empty, which means RSA-2048",
			config:   configWithKeyConfiguration(nil),
			cluster:  keyConfigurationTestCluster(),
			expected: nil,
		},
		{
			name:   "an explicit per-cluster value wins over the global default",
			config: configWithKeyConfiguration(ecdsaKeyConfiguration),
			cluster: func() *kubermaticv1.Cluster {
				cluster := keyConfigurationTestCluster()
				cluster.Spec.KeyConfiguration = &kubermaticv1.KeyConfiguration{
					Certificates: &kubermaticv1.KeySpec{
						Algorithm:  kubermaticv1.KeyAlgorithmRSA,
						RSAKeySize: 4096,
					},
				}
				return cluster
			}(),
			expected: &kubermaticv1.KeyConfiguration{
				Certificates: &kubermaticv1.KeySpec{
					Algorithm:  kubermaticv1.KeyAlgorithmRSA,
					RSAKeySize: 4096,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if err := MutateCreate(test.cluster, test.config, keyConfigurationTestSeed(), nil); err != nil {
				t.Fatalf("MutateCreate failed: %v", err)
			}

			got := test.cluster.Spec.KeyConfiguration
			switch {
			case test.expected == nil && got != nil:
				t.Fatalf("expected no key configuration, got %+v", got)
			case test.expected != nil && got == nil:
				t.Fatal("expected a key configuration, got none")
			case test.expected == nil:
				return
			}

			if got == test.expected {
				t.Error("the cluster shares the KeyConfiguration with the KubermaticConfiguration; it has to be a copy")
			}
			if a, b := keySpecString(got.ServiceAccountKey), keySpecString(test.expected.ServiceAccountKey); a != b {
				t.Errorf("expected serviceAccountKey %s, got %s", b, a)
			}
			if a, b := keySpecString(got.Certificates), keySpecString(test.expected.Certificates); a != b {
				t.Errorf("expected certificates %s, got %s", b, a)
			}
		})
	}
}

// TestMutateUpdateNeverStampsKeyConfiguration is the counterpart, and the reason
// the stamping does not live in DefaultClusterSpec: a cluster that was created
// before the global default was configured must never pick it up later, because
// its certificates would then change algorithm at their next renewal.
func TestMutateUpdateNeverStampsKeyConfiguration(t *testing.T) {
	config := configWithKeyConfiguration(ecdsaKeyConfiguration)

	oldCluster := keyConfigurationTestCluster()
	newCluster := keyConfigurationTestCluster()

	if err := MutateUpdate(oldCluster, newCluster, config, keyConfigurationTestSeed(), nil); err != nil {
		t.Fatalf("MutateUpdate failed: %v", err)
	}

	if newCluster.Spec.KeyConfiguration != nil {
		t.Errorf("an existing cluster picked up the global key configuration: %+v", newCluster.Spec.KeyConfiguration)
	}
}

func keySpecString(spec *kubermaticv1.KeySpec) string {
	if spec == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s/%s/%d", spec.Algorithm, spec.ECDSACurve, spec.RSAKeySize)
}
