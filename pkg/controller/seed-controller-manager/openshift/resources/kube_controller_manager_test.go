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
