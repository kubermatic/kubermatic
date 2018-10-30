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

// MachineDeployment returns a Machine Deployment object for the given Node Set spec.
func MachineDeployment(c *kubermaticv1.Cluster, ns *apiv1.NodeSet, dc provider.DatacenterMeta, keys []*kubermaticv1.UserSSHKey) (*clusterv1alpha1.MachineDeployment, error) {
	md := clusterv1alpha1.MachineDeployment{}

	md.Name = fmt.Sprintf("machinedeployment-%s", ns.Name)
	md.Namespace = metav1.NamespaceSystem

	md.Spec.Replicas = ns.Spec.Replicas
	md.Spec.Selector = ns.Spec.Selector
	md.Spec.Template.Spec.Versions.Kubelet = ns.Spec.Template.Versions.Kubelet
	md.Spec.Strategy = ns.Spec.Strategy
	md.Spec.MinReadySeconds = ns.Spec.MinReadySeconds
	md.Spec.RevisionHistoryLimit = ns.Spec.RevisionHistoryLimit
	md.Spec.Paused = ns.Spec.Paused
	md.Spec.ProgressDeadlineSeconds = ns.Spec.ProgressDeadlineSeconds

	// MachineDeploymentSpec's label selector must match the machine template's labels as docs say.
	md.Spec.Template.ObjectMeta.Labels = md.Spec.Selector.MatchLabels

	config := providerconfig.Config{}
	config.SSHPublicKeys = make([]string, len(keys))
	for i, key := range keys {
		config.SSHPublicKeys[i] = key.Spec.PublicKey
	}

	var err error
	var cloudExt *runtime.RawExtension

	// Cloud specifics
	switch {
	case ns.Spec.Template.Cloud.AWS != nil:
		config.CloudProvider = providerconfig.CloudProviderAWS
		cloudExt, err = getAWSProviderSpec(c, ns.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case ns.Spec.Template.Cloud.Azure != nil:
		config.CloudProvider = providerconfig.CloudProviderAzure
		cloudExt, err = getAzureProviderSpec(c, ns.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case ns.Spec.Template.Cloud.VSphere != nil:
		config.CloudProvider = providerconfig.CloudProviderVsphere

		// We use OverwriteCloudConfig for Vsphere to ensure we always
		// use the credentials passed in via frontend for the cloud-provider
		// functionality
		templateData := resources.NewTemplateData(c, &dc, "", nil, nil, nil, "", "", "", resource.Quantity{}, "", "", false, false, "", nil)
		overwriteCloudConfig, err := cloudconfig.CloudConfig(templateData)
		if err != nil {
			return nil, err
		}
		config.OverwriteCloudConfig = &overwriteCloudConfig

		cloudExt, err = getVSphereProviderSpec(c, ns.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case ns.Spec.Template.Cloud.Openstack != nil:
		config.CloudProvider = providerconfig.CloudProviderOpenstack
		cloudExt, err = getOpenstackProviderSpec(c, ns.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case ns.Spec.Template.Cloud.Hetzner != nil:
		config.CloudProvider = providerconfig.CloudProviderHetzner
		cloudExt, err = getHetznerProviderSpec(c, ns.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case ns.Spec.Template.Cloud.Digitalocean != nil:
		config.CloudProvider = providerconfig.CloudProviderDigitalocean
		cloudExt, err = getDigitaloceanProviderSpec(c, ns.Spec.Template, dc)
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
	case ns.Spec.Template.OperatingSystem.ContainerLinux != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCoreos
		osExt, err = getCoreosOperatingSystemSpec(ns.Spec.Template)
		if err != nil {
			return nil, err
		}
	case ns.Spec.Template.OperatingSystem.Ubuntu != nil:
		config.OperatingSystem = providerconfig.OperatingSystemUbuntu
		osExt, err = getUbuntuOperatingSystemSpec(ns.Spec.Template)
		if err != nil {
			return nil, err
		}
	case ns.Spec.Template.OperatingSystem.CentOS != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCentOS
		osExt, err = getCentOSOperatingSystemSpec(ns.Spec.Template)
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

	md.Spec.Template.Spec.ProviderConfig.Value = &runtime.RawExtension{Raw: b}

	return &md, nil
}
