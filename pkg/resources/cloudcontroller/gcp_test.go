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

package cloudcontroller

import (
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/resources"
)

func TestGCPInitContainerUtilImage(t *testing.T) {
	testcases := []struct {
		name              string
		utilRepository    string
		overwriteRegistry string
		expectedImage     string
	}{
		{
			name:          "default repository is rendered",
			expectedImage: "quay.io/kubermatic/util:2.10.0",
		},
		{
			name:           "overridden repository wins for the path",
			utilRepository: "registry.corp/kkp/util",
			expectedImage:  "registry.corp/kkp/util:2.10.0",
		},
		{
			name:              "overwrite registry is composed on the default repository",
			overwriteRegistry: "mirror.corp",
			expectedImage:     "mirror.corp/kubermatic/util:2.10.0",
		},
		{
			name:              "overwrite registry is composed on the overridden repository",
			utilRepository:    "registry.corp/kkp/util",
			overwriteRegistry: "mirror.corp",
			expectedImage:     "mirror.corp/kkp/util:2.10.0",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			config := &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "kkp.example.com",
					},
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						OverwriteRegistry: tc.overwriteRegistry,
					},
					Util: kubermaticv1.KubermaticUtilConfiguration{
						DockerRepository: tc.utilRepository,
					},
				},
			}

			config, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
			if err != nil {
				t.Fatalf("failed to default configuration: %v", err)
			}

			data := resources.NewTemplateDataBuilder().
				WithOverwriteRegistry(tc.overwriteRegistry).
				WithKubermaticConfiguration(config).
				Build()

			image := getGCPInitContainer(data).Image
			if image != tc.expectedImage {
				t.Fatalf("expected decode-sa init container image %q, got %q", tc.expectedImage, image)
			}
		})
	}
}
