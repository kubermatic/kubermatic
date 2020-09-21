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
	"errors"
	"fmt"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	serviceAccountName    = "kubermatic-master"
	uiConfigConfigMapName = "ui-config"
	ingressName           = "kubermatic"
	apiDeploymentName     = "kubermatic-api"
	uiDeploymentName      = "kubermatic-dashboard"
	apiServiceName        = "kubermatic-api"
	uiServiceName         = "kubermatic-dashboard"
	certificateName       = "kubermatic"
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
		return ingressName, func(i *extensionsv1beta1.Ingress) (*extensionsv1beta1.Ingress, error) {
			if i.Annotations == nil {
				i.Annotations = make(map[string]string)
			}
			i.Annotations["kubernetes.io/ingress.class"] = cfg.Spec.Ingress.ClassName

			i.Spec.TLS = []extensionsv1beta1.IngressTLS{
				{
					Hosts:      []string{cfg.Spec.Ingress.Domain},
					SecretName: certificateSecretName,
				},
			}

			i.Spec.Backend = &extensionsv1beta1.IngressBackend{
				ServiceName: uiServiceName,
				ServicePort: intstr.FromInt(80),
			}

			// PathType has been added in Kubernetes 1.18 and defaults to
			// "ImplementationSpecific". To prevent reconcile loops in previous
			// Kubernetes versions, this code is carefully written to not
			// overwrite a PathType that has been defaulted by the apiserver.
			var pathType *extensionsv1beta1.PathType

			// As we control the entire rule set anyway, it's enough to find
			// the first pathType -- they will all be identical eventually.
			if rules := i.Spec.Rules; len(rules) > 0 {
				if http := rules[0].IngressRuleValue.HTTP; http != nil {
					for _, path := range http.Paths {
						pathType = path.PathType
						break
					}
				}
			}

			i.Spec.Rules = []extensionsv1beta1.IngressRule{
				{
					Host: cfg.Spec.Ingress.Domain,
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: []extensionsv1beta1.HTTPIngressPath{
								{
									Path:     "/api",
									PathType: pathType,
									Backend: extensionsv1beta1.IngressBackend{
										ServiceName: apiServiceName,
										ServicePort: intstr.FromInt(80),
									},
								},
								{
									Path:     "/",
									PathType: pathType,
									Backend:  *i.Spec.Backend,
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

func CertificateCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedCertificateCreatorGetter {
	return func() (string, reconciling.CertificateCreator) {
		return certificateName, func(c *certmanagerv1alpha2.Certificate) (*certmanagerv1alpha2.Certificate, error) {
			name := cfg.Spec.Ingress.CertificateIssuer.Name
			if name == "" {
				return nil, errors.New("no certificateIssuer configured in KubermaticConfiguration")
			}

			c.Spec.IssuerRef.Name = name
			c.Spec.IssuerRef.Kind = cfg.Spec.Ingress.CertificateIssuer.Kind

			if group := cfg.Spec.Ingress.CertificateIssuer.APIGroup; group != nil {
				c.Spec.IssuerRef.Group = *group
			}

			c.Spec.SecretName = certificateSecretName
			c.Spec.DNSNames = []string{cfg.Spec.Ingress.Domain}

			return c, nil
		}
	}
}
