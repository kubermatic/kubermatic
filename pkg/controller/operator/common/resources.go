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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
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

func NamespaceCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedNamespaceCreatorGetter {
	return func() (string, reconciling.NamespaceCreator) {
		return cfg.Namespace, func(n *corev1.Namespace) (*corev1.Namespace, error) {
			if n.Labels == nil {
				n.Labels = map[string]string{}
			}

			n.Labels[NameLabel] = cfg.Namespace

			return n, nil
		}
	}
}

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
