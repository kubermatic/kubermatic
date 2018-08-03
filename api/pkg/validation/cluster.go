package validation

import (
	"errors"
	"fmt"
	"regexp"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/apimachinery/pkg/api/equality"
)

var (
	// ErrCloudChangeNotAllowed describes that it is not allowed to change the cloud provider
	ErrCloudChangeNotAllowed = errors.New("not allowed to change the cloud provider")

	tokenValidator = regexp.MustCompile(`[bcdfghjklmnpqrstvwxz2456789]{6}\.[bcdfghjklmnpqrstvwxz2456789]{16}`)
)

// ValidateKubernetesToken checks if a given token is syntactically correct.
func ValidateKubernetesToken(token string) error {
	if !tokenValidator.MatchString(token) {
		return fmt.Errorf("token is malformed, must match %s", tokenValidator.String())
	}

	return nil
}

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

	if spec.Version == "" {
		return errors.New("invalid cloud spec \"Version\" is required but was not specified")
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
	if newSpec.Hetzner == nil && oldSpec.Hetzner != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.VSphere == nil && oldSpec.VSphere != nil {
		return ErrCloudChangeNotAllowed
	}

	return nil
}

// ValidateUpdateCluster validates if the cluster update is allowed
func ValidateUpdateCluster(newCluster, oldCluster *kubermaticv1.Cluster, cloudProviders map[string]provider.CloudProvider) error {
	if newCluster.Spec.Cloud == nil {
		return errors.New("deleting the cloud spec is not allowed")
	}
	if err := ValidateCloudChange(newCluster.Spec.Cloud, oldCluster.Spec.Cloud); err != nil {
		return err
	}

	if newCluster.Address.ExternalName != oldCluster.Address.ExternalName {
		return errors.New("changing the external name is not allowed")
	}

	if newCluster.Address.IP != oldCluster.Address.IP {
		return errors.New("changing the ip is not allowed")
	}

	if newCluster.Address.URL != oldCluster.Address.URL {
		return errors.New("changing the url is not allowed")
	}

	if err := ValidateKubernetesToken(newCluster.Address.AdminToken); err != nil {
		return fmt.Errorf("invalid admin token: %v", err)
	}

	if !equality.Semantic.DeepEqual(newCluster.Status, oldCluster.Status) {
		return errors.New("changing the status is not allowed")
	}

	if !equality.Semantic.DeepEqual(newCluster.ObjectMeta, oldCluster.ObjectMeta) {
		return errors.New("changing the metadata is not allowed")
	}

	if !equality.Semantic.DeepEqual(newCluster.TypeMeta, oldCluster.TypeMeta) {
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

	if spec.Azure != nil {
		if spec.Azure.TenantID == "" {
			return errors.New("no tenant ID specified")
		}
		if spec.Azure.SubscriptionID == "" {
			return errors.New("no subscription ID specified")
		}
		if spec.Azure.ClientID == "" {
			return errors.New("no client ID specified")
		}
		if spec.Azure.ClientSecret == "" {
			return errors.New("no client secret specified")
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

	if spec.Hetzner != nil {
		if spec.Hetzner.Token == "" {
			return errors.New("no token specified")
		}
		return nil
	}

	if spec.VSphere != nil {
		if spec.VSphere.Username == "" {
			return errors.New("no username specified")
		}

		if spec.VSphere.Password == "" {
			return errors.New("no password provided")
		}

		return nil
	}

	return errors.New("no cloud provider specified")
}
