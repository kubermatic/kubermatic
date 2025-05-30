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
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/features"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version"
	clusterversion "k8c.io/kubermatic/v2/pkg/version/cluster"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	kubenetutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var (
	// ErrCloudChangeNotAllowed describes that it is not allowed to change the cloud provider.
	ErrCloudChangeNotAllowed  = errors.New("not allowed to change the cloud provider")
	azureLoadBalancerSKUTypes = sets.New("", string(kubermaticv1.AzureStandardLBSKU), string(kubermaticv1.AzureBasicLBSKU))

	errPodSecurityPolicyAdmissionPluginWithVersionGte125 = errors.New("admission plugin \"PodSecurityPolicy\" is not supported in Kubernetes v1.25 and later")
)

const (
	// UnsafeCNIUpgradeLabel allows unsafe CNI version upgrade (difference in versions more than one minor version).
	UnsafeCNIUpgradeLabel = "unsafe-cni-upgrade"
	// UnsafeCNIMigrationLabel allows unsafe CNI type migration.
	UnsafeCNIMigrationLabel = "unsafe-cni-migration"
	// UnsafeExposeStrategyMigrationLabel allows unsafe expose strategy migration.
	UnsafeExposeStrategyMigrationLabel = "unsafe-expose-strategy-migration"

	// MaxClusterNameLength is the maximum allowed length for cluster names.
	// This is restricted by the many uses of cluster names, from embedding
	// them in namespace names (and prefixing them) to using them in role
	// names (when using AWS).
	// AWS role names have a max length of 64 characters, "kubernetes-" and
	// "-control-plane" being added by KKP. This leaves 39 usable characters
	// and to give some wiggle room, we define the max length to be 36.
	MaxClusterNameLength = 36

	// EARKeyLength is required key length for encryption at rest.
	EARKeyLength = 32

	podSecurityPolicyAdmissionPluginName = "PodSecurityPolicy"
)

// ValidateClusterSpec validates the given cluster spec. If this is not called from within another validation
// routine, parentFieldPath can be nil.
//
//nolint:gocyclo // there just needs to be a place that validates the spec and the spec is simply large; splitting this function into smaller ones would not help readability
func ValidateClusterSpec(spec *kubermaticv1.ClusterSpec, dc *kubermaticv1.Datacenter, kubeLBSeedSettings *kubermaticv1.KubeLBSeedSettings, enabledFeatures features.FeatureGate, versionManager *version.Manager, currentVersion *semver.Semver, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.HumanReadableName == "" {
		allErrs = append(allErrs, field.Required(parentFieldPath.Child("humanReadableName"), "no name specified"))
	}

	if err := ValidateVersion(spec, versionManager, currentVersion, parentFieldPath.Child("version")); err != nil {
		allErrs = append(allErrs, err)
	}

	// Validate if container runtime is valid for this cluster (in particular this checks for docker support).
	if err := ValidateContainerRuntime(spec); err != nil {
		allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("containerRuntime"), spec.ContainerRuntime, fmt.Sprintf("failed to validate container runtime: %s", err)))
	}

	if !kubermaticv1.AllExposeStrategies.Has(spec.ExposeStrategy) {
		allErrs = append(allErrs, field.NotSupported(parentFieldPath.Child("exposeStrategy"), spec.ExposeStrategy, kubermaticv1.AllExposeStrategies.Items()))
	}

	// Validate APIServerAllowedIPRanges for LoadBalancer expose strategy
	if spec.ExposeStrategy != kubermaticv1.ExposeStrategyLoadBalancer && spec.APIServerAllowedIPRanges != nil && len(spec.APIServerAllowedIPRanges.CIDRBlocks) > 0 {
		allErrs = append(allErrs, field.Forbidden(parentFieldPath.Child("APIServerAllowedIPRanges"), "Access control for API server is supported only for LoadBalancer expose strategy"))
	}

	// Validate TunnelingAgentIP for Tunneling Expose strategy
	if spec.ExposeStrategy != kubermaticv1.ExposeStrategyTunneling && spec.ClusterNetwork.TunnelingAgentIP != "" {
		allErrs = append(allErrs, field.Forbidden(parentFieldPath.Child("TunnelingAgentIP"), "Tunneling agent IP can be configured only for Tunneling Expose strategy"))
	}

	// External CCM is not supported for all providers and all Kubernetes versions.
	if spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
		if !resources.ExternalCloudControllerFeatureSupported(dc, &spec.Cloud, spec.Version, versionManager.GetIncompatibilities()...) {
			allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("features").Key(kubermaticv1.ClusterFeatureExternalCloudProvider), true, "external cloud-controller-manager is not supported for this cluster / provider combination"))
		}
	}

	if spec.CNIPlugin != nil {
		if !cni.GetSupportedCNIPlugins().Has(spec.CNIPlugin.Type.String()) {
			allErrs = append(allErrs, field.NotSupported(parentFieldPath.Child("cniPlugin", "type"), spec.CNIPlugin.Type.String(), sets.List(cni.GetSupportedCNIPlugins())))
		} else if versions, err := cni.GetAllowedCNIPluginVersions(spec.CNIPlugin.Type); err != nil || !versions.Has(spec.CNIPlugin.Version) {
			allErrs = append(allErrs, field.NotSupported(parentFieldPath.Child("cniPlugin", "version"), spec.CNIPlugin.Version, sets.List(versions)))
		}

		// Dual-stack is not supported on Canal < v3.22
		if spec.ClusterNetwork.IPFamily == kubermaticv1.IPFamilyDualStack && spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCanal {
			gte322Constraint, _ := semverlib.NewConstraint(">= 3.22")
			cniVer, _ := semverlib.NewVersion(spec.CNIPlugin.Version)
			if cniVer != nil && !gte322Constraint.Check(cniVer) {
				allErrs = append(allErrs, field.Forbidden(parentFieldPath.Child("cniPlugin"), "dual-stack not allowed on Canal CNI version lower than 3.22"))
			}
		}
	}

	allErrs = append(allErrs, ValidateLeaderElectionSettings(&spec.ComponentsOverride.ControllerManager.LeaderElectionSettings, parentFieldPath.Child("componentsOverride", "controllerManager", "leaderElection"))...)
	allErrs = append(allErrs, ValidateLeaderElectionSettings(&spec.ComponentsOverride.Scheduler.LeaderElectionSettings, parentFieldPath.Child("componentsOverride", "scheduler", "leaderElection"))...)

	externalCCM := false
	if val, ok := spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; ok {
		externalCCM = val
	}

	// general cloud spec logic
	if errs := ValidateCloudSpec(spec.Cloud, dc, spec.ClusterNetwork.IPFamily, parentFieldPath.Child("cloud"), externalCCM); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if errs := validateMachineNetworksFromClusterSpec(spec, parentFieldPath); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if errs := ValidateClusterNetworkConfig(&spec.ClusterNetwork, dc, spec.CNIPlugin, parentFieldPath.Child("networkConfig")); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	portRangeFld := field.NewPath("componentsOverride", "apiserver", "nodePortRange")
	if err := ValidateNodePortRange(spec.ComponentsOverride.Apiserver.NodePortRange, portRangeFld); err != nil {
		allErrs = append(allErrs, err)
	}

	if errs := validateEncryptionConfiguration(spec, parentFieldPath.Child("encryptionConfiguration")); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	// KubeLB can only be enabled on the cluster when
	// a) It's either enforced or enabled at the datacenter level.
	// b) It's enabled for all datacenters at the seed level.
	if spec.IsKubeLBEnabled() && !kubeLBSeedSettings.EnableForAllDatacenters && (dc.Spec.KubeLB == nil || (!dc.Spec.KubeLB.Enabled && !dc.Spec.KubeLB.Enforced)) {
		allErrs = append(allErrs, field.Forbidden(parentFieldPath.Child("kubeLB"), "KubeLB is not enabled on this datacenter"))
	}

	if err := validateCoreDNSReplicas(spec, parentFieldPath); err != nil {
		allErrs = append(allErrs, err)
	}

	if spec.IsAuthorizationConfigurationFileEnabled() && spec.IsWebhookAuthorizationEnabled() {
		allErrs = append(allErrs, field.Forbidden(parentFieldPath.Child("authorizationConfig"), "AuthorizationWebhookConfiguration and AuthorizationConfigurationFile cannot be used together"))
	}
	return allErrs
}

func ValidateNewClusterSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec, dc *kubermaticv1.Datacenter, seed *kubermaticv1.Seed, cloudProvider provider.CloudProvider, versionManager *version.Manager, enabledFeatures features.FeatureGate, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if errs := ValidateClusterSpec(spec, dc, seed.Spec.KubeLB, enabledFeatures, versionManager, nil, parentFieldPath); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	// The cloudProvider is built based on the *datacenter*, but does not necessarily match the CloudSpec.
	// To prevent a cloud provider to accidentally access nil fields, we check here again that the datacenter
	// type, providerName and provider data all match before calling the provider's validation logic.
	// No error needs to be reported here if there's a mismatch, as ValidateClusterSpec() already reported one.
	if cloudProvider != nil && validateDatacenterMatchesProvider(spec.Cloud, dc) == nil {
		if err := cloudProvider.ValidateCloudSpec(ctx, spec.Cloud); err != nil {
			// Just using spec.Cloud for the error leads to a Go-representation of the struct being printed in
			// the error message, which looks awful an is not helpful. However any other encoding (e.g. JSON)
			// could lead to us leaking credentials that were given in the CloudSpec, so to be safe, we never
			// reveal the CloudSpec in an error.
			allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("cloud"), "<redacted>", err.Error()))
		}
	}

	return allErrs
}

// ValidateClusterUpdate validates the new cluster and if no forbidden changes were attempted.
func ValidateClusterUpdate(ctx context.Context, newCluster, oldCluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, seed *kubermaticv1.Seed, cloudProvider provider.CloudProvider, versionManager *version.Manager, features features.FeatureGate) field.ErrorList {
	specPath := field.NewPath("spec")
	allErrs := field.ErrorList{}

	// perform general basic checks on the new cluster spec
	if errs := ValidateClusterSpec(&newCluster.Spec, dc, seed.Spec.KubeLB, features, versionManager, &oldCluster.Spec.Version, specPath); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if cloudProvider != nil {
		if err := cloudProvider.ValidateCloudSpecUpdate(ctx, oldCluster.Spec.Cloud, newCluster.Spec.Cloud); err != nil {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("cloud"), err.Error()))
		}
	}

	// ensure neither cloud nor datacenter were changed
	if err := ValidateCloudChange(newCluster.Spec.Cloud, oldCluster.Spec.Cloud); err != nil {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("cloud"), err.Error()))
	}

	if address := newCluster.Status.Address; address.AdminToken != "" {
		if err := kuberneteshelper.ValidateKubernetesToken(address.AdminToken); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("address", "adminToken"), address.AdminToken, err.Error()))
		}
	}

	// Validate ExternalCloudProvider feature flag immutability.
	// Once the feature flag is enabled, it must not be disabled.
	if vOld, v := oldCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider],
		newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; vOld && !v {
		allErrs = append(allErrs, field.Invalid(specPath.Child("features").Key(kubermaticv1.ClusterFeatureExternalCloudProvider), v, fmt.Sprintf("feature gate %q cannot be disabled once it's enabled", kubermaticv1.ClusterFeatureExternalCloudProvider)))
	}

	// Validate EtcdLauncher feature flag immutability.
	// Once the feature flag is enabled, it must not be disabled.
	if vOld, v := oldCluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher],
		newCluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher]; vOld && !v {
		allErrs = append(allErrs, field.Invalid(specPath.Child("features").Key(kubermaticv1.ClusterFeatureEtcdLauncher), v, fmt.Sprintf("feature gate %q cannot be disabled once it's enabled", kubermaticv1.ClusterFeatureEtcdLauncher)))
	}

	// Validate datacenter setting for disabling CSI driver installation if true, is not being over-written.
	if dc.Spec.DisableCSIDriver && !newCluster.Spec.DisableCSIDriver {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("DisableCSIDriver"), "CSI driver installation is disabled on the datacenter, can't be enabled on cluster"))
	}

	// Validate CSI addon is not being disabled while CSI driver(s) created by it is still in use.
	if !oldCluster.Spec.DisableCSIDriver && newCluster.Spec.DisableCSIDriver {
		csiAddonCondition, ok := oldCluster.Status.Conditions[kubermaticv1.ClusterConditionCSIAddonInUse]
		if ok {
			if csiAddonCondition.Status != corev1.ConditionFalse {
				allErrs = append(allErrs, field.Forbidden(specPath.Child("DisableCSIDriver"), "CSI addon is in use or unknown state "+oldCluster.Status.Conditions[kubermaticv1.ClusterConditionCSIAddonInUse].Reason))
			}
		}
	}

	if oldCluster.Spec.ExposeStrategy != "" {
		if _, ok := newCluster.Labels[UnsafeExposeStrategyMigrationLabel]; !ok { // allow expose strategy migration if labeled explicitly
			allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
				newCluster.Spec.ExposeStrategy,
				oldCluster.Spec.ExposeStrategy,
				specPath.Child("exposeStrategy"),
			)...)
		}
	}

	if oldCluster.Spec.ComponentsOverride.Apiserver.NodePortRange != "" {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			newCluster.Spec.ComponentsOverride.Apiserver.NodePortRange,
			oldCluster.Spec.ComponentsOverride.Apiserver.NodePortRange,
			specPath.Child("componentsOverride", "apiserver", "nodePortRange"),
		)...)
	}

	if oldCluster.Spec.EnableUserSSHKeyAgent != nil {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			newCluster.Spec.EnableUserSSHKeyAgent,
			oldCluster.Spec.EnableUserSSHKeyAgent,
			specPath.Child("enableUserSSHKeyAgent"),
		)...)
	} else if newCluster.Spec.EnableUserSSHKeyAgent != nil && !*newCluster.Spec.EnableUserSSHKeyAgent {
		path := field.NewPath("cluster", "spec", "enableUserSSHKeyAgent")
		allErrs = append(allErrs, field.Invalid(path, *newCluster.Spec.EnableUserSSHKeyAgent, "UserSSHKey agent is enabled by default for user clusters created prior KKP 2.16 version"))
	}
	allErrs = append(allErrs, validateClusterNetworkingConfigUpdateImmutability(&newCluster.Spec.ClusterNetwork, &oldCluster.Spec.ClusterNetwork, newCluster.Labels, specPath.Child("clusterNetwork"))...)

	// even though ErrorList later in ToAggregate() will filter out nil errors, it does so by
	// stringifying them. A field.Error that is nil will panic when doing so, so one cannot simply
	// append a nil *field.Error to allErrs.
	if err := validateCNIUpdate(newCluster.Spec.CNIPlugin, oldCluster.Spec.CNIPlugin, newCluster.Labels, newCluster.Spec.Version); err != nil {
		allErrs = append(allErrs, err)
	}

	if errs := validateEncryptionUpdate(newCluster, oldCluster); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if !equality.Semantic.DeepEqual(newCluster.TypeMeta, oldCluster.TypeMeta) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("typeMeta"), "type meta cannot be changed"))
	}

	return allErrs
}

func ValidateVersion(spec *kubermaticv1.ClusterSpec, versionManager *version.Manager, currentVersion *semver.Semver, fldPath *field.Path) *field.Error {
	if spec.Version.Semver() == nil || spec.Version.String() == "" {
		return field.Required(fldPath, "version is required but was not specified")
	}

	var (
		validVersions []string
		versionValid  bool
		versions      []*version.Version
		err           error
	)

	conditions := clusterversion.GetVersionConditions(spec)

	// if a current version is passed, we are doing a version upgrade.
	if currentVersion != nil {
		// we return early here so we don't reject ClusterSpecs if the version hasn't changed.
		if spec.Version.Equal(currentVersion) {
			return nil
		}

		versions, err = versionManager.GetPossibleUpdates(currentVersion.String(), kubermaticv1.ProviderType(spec.Cloud.ProviderName), conditions...)
		if err != nil {
			return field.InternalError(fldPath, fmt.Errorf("failed to get available version updates: %w", err))
		}
	} else {
		versions, err = versionManager.GetVersionsForProvider(kubermaticv1.ProviderType(spec.Cloud.ProviderName), conditions...)
		if err != nil {
			return field.InternalError(fldPath, fmt.Errorf("failed to get available versions: %w", err))
		}
	}

	for _, availableVersion := range versions {
		validVersions = append(validVersions, availableVersion.Version.String())
		if spec.Version.Semver().Equal(availableVersion.Version) {
			versionValid = true
			break
		}
	}

	if !versionValid {
		return field.NotSupported(fldPath, spec.Version.String(), validVersions)
	}

	if err := validatePodSecurityPolicyAdmissionPluginForVersion(spec); err != nil {
		return field.Forbidden(fldPath, err.Error())
	}

	return nil
}

func ValidateClusterNetworkConfig(n *kubermaticv1.ClusterNetworkingConfig, dc *kubermaticv1.Datacenter, cni *kubermaticv1.CNIPluginSettings, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// Maximum 2 (one IPv4 + one IPv6) CIDR blocks are allowed
	if len(n.Pods.CIDRBlocks) > 2 {
		allErrs = append(allErrs, field.TooMany(fldPath.Child("pods", "cidrBlocks"), len(n.Pods.CIDRBlocks), 2))
	}
	if len(n.Services.CIDRBlocks) > 2 {
		allErrs = append(allErrs, field.TooMany(fldPath.Child("services", "cidrBlocks"), len(n.Services.CIDRBlocks), 2))
	}
	if len(n.Pods.CIDRBlocks) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("pods", "cidrBlocks"), "pod CIDR must be provided"))
	}
	if len(n.Services.CIDRBlocks) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("services", "cidrBlocks"), "service CIDR must be provided"))
	}
	if len(n.Pods.CIDRBlocks) < len(n.Services.CIDRBlocks) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("pods", "cidrBlocks"), n.Pods.CIDRBlocks,
			fmt.Sprintf("%d pod CIDRs must be provided", len(n.Services.CIDRBlocks))),
		)
	}
	if len(n.Services.CIDRBlocks) < len(n.Pods.CIDRBlocks) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("services", "cidrBlocks"), n.Services.CIDRBlocks,
			fmt.Sprintf("%d services CIDRs must be provided", len(n.Pods.CIDRBlocks))),
		)
	}

	// Verify that provided CIDRs are well-formed
	if err := validateClusterCIDRBlocks(n.Pods.CIDRBlocks, fldPath.Child("pods", "cidrBlocks")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateClusterCIDRBlocks(n.Services.CIDRBlocks, fldPath.Child("services", "cidrBlocks")); err != nil {
		allErrs = append(allErrs, err)
	}

	// Verify that IP family is consistent with provided pod CIDRs
	if (n.IPFamily == kubermaticv1.IPFamilyIPv4) && len(n.Pods.CIDRBlocks) != 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("ipFamily"), n.IPFamily,
			fmt.Sprintf("IP family %q does not match with provided pods CIDRs %q", n.IPFamily, n.Pods.CIDRBlocks)),
		)
	}
	if n.IPFamily == kubermaticv1.IPFamilyDualStack && len(n.Pods.CIDRBlocks) != 2 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("ipFamily"), n.IPFamily,
			fmt.Sprintf("IP family %q does not match with provided pods CIDRs %q", n.IPFamily, n.Pods.CIDRBlocks)),
		)
	}

	// Verify that node CIDR mask sizes are longer than the mask size of pod CIDRs
	if err := validateNodeCIDRMaskSize(n.NodeCIDRMaskSizeIPv4, n.Pods.GetIPv4CIDR(), fldPath.Child("nodeCidrMaskSizeIPv4")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateNodeCIDRMaskSize(n.NodeCIDRMaskSizeIPv6, n.Pods.GetIPv6CIDR(), fldPath.Child("nodeCidrMaskSizeIPv6")); err != nil {
		allErrs = append(allErrs, err)
	}

	// TODO Remove all hardcodes before allowing arbitrary domain names.
	if n.DNSDomain != "cluster.local" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("dnsDomain"), n.DNSDomain, "dnsDomain must be 'cluster.local'"))
	}

	if errs := validateProxyMode(n, cni, fldPath); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if n.IPFamily == kubermaticv1.IPFamilyDualStack && dc != nil {
		cloudProvider, err := kubermaticv1helper.DatacenterCloudProviderName(&dc.Spec)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath, nil,
				fmt.Sprintf("could not determine cloud provider: %v", err)))
		}

		cloudProviderType := kubermaticv1.ProviderType(cloudProvider)

		if cloudProviderType.IsIPv6KnownProvider() && !dc.IsIPv6Enabled(cloudProviderType) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("ipFamily"), n.IPFamily,
				fmt.Sprintf("IP family %q requires ipv6 to be enabled for the datacenter", n.IPFamily)),
			)
		}
	}

	if n.KonnectivityEnabled != nil && !*n.KonnectivityEnabled { //nolint:staticcheck
		allErrs = append(allErrs, field.Invalid(fldPath.Child("konnectivityEnabled"), n.KonnectivityEnabled, //nolint:staticcheck
			"Konnectivity can no longer be disabled"),
		)
	}

	return allErrs
}

func validateProxyMode(n *kubermaticv1.ClusterNetworkingConfig, cni *kubermaticv1.CNIPluginSettings, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if n.ProxyMode != resources.IPVSProxyMode && n.ProxyMode != resources.IPTablesProxyMode && n.ProxyMode != resources.EBPFProxyMode {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("proxyMode"), n.ProxyMode,
			[]string{resources.IPVSProxyMode, resources.IPTablesProxyMode, resources.EBPFProxyMode}))
	}

	if n.ProxyMode == resources.EBPFProxyMode && (cni == nil || cni.Type == kubermaticv1.CNIPluginTypeCanal) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("proxyMode"), n.ProxyMode,
			fmt.Sprintf("%s proxy mode is not valid for %s CNI", resources.EBPFProxyMode, kubermaticv1.CNIPluginTypeCanal)))
	}

	if n.ProxyMode == resources.EBPFProxyMode && (n.KonnectivityEnabled == nil || !*n.KonnectivityEnabled) { //nolint:staticcheck
		allErrs = append(allErrs, field.Invalid(fldPath.Child("proxyMode"), n.ProxyMode,
			fmt.Sprintf("%s proxy mode can be used only when Konnectivity is enabled", resources.EBPFProxyMode)))
	}

	return allErrs
}

func validateEncryptionConfiguration(spec *kubermaticv1.ClusterSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.EncryptionConfiguration != nil && spec.EncryptionConfiguration.Enabled {
		if enabled, ok := spec.Features[kubermaticv1.ClusterFeatureEncryptionAtRest]; !ok || !enabled {
			allErrs = append(allErrs, field.Forbidden(fieldPath.Child("enabled"),
				fmt.Sprintf("cannot enable encryption configuration if feature gate '%s' is not set", kubermaticv1.ClusterFeatureEncryptionAtRest)))
		}

		// TODO: Update with implementations of other encryption providers (KMS)

		if spec.EncryptionConfiguration.Secretbox == nil {
			allErrs = append(allErrs, field.Required(fieldPath.Child("secretbox"),
				"exactly one encryption provider (secretbox, kms) needs to be configured"))
		} else {
			for i, key := range spec.EncryptionConfiguration.Secretbox.Keys {
				childPath := fieldPath.Child("secretbox", "keys").Index(i)
				if key.Name == "" {
					allErrs = append(allErrs, field.Required(childPath.Child("name"),
						"secretbox key name is required"))
				}

				if key.Value == "" && key.SecretRef == nil {
					allErrs = append(allErrs, field.Required(childPath,
						"either 'value' or 'secretRef' must be set"))
				}

				if key.Value != "" && key.SecretRef != nil {
					allErrs = append(allErrs, field.Invalid(childPath, key.Name,
						"'value' and 'secretRef' cannot be set at the same time"))
				}

				if key.Value != "" {
					if err := validateKeyLength(key.Value); err != nil {
						allErrs = append(allErrs, field.Invalid(childPath, key.Name, fmt.Sprint(err)))
					}
				}
			}
		}

		// END TODO
	}

	return allErrs
}

// validateKeyLength base64 decodes key and checks length.
func validateKeyLength(key string) error {
	data, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}

	if len(data) != EARKeyLength {
		return fmt.Errorf("key length should be 32 it is %d", len(data))
	}

	return nil
}

func validateEncryptionUpdate(oldCluster *kubermaticv1.Cluster, newCluster *kubermaticv1.Cluster) field.ErrorList {
	allErrs := field.ErrorList{}

	if enabled, ok := oldCluster.Spec.Features[kubermaticv1.ClusterFeatureEncryptionAtRest]; ok && enabled {
		if oldCluster.Status.Encryption != nil {
			if oldCluster.Status.Encryption.Phase != "" && oldCluster.Status.Encryption.Phase != kubermaticv1.ClusterEncryptionPhaseActive {
				if !equality.Semantic.DeepEqual(oldCluster.Spec.EncryptionConfiguration, newCluster.Spec.EncryptionConfiguration) {
					allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "encryptionConfiguration"),
						fmt.Sprintf("no changes to encryption configuration are allowed while encryption phase is '%s'", oldCluster.Status.Encryption.Phase),
					))
				}
			}

			encryptionConfigExists :=
				oldCluster.Spec.EncryptionConfiguration != nil &&
					newCluster.Spec.EncryptionConfiguration != nil

			if encryptionConfigExists {
				encryptionConfigEnabled :=
					oldCluster.Spec.EncryptionConfiguration.Enabled &&
						newCluster.Spec.EncryptionConfiguration.Enabled

				if encryptionConfigEnabled && !equality.Semantic.DeepEqual(oldCluster.Spec.EncryptionConfiguration.Resources, newCluster.Spec.EncryptionConfiguration.Resources) {
					allErrs = append(
						allErrs,
						field.Forbidden(
							field.NewPath("spec", "encryptionConfiguration", "resources"),
							"list of encrypted resources cannot be changed. Please disable encryption and re-configure",
						),
					)
				}
			}
		}
	}

	// prevent removing the feature flag while the cluster is still in some encryption-active configuration or state
	if enabled, ok := newCluster.Spec.Features[kubermaticv1.ClusterFeatureEncryptionAtRest]; (!ok || !enabled) && (newCluster.IsEncryptionEnabled() || newCluster.IsEncryptionActive()) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("features"),
			fmt.Sprintf("cannot disable %q feature flag while encryption is still configured or active", kubermaticv1.ClusterFeatureEncryptionAtRest),
		))
	}

	return allErrs
}

func validateClusterCIDRBlocks(cidrBlocks []string, fldPath *field.Path) *field.Error {
	for i, cidr := range cidrBlocks {
		addr, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return field.Invalid(fldPath.Index(i), cidr, fmt.Sprintf("couldn't parse CIDR %q: %v", cidr, err))
		}
		// At this point, KKP only supports IPv4 as the primary CIDR and IPv6 as the secondary CIDR.
		// The first provided CIDR has to be IPv4
		if i == 0 && addr.To4() == nil {
			return field.Invalid(fldPath.Child("pods", "cidrBlocks").Index(i), cidr,
				fmt.Sprintf("invalid address family for primary CIDR %q: has to be IPv4", cidr))
		}
		// The second provided CIDR has to be IPv6
		if i == 1 && addr.To4() != nil {
			return field.Invalid(fldPath.Child("pods", "cidrBlocks").Index(i), cidr,
				fmt.Sprintf("invalid address family for secondary CIDR %q: has to be IPv6", cidr))
		}
	}
	return nil
}

func validateNodeCIDRMaskSize(nodeCIDRMaskSize *int32, podCIDR string, fldPath *field.Path) *field.Error {
	if podCIDR == "" || nodeCIDRMaskSize == nil {
		return nil
	}
	_, podCIDRNet, err := net.ParseCIDR(podCIDR)
	if err != nil {
		return field.Invalid(fldPath, podCIDR, fmt.Sprintf("couldn't parse CIDR %q: %v", podCIDR, err))
	}
	podCIDRMaskSize, _ := podCIDRNet.Mask.Size()

	if int32(podCIDRMaskSize) >= *nodeCIDRMaskSize {
		return field.Invalid(fldPath, nodeCIDRMaskSize,
			fmt.Sprintf("node CIDR mask size (%d) must be longer than the mask size of the pod CIDR (%q)", *nodeCIDRMaskSize, podCIDR))
	}
	return nil
}

func validateMachineNetworksFromClusterSpec(spec *kubermaticv1.ClusterSpec, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	networks := spec.MachineNetworks
	basePath := parentFieldPath.Child("machineNetworks")

	if len(networks) == 0 {
		return allErrs
	}

	if len(networks) > 0 && spec.Cloud.VSphere == nil {
		allErrs = append(allErrs, field.Invalid(basePath, networks, "machine networks are only supported with the vSphere provider"))
	}

	for i, network := range networks {
		_, _, err := net.ParseCIDR(network.CIDR)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(basePath.Index(i), network.CIDR, fmt.Sprintf("could not parse CIDR: %v", err)))
		}

		if net.ParseIP(network.Gateway) == nil {
			allErrs = append(allErrs, field.Invalid(basePath.Index(i), network.Gateway, fmt.Sprintf("could not parse gateway: %v", err)))
		}

		if len(network.DNSServers) > 0 {
			for j, dnsServer := range network.DNSServers {
				if net.ParseIP(dnsServer) == nil {
					allErrs = append(allErrs, field.Invalid(basePath.Index(i).Child("dnsServers").Index(j), dnsServer, fmt.Sprintf("could not parse DNS server: %v", err)))
				}
			}
		}
	}

	return allErrs
}

// ValidateCloudChange validates if the cloud provider has been changed.
func ValidateCloudChange(newSpec, oldSpec kubermaticv1.CloudSpec) error {
	if newSpec.DatacenterName != oldSpec.DatacenterName {
		return errors.New("changing the datacenter is not allowed")
	}

	oldCloudProvider, err := kubermaticv1helper.ClusterCloudProviderName(oldSpec)
	if err != nil {
		return fmt.Errorf("could not determine old cloud provider: %w", err)
	}

	newCloudProvider, err := kubermaticv1helper.ClusterCloudProviderName(newSpec)
	if err != nil {
		return fmt.Errorf("could not determine new cloud provider: %w", err)
	}

	if oldCloudProvider != newCloudProvider {
		return ErrCloudChangeNotAllowed
	}

	return nil
}

func validateDatacenterMatchesProvider(spec kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) error {
	clusterCloudProvider, err := kubermaticv1helper.ClusterCloudProviderName(spec)
	if err != nil {
		return fmt.Errorf("could not determine cluster cloud provider: %w", err)
	}

	dcCloudProvider, err := kubermaticv1helper.DatacenterCloudProviderName(&dc.Spec)
	if err != nil {
		return fmt.Errorf("could not determine datacenter cloud provider: %w", err)
	}

	if clusterCloudProvider != dcCloudProvider {
		return fmt.Errorf("expected datacenter provider to be %q, but got %q", clusterCloudProvider, dcCloudProvider)
	}

	if spec.ProviderName != dcCloudProvider {
		return fmt.Errorf("expected providerName to be %q, but got %q", dcCloudProvider, spec.ProviderName)
	}

	return nil
}

// ValidateCloudSpec validates if the cloud spec is valid
// If this is not called from within another validation
// routine, parentFieldPath can be nil.
func ValidateCloudSpec(spec kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter, ipFamily kubermaticv1.IPFamily, parentFieldPath *field.Path, externalCCM bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.DatacenterName == "" {
		allErrs = append(allErrs, field.Required(parentFieldPath.Child("dc"), "no node datacenter specified"))
	}

	providerName, err := kubermaticv1helper.ClusterCloudProviderName(spec)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(parentFieldPath, "<redacted>", err.Error()))
	}

	// if this field is set, it must match the given provider;
	// if the field is not set, the mutation webhook will fill it in
	if spec.ProviderName != "" {
		if spec.ProviderName != providerName {
			msg := fmt.Sprintf("expected providerName to be %q", providerName)
			allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("providerName"), spec.ProviderName, msg))
		}
	}

	if dc != nil {
		if err := validateDatacenterMatchesProvider(spec, dc); err != nil {
			allErrs = append(allErrs, field.Invalid(parentFieldPath, nil, err.Error()))
		}
	}

	var providerErr error

	switch {
	case spec.AWS != nil:
		providerErr = validateAWSCloudSpec(spec.AWS)
	case spec.Alibaba != nil:
		providerErr = validateAlibabaCloudSpec(spec.Alibaba)
	case spec.Anexia != nil:
		providerErr = validateAnexiaCloudSpec(spec.Anexia)
	case spec.Azure != nil:
		providerErr = validateAzureCloudSpec(spec.Azure)
	case spec.Baremetal != nil:
		providerErr = validateBaremetalCloudSpec(spec.Baremetal)
	case spec.BringYourOwn != nil:
		providerErr = nil
	case spec.Edge != nil:
		providerErr = nil
	case spec.Digitalocean != nil:
		providerErr = validateDigitaloceanCloudSpec(spec.Digitalocean)
	case spec.Fake != nil:
		providerErr = validateFakeCloudSpec(spec.Fake)
	case spec.GCP != nil:
		providerErr = validateGCPCloudSpec(spec.GCP, dc, ipFamily, gcp.GetGCPSubnetwork)
	case spec.Hetzner != nil:
		providerErr = validateHetznerCloudSpec(spec.Hetzner)
	case spec.Kubevirt != nil:
		providerErr = validateKubevirtCloudSpec(spec.Kubevirt)
	case spec.Openstack != nil:
		providerErr = validateOpenStackCloudSpec(spec.Openstack, dc, externalCCM)
	//nolint:staticcheck // Deprecated Packet provider is still used for backward compatibility until v2.29
	case spec.Packet != nil:
		providerErr = validatePacketCloudSpec(spec.Packet)
	case spec.VSphere != nil:
		providerErr = validateVSphereCloudSpec(spec.VSphere)
	case spec.Nutanix != nil:
		providerErr = validateNutanixCloudSpec(spec.Nutanix)
	case spec.VMwareCloudDirector != nil:
		providerErr = validateVMwareCloudDirectorCloudSpec(spec.VMwareCloudDirector)
	default:
		providerErr = errors.New("no cloud provider specified")
	}

	if providerErr != nil {
		allErrs = append(allErrs, field.Invalid(parentFieldPath, "<redacted>", providerErr.Error()))
	}

	return allErrs
}

func validateOpenStackCloudSpec(spec *kubermaticv1.OpenstackCloudSpec, dc *kubermaticv1.Datacenter, externalCCM bool) error {
	// validate applicationCredentials
	if spec.ApplicationCredentialID != "" && spec.ApplicationCredentialSecret == "" {
		return errors.New("no applicationCredentialSecret specified")
	}
	if spec.ApplicationCredentialID != "" && spec.ApplicationCredentialSecret != "" {
		return nil
	}

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
	if spec.NodePortsAllowedIPRange != "" {
		if _, _, err := net.ParseCIDR(spec.NodePortsAllowedIPRange); err != nil {
			return err
		}
	}
	if err := spec.NodePortsAllowedIPRanges.Validate(); err != nil {
		return err
	}

	var errs []error
	if spec.Project == "" && spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" && spec.CredentialsReference.Namespace == "" {
		errs = append(errs, fmt.Errorf("%q and %q cannot be empty at the same time", resources.OpenstackProject, resources.OpenstackTenant))
	}
	if spec.ProjectID == "" && spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" && spec.CredentialsReference.Namespace == "" {
		errs = append(errs, fmt.Errorf("%q and %q cannot be empty at the same time", resources.OpenstackProjectID, resources.OpenstackTenantID))
	}
	if len(errs) > 0 {
		return errors.New("no tenant name or ID specified")
	}

	if dc != nil && spec.FloatingIPPool == "" && dc.Spec.Openstack != nil && dc.Spec.Openstack.EnforceFloatingIP {
		return errors.New("no floating ip pool specified")
	}

	if !externalCCM && (spec.EnableIngressHostname != nil || spec.IngressHostnameSuffix != nil) {
		return errors.New("cannot enable ingress hostname feature without external CCM")
	}

	if spec.IngressHostnameSuffix != nil && *spec.IngressHostnameSuffix != "" && (spec.EnableIngressHostname == nil || !*spec.EnableIngressHostname) {
		return errors.New("cannot set ingress hostname suffix if ingress hostname is not enabled")
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
	if spec.NodePortsAllowedIPRange != "" {
		if _, _, err := net.ParseCIDR(spec.NodePortsAllowedIPRange); err != nil {
			return err
		}
	}
	if err := spec.NodePortsAllowedIPRanges.Validate(); err != nil {
		return err
	}

	if spec.DisableIAMReconciling && spec.ControlPlaneRoleARN == "" {
		return fmt.Errorf("roleARN is required when IAM reconciling is disabled")
	}

	if spec.DisableIAMReconciling && spec.InstanceProfileName == "" {
		return fmt.Errorf("instanceProfileName is required when IAM reconciling is disabled")
	}

	return nil
}

func validateGCPCloudSpec(spec *kubermaticv1.GCPCloudSpec, dc *kubermaticv1.Datacenter, ipFamily kubermaticv1.IPFamily, gcpSubnetworkGetter gcp.GCPSubnetworkGetter) error {
	if spec.ServiceAccount == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.GCPServiceAccount); err != nil {
			return err
		}
	}
	if spec.NodePortsAllowedIPRange != "" {
		if _, _, err := net.ParseCIDR(spec.NodePortsAllowedIPRange); err != nil {
			return err
		}
	}
	if err := spec.NodePortsAllowedIPRanges.Validate(); err != nil {
		return err
	}
	if ipFamily == kubermaticv1.IPFamilyDualStack {
		if spec.Network == "" || spec.Subnetwork == "" {
			return errors.New("network and subnetwork should be defined for GCP dual-stack (IPv4 + IPv6) cluster")
		}

		subnetworkParts := strings.Split(spec.Subnetwork, "/")
		if len(subnetworkParts) != 6 {
			return errors.New("invalid GCP subnetwork path")
		}
		subnetworkRegion := subnetworkParts[3]
		subnetworkName := subnetworkParts[5]

		if dc.Spec.GCP.Region != subnetworkRegion {
			return errors.New("GCP subnetwork should belong to same cluster region")
		}

		if spec.ServiceAccount != "" {
			gcpSubnetwork, err := gcpSubnetworkGetter(context.Background(), spec.ServiceAccount, subnetworkRegion, subnetworkName)
			if err != nil {
				return err
			}
			if ipFamily != gcpSubnetwork.IPFamily {
				return errors.New("GCP subnetwork should belong to same cluster network stack type")
			}
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

//nolint:staticcheck // Deprecated Packet provider is still used for backward compatibility until v2.29
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

	if spec.Networks != nil && spec.VMNetName != "" {
		return errors.New("networks and vmNetName cannot be set at the same time")
	}

	return nil
}

func validateVMwareCloudDirectorCloudSpec(spec *kubermaticv1.VMwareCloudDirectorCloudSpec) error {
	if spec.Organization == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.VMwareCloudDirectorOrganization); err != nil {
			return err
		}
	}
	if spec.VDC == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.VMwareCloudDirectorVDC); err != nil {
			return err
		}
	}

	if spec.APIToken != "" || kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.VMwareCloudDirectorAPIToken) == nil {
		return nil
	}

	if spec.Username == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.VMwareCloudDirectorUsername); err != nil {
			return err
		}
	}
	if spec.Password == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.VMwareCloudDirectorPassword); err != nil {
			return err
		}
	}

	if spec.OVDCNetwork == "" && len(spec.OVDCNetworks) == 0 {
		return errors.New("one of ovdcNetwork or ovdcNetworks needs to be specified")
	} else if spec.OVDCNetwork != "" && len(spec.OVDCNetworks) > 0 {
		return errors.New("ovdcNetwork and ovdcNetworks cannot be set at the same time")
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
	if !azureLoadBalancerSKUTypes.Has(string(spec.LoadBalancerSKU)) {
		return fmt.Errorf("azure LB SKU cannot be %q, allowed values are %v", spec.LoadBalancerSKU, sets.List(azureLoadBalancerSKUTypes))
	}
	if spec.NodePortsAllowedIPRange != "" {
		if _, _, err := net.ParseCIDR(spec.NodePortsAllowedIPRange); err != nil {
			return err
		}
	}
	if err := spec.NodePortsAllowedIPRanges.Validate(); err != nil {
		return err
	}

	return nil
}

func validateBaremetalCloudSpec(spec *kubermaticv1.BaremetalCloudSpec) error {
	if spec.Tinkerbell != nil && spec.Tinkerbell.Kubeconfig == "" {
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.TinkerbellKubeconfig); err != nil {
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
		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.KubeVirtKubeconfig); err != nil {
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

func validateNutanixCloudSpec(spec *kubermaticv1.NutanixCloudSpec) error {
	if spec.Username == "" {
		if spec.CredentialsReference == nil {
			return errors.New("no username or credentials reference specified")
		}

		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.NutanixUsername); err != nil {
			return err
		}
	}

	if spec.Password == "" {
		if spec.CredentialsReference == nil {
			return errors.New("no password or credentials reference specified")
		}

		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.NutanixPassword); err != nil {
			return err
		}
	}

	if spec.ClusterName == "" {
		return errors.New("no cluster name specified")
	}

	if spec.CSI == nil {
		return nil
	}

	// validate csi
	if spec.CSI.Username == "" {
		if spec.CredentialsReference == nil {
			return errors.New("no CSI username or credentials reference specified")
		}

		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.NutanixCSIUsername); err != nil {
			return err
		}
	}

	if spec.CSI.Password == "" {
		if spec.CredentialsReference == nil {
			return errors.New("no CSI password or credentials reference specified")
		}

		if err := kuberneteshelper.ValidateSecretKeySelector(spec.CredentialsReference, resources.NutanixCSIPassword); err != nil {
			return err
		}
	}

	if spec.CSI.Endpoint == "" {
		return errors.New("CSI Endpoint mut not be empty")
	}

	// should never happen due to defaulting
	if spec.CSI.Port == nil {
		return errors.New("CSI Port mut not be empty")
	}

	return nil
}

func ValidateUpdateWindow(updateWindow *kubermaticv1.UpdateWindow) error {
	if updateWindow != nil && updateWindow.Start != "" && updateWindow.Length != "" {
		_, err := time.ParseDuration(updateWindow.Length)
		if err != nil {
			return fmt.Errorf("error parsing update Length: %w", err)
		}

		var layout string
		if strings.Contains(updateWindow.Start, " ") {
			layout = "Mon 15:04"
		} else {
			layout = "15:04"
		}

		_, err = time.Parse(layout, updateWindow.Start)
		if err != nil {
			return fmt.Errorf("error parsing start day: %w", err)
		}
	}
	return nil
}

func ValidateContainerRuntime(spec *kubermaticv1.ClusterSpec) error {
	if !sets.New("containerd").Has(spec.ContainerRuntime) {
		return fmt.Errorf("container runtime not supported: %s", spec.ContainerRuntime)
	}

	return nil
}

func ValidateLeaderElectionSettings(l *kubermaticv1.LeaderElectionSettings, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if l.LeaseDurationSeconds != nil && *l.LeaseDurationSeconds < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("leaseDurationSeconds"), l.LeaseDurationSeconds, "lease duration seconds cannot be negative"))
	}
	if l.RenewDeadlineSeconds != nil && *l.RenewDeadlineSeconds < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("renewDeadlineSeconds"), l.RenewDeadlineSeconds, "renew deadline seconds cannot be negative"))
	}
	if l.RetryPeriodSeconds != nil && *l.RetryPeriodSeconds < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("retryPeriodSeconds"), l.RetryPeriodSeconds, "retry period seconds cannot be negative"))
	}
	if lds, rds := l.LeaseDurationSeconds, l.RenewDeadlineSeconds; (lds == nil) != (rds == nil) {
		allErrs = append(allErrs, field.Forbidden(fldPath, "leader election lease duration and renew deadline should be either both specified or unspecified"))
	}
	if lds, rds := l.LeaseDurationSeconds, l.RenewDeadlineSeconds; lds != nil && rds != nil && *lds < *rds {
		allErrs = append(allErrs, field.Forbidden(fldPath, "control plane leader election renew deadline cannot be smaller than lease duration"))
	}

	return allErrs
}

func ValidateNodePortRange(nodePortRange string, fldPath *field.Path) *field.Error {
	if nodePortRange == "" {
		return field.Required(fldPath, "node port range is required")
	}

	portRange, err := kubenetutil.ParsePortRange(nodePortRange)
	if err != nil {
		return field.Invalid(fldPath, nodePortRange, err.Error())
	}

	if portRange.Base == 0 || portRange.Size == 0 {
		return field.Invalid(fldPath, nodePortRange, "invalid nodeport range")
	}

	return nil
}

func validateClusterNetworkingConfigUpdateImmutability(c, oldC *kubermaticv1.ClusterNetworkingConfig, labels map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if oldC.IPFamily != "" {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			c.IPFamily,
			oldC.IPFamily,
			fldPath.Child("ipFamily"),
		)...)
	}

	if len(oldC.Pods.CIDRBlocks) != 0 {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			c.Pods.CIDRBlocks,
			oldC.Pods.CIDRBlocks,
			fldPath.Child("pods", "cidrBlocks"),
		)...)
	}

	if len(oldC.Services.CIDRBlocks) != 0 {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			c.Services.CIDRBlocks,
			oldC.Services.CIDRBlocks,
			fldPath.Child("services", "cidrBlocks"),
		)...)
	}

	if oldC.ProxyMode != "" {
		if _, ok := labels[UnsafeCNIMigrationLabel]; !ok { // allow proxy mode change by CNI migration
			allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
				c.ProxyMode,
				oldC.ProxyMode,
				fldPath.Child("proxyMode"),
			)...)
		}
	}

	if oldC.DNSDomain != "" {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			c.DNSDomain,
			oldC.DNSDomain,
			fldPath.Child("dnsDomain"),
		)...)
	}

	if oldC.NodeLocalDNSCacheEnabled != nil {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			c.NodeLocalDNSCacheEnabled,
			oldC.NodeLocalDNSCacheEnabled,
			fldPath.Child("nodeLocalDNSCacheEnabled"),
		)...)
	}

	return allErrs
}

func validateCNIUpdate(newCni *kubermaticv1.CNIPluginSettings, oldCni *kubermaticv1.CNIPluginSettings, labels map[string]string, k8sVersion semver.Semver) *field.Error {
	basePath := field.NewPath("spec", "cniPlugin")

	// if there was no CNI setting, we allow the mutation to happen
	// allowed for backward compatibility with older KKP with existing clusters with no CNI settings
	if newCni == nil && oldCni == nil {
		return nil
	}

	if oldCni != nil && newCni == nil {
		return field.Required(basePath, "CNI plugin settings cannot be removed")
	}

	if oldCni == nil && newCni != nil {
		return nil // allowed for automated setting of CNI type and version
	}

	if newCni.Type != oldCni.Type {
		if _, ok := labels[UnsafeCNIMigrationLabel]; ok {
			return nil // allowed for CNI type migration path
		}

		return field.Forbidden(basePath.Child("type"), fmt.Sprintf("cannot change CNI plugin type, unless %s label is present", UnsafeCNIMigrationLabel))
	}

	if newCni.Version != oldCni.Version {
		newV, err := semverlib.NewVersion(newCni.Version)
		if err != nil {
			return field.Invalid(basePath.Child("version"), newCni.Version, fmt.Sprintf("couldn't parse CNI version `%s`: %v", newCni.Version, err))
		}

		oldV, err := semverlib.NewVersion(oldCni.Version)
		if err != nil {
			return field.Invalid(basePath.Child("version"), oldCni.Version, fmt.Sprintf("couldn't parse CNI version `%s`: %v", oldCni.Version, err))
		}

		majorVersionChange := newV.Major() != oldV.Major()
		minorVersionChange := newV.Minor() != oldV.Minor()
		oneMinorVersionUpgrade := newV.Minor()-oldV.Minor() == 1
		oneMinorVersionDowngrade := oldV.Minor()-newV.Minor() == 1

		// Major version changes and minor version changes greater than 1 version needs to be explicitly
		// allowed via AllowedCNIVersionTransition entries.
		if majorVersionChange || (minorVersionChange && !oneMinorVersionUpgrade && !oneMinorVersionDowngrade) {
			// allow explicitly defined version transitions
			allowedTransitions := cni.GetAllowedCNIVersionTransitions(newCni.Type)
			for _, t := range allowedTransitions {
				if checkVersionConstraint(k8sVersion.Semver(), t.K8sVersion) &&
					checkVersionConstraint(oldV, t.OldCNIVersion) &&
					checkVersionConstraint(newV, t.NewCNIVersion) {
					return nil
				}
			}
			if _, ok := labels[UnsafeCNIUpgradeLabel]; !ok {
				return field.Forbidden(basePath.Child("version"), fmt.Sprintf("cannot upgrade CNI from %s to %s, only one minor version difference is allowed unless %s label is present", oldCni.Version, newCni.Version, UnsafeCNIUpgradeLabel))
			}
		}
	}

	return nil
}

func checkVersionConstraint(version *semverlib.Version, constraint string) bool {
	if constraint == "" {
		return true // if constraint is not set, assume it is satisfied
	}
	c, err := semverlib.NewConstraint(constraint)
	if err != nil {
		return false
	}
	return c.Check(version)
}

func validatePodSecurityPolicyAdmissionPluginForVersion(spec *kubermaticv1.ClusterSpec) error {
	// Admissin plugin "PodSecurityPolicy" was removed in Kubernetes v1.25 and is no longer supported.
	if spec.UsePodSecurityPolicyAdmissionPlugin {
		return errPodSecurityPolicyAdmissionPluginWithVersionGte125
	}
	for _, admissionPlugin := range spec.AdmissionPlugins {
		if admissionPlugin == podSecurityPolicyAdmissionPluginName {
			return errPodSecurityPolicyAdmissionPluginWithVersionGte125
		}
	}

	return nil
}

func validateCoreDNSReplicas(spec *kubermaticv1.ClusterSpec, fldPath *field.Path) *field.Error {
	newSettings := spec.ComponentsOverride.CoreDNS
	oldSettings := spec.ClusterNetwork

	if newSettings != nil && newSettings.Replicas != nil && oldSettings.CoreDNSReplicas != nil && *oldSettings.CoreDNSReplicas != *newSettings.Replicas {
		return field.Invalid(fldPath.Child("componentsOverride", "coreDNS", "replicas"), *newSettings.Replicas, "both the new spec.componentsOverride.coreDNS.replicas and deprecated spec.clusterNetwork.coreDNSReplicas fields are set")
	}

	return nil
}
