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
	"testing"
)

const (
	componentName = "cli"
	version       = "4.1.18"
	digestImage   = openshiftImage + "@sha256:528f2ead3d1605bdf818579976d97df5dd86df0a2a5d80df9aa8209c82333a86"
)

func TestOpenshiftImageWithRegistry(t *testing.T) {
	tests := []struct {
		name          string
		image         string
		registry      string
		expectedImage string
	}{
		{
			name:          "Image with digest",
			image:         digestImage,
			registry:      "localhost:5000",
			expectedImage: "localhost:5000/openshift-release-dev/ocp-v4.0-art-dev:4.1.18-cli",
		},
		{
			name:          "Image with normal tag",
			image:         "quay.io/kubermatic/openshift:v4.1.18",
			registry:      "docker.io",
			expectedImage: "docker.io/kubermatic/openshift:v4.1.18",
		},
		{
			name:          "Same registry",
			image:         digestImage,
			registry:      "quay.io",
			expectedImage: digestImage,
		},
		{
			name:          "Empty registry",
			image:         digestImage,
			registry:      "",
			expectedImage: digestImage,
		},
	}
	for _, test := range tests {
		image, err := OpenshiftImageWithRegistry(test.image, componentName, version, test.registry)
		if err != nil {
			t.Fatalf("failed to run OpenshiftImageWithRegistry: %v", err)
		}
		if test.expectedImage != image {
			t.Fatalf("Invalid Openshift image returned. Expected [%v], got [%v]", test.expectedImage, image)
		}
	}
}
