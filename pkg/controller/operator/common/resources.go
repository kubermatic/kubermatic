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
	semverlib "github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	// NameLabel is the label containing the application's name.
	NameLabel = "app.kubernetes.io/name"

	// VersionLabel is the label containing the application's version.
	VersionLabel = resources.VersionLabel

	// InstanceLabel is A unique name identifying the instance of an application.
	InstanceLabel = "app.kubernetes.io/instance"

	// ComponentLabel is the label of the component within the architecture.
	ComponentLabel = "app.kubernetes.io/component"

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

	// ResourceQuotaAdmissionWebhookName is the name of the validating webhook for ResourceQuotas.
	ResourceQuotaAdmissionWebhookName = "kubermatic-resourcequotas"

	// ApplicationDefinitionAdmissionWebhookName is the name of the validating webhook for ApplicationDefnition.
	ApplicationDefinitionAdmissionWebhookName = "kubermatic-application-definitions"

	// we use a shared certificate/CA for all webhooks, because multiple webhooks
	// run in the same controller manager so it's much easier if they all use the
	// same certs.
	webhookCommonName            = "webhook"
	WebhookServingCASecretName   = "webhook-ca"
	WebhookServingCertSecretName = "webhook-cert"

	IngressName                           = "kubermatic"
	MasterControllerManagerDeploymentName = "kubermatic-master-controller-manager"
	SeedControllerManagerDeploymentName   = "kubermatic-seed-controller-manager"
	WebhookDeploymentName                 = "kubermatic-webhook"

	CleanupFinalizer = "kubermatic.k8c.io/cleanup"

	// SkipReconcilingAnnotation can be used on Seed resources to make
	// the operator ignore them and not reconcile the seed components into
	// the cluster. This should only be used during cluster migrations.
	SkipReconcilingAnnotation = "kubermatic.k8c.io/skip-reconciling"
)

func DockercfgSecretCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return DockercfgSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Type = corev1.SecretTypeDockerConfigJson

			return createSecretData(s, map[string]string{
				corev1.DockerConfigJsonKey: cfg.Spec.ImagePullSecret,
			}), nil
		}
	}
}

// CRDCreator will reconcile a CRD, but only if the existing CRD is older or the same
// version (i.e. this function will never downgrade a CRD). Up- and downgrading is only
// defined for KKP CRDs which have a version annotation.
func CRDCreator(crd *apiextensionsv1.CustomResourceDefinition, log *zap.SugaredLogger, versions kubermaticversion.Versions) reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		log = log.With("crd", crd.Name)

		return crd.Name, func(obj *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			currentVersion := versions.KubermaticCommit

			if obj != nil {
				existingVersion := obj.GetAnnotations()[resources.VersionLabel]
				if existingVersion != "" {
					existing, err := semverlib.NewVersion(existingVersion)
					if err != nil {
						log.Warnw("CRD has invalid version annotation", "annotation", existingVersion, zap.Error(err))
						// continue to update the CRD
					} else {
						current, err := semverlib.NewVersion(currentVersion)
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
