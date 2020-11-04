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

package machine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig"
	"k8c.io/kubermatic/v2/pkg/validation"
	"k8c.io/kubermatic/v2/pkg/validation/nodeupdate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Deployment returns a Machine Deployment object for the given Node Deployment spec.
func Deployment(c *kubermaticv1.Cluster, nd *apiv1.NodeDeployment, dc *kubermaticv1.Datacenter, keys []*kubermaticv1.UserSSHKey, data resources.CredentialsData) (*clusterv1alpha1.MachineDeployment, error) {
	md := &clusterv1alpha1.MachineDeployment{}

	if nd.Name != "" {
		md.Name = nd.Name
	} else {
		// GenerateName can be set only if Name is empty to avoid confusing error:
		// https://github.com/kubernetes/kubernetes/issues/32220
		md.GenerateName = fmt.Sprintf("%s-worker-", c.Spec.HumanReadableName)
	}

	md.Namespace = metav1.NamespaceSystem
	md.Finalizers = []string{metav1.FinalizerDeleteDependents}

	md.Spec.Selector.MatchLabels = map[string]string{
		"machine": fmt.Sprintf("md-%s-%s", c.Name, rand.String(10)),
	}
	md.Spec.Template.Labels = md.Spec.Selector.MatchLabels
	md.Spec.Template.Spec.Labels = nd.Spec.Template.Labels
	if md.Spec.Template.Spec.Labels == nil {
		md.Spec.Template.Spec.Labels = make(map[string]string)
	}
	md.Spec.Template.Spec.Labels["system/cluster"] = c.Name
	projectID, ok := c.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		md.Spec.Template.Spec.Labels["system/project"] = projectID
	}

	var taints []corev1.Taint
	for _, taint := range nd.Spec.Template.Taints {
		taints = append(taints, corev1.Taint{
			Value:  taint.Value,
			Key:    taint.Key,
			Effect: corev1.TaintEffect(taint.Effect),
		})
	}
	md.Spec.Template.Spec.Taints = taints

	// Create a copy to avoid changing the ND when changing the MD
	replicas := nd.Spec.Replicas
	md.Spec.Replicas = &replicas

	md.Spec.Template.Spec.Versions.Kubelet = nd.Spec.Template.Versions.Kubelet

	if nd.Spec.DynamicConfig != nil && *nd.Spec.DynamicConfig {
		kubeletVersion, err := semver.NewVersion(nd.Spec.Template.Versions.Kubelet)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kubelet version: %v", err)
		}

		md.Spec.Template.Spec.ConfigSource = &corev1.NodeConfigSource{
			ConfigMap: &corev1.ConfigMapNodeConfigSource{
				Namespace:        "kube-system",
				Name:             fmt.Sprintf("kubelet-config-%d.%d", kubeletVersion.Major(), kubeletVersion.Minor()),
				KubeletConfigKey: "kubelet",
			},
		}
	}

	if len(c.Spec.MachineNetworks) > 0 {
		// TODO(mrIncompetent): Rename this finalizer to not contain the word "kubermatic" (For whitelabeling purpose)
		md.Spec.Template.Annotations = map[string]string{
			"machine-controller.kubermatic.io/initializers": "ipam",
		}
	}

	if nd.Spec.Paused != nil {
		md.Spec.Paused = *nd.Spec.Paused
	}

	config, err := getProviderConfig(c, nd, dc, keys, data)
	if err != nil {
		return nil, err
	}

	err = getProviderOS(config, nd)
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

func getProviderConfig(c *kubermaticv1.Cluster, nd *apiv1.NodeDeployment, dc *kubermaticv1.Datacenter, keys []*kubermaticv1.UserSSHKey, data resources.CredentialsData) (*providerconfig.Config, error) {
	config := providerconfig.Config{}
	config.SSHPublicKeys = make([]string, len(keys))
	for i, key := range keys {
		config.SSHPublicKeys[i] = key.Spec.PublicKey
	}

	var (
		cloudExt *runtime.RawExtension
		err      error
	)

	credentials, err := resources.GetCredentials(data)
	if err != nil {
		return nil, err
	}

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
		overwriteCloudConfig, err := cloudconfig.CloudConfig(c, dc, credentials)
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
		if err := validation.ValidateCreateNodeSpec(c, &nd.Spec.Template, dc); err != nil {
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
	case nd.Spec.Template.Cloud.Packet != nil:
		config.CloudProvider = providerconfig.CloudProviderPacket
		cloudExt, err = getPacketProviderSpec(c, nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.Cloud.GCP != nil:
		config.CloudProvider = providerconfig.CloudProviderGoogle
		cloudExt, err = getGCPProviderSpec(c, nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.Cloud.Kubevirt != nil:
		config.CloudProvider = providerconfig.CloudProviderKubeVirt
		cloudExt, err = getKubevirtProviderSpec(nd.Spec.Template)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.Cloud.Alibaba != nil:
		config.CloudProvider = providerconfig.CloudProviderAlibaba
		cloudExt, err = getAlibabaProviderSpec(c, nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	case nd.Spec.Template.Cloud.Anexia != nil:
		config.CloudProvider = providerconfig.CloudProviderAnexia
		cloudExt, err = getAnexiaProviderSpec(nd.Spec.Template, dc)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown cloud provider")
	}
	config.CloudProviderSpec = *cloudExt

	return &config, nil
}

func getProviderOS(config *providerconfig.Config, nd *apiv1.NodeDeployment) error {
	var (
		err   error
		osExt *runtime.RawExtension
	)

	// OS specifics
	switch {
	case nd.Spec.Template.OperatingSystem.ContainerLinux != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCoreos
		osExt, err = getCoreosOperatingSystemSpec(nd.Spec.Template)
		if err != nil {
			return err
		}
	case nd.Spec.Template.OperatingSystem.Ubuntu != nil:
		config.OperatingSystem = providerconfig.OperatingSystemUbuntu
		osExt, err = getUbuntuOperatingSystemSpec(nd.Spec.Template)
		if err != nil {
			return err
		}
	case nd.Spec.Template.OperatingSystem.CentOS != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCentOS
		osExt, err = getCentOSOperatingSystemSpec(nd.Spec.Template)
		if err != nil {
			return err
		}
	case nd.Spec.Template.OperatingSystem.SLES != nil:
		config.OperatingSystem = providerconfig.OperatingSystemSLES
		osExt, err = getSLESOperatingSystemSpec(nd.Spec.Template)
		if err != nil {
			return err
		}
	case nd.Spec.Template.OperatingSystem.RHEL != nil:
		config.OperatingSystem = providerconfig.OperatingSystemRHEL
		osExt, err = getRHELOperatingSystemSpec(nd.Spec.Template)
		if err != nil {
			return err
		}
	case nd.Spec.Template.OperatingSystem.Flatcar != nil:
		config.OperatingSystem = providerconfig.OperatingSystemFlatcar
		osExt, err = getFlatcarOperatingSystemSpec(nd.Spec.Template)
		if err != nil {
			return err
		}
	default:
		return errors.New("no machine os was provided")
	}
	config.OperatingSystemSpec = *osExt

	return nil
}

// Validate if the node deployment structure fulfills certain requirements. It returns node deployment with updated
// kubelet version if it wasn't specified.
func Validate(nd *apiv1.NodeDeployment, controlPlaneVersion *semver.Version) (*apiv1.NodeDeployment, error) {
	if nd.Spec.Template.Cloud.Openstack == nil &&
		nd.Spec.Template.Cloud.Digitalocean == nil &&
		nd.Spec.Template.Cloud.AWS == nil &&
		nd.Spec.Template.Cloud.Hetzner == nil &&
		nd.Spec.Template.Cloud.VSphere == nil &&
		nd.Spec.Template.Cloud.Azure == nil &&
		nd.Spec.Template.Cloud.Packet == nil &&
		nd.Spec.Template.Cloud.GCP == nil &&
		nd.Spec.Template.Cloud.Kubevirt == nil &&
		nd.Spec.Template.Cloud.Alibaba == nil &&
		nd.Spec.Template.Cloud.Anexia == nil {
		return nil, fmt.Errorf("node deployment needs to have cloud provider data")
	}

	if nd.Spec.Template.Versions.Kubelet != "" {
		kubeletVersion, err := semver.NewVersion(nd.Spec.Template.Versions.Kubelet)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kubelet version: %v", err)
		}

		if err = nodeupdate.EnsureVersionCompatible(controlPlaneVersion, kubeletVersion); err != nil {
			return nil, err
		}

		nd.Spec.Template.Versions.Kubelet = kubeletVersion.String()
	} else {
		nd.Spec.Template.Versions.Kubelet = controlPlaneVersion.String()
	}

	// The default
	allowedTaintEffects := sets.NewString(
		string(corev1.TaintEffectNoExecute),
		string(corev1.TaintEffectNoSchedule),
		string(corev1.TaintEffectPreferNoSchedule),
	)
	for _, taint := range nd.Spec.Template.Taints {
		if taint.Key == "" {
			return nil, errors.New("taint key must be set")
		}
		if taint.Value == "" {
			return nil, errors.New("taint value must be set")
		}
		if !allowedTaintEffects.Has(taint.Effect) {
			return nil, fmt.Errorf("taint effect '%s' not allowed. Allowed: %s", taint.Effect, strings.Join(allowedTaintEffects.List(), ", "))
		}
	}

	return nd, nil
}
