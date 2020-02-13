package kubeone

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type kubeone struct {
}

func (k *kubeone) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (k *kubeone) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	return nil
}

func (k *kubeone) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if !cluster.Spec.Cloud.Kubeone.Edge {
		return nil, fmt.Errorf("Non edge clusters are not supported at the moment")
	}

	return cluster, nil
}

func (k *kubeone) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (k *kubeone) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

func NewCloudProvider() provider.CloudProvider {
	return &kubeone{}
}
