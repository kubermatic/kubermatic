package defaulting

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

// DefaultCreateClusterSpec defalts the cluster spec when creating a new cluster
func DefaultCreateClusterSpec(
	spec *kubermaticv1.ClusterSpec,
	cloudProviders map[string]provider.CloudProvider) error {

	providerName, err := provider.ClusterCloudProviderName(spec.Cloud)
	if err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}
	cloudProvider, exists := cloudProviders[providerName]
	if !exists {
		// Validation happens further down the chain so we just
		// return here
		return nil
	}

	if err := cloudProvider.DefaultCloudSpec(spec.Cloud); err != nil {
		return fmt.Errorf("failed to default cloud spec: %v", err)
	}

	return nil
}
