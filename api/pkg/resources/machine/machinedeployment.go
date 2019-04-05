package machine

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Masterminds/semver"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/validation"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// Deployment returns a Machine Deployment object for the given Node Deployment spec.
func Deployment(c *kubermaticv1.Cluster, nd *apiv1.NodeDeployment, dc provider.DatacenterMeta, keys []*kubermaticv1.UserSSHKey) (*clusterv1alpha1.MachineDeployment, error) {
	md := &clusterv1alpha1.MachineDeployment{}

	if nd.Name != "" {
		md.Name = nd.Name
	} else {
		// GenerateName can be set only if Name is empty to avoid confusing error:
		// https://github.com/kubernetes/kubernetes/issues/32220
		md.GenerateName = fmt.Sprintf("kubermatic-%s-", c.Name)
	}

	md.Namespace = metav1.NamespaceSystem

	md.Spec.Selector.MatchLabels = map[string]string{
		"machine": fmt.Sprintf("md-%s-%s", c.Name, rand.String(10)),
	}
	md.Spec.Template.Labels = md.Spec.Selector.MatchLabels
	md.Spec.Template.Spec.Labels = nd.Spec.Template.Labels

	md.Spec.Replicas = &nd.Spec.Replicas
	md.Spec.Template.Spec.Versions.Kubelet = nd.Spec.Template.Versions.Kubelet

	if len(c.Spec.MachineNetworks) > 0 {
		md.Spec.Template.Annotations = map[string]string{
			"machine-controller.kubermatic.io/initializers": "ipam",
		}
	}

	if nd.Spec.Paused != nil {
		md.Spec.Paused = *nd.Spec.Paused
	}

	config, err := getProviderConfig(c, nd, dc, keys)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	md.Spec.Template.Spec.ProviderSpec.Value = &runtime.RawExtension{Raw: b}

	return md, nil
}

func getProviderConfig(c *kubermaticv1.Cluster, nd *apiv1.NodeDeployment, dc provider.DatacenterMeta, keys []*kubermaticv1.UserSSHKey) (*providerconfig.Config, error) {
	config := providerconfig.Config{}
	config.SSHPublicKeys = make([]string, len(keys))
	for i, key := range keys {
		config.SSHPublicKeys[i] = key.Spec.PublicKey
	}

	var cloudExt *runtime.RawExtension
	var err error

	switch {
	case nd.Spec.Template.Cloud.AWS != nil:
		config.CloudProvider = providerconfig.CloudProviderAWS
		cloudExt, err = getAWSProviderSpec(c, nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.Cloud.Azure != nil:
		config.CloudProvider = providerconfig.CloudProviderAzure
		cloudExt, err = getAzureProviderSpec(c, nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.Cloud.VSphere != nil:
		config.CloudProvider = providerconfig.CloudProviderVsphere

		// We use OverwriteCloudConfig for VSphere to ensure we always use the credentials
		// passed in via frontend for the cloud-provider functionality.
		overwriteCloudConfig, err := cloudconfig.CloudConfig(c, &dc)
		if err != nil {
			return nil, err
		}
		config.OverwriteCloudConfig = &overwriteCloudConfig

		cloudExt, err = getVSphereProviderSpec(c, nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.Cloud.Openstack != nil:
		config.CloudProvider = providerconfig.CloudProviderOpenstack
		if err := validation.ValidateCreateNodeSpec(c, &nd.Spec.Template, &dc); err != nil {
			return nil, err
		}

		cloudExt, err = getOpenstackProviderSpec(c, nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.Cloud.Hetzner != nil:
		config.CloudProvider = providerconfig.CloudProviderHetzner
		cloudExt, err = getHetznerProviderSpec(c, nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.Cloud.Digitalocean != nil:
		config.CloudProvider = providerconfig.CloudProviderDigitalocean
		cloudExt, err = getDigitaloceanProviderSpec(c, nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown cloud provider")
	}
	config.CloudProviderSpec = *cloudExt

	var osExt *runtime.RawExtension

	// OS specifics
	switch {
	case nd.Spec.Template.OperatingSystem.ContainerLinux != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCoreos
		osExt, err = getCoreosOperatingSystemSpec(nd.Spec.Template)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.OperatingSystem.Ubuntu != nil:
		config.OperatingSystem = providerconfig.OperatingSystemUbuntu
		osExt, err = getUbuntuOperatingSystemSpec(nd.Spec.Template)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.OperatingSystem.CentOS != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCentOS
		osExt, err = getCentOSOperatingSystemSpec(nd.Spec.Template)
		if err != nil {
			return nil, err
		}
	default:

		return nil, errors.New("unknown OS")
	}

	config.OperatingSystemSpec = *osExt
	return &config, nil
}

// Validate if the node deployment structure fulfills certain requirements. It returns node deployment with updated
// kubelet version if it wasn't specified.
func Validate(nd *apiv1.NodeDeployment, controlPlaneVersion *semver.Version) (*apiv1.NodeDeployment, error) {
	if nd.Spec.Template.Cloud.Openstack == nil &&
		nd.Spec.Template.Cloud.Digitalocean == nil &&
		nd.Spec.Template.Cloud.AWS == nil &&
		nd.Spec.Template.Cloud.Hetzner == nil &&
		nd.Spec.Template.Cloud.VSphere == nil &&
		nd.Spec.Template.Cloud.Azure == nil {
		return nil, fmt.Errorf("node deployment needs to have cloud provider data")
	}

	if nd.Spec.Template.Versions.Kubelet != "" {
		kubeletVersion, err := semver.NewVersion(nd.Spec.Template.Versions.Kubelet)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kubelet version: %v", err)
		}

		if err = common.EnsureVersionCompatible(controlPlaneVersion, kubeletVersion); err != nil {
			return nil, err
		}

		nd.Spec.Template.Versions.Kubelet = kubeletVersion.String()
	} else {
		nd.Spec.Template.Versions.Kubelet = controlPlaneVersion.String()
	}

	return nd, nil
}
