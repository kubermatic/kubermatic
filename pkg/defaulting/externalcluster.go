/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package defaulting

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
)

// DefaultExternalClusterSpec defaults the cluster spec when creating a new external cluster.
// This function assumes that the KubermaticConfiguration has already been defaulted
// (as the KubermaticConfigurationGetter does that automatically).
func DefaultExternalClusterSpec(ctx context.Context, spec *kubermaticv1.ExternalClusterSpec) error {
	// Ensure provider name matches the given spec
	providerName, err := kubermaticv1helper.ExternalClusterCloudProviderName(spec.CloudSpec)
	if err != nil {
		return fmt.Errorf("failed to determine cloud provider: %w", err)
	}

	spec.CloudSpec.ProviderName = kubermaticv1.ExternalClusterProvider(providerName)

	return nil
}
