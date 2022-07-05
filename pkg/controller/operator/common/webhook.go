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

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/servingcerthelper"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	seedNameEnvVariable = "SEED_NAME"
)

func webhookPodLabels() map[string]string {
	return map[string]string{
		NameLabel: WebhookDeploymentName,
	}
}

func WebhookServiceAccountCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return WebhookServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func WebhookClusterRoleName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:kubermatic-webhook", cfg.Namespace)
}

func WebhookClusterRoleBindingName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:kubermatic-webhook", cfg.Namespace)
}

func WebhookClusterRoleCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return WebhookClusterRoleName(cfg), func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kubermatic.k8c.io"},
					Resources: []string{"clustertemplates", "projects", "ipamallocations"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return r, nil
		}
	}
}

func WebhookClusterRoleBindingCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return WebhookClusterRoleBindingName(cfg), func(rb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     WebhookClusterRoleName(cfg),
			}

			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      WebhookServiceAccountName,
					Namespace: cfg.Namespace,
				},
			}

			return rb, nil
		}
	}
}

func WebhookRoleCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return WebhookRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kubermatic.k8c.io"},
					Resources: []string{"seeds", "kubermaticconfigurations", "resourcequotas"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return r, nil
		}
	}
}

func WebhookRoleBindingCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return WebhookRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     WebhookRoleName,
			}

			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      WebhookServiceAccountName,
					Namespace: cfg.Namespace,
				},
			}

			return rb, nil
		}
	}
}

// WebhookDeploymentCreator returns a DeploymentCreator for the Kubermatic webhook.
// The removeSeed flag should always be set to false, except for during seed cleanup.
// This is important because on shared master+seed clusters, when the Seed is removed,
// the -seed-name flag must be gone. But because the creator is careful to not accidentally
// remove the flag (so that the master-operator does not wipe the seed-operator's work),
// a separate parameter is needed to indicate that yes, we want to in fact remove the flag.
func WebhookDeploymentCreator(cfg *kubermaticv1.KubermaticConfiguration, versions kubermatic.Versions, seed *kubermaticv1.Seed, removeSeed bool) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return WebhookDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = cfg.Spec.Webhook.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: webhookPodLabels(),
			}
			d.Spec.Template.Spec.ServiceAccountName = WebhookServiceAccountName

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8080",
				"fluentbit.io/parser":  "json_iso",
			}

			args := []string{
				"-webhook-cert-dir=/opt/webhook-serving-cert/",
				fmt.Sprintf("-webhook-cert-name=%s", resources.ServingCertSecretKey),
				fmt.Sprintf("-webhook-key-name=%s", resources.ServingCertKeySecretKey),
				fmt.Sprintf("-ca-bundle=/opt/ca-bundle/%s", resources.CABundleConfigMapKey),
				fmt.Sprintf("-namespace=%s", cfg.Namespace),
			}

			if gates := StringifyFeatureGates(cfg); gates != "" {
				args = append(args, fmt.Sprintf("-feature-gates=%s", gates))
			}

			if cfg.Spec.Webhook.PProfEndpoint != nil && *cfg.Spec.Webhook.PProfEndpoint != "" {
				args = append(args, fmt.Sprintf("-pprof-listen-address=%s", *cfg.Spec.Webhook.PProfEndpoint))
			}

			if cfg.Spec.Webhook.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			// This Deployment lives on master and seed clusters. On seed clusters
			// we need to add -seed-name. On a shared master+seed cluster, we must
			// ensure that the 2 controllers will not overwrite each other (master-operator
			// removing the -seed-name flag, seed-operator adding it again). Instead
			// of fiddling with CLI flags, we just use an env variable to store the seed.
			envVars := ProxyEnvironmentVars(cfg)

			if !removeSeed {
				seedName := ""
				if seed != nil {
					seedName = seed.Name
				} else if d != nil && len(d.Spec.Template.Spec.Containers) > 0 {
					// check if the old Deployment had a seed env var
					for _, e := range d.Spec.Template.Spec.Containers[0].Env {
						if e.Name == seedNameEnvVariable {
							seedName = e.Value
							break
						}
					}
				}

				if seedName != "" {
					args = append(args, "-seed-name=$(SEED_NAME)")
					envVars = append(envVars, corev1.EnvVar{
						Name:  seedNameEnvVariable,
						Value: seedName,
					})
				}
			}

			volumes := []corev1.Volume{
				{
					Name: "ca-bundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cfg.Spec.CABundle.Name,
							},
						},
					},
				},
				{
					Name: "webhook-serving-cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: WebhookServingCertSecretName,
						},
					},
				},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					Name:      "ca-bundle",
					MountPath: "/opt/ca-bundle/",
					ReadOnly:  true,
				},
				{
					Name:      "webhook-serving-cert",
					MountPath: "/opt/webhook-serving-cert/",
					ReadOnly:  true,
				},
			}

			if cfg.Spec.ImagePullSecret != "" {
				volumes = append(volumes, corev1.Volume{
					Name: "dockercfg",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: DockercfgSecretName,
						},
					},
				})
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      "dockercfg",
					MountPath: "/opt/docker/",
					ReadOnly:  true,
				})
			}

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "webhook",
					Image:   cfg.Spec.Webhook.DockerRepository + ":" + versions.Kubermatic,
					Command: []string{"kubermatic-webhook"},
					Args:    args,
					Env:     envVars,
					Ports: []corev1.ContainerPort{
						{
							Name:          "admission",
							ContainerPort: 9443,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "metrics",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: volumeMounts,
					Resources:    cfg.Spec.Webhook.Resources,
					ReadinessProbe: &corev1.Probe{
						InitialDelaySeconds: 3,
						TimeoutSeconds:      2,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.Parse("metrics"),
							},
						},
					},
				},
			}

			return d, nil
		}
	}
}

func WebhookServingCASecretCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	creator := certificates.GetCACreator(webhookCommonName)

	return func() (string, reconciling.SecretCreator) {
		return WebhookServingCASecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s, err := creator(s)
			if err != nil {
				return s, fmt.Errorf("failed to reconcile webhook CA: %w", err)
			}

			return s, nil
		}
	}
}

func WebhookServingCertSecretCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedSecretCreatorGetter {
	altNames := []string{
		fmt.Sprintf("%s.%s", WebhookServiceName, cfg.Namespace),
		fmt.Sprintf("%s.%s.svc", WebhookServiceName, cfg.Namespace),
	}

	caGetter := func() (*triple.KeyPair, error) {
		se := corev1.Secret{}
		key := types.NamespacedName{
			Namespace: cfg.Namespace,
			Name:      WebhookServingCASecretName,
		}

		if err := client.Get(ctx, key, &se); err != nil {
			return nil, fmt.Errorf("CA certificate could not be retrieved: %w", err)
		}

		keypair, err := triple.ParseRSAKeyPair(se.Data[resources.CACertSecretKey], se.Data[resources.CAKeySecretKey])
		if err != nil {
			return nil, fmt.Errorf("CA certificate secret contains no valid key pair: %w", err)
		}

		return keypair, nil
	}

	return servingcerthelper.ServingCertSecretCreator(caGetter, WebhookServingCertSecretName, webhookCommonName, altNames, nil)
}

func SeedAdmissionWebhookName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("kubermatic-seeds-%s", cfg.Namespace)
}

func SeedAdmissionWebhookCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return SeedAdmissionWebhookName(cfg), func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.AllScopes

			ca, err := WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "seeds.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/validate-kubermatic-k8c-io-v1-seed"),
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

func KubermaticConfigurationAdmissionWebhookName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("kubermatic-configuration-%s", cfg.Namespace)
}

func KubermaticConfigurationAdmissionWebhookCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return KubermaticConfigurationAdmissionWebhookName(cfg), func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.AllScopes

			ca, err := WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "kubermaticconfigurations.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/validate-kubermatic-k8c-io-v1-kubermaticconfiguration"),
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
								Resources:   []string{"kubermaticconfigurations"},
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

func ApplicationDefinitionValidatingWebhookConfigurationCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return ApplicationDefinitionAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.AllScopes

			ca, err := WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "applicationdefinitions.apps.kubermatic.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/validate-application-definition"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{appskubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"applicationdefinitions"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

// WebhookServiceCreator creates the Service for all KKP webhooks.
func WebhookServiceCreator(cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return WebhookServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.Selector = webhookPodLabels()

			if len(s.Spec.Ports) != 1 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Name = "https"
			s.Spec.Ports[0].Port = 443
			s.Spec.Ports[0].TargetPort = intstr.FromInt(9443)
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP

			return s, nil
		}
	}
}

func WebhookCABundle(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) ([]byte, error) {
	secret := corev1.Secret{}
	key := types.NamespacedName{
		Name:      WebhookServingCASecretName,
		Namespace: cfg.Namespace,
	}

	err := client.Get(ctx, key, &secret)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve admission webhook CA Secret %s: %w", WebhookServingCASecretName, err)
	}

	cert, ok := secret.Data[resources.CACertSecretKey]
	if !ok {
		return nil, fmt.Errorf("Secret %s does not contain CA certificate at key %s", WebhookServingCASecretName, resources.CACertSecretKey)
	}

	return cert, nil
}
