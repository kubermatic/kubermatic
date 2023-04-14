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

package kubermatic

import (
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

const (
	apiServiceAccountName = "kubermatic-api"
	uiConfigConfigMapName = "ui-config"
	ingressName           = "kubermatic"
	APIDeploymentName     = "kubermatic-api"
	UIDeploymentName      = "kubermatic-dashboard"
	apiServiceName        = "kubermatic-api"
	uiServiceName         = "kubermatic-dashboard"
	certificateSecretName = "kubermatic-tls"
)

func UIConfigConfigMapReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return uiConfigConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if c.Data == nil {
				c.Data = make(map[string]string)
			}

			c.Data["config.json"] = cfg.Spec.UI.Config

			return c, nil
		}
	}
}

func IngressReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedIngressReconcilerFactory {
	return func() (string, reconciling.IngressReconciler) {
		return ingressName, func(i *networkingv1.Ingress) (*networkingv1.Ingress, error) {
			if i.Annotations == nil {
				i.Annotations = make(map[string]string)
			}
			i.Annotations["kubernetes.io/ingress.class"] = cfg.Spec.Ingress.ClassName

			// NGINX ingress annotations to avoid timeout of websocket connections after 1 minute.
			// Needed for Web Terminal feature, for example.
			i.Annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"] = "3600" // 1 hour
			i.Annotations["nginx.ingress.kubernetes.io/proxy-send-timeout"] = "3600" // 1 hour

			// If a Certificate is being issued, configure cert-manager by
			// setting up the required annotations.
			issuer := cfg.Spec.Ingress.CertificateIssuer

			if issuer != nil && issuer.Name != "" {
				delete(i.Annotations, certmanagerv1.IngressIssuerNameAnnotationKey)
				delete(i.Annotations, certmanagerv1.IngressClusterIssuerNameAnnotationKey)

				switch issuer.Kind {
				case certmanagerv1.IssuerKind:
					i.Annotations[certmanagerv1.IngressIssuerNameAnnotationKey] = issuer.Name
				case certmanagerv1.ClusterIssuerKind:
					i.Annotations[certmanagerv1.IngressClusterIssuerNameAnnotationKey] = issuer.Name
				default:
					return nil, fmt.Errorf("unknown Certificate Issuer Kind %q configured", issuer.Kind)
				}

				i.Spec.TLS = []networkingv1.IngressTLS{
					{
						Hosts:      []string{cfg.Spec.Ingress.Domain},
						SecretName: certificateSecretName,
					},
				}
			}

			i.Spec.DefaultBackend = &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: uiServiceName,
					Port: networkingv1.ServiceBackendPort{
						Number: 80,
					},
				},
			}

			pathType := networkingv1.PathTypePrefix

			i.Spec.Rules = []networkingv1.IngressRule{
				{
					Host: cfg.Spec.Ingress.Domain,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/api",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: apiServiceName,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
								{
									Path:     "/",
									PathType: &pathType,
									Backend:  *i.Spec.DefaultBackend,
								},
							},
						},
					},
				},
			}

			return i, nil
		}
	}
}
