package validation

import (
	"errors"
	"fmt"

	"github.com/go-test/deep"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

var (
	// ErrCloudChangeNotAllowed describes that it is not allowed to change the cloud provider
	ErrCloudChangeNotAllowed = errors.New("not allowed to change the cloud provider")
)

// ValidateCreateClusterSpec validates the given cluster spec
func ValidateCreateClusterSpec(spec *kubermaticv1.ClusterSpec, cloudProviders map[string]provider.CloudProvider) error {
	if spec.HumanReadableName == "" {
		return errors.New("no name specified")
	}

	if spec.Cloud == nil {
		return errors.New("no cloud spec given")
	}

	if err := ValidateCloudSpec(spec.Cloud); err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	providerName, err := provider.ClusterCloudProviderName(spec.Cloud)
	if err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	cloudProvider, exists := cloudProviders[providerName]
	if !exists {
		return fmt.Errorf("invalid cloud provider '%s' specified: %v", err, providerName)
	}

	if err := cloudProvider.ValidateCloudSpec(spec.Cloud); err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	return nil
}

// ValidateCloudChange validates if the cloud provider has been changed
func ValidateCloudChange(newSpec, oldSpec *kubermaticv1.CloudSpec) error {
	if newSpec.Openstack == nil && oldSpec.Openstack != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.AWS == nil && oldSpec.AWS != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.Digitalocean == nil && oldSpec.Digitalocean != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.BringYourOwn == nil && oldSpec.BringYourOwn != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.Fake == nil && oldSpec.Fake != nil {
		return ErrCloudChangeNotAllowed
	}

	return nil
}

// ValidateUpdateCluster validates if the cluster update is allowed
func ValidateUpdateCluster(newCluster, oldCluster *kubermaticv1.Cluster, cloudProviders map[string]provider.CloudProvider) error {
	if err := ValidateCloudChange(newCluster.Spec.Cloud, oldCluster.Spec.Cloud); err != nil {
		return err
	}

	if newCluster.Address.ExternalName != oldCluster.Address.ExternalName {
		return errors.New("changing the external name is not allowed")
	}

	if newCluster.Address.ExternalPort != oldCluster.Address.ExternalPort {
		return errors.New("changing the external port is not allowed")
	}

	if newCluster.Address.IP != oldCluster.Address.IP {
		return errors.New("changing the ip is not allowed")
	}

	if newCluster.Address.URL != oldCluster.Address.URL {
		return errors.New("changing the url is not allowed")
	}

	if diff := deep.Equal(newCluster.Status, oldCluster.Status); diff != nil {
		return errors.New("changing the status is not allowed")
	}

	if diff := deep.Equal(newCluster.ObjectMeta, oldCluster.ObjectMeta); diff != nil {
		return errors.New("changing the metadata is not allowed")
	}

	if diff := deep.Equal(newCluster.TypeMeta, oldCluster.TypeMeta); diff != nil {
		return errors.New("changing the type metadata is not allowed")
	}

	if err := ValidateCloudSpec(newCluster.Spec.Cloud); err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	providerName, err := provider.ClusterCloudProviderName(newCluster.Spec.Cloud)
	if err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	cloudProvider, exists := cloudProviders[providerName]
	if !exists {
		return fmt.Errorf("invalid cloud provider '%s' specified: %v", err, providerName)
	}

	if err := cloudProvider.ValidateCloudSpec(newCluster.Spec.Cloud); err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	return nil
}

// ValidateCloudSpec validates if the cloud spec is valid
func ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	if spec.DatacenterName == "" {
		return errors.New("no node datacenter specified")
	}

	if spec.Fake != nil {
		if spec.Fake.Token == "" {
			return errors.New("no token specified")
		}
		return nil
	}

	if spec.Digitalocean != nil {
		if spec.Digitalocean.Token == "" {
			return errors.New("no token specified")
		}
		return nil
	}

	if spec.BringYourOwn != nil {
		return nil
	}

	if spec.AWS != nil {
		if spec.AWS.SecretAccessKey == "" {
			return errors.New("no secret access key specified")
		}
		if spec.AWS.AccessKeyID == "" {
			return errors.New("no access key ID specified")
		}
		return nil
	}

	if spec.Openstack != nil {
		if spec.Openstack.Domain == "" {
			return errors.New("no domain specified")
		}
		if spec.Openstack.Username == "" {
			return errors.New("no username specified")
		}
		if spec.Openstack.Password == "" {
			return errors.New("no password specified")
		}
		if spec.Openstack.Tenant == "" {
			return errors.New("no tenant specified")
		}
		return nil
	}

	return errors.New("no cloud provider specified")
}
