package alibaba

import (
	"errors"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

type alibaba struct {
	dc                *kubermaticv1.DatacenterSpecAlibaba
	secretKeySelector provider.SecretKeySelectorValueFunc
}

func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*alibaba, error) {
	if dc.Spec.Alibaba == nil {
		return nil, errors.New("datacenter is not an Alibaba datacenter")
	}
	return &alibaba{
		dc:                dc.Spec.Alibaba,
		secretKeySelector: secretKeyGetter,
	}, nil
}

func (k *alibaba) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (g *alibaba) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	if spec.Alibaba.AccessKeyID == "" {
		return fmt.Errorf("accessKeyID cannot be empty")
	}
	if spec.Alibaba.AccessKeySecret == "" {
		return fmt.Errorf("accessKeySecret cannot be empty")
	}
	return nil
}

func (k *alibaba) InitializeCloudProvider(c *kubermaticv1.Cluster, p provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return c, nil
}

func (k *alibaba) CleanUpCloudProvider(c *kubermaticv1.Cluster, p provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return c, nil
}

func (k *alibaba) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (accessKeyID, accessKeySecret string, err error) {
	accessKeyID = cloud.Alibaba.AccessKeyID
	accessKeySecret = cloud.Alibaba.AccessKeySecret

	if accessKeyID == "" {
		if cloud.Alibaba.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		accessKeyID, err = secretKeySelector(cloud.Alibaba.CredentialsReference, resources.AlibabaAccessKeyID)
		if err != nil {
			return "", "", err
		}
	}

	if accessKeySecret == "" {
		if cloud.Alibaba.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		accessKeySecret, err = secretKeySelector(cloud.Alibaba.CredentialsReference, resources.AlibabaAccessKeySecret)
		if err != nil {
			return "", "", err
		}
	}

	return accessKeyID, accessKeySecret, nil
}
