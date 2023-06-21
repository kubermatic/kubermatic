/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package cluster

import (
	"fmt"

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

// MutateCreate is an addition to regular defaulting for new clusters.
func MutateCreate(newCluster *kubermaticv1.Cluster, config *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, cloudProvider provider.CloudProvider) *field.Error {
	if newCluster.Spec.Features == nil {
		newCluster.Spec.Features = map[string]bool{}
	}

	// Network policies for Apiserver are deployed by default
	if _, ok := newCluster.Spec.Features[kubermaticv1.ApiserverNetworkPolicy]; !ok {
		newCluster.Spec.Features[kubermaticv1.ApiserverNetworkPolicy] = true
	}

	datacenter, fieldErr := defaulting.DatacenterForClusterSpec(&newCluster.Spec, seed)
	if fieldErr != nil {
		return fieldErr
	}

	// Always enable external CCM for supported providers in new clusters unless the user
	// explicitly disabled the external CCM. For regular users this is not important (most
	// won't disable the CCM), but the ccm-migration e2e tests require to create a cluster
	// without external CCM.
	supported := resources.ExternalCloudControllerFeatureSupported(datacenter, &newCluster.Spec.Cloud, newCluster.Spec.Version, version.NewFromConfiguration(config).GetIncompatibilities()...)
	enabled, configured := newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]

	if supported && (enabled || !configured) {
		newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] = true

		if resources.ExternalCloudControllerClusterName(&newCluster.Spec.Cloud) {
			newCluster.Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] = true
		}

		// For new clusters this can be enabled by default, for existing clusters a
		// more involved migration, outside of the CCM/CSI migration, is required.
		if newCluster.Spec.Cloud.VSphere != nil {
			newCluster.Spec.Features[kubermaticv1.ClusterFeatureVsphereCSIClusterID] = true
		}
	}

	return nil
}

func MutateUpdate(oldCluster, newCluster *kubermaticv1.Cluster, config *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, cloudProvider provider.CloudProvider) *field.Error {
	// If the ExternalCloudProvider feature is enabled for the first time, mark the occasion
	// by adding annotations to keep track of the required migration. This is only required for
	// some providers that have more complex migration procedures; providers like Hetzner for
	// example do not require a CCM migration.
	if v, oldV := newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider],
		oldCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; v && !oldV {
		switch {
		case newCluster.Spec.Cloud.Openstack != nil:
			addCCMCSIMigrationAnnotations(newCluster)
			newCluster.Spec.Cloud.Openstack.UseOctavia = pointer.Bool(true)

		case newCluster.Spec.Cloud.VSphere != nil:
			addCCMCSIMigrationAnnotations(newCluster)

		case newCluster.Spec.Cloud.Azure != nil:
			addCCMCSIMigrationAnnotations(newCluster)

		case newCluster.Spec.Cloud.AWS != nil:
			addCCMCSIMigrationAnnotations(newCluster)
		}

		if resources.ExternalCloudControllerClusterName(&newCluster.Spec.Cloud) {
			newCluster.Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] = true
		}
	}

	// For KubeVirt, we want to mutate and always have ClusterFeatureCCMClusterName = true
	// It's not handled by the previous loop for the migration 2.21 to 2.22
	// as ExternalCloudProvider feature not is set for the first time.
	if newCluster.Spec.Cloud.Kubevirt != nil {
		newCluster.Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] = true
	}

	// just because spec.Version might say 1.23 doesn't say that the cluster is already on 1.23,
	// so for all feature toggles and migrations we should base this on the actual, current apiserver
	curVersion := newCluster.Status.Versions.ControlPlane
	if curVersion.String() == "" {
		curVersion = newCluster.Spec.Version
	}

	if newCluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCanal {
		// This part handles Canal version upgrade for clusters with Kubernetes version 1.25 and higher,
		// where the minimal Canal version is v3.23. We need to check the target cluster version here to ensure,
		// the upgrade happens at the same time.
		cniVersion, err := semverlib.NewVersion(newCluster.Spec.CNIPlugin.Version)
		if err != nil {
			return field.Invalid(field.NewPath("spec", "cniPlugin", "version"), newCluster.Spec.CNIPlugin.Version, err.Error())
		}
		lowerThan323, err := semverlib.NewConstraint("< 3.23")
		if err != nil {
			return field.InternalError(nil, fmt.Errorf("semver constraint parsing failed: %w", err))
		}
		equalOrHigherThan125, err := semverlib.NewConstraint(">= 1.25")
		if err != nil {
			return field.InternalError(nil, fmt.Errorf("semver constraint parsing failed: %w", err))
		}
		if lowerThan323.Check(cniVersion) && newCluster.Spec.Version.String() != "" && equalOrHigherThan125.Check(newCluster.Spec.Version.Semver()) {
			newCluster.Spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCanal,
				Version: "v3.23",
			}
		}
	}

	return nil
}

func addCCMCSIMigrationAnnotations(cluster *kubermaticv1.Cluster) {
	if cluster.ObjectMeta.Annotations == nil {
		cluster.ObjectMeta.Annotations = map[string]string{}
	}

	cluster.ObjectMeta.Annotations[kubermaticv1.CCMMigrationNeededAnnotation] = ""
	cluster.ObjectMeta.Annotations[kubermaticv1.CSIMigrationNeededAnnotation] = ""
}
