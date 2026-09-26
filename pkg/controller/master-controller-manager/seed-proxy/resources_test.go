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

package seedproxy

import (
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSeedProxyDeploymentUtilImage(t *testing.T) {
	seed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "europe-west3",
			Namespace: "kubermatic",
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName(seed),
			Namespace: seed.Namespace,
		},
	}

	testcases := []struct {
		name              string
		utilRepository    string
		overwriteRegistry string
		expectedImage     string
	}{
		{
			name:          "default repository is rendered verbatim",
			expectedImage: "quay.io/kubermatic/util:2.10.0",
		},
		{
			name:           "overridden repository is rendered verbatim",
			utilRepository: "registry.corp/kkp/util",
			expectedImage:  "registry.corp/kkp/util:2.10.0",
		},
		{
			name:              "user-cluster overwrite registry does not rewrite the image",
			utilRepository:    "registry.corp/kkp/util",
			overwriteRegistry: "mirror.corp",
			expectedImage:     "registry.corp/kkp/util:2.10.0",
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

			_, reconciler := masterDeploymentReconciler(seed, secret, config)()

			deployment, err := reconciler(&appsv1.Deployment{})
			if err != nil {
				t.Fatalf("failed to reconcile Deployment: %v", err)
			}

			image := deployment.Spec.Template.Spec.Containers[0].Image
			if image != tc.expectedImage {
				t.Fatalf("expected proxy container image %q, got %q", tc.expectedImage, image)
			}
		})
	}
}
