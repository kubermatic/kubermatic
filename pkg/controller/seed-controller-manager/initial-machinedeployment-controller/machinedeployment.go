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

package initialmachinedeployment

import (
	"errors"
	"fmt"

	semverlib "github.com/Masterminds/semver/v3"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/machine"
	"k8c.io/kubermatic/v2/pkg/validation/nodeupdate"
	osmresources "k8c.io/operating-system-manager/pkg/controllers/osc/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

// CompleteMachineDeployment returns a Machine Deployment object for the given Node Deployment spec.
func CompleteMachineDeployment(md *clusterv1alpha1.MachineDeployment, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.Datacenter, keys []*kubermaticv1.UserSSHKey) (*clusterv1alpha1.MachineDeployment, error) {
	md.Namespace = metav1.NamespaceSystem
	if md.Name == "" {
		md.GenerateName = fmt.Sprintf("%s-worker-", cluster.Name)
	}

	config, err := providerconfig.GetConfig(md.Spec.Template.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}

	if err := completeCloudProviderSpec(config, cluster, datacenter, keys); err != nil {
		return nil, err
	}

	if md.Annotations == nil {
		md.Annotations = make(map[string]string)
	}

	osp := datacenter.Spec.DefaultOperatingSystemProfiles[config.OperatingSystem]
	if osp != "" {
		md.Annotations[osmresources.MachineDeploymentOSPAnnotation] = osp
	}

	md.Finalizers = []string{metav1.FinalizerDeleteDependents}

	// inject a known, good set of labels+matchLabels to ensure the MD makes sense
	md.Spec.Selector.MatchLabels = map[string]string{
		"machine": fmt.Sprintf("md-%s-%s", cluster.Name, rand.String(10)),
	}
	md.Spec.Template.Labels = md.Spec.Selector.MatchLabels
	md.Spec.Template.Spec.Labels = md.Spec.Template.Labels

	// Do not confuse the convenience labels with the labels inside the
	// providerSpec, which ultimately get applied on the cloud provider resources.
	// That's why these labels do not depend on the given cloud provider.
	md.Spec.Template.Spec.Labels["system/cluster"] = cluster.Name
	projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		md.Spec.Template.Spec.Labels["system/project"] = projectID
	}

	// ensure a version is set; if one is set already, Validate() took care to ensure
	// it's compatible
	if md.Spec.Template.Spec.Versions.Kubelet == "" {
		md.Spec.Template.Spec.Versions.Kubelet = cluster.Spec.Version.String()
	}

	// Deprecated: This is not supported for 1.24 and higher and is blocked by
	// Validate for 1.24+. Can be removed once 1.23 support is dropped.
	if md.Spec.Template.Spec.ConfigSource != nil && md.Spec.Template.Spec.ConfigSource.ConfigMap != nil {
		kubeletVersion, err := semverlib.NewVersion(md.Spec.Template.Spec.Versions.Kubelet)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kubelet version %q: %w", md.Spec.Template.Spec.Versions.Kubelet, err)
		}

		md.Spec.Template.Spec.ConfigSource.ConfigMap = &corev1.ConfigMapNodeConfigSource{
			Namespace:        metav1.NamespaceSystem,
			Name:             fmt.Sprintf("kubelet-config-%d.%d", kubeletVersion.Major(), kubeletVersion.Minor()),
			KubeletConfigKey: "kubelet",
		}
	}

	if len(cluster.Spec.MachineNetworks) > 0 {
		if md.Spec.Template.Annotations == nil {
			md.Spec.Template.Annotations = make(map[string]string)
		}

		// TODO: Rename this finalizer to not contain the word "kubermatic" (For whitelabeling purpose)
		md.Spec.Template.Annotations["machine-controller.kubermatic.io/initializers"] = "ipam"
	}

	providerSpec, err := machine.CreateProviderSpec(config)
	if err != nil {
		return nil, err
	}
	md.Spec.Template.Spec.ProviderSpec = *providerSpec

	return md, nil
}

// completeCloudProviderSpec decodes the cloud provider spec, fills in missing values based on the datacenter and cluster,
// and then re-encodes the cloud provider spec into the provider config.
func completeCloudProviderSpec(config *providerconfig.Config, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.Datacenter, keys []*kubermaticv1.UserSSHKey) error {
	// determine KKP-name for the cloud provider
	kkpCloudProvider, err := machine.KubermaticProviderType(config.CloudProvider)
	if err != nil {
		return err
	}

	// decode the cloud provider spec
	cloudProviderSpec, err := machine.DecodeCloudProviderSpec(kkpCloudProvider, *config)
	if err != nil {
		return err
	}

	// fill in missing values
	cloudProviderSpec, err = machine.CompleteCloudProviderSpec(cloudProviderSpec, kkpCloudProvider, cluster, datacenter, config.OperatingSystem)
	if err != nil {
		return err
	}

	// re-encode the spec back into the config
	config.CloudProviderSpec, err = machine.EncodeAsRawExtension(cloudProviderSpec)
	if err != nil {
		return err
	}

	// assign list of SSH keys currently assigned to the cluster
	config.SSHPublicKeys = []string{}
	for _, key := range keys {
		config.SSHPublicKeys = append(config.SSHPublicKeys, key.Spec.PublicKey)
	}

	return nil
}

// Validate if the node deployment structure fulfills certain requirements. It returns node deployment with updated
// kubelet version if it wasn't specified.
func ValidateMachineDeployment(md *clusterv1alpha1.MachineDeployment, controlPlaneVersion *semverlib.Version) error {
	var (
		kubeletVersion = controlPlaneVersion
		err            error
	)

	if kubelet := md.Spec.Template.Spec.Versions.Kubelet; kubelet != "" {
		kubeletVersion, err = semverlib.NewVersion(kubelet)
		if err != nil {
			return fmt.Errorf("failed to parse kubelet version %q: %w", kubelet, err)
		}

		if err = nodeupdate.EnsureVersionCompatible(controlPlaneVersion, kubeletVersion); err != nil {
			return err
		}
	}

	constraint124, err := semverlib.NewConstraint(">= 1.24")
	if err != nil {
		return fmt.Errorf("failed to parse 1.24 constraint: %w", err)
	}

	if md.Spec.Template.Spec.ConfigSource != nil && constraint124.Check(kubeletVersion) {
		return errors.New("dynamic config cannot be configured for Kubernetes 1.24 or higher")
	}

	return nil
}
