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

	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	serviceAccountName    = "kubermatic-master"
	uiConfigConfigMapName = "ui-config"
	ingressName           = "kubermatic"
	apiDeploymentName     = "kubermatic-api"
	uiDeploymentName      = "kubermatic-dashboard"
	apiServiceName        = "kubermatic-api"
	uiServiceName         = "kubermatic-dashboard"
	certificateSecretName = "kubermatic-tls"
)

func ClusterRoleBindingName(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:%s-master:cluster-admin", cfg.Namespace, cfg.Name)
}

func UIConfigConfigMapCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return uiConfigConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if c.Data == nil {
				c.Data = make(map[string]string)
			}

			c.Data["config.json"] = cfg.Spec.UI.Config

			return c, nil
		}
	}
}

func ServiceAccountCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func ClusterRoleBindingCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedClusterRoleBindingCreatorGetter {
	name := ClusterRoleBindingName(cfg)

	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: cfg.Namespace,
				},
			}

			return crb, nil
		}
	}
}

func IngressCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedIngressCreatorGetter {
	return func() (string, reconciling.IngressCreator) {
		return ingressName, func(i *networkingv1.Ingress) (*networkingv1.Ingress, error) {
			if i.Annotations == nil {
				i.Annotations = make(map[string]string)
			}
			i.Annotations["kubernetes.io/ingress.class"] = cfg.Spec.Ingress.ClassName

			// If a Certificate is being issued, configure cert-manager by
			// setting up the required annoations.
			issuer := cfg.Spec.Ingress.CertificateIssuer

			if issuer.Name != "" {
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
