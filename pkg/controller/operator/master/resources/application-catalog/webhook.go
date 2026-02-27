/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package applicationcatalogmanager

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ApplicationCatalogWebhookDeploymentName = "application-catalog-webhook"
	ApplicationCatalogWebhookServiceName    = "application-catalog-webhook"

	// ApplicationCatalogAdmissionWebhookName is the name of the validating and mutating webhooks for ApplicationCatalog.
	ApplicationCatalogAdmissionWebhookName = "application-catalog"

	// Webhook paths as defined in application-catalog-manager webhook handlers.
	applicationCatalogMutationPath   = "/mutate-applicationcatalog-k8c-io-v1alpha1-applicationcatalog"
	applicationCatalogValidationPath = "/validate-applicationcatalog-k8c-io-v1alpha1-applicationcatalog"
)

var (
	// Default resource requirements for application-catalog-webhook deployment.
	defaultWebhookResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
			corev1.ResourceCPU:    resource.MustParse("50m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("128Mi"),
			corev1.ResourceCPU:    resource.MustParse("200m"),
		},
	}
)

func catalogWebhookPodLabels() map[string]string {
	return map[string]string{
		common.NameLabel:      ApplicationCatalogWebhookDeploymentName,
		common.ComponentLabel: ComponentLabelValue,
	}
}

// CatalogWebhookDeploymentReconciler returns a DeploymentReconciler for the application-catalog webhook.
func CatalogWebhookDeploymentReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return ApplicationCatalogWebhookDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := catalogWebhookPodLabels()

			d.Spec.Replicas = ptr.To(int32(1))
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			kubernetes.EnsureLabels(&d.Spec.Template, labels)
			kubernetes.EnsureAnnotations(&d.Spec.Template, map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8080",
				"fluentbit.io/parser":  "json_iso",
			})

			d.Spec.Template.Spec.ServiceAccountName = ApplicationCatalogServiceAccountName

			args := []string{
				"--webhook-port=9443",
				"--cert-dir=/opt/webhook-serving-cert/",
				"--metrics-bind-address=0.0.0.0:8080",
				"--health-probe-bind-address=0.0.0.0:8081",
			}

			if cfg.Spec.Applications.CatalogManager.WebhookSettings.LogLevel == "debug" {
				args = append(args, "--log-debug=true")
			} else {
				args = append(args, "--log-debug=false")
			}

			volumes := []corev1.Volume{
				{
					Name: "webhook-serving-cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: common.WebhookServingCertSecretName,
						},
					},
				},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					Name:      "webhook-serving-cert",
					MountPath: "/opt/webhook-serving-cert/",
					ReadOnly:  true,
				},
			}

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.SecurityContext = &common.PodSecurityContext

			image := getImage(cfg)
			webhookResources := getWebhookResources(cfg)

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "webhook",
					Image:   image,
					Command: []string{"/usr/local/bin/webhook"},
					Args:    args,
					Env:     common.KubermaticProxyEnvironmentVars(&cfg.Spec.Proxy),
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
					Resources:    webhookResources,
					ReadinessProbe: &corev1.Probe{
						InitialDelaySeconds: 3,
						TimeoutSeconds:      2,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/readyz",
								Scheme: corev1.URISchemeHTTP,
								Port:   intstr.FromInt(8081),
							},
						},
					},
					LivenessProbe: &corev1.Probe{
						InitialDelaySeconds: 10,
						TimeoutSeconds:      10,
						PeriodSeconds:       15,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Scheme: corev1.URISchemeHTTP,
								Port:   intstr.FromInt(8081),
							},
						},
					},
					SecurityContext: &common.ContainerSecurityContext,
				},
			}

			return d, nil
		}
	}
}

func getWebhookResources(cfg *kubermaticv1.KubermaticConfiguration) corev1.ResourceRequirements {
	ws := cfg.Spec.Applications.CatalogManager.WebhookSettings
	if ws.Resources.Requests != nil || ws.Resources.Limits != nil {
		return ws.Resources
	}

	return defaultWebhookResourceRequirements
}

// CatalogWebhookServiceReconciler creates the Service for the application-catalog webhook.
func CatalogWebhookServiceReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return ApplicationCatalogWebhookServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.Selector = catalogWebhookPodLabels()

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

// ApplicationCatalogValidatingWebhookConfigurationReconciler returns a reconciler for the ApplicationCatalog validating webhook.
func ApplicationCatalogValidatingWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return ApplicationCatalogAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.ClusterScope

			ca, err := common.WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "applicationcatalogs.applicationcatalog.k8c.io",
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      ApplicationCatalogWebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To(applicationCatalogValidationPath),
							Port:      ptr.To[int32](443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"applicationcatalog.k8c.io"},
								APIVersions: []string{"*"},
								Resources:   []string{"applicationcatalogs"},
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

// ApplicationCatalogMutatingWebhookConfigurationReconciler returns a reconciler for the ApplicationCatalog mutating webhook.
func ApplicationCatalogMutatingWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedMutatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.MutatingWebhookConfigurationReconciler) {
		return ApplicationCatalogAdmissionWebhookName, func(hook *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			reinvocationPolicy := admissionregistrationv1.NeverReinvocationPolicy
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.ClusterScope

			ca, err := common.WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.MutatingWebhook{
				{
					Name:                    "applicationcatalogs.applicationcatalog.k8c.io",
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ReinvocationPolicy:      &reinvocationPolicy,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      ApplicationCatalogWebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To(applicationCatalogMutationPath),
							Port:      ptr.To[int32](443),
						},
					},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"applicationcatalog.k8c.io"},
								APIVersions: []string{"*"},
								Resources:   []string{"applicationcatalogs"},
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
