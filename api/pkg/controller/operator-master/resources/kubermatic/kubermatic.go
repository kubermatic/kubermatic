package kubermatic

import (
	"fmt"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	dockercfgSecretName                   = "dockercfg"
	kubeconfigSecretName                  = "kubeconfig"
	presetsSecretName                     = "presets"
	dexCASecretName                       = "dex-ca"
	masterFilesSecretName                 = "extra-files"
	serviceAccountName                    = "kubermatic"
	uiConfigConfigMapName                 = "ui-config"
	backupContainersConfigMapName         = "backup-containers"
	ingressName                           = "kubermatic"
	apiDeploymentName                     = "kubermatic-api-v1"
	uiDeploymentName                      = "kubermatic-ui-v2"
	masterControllerManagerDeploymentName = "kubermatic-master-controller-manager-v1"
	apiServiceName                        = "kubermatic-api"
	uiServiceName                         = "kubermatic-ui"
)

func clusterRoleBindingName(ns string) string {
	return fmt.Sprintf("%s:kubermatic:cluster-admin", ns)
}

func NamespaceCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedNamespaceCreatorGetter {
	return func() (string, reconciling.NamespaceCreator) {
		return cfg.Spec.Namespace, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
			return ns, nil
		}
	}
}

func DockercfgSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return dockercfgSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Type = corev1.SecretTypeDockerConfigJson

			return createSecretData(s, map[string]string{
				corev1.DockerConfigJsonKey: cfg.Spec.ImagePullSecret,
			}), nil
		}
	}
}

func KubeconfigSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return kubeconfigSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			return createSecretData(s, map[string]string{
				"kubeconfig": cfg.Spec.Auth.CABundle,
			}), nil
		}
	}
}

func DexCASecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return dexCASecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			return createSecretData(s, map[string]string{
				"caBundle.pem": cfg.Spec.Auth.CABundle,
			}), nil
		}
	}
}

func MasterFilesSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return masterFilesSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			return createSecretData(s, cfg.Spec.MasterFiles), nil
		}
	}
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

func BackupContainersConfigMapCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return backupContainersConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if c.Data == nil {
				c.Data = make(map[string]string)
			}

			c.Data["store-container.yaml"] = cfg.Spec.SeedController.BackupStoreContainer
			c.Data["cleanup-container.yaml"] = cfg.Spec.SeedController.BackupCleanupContainer

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
	name := clusterRoleBindingName(cfg.Spec.Namespace)

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
					Namespace: cfg.Spec.Namespace,
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
					Hosts: []string{cfg.Spec.Domain},
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
