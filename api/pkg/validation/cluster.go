package validation

import (
	"errors"
	"fmt"
	"net"
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
func ValidateCreateClusterSpec(spec *kubermaticv1.ClusterSpec, cloudProviders map[string]provider.CloudProvider, dc provider.DatacenterMeta) error {
	if spec.HumanReadableName == "" {
		return errors.New("no name specified")
	}

	if err := ValidateCloudSpec(spec.Cloud, dc); err != nil {
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

	if spec.Version.Semver() == nil || spec.Version.String() == "" {
		return errors.New(`invalid cloud spec "Version" is required but was not specified`)
	}

	if err = cloudProvider.ValidateCloudSpec(spec.Cloud); err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	if err = validateMachineNetworksFromClusterSpec(spec); err != nil {
		return fmt.Errorf("machine network validation failed, see: %v", err)
	}

	return nil
}

func validateMachineNetworksFromClusterSpec(spec *kubermaticv1.ClusterSpec) error {
	networks := spec.MachineNetworks

	if len(networks) == 0 {
		return nil
	}

	if len(networks) > 0 && spec.Version.Semver().Minor() < 9 {
		return errors.New("cant specify machinenetworks on kubernetes <= 1.9.0")
	}

	if len(networks) > 0 && spec.Cloud.VSphere == nil {
		return errors.New("machineNetworks are only supported with the vSphere provider")
	}

	for _, network := range networks {
		_, _, err := net.ParseCIDR(network.CIDR)
		if err != nil {
			return fmt.Errorf("couldn't parse cidr `%s`, see: %v", network.CIDR, err)
		}

		if net.ParseIP(network.Gateway) == nil {
			return fmt.Errorf("couldn't parse gateway `%s`", network.Gateway)
		}

		if len(network.DNSServers) > 0 {
			for _, dnsServer := range network.DNSServers {
				if net.ParseIP(dnsServer) == nil {
					return fmt.Errorf("couldn't parse dns server `%s`", dnsServer)
				}
			}
		}
	}

	return nil
}

// ValidateCloudChange validates if the cloud provider has been changed
func ValidateCloudChange(newSpec, oldSpec kubermaticv1.CloudSpec) error {
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
	if newSpec.Packet == nil && oldSpec.Packet != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.GCP == nil && oldSpec.GCP != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.Azure == nil && oldSpec.Azure != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.DatacenterName != oldSpec.DatacenterName {
		return errors.New("changing the datacenter is not allowed")
	}

	return nil
}

// ValidateUpdateCluster validates if the cluster update is allowed
func ValidateUpdateCluster(newCluster, oldCluster *kubermaticv1.Cluster, cloudProviders map[string]provider.CloudProvider, dc provider.DatacenterMeta) error {
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

	if err := ValidateCloudSpec(newCluster.Spec.Cloud, dc); err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	// We ignore the error, since we're here to check the new config, not the old one.
	oldProviderName, _ := provider.ClusterCloudProviderName(oldCluster.Spec.Cloud)

	providerName, err := provider.ClusterCloudProviderName(newCluster.Spec.Cloud)
	if err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	if oldProviderName != providerName {
		return fmt.Errorf("changing to a different provider is not allowed")
	}

	cloudProvider, exists := cloudProviders[providerName]
	if !exists {
		return fmt.Errorf("invalid cloud provider '%s' specified: %v", err, providerName)
	}

	if err := cloudProvider.ValidateCloudSpec(newCluster.Spec.Cloud); err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	if err := cloudProvider.ValidateCloudSpecUpdate(oldCluster.Spec.Cloud, newCluster.Spec.Cloud); err != nil {
		return fmt.Errorf("invalid cloud spec modification: %v", err)
	}

	return nil
}

// ValidateCloudSpec validates if the cloud spec is valid
func ValidateCloudSpec(spec kubermaticv1.CloudSpec, dc provider.DatacenterMeta) error {
	if spec.DatacenterName == "" {
		return errors.New("no node datacenter specified")
	}

	switch true {
	case spec.Fake != nil:
		return validateFakeCloudSpec(spec.Fake)
	case spec.AWS != nil:
		return validateAWSCloudSpec(spec.AWS)
	case spec.Digitalocean != nil:
		return validateDigitaloceanCloudSpec(spec.Digitalocean)
	case spec.Openstack != nil:
		return validateOpenStackCloudSpec(spec.Openstack, dc)
	case spec.Azure != nil:
		return validateAzureCloudSpec(spec.Azure)
	case spec.VSphere != nil:
		return validateVSphereCloudSpec(spec.VSphere)
	case spec.GCP != nil:
		return validateGCPCloudSpec(spec.GCP)
	case spec.Packet != nil:
		return validatePacketCloudSpec(spec.Packet)
	case spec.Hetzner != nil:
		return validateHetznerCloudSpec(spec.Hetzner)
	default:
		return errors.New("no cloud provider specified")
	}
}

func validateOpenStackCloudSpec(spec *kubermaticv1.OpenstackCloudSpec, dc provider.DatacenterMeta) error {
	if spec.Domain == "" {
		return errors.New("no domain specified")
	}
	if spec.Username == "" {
		return errors.New("no username specified")
	}
	if spec.Password == "" {
		return errors.New("no password specified")
	}
	if spec.Tenant == "" && spec.TenantID == "" {
		return errors.New("no tenant name or ID specified")
	}
	if spec.FloatingIPPool == "" && dc.Spec.Openstack != nil && dc.Spec.Openstack.EnforceFloatingIP {
		return errors.New("no floating ip pool specified")
	}
	return nil
}

func validateAWSCloudSpec(spec *kubermaticv1.AWSCloudSpec) error {
	if spec.SecretAccessKey == "" {
		return errors.New("no secret access key specified")
	}
	if spec.AccessKeyID == "" {
		return errors.New("no access key ID specified")
	}
	return nil
}

func validateGCPCloudSpec(spec *kubermaticv1.GCPCloudSpec) error {
	if spec.ServiceAccount == "" {
		return errors.New("no serviceaccount specified")
	}
	return nil
}

func validateHetznerCloudSpec(spec *kubermaticv1.HetznerCloudSpec) error {
	if spec.Token == "" {
		return errors.New("no token specified")
	}

	return nil
}

func validatePacketCloudSpec(spec *kubermaticv1.PacketCloudSpec) error {
	if spec.APIKey == "" {
		return errors.New("no API key specified")
	}
	if spec.ProjectID == "" {
		return errors.New("no project ID specified")
	}

	return nil
}

func validateVSphereCloudSpec(spec *kubermaticv1.VSphereCloudSpec) error {
	if spec.Username == "" {
		return errors.New("no username specified")
	}
	if spec.Password == "" {
		return errors.New("no password specified")
	}

	return nil
}

func validateAzureCloudSpec(spec *kubermaticv1.AzureCloudSpec) error {
	if spec.TenantID == "" {
		return errors.New("no tenant ID specified")
	}
	if spec.SubscriptionID == "" {
		return errors.New("no subscription ID specified")
	}
	if spec.ClientID == "" {
		return errors.New("no client ID specified")
	}
	if spec.ClientSecret == "" {
		return errors.New("no client secret specified")
	}

	return nil
}

func validateDigitaloceanCloudSpec(spec *kubermaticv1.DigitaloceanCloudSpec) error {
	if spec.Token == "" {
		return errors.New("no token specified")
	}

	return nil
}

func validateFakeCloudSpec(spec *kubermaticv1.FakeCloudSpec) error {
	if spec.Token == "" {
		return errors.New("no token specified")
	}

	return nil
}
