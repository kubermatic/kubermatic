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
	"flag"
	"strings"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	testhelper "github.com/kubermatic/kubermatic/api/pkg/test"

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
