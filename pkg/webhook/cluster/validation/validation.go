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
	"fmt"
	"net/http"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/validation"

	admissionv1 "k8s.io/api/admission/v1"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	// unsafeCNIUpgradeLabel allows unsafe CNI version upgrade (difference in versions more than one minor version).
	unsafeCNIUpgradeLabel = "unsafe-cni-upgrade"
	// unsafeCNIMigrationLabel allows unsafe CNI type migration.
	unsafeCNIMigrationLabel = "unsafe-cni-migration"
)

var (
	supportedCNIPlugins        = sets.NewString(kubermaticv1.CNIPluginTypeCanal.String(), kubermaticv1.CNIPluginTypeCilium.String(), kubermaticv1.CNIPluginTypeNone.String())
	supportedCNIPluginVersions = map[kubermaticv1.CNIPluginType]sets.String{
		kubermaticv1.CNIPluginTypeCanal:  sets.NewString("v3.8", "v3.19", "v3.20"),
		kubermaticv1.CNIPluginTypeCilium: sets.NewString("v1.10"),
		kubermaticv1.CNIPluginTypeNone:   sets.NewString(""),
	}
)

// AdmissionHandler for validating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log      logr.Logger
	decoder  *admission.Decoder
	features features.FeatureGate
}

// NewAdmissionHandler returns a new cluster validation AdmissionHandler.
func NewAdmissionHandler(features features.FeatureGate) *AdmissionHandler {
	return &AdmissionHandler{
		features: features,
	}
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("cluster-validation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	allErrs := field.ErrorList{}
	cluster := &kubermaticv1.Cluster{}
	oldCluster := &kubermaticv1.Cluster{}
	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		allErrs = append(allErrs, h.validateCreate(cluster)...)
	case admissionv1.Update:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding cluster: %w", err))
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldCluster); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding old cluster: %w", err))
		}
		allErrs = append(allErrs, h.validateUpdate(cluster, oldCluster)...)
	case admissionv1.Delete:
		// NOP we always allow delete operarions at the moment
	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on cluster resources", req.Operation))
	}
	if len(allErrs) > 0 {
		return webhook.Denied(fmt.Sprintf("cluster validation request %s denied: %v", req.UID, allErrs))
	}
	return webhook.Allowed(fmt.Sprintf("cluster validation request %s allowed", req.UID))
}

func (h *AdmissionHandler) validateCreate(c *kubermaticv1.Cluster) field.ErrorList {
	allErrs := field.ErrorList{}
	specFldPath := field.NewPath("spec")

	if !kubermaticv1.AllExposeStrategies.Has(c.Spec.ExposeStrategy) {
		allErrs = append(allErrs, field.NotSupported(specFldPath.Child("exposeStrategy"), c.Spec.ExposeStrategy, kubermaticv1.AllExposeStrategies.Items()))
	}
	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling &&
		!h.features.Enabled(features.TunnelingExposeStrategy) {
		allErrs = append(allErrs, field.Forbidden(specFldPath.Child("exposeStrategy"), "cannot create cluster with Tunneling expose strategy because the TunnelingExposeStrategy feature gate is not enabled"))
	}
	if c.Spec.CNIPlugin != nil {
		if !supportedCNIPlugins.Has(c.Spec.CNIPlugin.Type.String()) {
			allErrs = append(allErrs, field.NotSupported(specFldPath.Child("cniPlugin", "type"), c.Spec.CNIPlugin.Type.String(), supportedCNIPlugins.List()))
		} else if !supportedCNIPluginVersions[c.Spec.CNIPlugin.Type].Has(c.Spec.CNIPlugin.Version) {
			allErrs = append(allErrs, field.NotSupported(specFldPath.Child("cniPlugin", "version"), c.Spec.CNIPlugin.Version, supportedCNIPluginVersions[c.Spec.CNIPlugin.Type].List()))
		}
	}
	allErrs = append(allErrs, validation.ValidateLeaderElectionSettings(&c.Spec.ComponentsOverride.ControllerManager.LeaderElectionSettings, specFldPath.Child("componentsOverride", "controllerManager", "leaderElection"))...)
	allErrs = append(allErrs, validation.ValidateLeaderElectionSettings(&c.Spec.ComponentsOverride.Scheduler.LeaderElectionSettings, specFldPath.Child("componentsOverride", "scheduler", "leaderElection"))...)
	allErrs = append(allErrs, validation.ValidateClusterNetworkConfig(&c.Spec.ClusterNetwork, c.Spec.CNIPlugin, specFldPath.Child("clusterNetwork"), false)...)

	allErrs = append(allErrs, validation.ValidateNodePortRange(
		c.Spec.ComponentsOverride.Apiserver.NodePortRange,
		specFldPath.Child("componentsOverride", "apiserver", "nodePortRange"), true)...)

	return allErrs
}

func (h *AdmissionHandler) validateUpdate(c, oldC *kubermaticv1.Cluster) field.ErrorList {
	allErrs := field.ErrorList{}
	specFldPath := field.NewPath("spec")

	if !kubermaticv1.AllExposeStrategies.Has(c.Spec.ExposeStrategy) {
		allErrs = append(allErrs, field.NotSupported(specFldPath.Child("exposeStrategy"), c.Spec.ExposeStrategy, kubermaticv1.AllExposeStrategies.Items()))
	}
	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling &&
		!h.features.Enabled(features.TunnelingExposeStrategy) {
		allErrs = append(allErrs, field.Forbidden(specFldPath.Child("exposeStrategy"), "cannot create cluster with Tunneling expose strategy because the TunnelingExposeStrategy feature gate is not enabled"))
	}
	allErrs = append(allErrs, validation.ValidateLeaderElectionSettings(&c.Spec.ComponentsOverride.ControllerManager.LeaderElectionSettings, specFldPath.Child("componentsOverride", "controllerManager", "leaderElection"))...)
	allErrs = append(allErrs, validation.ValidateLeaderElectionSettings(&c.Spec.ComponentsOverride.Scheduler.LeaderElectionSettings, specFldPath.Child("componentsOverride", "scheduler", "leaderElection"))...)
	allErrs = append(allErrs, validation.ValidateClusterNetworkConfig(&c.Spec.ClusterNetwork, c.Spec.CNIPlugin, specFldPath.Child("clusterNetwork"), false)...)

	allErrs = append(allErrs, validation.ValidateNodePortRange(
		c.Spec.ComponentsOverride.Apiserver.NodePortRange,
		specFldPath.Child("componentsOverride", "apiserver", "nodePortRange"), false)...)

	allErrs = append(allErrs, validateUpdateImmutability(c, oldC)...)
	allErrs = append(allErrs, validateCNIUpdate(c.Spec.CNIPlugin, oldC.Spec.CNIPlugin, c.Labels)...)

	return allErrs
}

func validateUpdateImmutability(c, oldC *kubermaticv1.Cluster) field.ErrorList {
	// Immutability should be validated only for update requests
	allErrs := field.ErrorList{}
	specFldPath := field.NewPath("spec")

	// Validate ExternalCloudProvider feature flag immutability.
	// Once the feature flag is enabled, it must not be disabled.
	if vOld, v := oldC.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider],
		c.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; vOld && !v {
		allErrs = append(allErrs, field.Invalid(specFldPath.Child("features").Key(kubermaticv1.ClusterFeatureExternalCloudProvider), v, fmt.Sprintf("feature gate %q cannot be disabled once it's enabled", kubermaticv1.ClusterFeatureExternalCloudProvider)))
	}

	// Validate EtcdLauncher feature flag immutability.
	// Once the feature flag is enabled, it must not be disabled.
	if vOld, v := oldC.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher],
		c.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher]; vOld && !v {
		allErrs = append(allErrs, field.Invalid(specFldPath.Child("features").Key(kubermaticv1.ClusterFeatureEtcdLauncher), v, fmt.Sprintf("feature gate %q cannot be disabled once it's enabled", kubermaticv1.ClusterFeatureEtcdLauncher)))
	}

	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		c.Spec.ExposeStrategy,
		oldC.Spec.ExposeStrategy,
		specFldPath.Child("exposeStrategy"),
	)...)

	if oldC.Spec.EnableUserSSHKeyAgent != nil {
		allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
			c.Spec.EnableUserSSHKeyAgent,
			oldC.Spec.EnableUserSSHKeyAgent,
			specFldPath.Child("enableUserSSHKeyAgent"),
		)...)
	} else if c.Spec.EnableUserSSHKeyAgent != nil && !*c.Spec.EnableUserSSHKeyAgent {
		path := field.NewPath("cluster", "spec", "enableUserSSHKeyAgent")
		allErrs = append(allErrs, field.Invalid(path, *c.Spec.EnableUserSSHKeyAgent, "UserSSHKey agent is enabled by default "+
			"for user clusters created prior KKP 2.16 version"))

	}

	allErrs = append(allErrs, validateClusterNetworkingConfigUpdateImmutability(&c.Spec.ClusterNetwork, &oldC.Spec.ClusterNetwork, specFldPath.Child("clusterNetwork"))...)
	allErrs = append(allErrs, validateComponentSettingsImmutability(&c.Spec.ComponentsOverride, &oldC.Spec.ComponentsOverride, specFldPath.Child("componentsOverride"))...)

	return allErrs
}

func validateComponentSettingsImmutability(c, oldC *kubermaticv1.ComponentSettings, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		c.Apiserver.NodePortRange,
		oldC.Apiserver.NodePortRange,
		fldPath.Child("apiserver", "nodePortRange"),
	)...)

	return allErrs
}

func validateClusterNetworkingConfigUpdateImmutability(c, oldC *kubermaticv1.ClusterNetworkingConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		c.Pods.CIDRBlocks,
		oldC.Pods.CIDRBlocks,
		fldPath.Child("pods", "cidrBlocks"),
	)...)
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		c.Services.CIDRBlocks,
		oldC.Services.CIDRBlocks,
		fldPath.Child("services", "cidrBlocks"),
	)...)
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		c.ProxyMode,
		oldC.ProxyMode,
		fldPath.Child("proxyMode"),
	)...)
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		c.DNSDomain,
		oldC.DNSDomain,
		fldPath.Child("dnsDomain"),
	)...)
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		c.NodeLocalDNSCacheEnabled,
		oldC.NodeLocalDNSCacheEnabled,
		fldPath.Child("nodeLocalDNSCacheEnabled"),
	)...)

	return allErrs
}

func validateCNIUpdate(cni *kubermaticv1.CNIPluginSettings, oldCni *kubermaticv1.CNIPluginSettings, labels map[string]string) field.ErrorList {
	specFldPath := field.NewPath("spec")

	if cni == nil && oldCni == nil {
		return field.ErrorList{} // allowed for backward compatibility with older KKP with existing clusters with no CNI settings
	}
	if oldCni != nil && cni == nil {
		return field.ErrorList{field.Required(specFldPath.Child("cniPlugin"), "CNI plugin settings cannot be removed")}
	}
	if oldCni == nil && cni != nil {
		if _, ok := labels[unsafeCNIUpgradeLabel]; ok {
			return field.ErrorList{} // allowed for migration path from older KKP with existing clusters with no CNI settings
		}
		return field.ErrorList{field.Forbidden(specFldPath.Child("cniPlugin"),
			fmt.Sprintf("cannot add CNI plugin settings, unless %s label is present", unsafeCNIUpgradeLabel))}
	}
	if cni.Type != oldCni.Type {
		if _, ok := labels[unsafeCNIMigrationLabel]; ok {
			return field.ErrorList{} // allowed for CNI type migration path
		}
		return field.ErrorList{field.Forbidden(specFldPath.Child("cniPlugin", "type"),
			fmt.Sprintf("cannot change CNI plugin type, unless %s label is present", unsafeCNIMigrationLabel))}
	}
	if cni.Version != oldCni.Version {
		newV, err := semver.NewVersion(cni.Version)
		if err != nil {
			return field.ErrorList{field.Invalid(specFldPath.Child("cniPlugin", "version"), cni.Version,
				fmt.Sprintf("couldn't parse CNI version `%s`: %v", cni.Version, err))}
		}
		oldV, err := semver.NewVersion(oldCni.Version)
		if err != nil {
			return field.ErrorList{field.Invalid(specFldPath.Child("cniPlugin", "version"), oldCni.Version,
				fmt.Sprintf("couldn't parse CNI version `%s`: %v", oldCni.Version, err))}
		}
		if newV.Major() != oldV.Major() || (newV.Minor() != oldV.Minor()+1 && oldV.Minor() != newV.Minor()+1) {
			if _, ok := labels[unsafeCNIUpgradeLabel]; !ok {
				return field.ErrorList{field.Forbidden(specFldPath.Child("cniPlugin", "version"),
					fmt.Sprintf("cannot upgrade CNI from %s to %s, only one minor version difference is allowed unless %s label is present", oldCni.Version, cni.Version, unsafeCNIUpgradeLabel))}
			}
		}
	}
	return field.ErrorList{}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/validate-kubermatic-k8s-io-cluster", &webhook.Admission{Handler: h})
}
