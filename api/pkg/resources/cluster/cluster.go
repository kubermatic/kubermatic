package cluster

import (
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/defaulting"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/validation"
)

// Spec builds ClusterSpec kubermatic Custom Resource from API Cluster
func Spec(apiCluster apiv1.Cluster, cloudProviders map[string]provider.CloudProvider) (*kubermaticv1.ClusterSpec, error) {
	spec := &kubermaticv1.ClusterSpec{
		HumanReadableName: apiCluster.Name,
		Cloud:             apiCluster.Spec.Cloud,
		MachineNetworks:   apiCluster.Spec.MachineNetworks,
		Version:           apiCluster.Spec.Version,
	}

	if err := defaulting.DefaultCreateClusterSpec(spec, cloudProviders); err != nil {
		return nil, err
	}

	return spec, validation.ValidateCreateClusterSpec(spec, cloudProviders)
}
