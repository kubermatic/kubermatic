package resources

import (
	"flag"
	"strings"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	testhelper "github.com/kubermatic/kubermatic/pkg/test"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var update = flag.Bool("update", false, "update .golden files")

type openshiftAPIServerCreatorDataFake struct {
	cluster *kubermaticv1.Cluster
}

func (o *openshiftAPIServerCreatorDataFake) Cluster() *kubermaticv1.Cluster {
	return o.cluster
}

func TestOpenshiftAPIServerConfigMapCreator(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "Generate simple config",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := &openshiftAPIServerCreatorDataFake{cluster: &kubermaticv1.Cluster{}}
			creatorGetter := OpenshiftAPIServerConfigMapCreator(data)
			name, creator := creatorGetter()
			if name != openshiftAPIServerConfigMapName {
				t.Errorf("expected name to be %q was %q", openshiftAPIServerConfigMapName, name)
			}

			configMap, err := creator(&corev1.ConfigMap{})
			if err != nil {
				t.Fatalf("failed calling creator: %v", err)
			}

			serializedConfigmap, err := yaml.Marshal(configMap)
			if err != nil {
				t.Fatalf("failed to marshal configmap: %v", err)
			}

			testhelper.CompareOutput(t, strings.ReplaceAll(tc.name, " ", "_"), string(serializedConfigmap), *update, ".yaml")
		})
	}
}
