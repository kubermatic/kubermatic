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

package common

import (
	"fmt"
	"regexp"
	"strings"

	semverlib "github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	// NameLabel is the label containing the application's name.
	NameLabel = "app.kubernetes.io/name"

	// InstanceLabel is A unique name identifying the instance of an application.
	InstanceLabel = "app.kubernetes.io/instance"

	// ComponentLabel is the label of the component within the architecture.
	ComponentLabel = "app.kubernetes.io/component"

	// GatewayAccessLabelKey is the label key used to allow namespaces to attach
	// HTTPRoutes to the KKP Gateway. Namespaces with this label set to "true"
	// can route traffic through the Gateway.
	GatewayAccessLabelKey = "kubermatic.io/gateway-access"

	DockercfgSecretName = "dockercfg"

	WebhookServiceName        = "kubermatic-webhook"
	WebhookRoleName           = "kubermatic-webhook"
	WebhookRoleBindingName    = "kubermatic-webhook"
	WebhookServiceAccountName = "kubermatic-webhook"

	// SeedWebhookServiceName is deprecated and only exists to facilitate cleanup by the operator.
	SeedWebhookServiceName = "seed-webhook"
	// ClusterWebhookServiceName is deprecated and only exists to facilitate cleanup by the operator.
	ClusterWebhookServiceName = "cluster-webhook"

	// UserSSHKeyAdmissionWebhookName is the name of the validating and mutation webhooks for UserSSHKeys.
	UserSSHKeyAdmissionWebhookName = "kubermatic-usersshkeys"

	// UserAdmissionWebhookName is the name of the validating webhook for Users.
	UserAdmissionWebhookName = "kubermatic-users"

	// ResourceQuotaAdmissionWebhookName is the name of the validating and mutating webhook for ResourceQuotas.
	ResourceQuotaAdmissionWebhookName = "kubermatic-resourcequotas"

	// ExternalClusterAdmissionWebhookName is the name of the mutating webhook for ExternalClusters.
	ExternalClusterAdmissionWebhookName = "kubermatic-externalclusters"

	// ApplicationDefinitionAdmissionWebhookName is the name of the validating webhook for ApplicationDefnition.
	ApplicationDefinitionAdmissionWebhookName = "kubermatic-application-definitions"

	// GroupProjectBindingAdmissionWebhookName is the name of the validating webhook for GroupProjectBindings.
	GroupProjectBindingAdmissionWebhookName = "kubermatic-groupprojectbindings"

	// PoliciesAdmissionWebhookName is the name of the validating webhook that implements deletion policies.
	PoliciesAdmissionWebhookName = "kubermatic-policies"

	// PolicyTemplateAdmissionWebhookName is the name of the validating webhook for PolicyTemplates.
	PolicyTemplateAdmissionWebhookName = "kubermatic-policytemplates"

	// we use a shared certificate/CA for all webhooks, because multiple webhooks
	// run in the same controller manager so it's much easier if they all use the
	// same certs.
	webhookCommonName            = "webhook"
	WebhookServingCASecretName   = "webhook-ca"
	WebhookServingCertSecretName = "webhook-cert"

	IngressName                           = "kubermatic"
	GatewayName                           = "kubermatic"
	MasterControllerManagerDeploymentName = "kubermatic-master-controller-manager"
	SeedControllerManagerDeploymentName   = "kubermatic-seed-controller-manager"
	WebhookDeploymentName                 = "kubermatic-webhook"

	CleanupFinalizer = "kubermatic.k8c.io/cleanup"

	// SkipReconcilingAnnotation can be used on Seed resources to make
	// the operator ignore them and not reconcile the seed components into
	// the cluster. This should only be used during cluster migrations.
	SkipReconcilingAnnotation = "kubermatic.k8c.io/skip-reconciling"
)

var (
	// ContainerSecurityContext is a default common security context for containers
	// in the kubermatic/kubermatic container image.
	ContainerSecurityContext = corev1.SecurityContext{
		AllowPrivilegeEscalation: resources.Bool(false),
		ReadOnlyRootFilesystem:   resources.Bool(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				corev1.Capability("ALL"),
			},
		},
	}

	// PodSecurityContext is a default common security context for Pods
	// using the kubermatic/kubermatic image.
	PodSecurityContext = corev1.PodSecurityContext{
		RunAsNonRoot: resources.Bool(true),
		RunAsUser:    resources.Int64(65534),
		RunAsGroup:   resources.Int64(65534),
		FSGroup:      resources.Int64(65534),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
)

func DockercfgSecretReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return DockercfgSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Type = corev1.SecretTypeDockerConfigJson

			return createSecretData(s, map[string]string{
				corev1.DockerConfigJsonKey: cfg.Spec.ImagePullSecret,
			}), nil
		}
	}
}

// CRDReconciler will reconcile a CRD, but only if the existing CRD is older or the same
// version (i.e. this function will never downgrade a CRD). Up- and downgrading is only
// defined for KKP CRDs which have a version annotation.
func CRDReconciler(crd *apiextensionsv1.CustomResourceDefinition, log *zap.SugaredLogger, versions kubermaticversion.Versions) kkpreconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, kkpreconciling.CustomResourceDefinitionReconciler) {
		log = log.With("crd", crd.Name)

		return crd.Name, func(obj *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			currentVersion := versions.GitVersion

			if obj != nil {
				existingVersion := obj.GetAnnotations()[resources.VersionLabel]
				if existingVersion != "" {
					existing, err := semverlib.NewVersion(comparableVersionSuffix(existingVersion))
					if err != nil {
						log.Warnw("CRD has invalid version annotation", "annotation", existingVersion, zap.Error(err))
						// continue to update the CRD
					} else {
						current, err := semverlib.NewVersion(comparableVersionSuffix(currentVersion))
						if err != nil {
							// This should never happen.
							log.Warnw("Built-in CRD has invalid version annotation", "version", currentVersion, zap.Error(err))
							// continue to update the CRD
						} else if existing.GreaterThan(current) {
							log.Warnw("Refusing to downgrade CRD", "version", currentVersion, "crdversion", existingVersion)
							return obj, nil
						}
					}
				}
			}

			obj.Labels = crd.Labels
			obj.Annotations = crd.Annotations
			obj.Spec = crd.Spec

			// inject the current KKP version, so the operator and other controllers
			// can react to the changed CRDs (the KKP installer does the same when
			// updating CRDs on the master cluster)
			if obj.Annotations == nil {
				obj.Annotations = map[string]string{}
			}
			obj.Annotations[resources.VersionLabel] = currentVersion

			if crd.Spec.Conversion == nil {
				// reconcile fails if conversion is not set as it's set by default to None
				obj.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}
			}

			return obj, nil
		}
	}
}

var versionRegex = regexp.MustCompile(`^(.+)-([0-9]+)-g[0-9a-f]+$`)

// We compare versions when updating CRDs and this works fine when it comes
// to "v2.20.1" vs. "v2.20.3". But during development our versions look like
// git-describe output ("<last-tag>-<number of commits since tag>-g<hash>",
// for example "v2.21.0-7-gfd517a". The semverlib does not treat this suffix
// special, and so would say "v2.21.0-10-gfd517a" < "v2.21.0-7-gfd517a".
// Semverlib correctly handles the version suffix, so alpha.4 is smaller than
// alpha.12.
// This function knows about the KKP versioning scheme and turns the commit
// number into a zeta suffix, effectively turning "v2.21.0-7-gabcdef" into
// "v2.21.0-zeta.7".
func comparableVersionSuffix(version string) string {
	match := versionRegex.FindStringSubmatch(version)
	if match == nil {
		// a plain version number without any suffix needs to be treated special
		// as otherwise a comparison like "v1.0.0-1-gabcdef > v1.0.0" would not
		// become true, as semver treats the final tag ("v1.0.0") as the latest
		// version.
		parsed, err := semverlib.NewVersion(version)
		if err != nil {
			// let the outer version parsing deal with the error
			return version
		}

		if parsed.Prerelease() == "" {
			// Inject zeta as hopefully the highest we ever go in prereleases,
			// so that "v1.0.0-zeta.0" > "v1.0.0-beta"
			return fmt.Sprintf("%s-zeta.0", version)
		}

		return version
	}

	base := match[1]
	commits := match[2]

	// a version like "v1.2.3-7" is not valid, so we must treat
	// versions without a second segment special
	if !strings.Contains(base, "-") {
		return fmt.Sprintf("%s-zeta.%s", base, commits)
	}

	return fmt.Sprintf("%s.%s", base, commits)
}
