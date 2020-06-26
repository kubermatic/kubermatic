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

package resources

import (
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	testhelper "github.com/kubermatic/kubermatic/api/pkg/test"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestKubeControllerConfigMapCreation(t *testing.T) {
	testCases := []struct {
		name    string
		cluster *kubermaticv1.Cluster
	}{
		{
			name: "config",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"10.240.16.0"},
						},
						Services: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"10.11.10.0"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := &fakeKubeControllerManagerConfigData{cluster: tc.cluster}
			creatorGetter := KubeControllerManagerConfigMapCreatorFactory(data)
			_, creator := creatorGetter()

			configMap, err := creator(&corev1.ConfigMap{})
			if err != nil {
				t.Fatalf("failed calling creator: %v", err)
			}
			serializedConfigmap, err := yaml.Marshal(configMap)
			if err != nil {
				t.Fatalf("failed to marshal configmap: %v", err)
			}

			testhelper.CompareOutput(t, fmt.Sprintf("kube-%s", tc.name), string(serializedConfigmap), *update, ".yaml")
		})
	}
}

type fakeKubeControllerManagerConfigData struct {
	cluster *kubermaticv1.Cluster
}

func (f *fakeKubeControllerManagerConfigData) Cluster() *kubermaticv1.Cluster {
	return f.cluster
}

func (f *fakeKubeControllerManagerConfigData) GetKubernetesCloudProviderName() string {
	return "fake-cloud-provider"
}
