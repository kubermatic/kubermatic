package operatormaster

import (
	"fmt"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	dockercfgSecretName         = "dockercfg"
	kubeconfigSecretName        = "kubeconfig"
	presetsSecretName           = "presets"
	dexCASecretName             = "dex-ca"
	masterFilesSecretName       = "extra-files"
	serviceAccountName          = "kubermatic"
	uiConfigConfigMapName       = "ui-config"
	kubermaticAPIDeploymentName = "kubermatic-api-v1"
)

func clusterRoleBindingName(ns string) string {
	return fmt.Sprintf("%s:kubermatic:cluster-admin", ns)
}

func defaultLabels(cfg *operatorv1alpha1.KubermaticConfiguration) map[string]string {
	labels := map[string]string{
		ManagedByLabel: ControllerName,
	}

	return labels
}

func defaultAnnotations(cfg *operatorv1alpha1.KubermaticConfiguration) map[string]string {
	annotations := map[string]string{
		ConfigurationOwnerAnnotation: joinNamespaceName(cfg.Namespace, cfg.Name),
	}

	return annotations
}

func namespaceCreator(name string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedNamespaceCreatorGetter {
	return func() (string, reconciling.NamespaceCreator) {
		return name, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
			ns.Name = name
			ns.Labels = defaultLabels(cfg)
			ns.Annotations = defaultAnnotations(cfg)

			return ns, nil
		}
	}
}

func dockercfgSecretCreator(ns string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return dockercfgSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Name = dockercfgSecretName
			s.Namespace = ns
			s.Labels = defaultLabels(cfg)
			s.Annotations = defaultAnnotations(cfg)
			s.Type = corev1.SecretTypeDockerConfigJson

			if s.Data == nil {
				s.Data = make(map[string][]byte)
			}

			s.Data[corev1.DockerConfigJsonKey] = []byte(cfg.Spec.Secrets.ImagePullSecret)

			return s, nil
		}
	}
}

func kubeconfigSecretCreator(ns string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return kubeconfigSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Name = kubeconfigSecretName
			s.Namespace = ns
			s.Labels = defaultLabels(cfg)
			s.Annotations = defaultAnnotations(cfg)

			if s.Data == nil {
				s.Data = make(map[string][]byte)
			}

			s.Data["kubeconfig"] = []byte(cfg.Spec.Auth.CABundle)

			return s, nil
		}
	}
}

func dexCASecretCreator(ns string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return dexCASecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Name = dexCASecretName
			s.Namespace = ns
			s.Labels = defaultLabels(cfg)
			s.Annotations = defaultAnnotations(cfg)

			if s.Data == nil {
				s.Data = make(map[string][]byte)
			}

			s.Data["caBundle.pem"] = []byte(cfg.Spec.Auth.CABundle)

			return s, nil
		}
	}
}

func masterFilesSecretCreator(ns string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return masterFilesSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Name = masterFilesSecretName
			s.Namespace = ns
			s.Labels = defaultLabels(cfg)
			s.Annotations = defaultAnnotations(cfg)

			if s.Data == nil {
				s.Data = make(map[string][]byte)
			}

			for name, content := range cfg.Spec.MasterFiles {
				s.Data[name] = []byte(content)
			}

			return s, nil
		}
	}
}

func presetsSecretCreator(ns string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return presetsSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Name = presetsSecretName
			s.Namespace = ns
			s.Labels = defaultLabels(cfg)
			s.Annotations = defaultAnnotations(cfg)

			if s.Data == nil {
				s.Data = make(map[string][]byte)
			}

			s.Data["presets.yaml"] = []byte(cfg.Spec.Auth.CABundle)

			return s, nil
		}
	}
}

func uiConfigConfigMapCreator(ns string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return uiConfigConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			c.Name = uiConfigConfigMapName
			c.Namespace = ns
			c.Labels = defaultLabels(cfg)
			c.Annotations = defaultAnnotations(cfg)

			if c.Data == nil {
				c.Data = make(map[string]string)
			}

			c.Data["config.json"] = cfg.Spec.UI.Config

			return c, nil
		}
	}
}

func serviceAccountCreator(ns string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Name = serviceAccountName
			sa.Labels = defaultLabels(cfg)
			sa.Annotations = defaultAnnotations(cfg)

			return sa, nil
		}
	}
}

func clusterRoleBindingCreator(ns string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedClusterRoleBindingCreatorGetter {
	name := clusterRoleBindingName(ns)

	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Name = name
			crb.Labels = defaultLabels(cfg)
			crb.Annotations = defaultAnnotations(cfg)

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: ns,
				},
			}

			return crb, nil
		}
	}
}

func kubermaticAPIDeploymentCreator(ns string, cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return kubermaticAPIDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			probe := corev1.Probe{
				InitialDelaySeconds: 3,
				TimeoutSeconds:      2,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				FailureThreshold:    3,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/api/v1/healthz",
						Scheme: corev1.URISchemeHTTP,
						Port:   intstr.FromInt(8080),
					},
				},
			}

			d.Name = kubermaticAPIDeploymentName
			d.Namespace = ns
			d.Labels = defaultLabels(cfg)
			d.Annotations = defaultAnnotations(cfg)

			specLabels := map[string]string{
				NameLabel:    "kubermatic-api",
				VersionLabel: "v1",
			}

			d.Spec.Replicas = i32ptr(2)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: specLabels,
			}

			d.Spec.Template.Labels = specLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "glog",

				// TODO: add checksums for kubeconfig, datacenters etc. to trigger redeployments
			}

			d.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: dockercfgSecretName,
				},
			}

			args := []string{
				"-v=2",
				"-logtostderr",
				"-address=0.0.0.0:8080",
				"-internal-address=0.0.0.0:8085",
				// "-datacenters=/opt/datacenter/datacenters.yaml",
				"-kubeconfig=/opt/.kube/kubeconfig",
				fmt.Sprintf("-oidc-url=%s", cfg.Spec.Auth.TokenIssuer),
				fmt.Sprintf("-oidc-authenticator-client-id=%s", cfg.Spec.Auth.ClientID),
				fmt.Sprintf("-oidc-skip-tls-verify=%v", cfg.Spec.Auth.SkipTokenIssuerTLSVerify),
				fmt.Sprintf("-domain=%s", cfg.Spec.Domain),
				fmt.Sprintf("-service-account-signing-key=%s", cfg.Spec.Auth.ServiceAccountKey),
				// fmt.Sprintf("-feature-gates=%s", cfg.Spec.FeatureGates),
				//	-expose-strategy={{ .Values.kubermatic.exposeStrategy }}
			}

			if cfg.Spec.FeatureGates.OIDCKubeCfgEndpoint.Enabled {
				args = append(
					args,
					fmt.Sprintf("-oidc-issuer-redirect-uri=%s", cfg.Spec.Auth.IssuerRedirectURL),
					fmt.Sprintf("-oidc-issuer-client-id=%s", cfg.Spec.Auth.IssuerClientID),
					fmt.Sprintf("-oidc-issuer-client-secret=%s", cfg.Spec.Auth.IssuerClientSecret),
					fmt.Sprintf("-oidc-issuer-cookie-hash-key=%s", cfg.Spec.Auth.IssuerCookieKey),
				)
			}

			volumes := []corev1.Volume{
				{
					Name: "kubeconfig",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: i32ptr(420),
							SecretName:  kubeconfigSecretName,
						},
					},
				},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					MountPath: "/opt/.kube/",
					Name:      "kubeconfig",
					ReadOnly:  true,
				},
			}

			if len(cfg.Spec.MasterFiles) > 0 {
				args = append(
					args,
					"-versions=/opt/master-files/versions.yaml",
					"-updates=/opt/master-files/updates.yaml",
					"-master-resources=/opt/master-files",
				)

				volumes = append(volumes, corev1.Volume{
					Name: "master-files",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: i32ptr(420),
							SecretName:  masterFilesSecretName,
						},
					},
				})

				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					MountPath: "/opt/master-files/",
					Name:      "master-files",
					ReadOnly:  true,
				})
			}

			if cfg.Spec.UI.Presets != "" {
				args = append(args, "-presets=/opt/presets/presets.yaml")

				volumes = append(volumes, corev1.Volume{
					Name: "presets",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: i32ptr(420),
							SecretName:  presetsSecretName,
						},
					},
				})

				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					MountPath: "/opt/presets/",
					Name:      "presets",
					ReadOnly:  true,
				})
			}

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "api",
					Image:           "quay.io/kubermatic/api:865c75fef2128b1d7076f48f8f03c7b81f74ce5f",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"kubermatic-api"},
					Args:            args,
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "http",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: volumeMounts,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
					TerminationMessagePolicy: corev1.TerminationMessageReadFile,
					TerminationMessagePath:   "/dev/termination-log",
					ReadinessProbe:           &probe,
				},
			}

			return d, nil
		}
	}
}

func i32ptr(i int32) *int32 {
	return &i
}
