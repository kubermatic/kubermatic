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

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/servingcerthelper"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
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

func WebhookServiceAccountReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
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

func WebhookClusterRoleReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return WebhookClusterRoleName(cfg), func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kubermatic.k8c.io"},
					Resources: []string{"clustertemplates", "projects", "ipamallocations", "resourcequotas", "users"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"kubermatic.k8c.io"},
					Resources: []string{"externalclusters"},
					Verbs:     []string{"get", "list"},
				},
			}

			return r, nil
		}
	}
}

func WebhookClusterRoleBindingReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
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

func WebhookRoleReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return WebhookRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kubermatic.k8c.io"},
					Resources: []string{"seeds", "kubermaticconfigurations"},
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

func WebhookRoleBindingReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
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

// WebhookDeploymentReconciler returns a DeploymentReconciler for the Kubermatic webhook.
// The removeSeed flag should always be set to false, except for during seed cleanup.
// This is important because on shared master+seed clusters, when the Seed is removed,
// the -seed-name flag must be gone. But because the creator is careful to not accidentally
// remove the flag (so that the master-operator does not wipe the seed-operator's work),
// a separate parameter is needed to indicate that yes, we want to in fact remove the flag.
func WebhookDeploymentReconciler(cfg *kubermaticv1.KubermaticConfiguration, versions kubermatic.Versions, seed *kubermaticv1.Seed, removeSeed bool) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return WebhookDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := webhookPodLabels()
			d.Spec.Replicas = cfg.Spec.Webhook.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}
			d.Spec.Template.Spec.ServiceAccountName = WebhookServiceAccountName

			kubernetes.EnsureLabels(&d.Spec.Template, labels)
			kubernetes.EnsureAnnotations(&d.Spec.Template, map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8080",
				"fluentbit.io/parser":  "json_iso",
			})

			if len(cfg.Spec.Webhook.NodeSelector) > 0 {
				d.Spec.Template.Spec.NodeSelector = cfg.Spec.Webhook.NodeSelector
			}

			if len(cfg.Spec.Webhook.Tolerations) > 0 {
				d.Spec.Template.Spec.Tolerations = cfg.Spec.Webhook.Tolerations
			}

			if cfg.Spec.Webhook.Affinity.NodeAffinity != nil ||
				cfg.Spec.Webhook.Affinity.PodAffinity != nil ||
				cfg.Spec.Webhook.Affinity.PodAntiAffinity != nil {
				d.Spec.Template.Spec.Affinity = &cfg.Spec.Webhook.Affinity
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

			var envVars []corev1.EnvVar

			// The information if a Seed is present or not is stored in the seedNameEnvVariable.
			// This impacts the -seed-name flag and proxy settings.
			withSeed := false
			if seed != nil {
				envVars = append(envVars, corev1.EnvVar{
					Name:  seedNameEnvVariable,
					Value: seed.Name,
				})
				envVars = append(envVars, SeedProxyEnvironmentVars(seed.Spec.ProxySettings)...)
				withSeed = true
			} else if d != nil && len(d.Spec.Template.Spec.Containers) > 0 {
				// check if the old Deployment had a seed env var and is deployed on a Seed or Master+Seed combination.
				for _, e := range d.Spec.Template.Spec.Containers[0].Env {
					if e.Name == seedNameEnvVariable {
						withSeed = true
						break
					}
				}

				if withSeed {
					// The webhook is deployed with a Seed, so it reuses the existing env to keep
					// seedNameEnvVariable and Proxy Settings that where set before based on the actual Seed.
					envVars = d.Spec.Template.Spec.Containers[0].Env
				} else {
					// The webhook is deployed without a Seed and will use the Kubermatic Configuration Proxy settings.
					envVars = KubermaticProxyEnvironmentVars(&cfg.Spec.Proxy)
				}
			}

			// On seed clusters we need to add -seed-name flag. On a shared master+seed cluster,
			// we must ensure that the 2 controllers will not overwrite each other (master-operator
			// removing the -seed-name flag, seed-operator adding it again).
			if !removeSeed && withSeed {
				args = append(args, "-seed-name=$(SEED_NAME)")
			}

			volumes := []corev1.Volume{
				{
					Name: "ca-bundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.CABundleConfigMapName,
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
			d.Spec.Template.Spec.SecurityContext = &PodSecurityContext
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "webhook",
					Image:   cfg.Spec.Webhook.DockerRepository + ":" + versions.KubermaticContainerTag,
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
					SecurityContext: &ContainerSecurityContext,
				},
			}

			return d, nil
		}
	}
}

func WebhookServingCASecretReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedSecretReconcilerFactory {
	reconciler := certificates.GetCAReconciler(webhookCommonName)

	return func() (string, reconciling.SecretReconciler) {
		return WebhookServingCASecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s, err := reconciler(s)
			if err != nil {
				return s, fmt.Errorf("failed to reconcile webhook CA: %w", err)
			}

			return s, nil
		}
	}
}

func WebhookServingCertSecretReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedSecretReconcilerFactory {
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

	return servingcerthelper.ServingCertSecretReconciler(caGetter, WebhookServingCertSecretName, webhookCommonName, altNames, nil)
}

func SeedAdmissionWebhookName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("kubermatic-seeds-%s", cfg.Namespace)
}

func SeedAdmissionWebhookReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
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
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/validate-kubermatic-k8c-io-v1-seed"),
							Port:      ptr.To[int32](443),
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

func KubermaticConfigurationAdmissionWebhookReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
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
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/validate-kubermatic-k8c-io-v1-kubermaticconfiguration"),
							Port:      ptr.To[int32](443),
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

func ApplicationDefinitionValidatingWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
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
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/validate-application-definition"),
							Port:      ptr.To[int32](443),
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
								admissionregistrationv1.Delete,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func ApplicationDefinitionMutatingWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedMutatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.MutatingWebhookConfigurationReconciler) {
		return ApplicationDefinitionAdmissionWebhookName, func(hook *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			reinvocationPolicy := admissionregistrationv1.NeverReinvocationPolicy
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.ClusterScope

			ca, err := WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.MutatingWebhook{
				{
					Name:                    appskubermaticv1.ApplicationDefinitionResourceName + "." + appskubermaticv1.GroupName, // this should be a FQDN,
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ReinvocationPolicy:      &reinvocationPolicy,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/mutate-application-definition"),
							Port:      ptr.To[int32](443),
						},
					},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{appskubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{appskubermaticv1.ApplicationDefinitionResourceName},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
					NamespaceSelector: &metav1.LabelSelector{},
					ObjectSelector:    &metav1.LabelSelector{},
				},
			}
			return hook, nil
		}
	}
}

func PoliciesWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return PoliciesAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			allScopes := admissionregistrationv1.AllScopes
			clusterScope := admissionregistrationv1.ClusterScope

			ca, err := WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "policies." + kubermaticv1.GroupName, // this should be a FQDN,
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](10),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/validate-policies"),
							Port:      ptr.To[int32](443),
						},
					},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"*"},
								Scope:       &allScopes,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Delete,
							},
						},
						// allow to put deletion prevention annotations on cluster namespaces
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"v1"},
								Resources:   []string{"namespaces"},
								Scope:       &clusterScope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Delete,
							},
						},
					},
					NamespaceSelector: &metav1.LabelSelector{},
					ObjectSelector:    &metav1.LabelSelector{},
				},
			}
			return hook, nil
		}
	}
}

func PolicyTemplateValidatingWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return PolicyTemplateAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
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
					Name:                    "policytemplates.kubermatic.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/validate-kubermatic-k8c-io-v1-policytemplate"),
							Port:      ptr.To[int32](443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"policytemplates"},
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

// WebhookServiceReconciler creates the Service for all KKP webhooks.
func WebhookServiceReconciler(cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
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
