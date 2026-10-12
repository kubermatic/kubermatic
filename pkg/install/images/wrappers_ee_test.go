//go:build ee

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

package images

import (
	"testing"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
)

func TestGetAdditionalImagesIncludesKubeLB(t *testing.T) {
	testCases := []struct {
		name              string
		overwriteRegistry string
		expectedImages    []string
	}{
		{
			name: "default images",
			expectedImages: []string{
				"quay.io/kubermatic/kubelb-ccm-ee:v1.5.0",
				"docker.io/envoyproxy/envoy:distroless-v1.36.4",
				"docker.io/envoyproxy/gateway:v1.8.3",
			},
		},
		{
			name:              "registry override",
			overwriteRegistry: "registry.example.com/mirror",
			expectedImages: []string{
				"registry.example.com/mirror/kubermatic/kubelb-ccm-ee:v1.5.0",
				"registry.example.com/mirror/envoyproxy/envoy:distroless-v1.36.4",
				"registry.example.com/mirror/envoyproxy/gateway:v1.8.3",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := resources.NewTemplateDataBuilder().
				WithCluster(&kubermaticv1.Cluster{}).
				WithDatacenter(&kubermaticv1.Datacenter{}).
				WithSeed(&kubermaticv1.Seed{
					Spec: kubermaticv1.SeedSpec{
						Metering: &kubermaticv1.MeteringConfiguration{StorageSize: "1Gi"},
					},
				}).
				WithOverwriteRegistry(tc.overwriteRegistry).
				Build()

			images, err := getAdditionalImagesFromReconcilers(data)
			require.NoError(t, err)
			for _, expectedImage := range tc.expectedImages {
				require.Contains(t, images, expectedImage)
			}
		})
	}
}

func TestGetAdditionalImagesUsesConfiguredKubeLBImage(t *testing.T) {
	config := &kubermaticv1.KubermaticConfiguration{}
	config.Spec.UserCluster.KubeLB.ImageRepository = "registry.example.com/custom/ccm"
	config.Spec.UserCluster.KubeLB.ImageTag = "v1.5.0-custom"

	data, err := getTemplateData(
		config,
		&version.Version{Version: semverlib.MustParse("1.35.0")},
		kubermaticv1.CloudSpec{},
		nil,
		false,
		kubermatic.GetFakeVersions(),
		nil,
		&kubermaticv1.Seed{},
	)
	require.NoError(t, err)

	images, err := getAdditionalImagesFromReconcilers(data)
	require.NoError(t, err)
	require.Contains(t, images, "registry.example.com/custom/ccm:v1.5.0-custom")
	require.NotContains(t, images, "quay.io/kubermatic/kubelb-ccm-ee:v1.5.0")
	require.Contains(t, images, "docker.io/envoyproxy/envoy:distroless-v1.36.4")
	require.Contains(t, images, "docker.io/envoyproxy/gateway:v1.8.3")
}
