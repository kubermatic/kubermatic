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
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/servingcerthelper"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// NameLabel is the label containing the application's name.
	NameLabel = "app.kubernetes.io/name"

	// VersionLabel is the label containing the application's version.
	VersionLabel = "app.kubernetes.io/version"

	// KubernetesAddonsFileName is the name of the kubernetes addons manifest file
	// in the master files.
	KubernetesAddonsFileName = "kubernetes-addons.yaml"

	DockercfgSecretName  = "dockercfg"
	ExtraFilesSecretName = "extra-files"

	SeedWebhookServiceName    = "seed-webhook"
	ClusterWebhookServiceName = "cluster-webhook"

	// we use a shared certificate/CA for all webhooks, because multiple webhooks
	// run in the same controller manager so it's much easier if they all use the
	// same certs
	webhookCommonName            = "webhook"
	WebhookServingCASecretName   = "webhook-ca"
	WebhookServingCertSecretName = "webhook-cert"

	IngressName                           = "kubermatic"
	MasterControllerManagerDeploymentName = "kubermatic-master-controller-manager"
	SeedControllerManagerDeploymentName   = "kubermatic-seed-controller-manager"

	CleanupFinalizer = "operator.kubermatic.io/cleanup"

	// SkipReconcilingAnnotation can be used on Seed resources to make
	// the operator ignore them and not reconcile the seed components into
	// the cluster. This should only be used during cluster migrations.
	SkipReconcilingAnnotation = "operator.kubermatic.io/skip-reconciling"
)

func NamespaceCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedNamespaceCreatorGetter {
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

func DockercfgSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return DockercfgSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Type = corev1.SecretTypeDockerConfigJson

			return createSecretData(s, map[string]string{
				corev1.DockerConfigJsonKey: cfg.Spec.ImagePullSecret,
			}), nil
		}
	}
}

func WebhookServingCASecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	creator := certificates.GetCACreator(webhookCommonName)

	return func() (string, reconciling.SecretCreator) {
		return WebhookServingCASecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s, err := creator(s)
			if err != nil {
				return s, fmt.Errorf("failed to reconcile webhook CA: %v", err)
			}

			return s, nil
		}
	}
}

func WebhookServingCertSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedSecretCreatorGetter {
	altNames := []string{
		fmt.Sprintf("%s.%s", SeedWebhookServiceName, cfg.Namespace),
		fmt.Sprintf("%s.%s.svc", SeedWebhookServiceName, cfg.Namespace),
		fmt.Sprintf("%s.%s", ClusterWebhookServiceName, cfg.Namespace),
		fmt.Sprintf("%s.%s.svc", ClusterWebhookServiceName, cfg.Namespace),
	}

	caGetter := func() (*triple.KeyPair, error) {
		se := corev1.Secret{}
		key := types.NamespacedName{
			Namespace: cfg.Namespace,
			Name:      WebhookServingCASecretName,
		}

		if err := client.Get(context.Background(), key, &se); err != nil {
			return nil, fmt.Errorf("CA certificate could not be retrieved: %v", err)
		}

		keypair, err := triple.ParseRSAKeyPair(se.Data[resources.CACertSecretKey], se.Data[resources.CAKeySecretKey])
		if err != nil {
			return nil, fmt.Errorf("CA certificate secret contains no valid key pair: %v", err)
		}

		return keypair, nil
	}

	return servingcerthelper.ServingCertSecretCreator(caGetter, WebhookServingCertSecretName, webhookCommonName, altNames, nil)
}

func SeedAdmissionWebhookName(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	return fmt.Sprintf("kubermatic-seeds-%s", cfg.Namespace)
}

func SeedAdmissionWebhookCreator(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return SeedAdmissionWebhookName(cfg), func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.AllScopes

			ca, err := WebhookCABundle(cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find Seed Admission CA bundle: %v", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "seeds.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{"v1beta1"},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      SeedWebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/validate-kubermatic-k8s-io-seed"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							NameLabel: cfg.Namespace,
						},
					},
					ObjectSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"seeds"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.OperationAll,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

// SeedAdmissionServiceCreator creates the Service for the Seed Admission webhook.
// This service is created on master and seed clusters, because on masters the original
// copy of a Seed is validated, and on seed clusters the synced copy is validated
// (synced by the seed-sync controller).
func SeedAdmissionServiceCreator(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return SeedWebhookServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeClusterIP

			if len(s.Spec.Ports) != 1 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Name = "https"
			s.Spec.Ports[0].Port = 443
			s.Spec.Ports[0].TargetPort = intstr.FromInt(8100)
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP

			selector, err := determineSeedWebhookServiceSelector(cfg, client)
			if err != nil {
				return s, fmt.Errorf("failed to determine SeedAdmissionWebhook target service: %v", err)
			}

			s.Spec.Selector = selector

			return s, nil
		}
	}
}

// On master clusters, point to the master-controller-manager, otherwise
// point to the seed-controller-manager. On combined master+seeds, the
// master has precedence.
func determineSeedWebhookServiceSelector(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) (map[string]string, error) {
	deployment := appsv1.Deployment{}
	key := types.NamespacedName{
		Name:      MasterControllerManagerDeploymentName,
		Namespace: cfg.Namespace,
	}

	err := client.Get(context.Background(), key, &deployment)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for %s: %v", key, err)
	}

	if err == nil {
		return map[string]string{
			NameLabel: MasterControllerManagerDeploymentName,
		}, nil
	}

	key = types.NamespacedName{
		Name:      SeedControllerManagerDeploymentName,
		Namespace: cfg.Namespace,
	}

	err = client.Get(context.Background(), key, &deployment)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for %s: %v", key, err)
	}

	if err == nil {
		return map[string]string{
			NameLabel: SeedControllerManagerDeploymentName,
		}, nil
	}

	return nil, fmt.Errorf("neither master- nor seed-controller-manager exist in namespace %s", cfg.Namespace)
}

func WebhookCABundle(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) ([]byte, error) {
	secret := corev1.Secret{}
	key := types.NamespacedName{
		Name:      WebhookServingCASecretName,
		Namespace: cfg.Namespace,
	}

	err := client.Get(context.Background(), key, &secret)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve admission webhook CA Secret %s: %v", WebhookServingCASecretName, err)
	}

	cert, ok := secret.Data[resources.CACertSecretKey]
	if !ok {
		return nil, fmt.Errorf("Secret %s does not contain CA certificate at key %s", WebhookServingCASecretName, resources.CACertSecretKey)
	}

	return cert, nil
}
