package packet

import (
	"errors"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

const (
	defaultBillingCycle = "hourly"
)

type packet struct {
	secretKeySelector provider.SecretKeySelectorValueFunc
}

// NewCloudProvider creates a new packet provider.
func NewCloudProvider(secretKeyGetter provider.SecretKeySelectorValueFunc) provider.CloudProvider {
	return &packet{
		secretKeySelector: secretKeyGetter,
	}
}

// DefaultCloudSpec adds defaults to the CloudSpec.
func (p *packet) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec validates the given CloudSpec.
func (p *packet) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	_, _, err := GetCredentialsForCluster(spec, p.secretKeySelector)
	return err
}

// InitializeCloudProvider initializes a cluster, in particular
// updates BillingCycle to the defaultBillingCycle, if it is not set.
func (p *packet) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error
	if cluster.Spec.Cloud.Packet.BillingCycle == "" {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Packet.BillingCycle = defaultBillingCycle
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// CleanUpCloudProvider
func (p *packet) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (p *packet) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

func GetCredentialsForCluster(cloudSpec kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (apiKey, projectID string, err error) {
	apiKey = cloudSpec.Packet.APIKey
	projectID = cloudSpec.Packet.ProjectID

	if apiKey == "" {
		if cloudSpec.Packet.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		apiKey, err = secretKeySelector(cloudSpec.Packet.CredentialsReference, resources.PacketAPIKey)
		if err != nil {
			return "", "", err
		}
	}

	if projectID == "" {
		if cloudSpec.Packet.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		projectID, err = secretKeySelector(cloudSpec.Packet.CredentialsReference, resources.PacketProjectID)
		if err != nil {
			return "", "", err
		}
	}

	return apiKey, projectID, nil
}
