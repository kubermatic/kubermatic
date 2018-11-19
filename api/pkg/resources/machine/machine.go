package machine

import (
	"errors"
	"fmt"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// Machine returns a machine object for the given spec
func Machine(c *kubermaticv1.Cluster, node *apiv1.Node, dc provider.DatacenterMeta, keys []*kubermaticv1.UserSSHKey) (*clusterv1alpha1.Machine, error) {
	m := clusterv1alpha1.Machine{}

	m.Namespace = metav1.NamespaceSystem
	m.GenerateName = fmt.Sprintf("machine-kubermatic-%s-", c.Name)

	m.Spec.Versions.Kubelet = node.Spec.Versions.Kubelet

	config := providerconfig.Config{}
	config.SSHPublicKeys = make([]string, len(keys))
	for i, key := range keys {
		config.SSHPublicKeys[i] = key.Spec.PublicKey
	}

	var (
		err      error
		cloudExt *runtime.RawExtension
	)
	// Cloud specifics
	switch {
	case node.Spec.Cloud.AWS != nil:
		config.CloudProvider = providerconfig.CloudProviderAWS
		cloudExt, err = getAWSProviderSpec(c, node.Spec, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.Azure != nil:
		config.CloudProvider = providerconfig.CloudProviderAzure
		cloudExt, err = getAzureProviderSpec(c, node.Spec, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.VSphere != nil:
		config.CloudProvider = providerconfig.CloudProviderVsphere

		// We use OverwriteCloudConfig for Vsphere to ensure we always
		// use the credentials passed in via frontend for the cloud-provider
		// functionality
		templateData := resources.NewTemplateData(c, &dc, "", nil, nil, nil, "", "", "", resource.Quantity{}, "", "", false, false, "", nil, false, "", "")
		overwriteCloudConfig, err := cloudconfig.CloudConfig(templateData)
		if err != nil {
			return nil, err
		}
		config.OverwriteCloudConfig = &overwriteCloudConfig

		cloudExt, err = getVSphereProviderSpec(c, node.Spec, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.Openstack != nil:
		config.CloudProvider = providerconfig.CloudProviderOpenstack
		cloudExt, err = getOpenstackProviderSpec(c, node.Spec, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.Hetzner != nil:
		config.CloudProvider = providerconfig.CloudProviderHetzner
		cloudExt, err = getHetznerProviderSpec(c, node.Spec, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.Digitalocean != nil:
		config.CloudProvider = providerconfig.CloudProviderDigitalocean
		cloudExt, err = getDigitaloceanProviderSpec(c, node.Spec, dc)
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
	case node.Spec.OperatingSystem.ContainerLinux != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCoreos
		osExt, err = getCoreosOperatingSystemSpec(node.Spec)
		if err != nil {
			return nil, err
		}
	case node.Spec.OperatingSystem.Ubuntu != nil:
		config.OperatingSystem = providerconfig.OperatingSystemUbuntu
		osExt, err = getUbuntuOperatingSystemSpec(node.Spec)
		if err != nil {
			return nil, err
		}
	case node.Spec.OperatingSystem.CentOS != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCentOS
		osExt, err = getCentOSOperatingSystemSpec(node.Spec)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown OS")
	}
	config.OperatingSystemSpec = *osExt

	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	m.Spec.ProviderConfig.Value = &runtime.RawExtension{Raw: b}

	return &m, nil
}
