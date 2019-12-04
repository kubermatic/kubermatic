package kubermatic

import (
	"errors"
	"fmt"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	presetsSecretName     = "presets"
	serviceAccountName    = "kubermatic-master"
	uiConfigConfigMapName = "ui-config"
	ingressName           = "kubermatic"
	apiDeploymentName     = "kubermatic-api"
	uiDeploymentName      = "kubermatic-ui"
	apiServiceName        = "kubermatic-api"
	uiServiceName         = "kubermatic-ui"
	certificateName       = "kubermatic"
	certificateSecretName = "kubermatic-tls"
)

func ClusterRoleBindingName(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:%s-master:cluster-admin", cfg.Namespace, cfg.Name)
}

func PresetsSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return presetsSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			return createSecretData(s, map[string]string{
				"presets.yaml": cfg.Spec.UI.Presets,
			}), nil
		}
	}
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
			i.Annotations["kubernetes.io/ingress.class"] = "nginx"

			i.Spec.TLS = []extensionsv1beta1.IngressTLS{
				{
					Hosts:      []string{cfg.Spec.Domain},
					SecretName: certificateSecretName,
				},
			}

			i.Spec.Backend = &extensionsv1beta1.IngressBackend{
				ServiceName: uiServiceName,
				ServicePort: intstr.FromInt(80),
			}

			i.Spec.Rules = []extensionsv1beta1.IngressRule{
				{
					Host: cfg.Spec.Domain,
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: []extensionsv1beta1.HTTPIngressPath{
								{
									Path: "/api",
									Backend: extensionsv1beta1.IngressBackend{
										ServiceName: apiServiceName,
										ServicePort: intstr.FromInt(80),
									},
								},
								{
									Path:    "/",
									Backend: *i.Spec.Backend,
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
			name := cfg.Spec.CertificateIssuer.Name
			if name == "" {
				return nil, errors.New("no certificateIssuer configured in KubermaticConfiguration")
			}

			c.Spec.IssuerRef.Name = name
			c.Spec.IssuerRef.Kind = cfg.Spec.CertificateIssuer.Kind

			if group := cfg.Spec.CertificateIssuer.APIGroup; group != nil {
				c.Spec.IssuerRef.Group = *group
			}

			c.Spec.SecretName = certificateSecretName
			c.Spec.DNSNames = []string{cfg.Spec.Domain}

			return c, nil
		}
	}
}
