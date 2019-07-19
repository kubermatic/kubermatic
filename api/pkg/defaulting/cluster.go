package defaulting

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

// DefaultCreateClusterSpec defalts the cluster spec when creating a new cluster
func DefaultCreateClusterSpec(
	spec *kubermaticv1.ClusterSpec,
	cloudProvider provider.CloudProvider) error {

	if err := cloudProvider.DefaultCloudSpec(&spec.Cloud); err != nil {
		return fmt.Errorf("failed to default cloud spec: %v", err)
	}

	return nil
}
