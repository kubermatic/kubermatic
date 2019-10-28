package kubermatic

import (
	"fmt"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func apiPodLabels() map[string]string {
	return map[string]string{
		nameLabel:    "kubermatic-api",
		versionLabel: "v1",
	}
}

func APIDeploymentCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return apiDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
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

			d.Spec.Replicas = pointer.Int32Ptr(2)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: apiPodLabels(),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "glog",
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
				"-kubeconfig=/opt/.kube/kubeconfig",
				"-dynamic-datacenters=true",
				fmt.Sprintf("-oidc-url=%s", cfg.Spec.Auth.TokenIssuer),
				fmt.Sprintf("-oidc-authenticator-client-id=%s", cfg.Spec.Auth.ClientID),
				fmt.Sprintf("-oidc-skip-tls-verify=%v", cfg.Spec.Auth.SkipTokenIssuerTLSVerify),
				fmt.Sprintf("-domain=%s", cfg.Spec.Domain),
				fmt.Sprintf("-service-account-signing-key=%s", cfg.Spec.Auth.ServiceAccountKey),
				fmt.Sprintf("-expose-strategy=%s", cfg.Spec.ExposeStrategy),
				fmt.Sprintf("-feature-gates=%s", featureGates(cfg)),
			}

			if cfg.Spec.FeatureGates.Has("OIDCKubeCfgEndpoint") {
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
							SecretName: kubeconfigSecretName,
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
							SecretName: masterFilesSecretName,
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
							SecretName: presetsSecretName,
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
					Name:    "api",
					Image:   cfg.Spec.API.Image,
					Command: []string{"kubermatic-api"},
					Args:    args,
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
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
					ReadinessProbe: &probe,
				},
			}

			return d, nil
		}
	}
}

func APIPDBCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetCreatorGetter {
	name := "kubermatic-api-v1"

	return func() (string, reconciling.PodDisruptionBudgetCreator) {
		return name, func(pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
			min := intstr.FromInt(1)

			pdb.Spec.MinAvailable = &min
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: apiPodLabels(),
			}

			return pdb, nil
		}
	}
}

func APIServiceCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return apiServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeNodePort
			s.Spec.Selector = apiPodLabels()

			if len(s.Spec.Ports) < 2 {
				s.Spec.Ports = make([]corev1.ServicePort, 2)
			}

			s.Spec.Ports[0].Name = "http"
			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromInt(8080)
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP

			s.Spec.Ports[1].Name = "metrics"
			s.Spec.Ports[1].Port = 8085
			s.Spec.Ports[1].TargetPort = intstr.FromInt(8080)
			s.Spec.Ports[1].Protocol = corev1.ProtocolTCP

			return s, nil
		}
	}
}
