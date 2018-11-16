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
	spec := &kubermaticv1.ClusterSpec{}
	spec.HumanReadableName = apiCluster.Name
	spec.Cloud = apiCluster.Spec.Cloud
	spec.MachineNetworks = apiCluster.Spec.MachineNetworks
	spec.Version = apiCluster.Spec.Version

	if err := defaulting.DefaultCreateClusterSpec(spec, cloudProviders); err != nil {
		return nil, err
	}

	return spec, validation.ValidateCreateClusterSpec(spec, cloudProviders)
}
