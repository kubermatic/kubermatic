package cluster

import (
	"errors"
	"fmt"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/defaulting"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/validation"
)

// Spec builds ClusterSpec kubermatic Custom Resource from API Cluster
func Spec(apiCluster apiv1.Cluster, dc *kubermaticv1.Datacenter) (*kubermaticv1.ClusterSpec, error) {
	spec := &kubermaticv1.ClusterSpec{
		HumanReadableName: apiCluster.Name,
		Cloud:             apiCluster.Spec.Cloud,
		MachineNetworks:   apiCluster.Spec.MachineNetworks,
		OIDC:              apiCluster.Spec.OIDC,
		Version:           apiCluster.Spec.Version,
	}

	providerName, err := provider.ClusterCloudProviderName(spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("invalid cloud spec: %v", err)
	}
	if providerName == "" {
		return nil, errors.New("cluster has no cloudprovider")
	}
	cloudProvider := cloud.Provider(dc)
	if cloudProvider == nil {
		return nil, errors.New("no cloudprovider found for cluster")
	}

	if err := defaulting.DefaultCreateClusterSpec(spec, cloudProvider); err != nil {
		return nil, err
	}

	return spec, validation.ValidateCreateClusterSpec(spec, dc, cloudProvider)
}
