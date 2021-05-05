/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"

	"github.com/coreos/locksmith/pkg/timeutil"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/api/equality"
	utilerror "k8s.io/apimachinery/pkg/util/errors"
)

var (
	// ErrCloudChangeNotAllowed describes that it is not allowed to change the cloud provider
	ErrCloudChangeNotAllowed = errors.New("not allowed to change the cloud provider")
)

// ValidateCreateClusterSpec validates the given cluster spec
func ValidateCreateClusterSpec(spec *kubermaticv1.ClusterSpec, dc *kubermaticv1.Datacenter, cloudProvider provider.CloudProvider) error {
	if spec.HumanReadableName == "" {
		return errors.New("no name specified")
	}

	if err := ValidateCloudSpec(spec.Cloud, dc); err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	if spec.Version.Semver() == nil || spec.Version.String() == "" {
		return errors.New(`invalid cloud spec "Version" is required but was not specified`)
	}

	if err := cloudProvider.ValidateCloudSpec(spec.Cloud); err != nil {
		return fmt.Errorf("invalid cloud spec: %v", err)
	}

	if err := validateMachineNetworksFromClusterSpec(spec); err != nil {
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
		return errors.New("can't specify machinenetworks on kubernetes <= 1.9.0")
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
	if newSpec.Kubevirt == nil && oldSpec.Kubevirt != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.Alibaba == nil && oldSpec.Alibaba != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.Anexia == nil && oldSpec.Anexia != nil {
		return ErrCloudChangeNotAllowed
	}
	if newSpec.DatacenterName != oldSpec.DatacenterName {
		return errors.New("changing the datacenter is not allowed")
	}

	return nil
}

// ValidateUpdateCluster validates if the cluster update is allowed
func ValidateUpdateCluster(ctx context.Context, newCluster, oldCluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter,
	clusterProvider *kubernetesprovider.ClusterProvider, caBundle *x509.CertPool) error {
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

	if err := kuberneteshelper.ValidateKubernetesToken(newCluster.Address.AdminToken); err != nil {
		return fmt.Errorf("invalid admin token: %v", err)
	}

	if !equality.Semantic.DeepEqual(newCluster.Status, oldCluster.Status) {
		return errors.New("changing the status is not allowed")
	}

	// Editing labels is allowed even though it is part of metadata.
	oldCluster.Labels = newCluster.Labels

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

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, clusterProvider.GetSeedClusterAdminRuntimeClient())
	cloudProvider, err := cloud.Provider(dc, secretKeySelectorFunc, caBundle)
	if err != nil {
		return err
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
func ValidateCloudSpec(spec kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) error {
	if spec.DatacenterName == "" {
		return errors.New("no node datacenter specified")
	}

	switch {
	case spec.Fake != nil:
		if dc.Spec.Fake == nil {
			return fmt.Errorf("datacenter %q is not a fake datacenter", spec.DatacenterName)
		}
		return validateFakeCloudSpec(spec.Fake)
	case spec.AWS != nil:
		if dc.Spec.AWS == nil {
			return fmt.Errorf("datacenter %q is not a AWS datacenter", spec.DatacenterName)
		}
		return validateAWSCloudSpec(spec.AWS)
	case spec.Digitalocean != nil:
		if dc.Spec.Digitalocean == nil {
			return fmt.Errorf("datacenter %q is not a Digitalocean datacenter", spec.DatacenterName)
		}
		return validateDigitaloceanCloudSpec(spec.Digitalocean)
	case spec.Openstack != nil:
		if dc.Spec.Openstack == nil {
			return fmt.Errorf("datacenter %q is not an Openstack datacenter", spec.DatacenterName)
		}
		return validateOpenStackCloudSpec(spec.Openstack, dc)
	case spec.Azure != nil:
		if dc.Spec.Azure == nil {
			return fmt.Errorf("datacenter %q is not an Azure datacenter", spec.DatacenterName)
		}
		return validateAzureCloudSpec(spec.Azure)
	case spec.VSphere != nil:
		if dc.Spec.VSphere == nil {
			return fmt.Errorf("datacenter %q is not a vSphere datacenter", spec.DatacenterName)
		}
		return validateVSphereCloudSpec(spec.VSphere)
	case spec.GCP != nil:
		if dc.Spec.GCP == nil {
			return fmt.Errorf("datacenter %q is not a GCP datacenter", spec.DatacenterName)
		}
		return validateGCPCloudSpec(spec.GCP)
	case spec.Packet != nil:
		if dc.Spec.Packet == nil {
			return fmt.Errorf("datacenter %q is not a Packet datacenter", spec.DatacenterName)
		}
		return validatePacketCloudSpec(spec.Packet)
	case spec.Hetzner != nil:
		if dc.Spec.Hetzner == nil {
			return fmt.Errorf("datacenter %q is not a Hetzner datacenter", spec.DatacenterName)
		}
		return validateHetznerCloudSpec(spec.Hetzner)
	case spec.BringYourOwn != nil:
		if dc.Spec.BringYourOwn == nil {
			return fmt.Errorf("datacenter %q is not a bringyourown datacenter", spec.DatacenterName)
		}
		return nil
	case spec.Kubevirt != nil:
		if dc.Spec.Kubevirt == nil {
			return fmt.Errorf("datacenter %q is not a kubevirt datacenter", spec.DatacenterName)
		}
		return validateKubevirtCloudSpec(spec.Kubevirt)
	case spec.Alibaba != nil:
		if dc.Spec.Alibaba == nil {
			return fmt.Errorf("datacenter %q is not a alibaba datacenter", spec.DatacenterName)
		}
		return validateAlibabaCloudSpec(spec.Alibaba)
	case spec.Anexia != nil:
		if dc.Spec.Anexia == nil {
			return fmt.Errorf("datacenter %q is not a anexia datacenter", spec.DatacenterName)
		}
		return validateAnexiaCloudSpec(spec.Anexia)
	default:
		return errors.New("no cloud provider specified")
	}
}

func validateOpenStackCloudSpec(spec *kubermaticv1.OpenstackCloudSpec, dc *kubermaticv1.Datacenter) error {
	if spec.Domain == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.OpenstackDomain); err != nil {
			return err
		}
	}
	if spec.Username == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.OpenstackUsername); err != nil {
			return err
		}
	}
	if spec.Password == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.OpenstackPassword); err != nil {
			return err
		}
	}

	var errs []error
	if spec.Tenant == "" && spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		errs = append(errs, kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.OpenstackTenant))
	}
	if spec.TenantID == "" && spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		errs = append(errs, kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.OpenstackTenantID))
	}
	if utilerror.NewAggregate(errs) != nil {
		return errors.New("no tenant name or ID specified")
	}

	if spec.FloatingIPPool == "" && dc.Spec.Openstack != nil && dc.Spec.Openstack.EnforceFloatingIP {
		return errors.New("no floating ip pool specified")
	}
	return nil
}

func validateAWSCloudSpec(spec *kubermaticv1.AWSCloudSpec) error {
	if spec.AccessKeyID == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.AWSAccessKeyID); err != nil {
			return err
		}
	}
	if spec.SecretAccessKey == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.AWSSecretAccessKey); err != nil {
			return err
		}
	}
	return nil
}

func validateGCPCloudSpec(spec *kubermaticv1.GCPCloudSpec) error {
	if spec.ServiceAccount == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.GCPServiceAccount); err != nil {
			return err
		}
	}
	return nil
}

func validateHetznerCloudSpec(spec *kubermaticv1.HetznerCloudSpec) error {
	if spec.Token == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.HetznerToken); err != nil {
			return err
		}
	}

	return nil
}

func validatePacketCloudSpec(spec *kubermaticv1.PacketCloudSpec) error {
	if spec.APIKey == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.PacketAPIKey); err != nil {
			return err
		}
	}
	if spec.ProjectID == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.PacketProjectID); err != nil {
			return err
		}
	}
	return nil
}

func validateVSphereCloudSpec(spec *kubermaticv1.VSphereCloudSpec) error {
	if spec.Username == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.VsphereUsername); err != nil {
			return err
		}
	}
	if spec.Password == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.VspherePassword); err != nil {
			return err
		}
	}

	return nil
}

func validateAzureCloudSpec(spec *kubermaticv1.AzureCloudSpec) error {
	if spec.TenantID == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.AzureTenantID); err != nil {
			return err
		}
	}
	if spec.SubscriptionID == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.AzureSubscriptionID); err != nil {
			return err
		}
	}
	if spec.ClientID == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.AzureClientID); err != nil {
			return err
		}
	}
	if spec.ClientSecret == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.AzureClientSecret); err != nil {
			return err
		}
	}

	return nil
}

func validateDigitaloceanCloudSpec(spec *kubermaticv1.DigitaloceanCloudSpec) error {
	if spec.Token == "" {
		if spec.CredentialsReference == nil {
			return errors.New("no token or credentials reference specified")
		}

		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.DigitaloceanToken); err != nil {
			return err
		}
	}

	return nil
}

func validateFakeCloudSpec(spec *kubermaticv1.FakeCloudSpec) error {
	if spec.Token == "" {
		return errors.New("no token specified")
	}

	return nil
}

func validateKubevirtCloudSpec(spec *kubermaticv1.KubevirtCloudSpec) error {
	if spec.Kubeconfig == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.KubevirtKubeConfig); err != nil {
			return err
		}
	}

	return nil
}

func validateAlibabaCloudSpec(spec *kubermaticv1.AlibabaCloudSpec) error {
	if spec.AccessKeyID == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.AlibabaAccessKeyID); err != nil {
			return err
		}
	}
	if spec.AccessKeySecret == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.AlibabaAccessKeySecret); err != nil {
			return err
		}
	}
	return nil
}

func validateAnexiaCloudSpec(spec *kubermaticv1.AnexiaCloudSpec) error {
	if spec.Token == "" {
		if spec.CredentialsReference == nil {
			return errors.New("no token or credentials reference specified")
		}

		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.AnexiaToken); err != nil {
			return err
		}
	}

	return nil
}

func ValidateUpdateWindow(updateWindow *kubermaticv1.UpdateWindow) error {
	if updateWindow != nil && updateWindow.Start != "" && updateWindow.Length != "" {
		_, err := timeutil.ParsePeriodic(updateWindow.Start, updateWindow.Length)
		if err != nil {
			return fmt.Errorf("error parsing update window: %s", err)
		}
	}
	return nil
}

func ValidateLeaderElectionSettings(l kubermaticv1.LeaderElectionSettings) error {
	if l.LeaseDurationSeconds != nil && *l.LeaseDurationSeconds < 0 {
		return fmt.Errorf("lease duration seconds cannot be negative: %d", *l.LeaseDurationSeconds)
	}
	if l.RenewDeadlineSeconds != nil && *l.RenewDeadlineSeconds < 0 {
		return fmt.Errorf("renew deadline seconds cannot be negative: %d", *l.RenewDeadlineSeconds)
	}
	if l.RetryPeriodSeconds != nil && *l.RetryPeriodSeconds < 0 {
		return fmt.Errorf("retry period seconds cannot be negative: %d", *l.RetryPeriodSeconds)
	}
	if lds, rds := l.LeaseDurationSeconds, l.RenewDeadlineSeconds; (lds == nil) != (rds == nil) {
		return errors.New("leader election lease duration and renew deadline should be either both specified or unspecified")
	}
	if lds, rds := l.LeaseDurationSeconds, l.RenewDeadlineSeconds; lds != nil && *lds < *rds {
		return errors.New("control plane leader election renew deadline cannot be smaller than lease duration")
	}
	return nil
}
