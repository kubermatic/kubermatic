package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd/etcdrunning"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilpointer "k8s.io/utils/pointer"
)

var (
	apiServerDefaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("4Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

const (
	legacyAppLabelValue                 = "apiserver"
	apiServerOauthMetadataConfigMapName = "openshift-oauth-metadata"
	apiServerOauthMetadataConfigMapKey  = "oauthMetadata"
)

func APIServerOauthMetadataConfigMapCreator(data openshiftData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return apiServerOauthMetadataConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			oauthPort, err := data.GetOauthExternalNodePort()
			if err != nil {
				return nil, fmt.Errorf("failed to get external port for oauth service: %v", err)
			}
			oauthAddress := fmt.Sprintf("https://%s:%d", data.Cluster().Address.ExternalName, oauthPort)
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Data[apiServerOauthMetadataConfigMapKey] = fmt.Sprintf(`{
  "issuer": "%s",
  "authorization_endpoint": "%s/oauth/authorize",
  "token_endpoint": "%s/oauth/token",
  "scopes_supported": [
    "user:check-access",
    "user:full",
    "user:info",
    "user:list-projects",
    "user:list-scoped-projects"
  ],
  "response_types_supported": [
    "code",
    "token"
  ],
  "grant_types_supported": [
    "authorization_code",
    "implicit"
  ],
  "code_challenge_methods_supported": [
    "plain",
    "S256"
  ]
}`, oauthAddress, oauthAddress, oauthAddress)

			return cm, nil
		}
	}
}

// DeploymentCreator returns the function to create and update the API server deployment
func APIDeploymentCreator(ctx context.Context, data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.ApiserverDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {

			dep.Name = resources.ApiserverDeploymentName
			dep.Labels = resources.BaseAppLabels(legacyAppLabelValue, nil)

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.Apiserver.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.Apiserver.Replicas
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(legacyAppLabelValue, nil),
			}
			dep.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
			dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
				MaxSurge: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 0,
				},
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: resources.ImagePullSecretName},
				{Name: openshiftImagePullSecretName},
			}

			volumes := getAPIServerVolumes()

			podLabels, err := data.GetPodTemplateLabelsWithContext(ctx, legacyAppLabelValue, volumes, nil)
			if err != nil {
				return nil, err
			}

			externalNodePort, err := data.GetApiserverExternalNodePort(ctx)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/scrape_with_kube_cert": "true",
					"prometheus.io/path":                  "/metrics",
					"prometheus.io/port":                  strconv.Itoa(int(externalNodePort)),
				},
			}

			etcdEndpoints := etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName)

			// Configure user cluster DNS resolver for this pod.
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				etcdrunning.Container(etcdEndpoints, data),
			}

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn-client sidecar: %v", err)
			}

			dnatControllerSidecar, err := vpnsidecar.DnatControllerContainer(data, "dnat-controller", "")
			if err != nil {
				return nil, fmt.Errorf("failed to get dnat-controller sidecar: %v", err)
			}

			envVars, err := apiserver.GetEnvVars(data)
			if err != nil {
				return nil, err
			}

			image, err := hypershiftImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				*dnatControllerSidecar,
				{
					Name:    resources.ApiserverDeploymentName,
					Image:   image,
					Command: []string{"hypershift", "openshift-kube-apiserver"},
					Args:    []string{"--config=/etc/origin/master/master-config.yaml"},
					Env:     envVars,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: externalNodePort,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(int(externalNodePort)),
								Scheme: "HTTPS",
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    5,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(int(externalNodePort)),
								Scheme: "HTTPS",
							},
						},
						InitialDelaySeconds: 15,
						FailureThreshold:    8,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					VolumeMounts: getVolumeMounts(),
				},
			}
			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				resources.ApiserverDeploymentName: apiServerDefaultResourceRequirements.DeepCopy(),
				openvpnSidecar.Name:               openvpnSidecar.Resources.DeepCopy(),
				dnatControllerSidecar.Name:        dnatControllerSidecar.Resources.DeepCopy(),
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(resources.ApiserverDeploymentName, data.Cluster().Name)

			return dep, nil
		}
	}
}

func getVolumeMounts() []corev1.VolumeMount {
	volumesMounts := []corev1.VolumeMount{
		{
			MountPath: "/etc/kubernetes/tls",
			Name:      resources.ApiserverTLSSecretName,
			ReadOnly:  true,
		},
		{
			Name:      resources.KubeletClientCertificatesSecretName,
			MountPath: "/etc/kubernetes/kubelet",
			ReadOnly:  true,
		},
		{
			Name:      resources.CASecretName,
			MountPath: "/etc/kubernetes/pki/ca",
			ReadOnly:  true,
		},
		{
			Name:      resources.ServiceAccountKeySecretName,
			MountPath: "/etc/kubernetes/service-account-key",
			ReadOnly:  true,
		},
		{
			Name:      resources.CloudConfigConfigMapName,
			MountPath: "/etc/kubernetes/cloud",
			ReadOnly:  true,
		},
		{
			Name:      resources.ApiserverEtcdClientCertificateSecretName,
			MountPath: "/etc/etcd/pki/client",
			ReadOnly:  true,
		},
		{
			Name:      resources.ApiserverFrontProxyClientCertificateSecretName,
			MountPath: "/etc/kubernetes/pki/front-proxy/client",
			ReadOnly:  true,
		},
		{
			Name:      resources.FrontProxyCASecretName,
			MountPath: "/etc/kubernetes/pki/front-proxy/ca",
			ReadOnly:  true,
		},
		{
			Name:      openshiftKubeAPIServerConfigMapName,
			MountPath: "/etc/origin/master",
		},
		{
			Name:      apiserverLoopbackKubeconfigName,
			MountPath: "/etc/origin/master/loopback-kubeconfig",
		},
		{
			Name:      ServiceSignerCASecretName,
			MountPath: "/etc/origin/master/service-signer-ca",
		},
		{
			Name:      apiServerOauthMetadataConfigMapName,
			MountPath: "/etc/kubernetes/oauth-metadata",
		},
	}

	return volumesMounts
}

func getAPIServerVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.ApiserverTLSSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ApiserverTLSSecretName,
				},
			},
		},
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		},
		{
			Name: resources.KubeletClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.KubeletClientCertificatesSecretName,
				},
			},
		},
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CASecretName,
					Items: []corev1.KeyToPath{
						{
							Path: resources.CACertSecretKey,
							Key:  resources.CACertSecretKey,
						},
					},
				},
			},
		},
		{
			Name: resources.ServiceAccountKeySecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ServiceAccountKeySecretName,
				},
			},
		},
		{
			Name: resources.CloudConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CloudConfigConfigMapName,
					},
				},
			},
		},
		{
			Name: resources.ApiserverEtcdClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ApiserverEtcdClientCertificateSecretName,
				},
			},
		},
		{
			Name: resources.ApiserverFrontProxyClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ApiserverFrontProxyClientCertificateSecretName,
				},
			},
		},
		{
			Name: resources.FrontProxyCASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.FrontProxyCASecretName,
				},
			},
		},
		{
			Name: resources.KubeletDnatControllerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.KubeletDnatControllerKubeconfigSecretName,
				},
			},
		},
		{
			Name: openshiftKubeAPIServerConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: openshiftKubeAPIServerConfigMapName},
				},
			},
		},
		{
			Name: apiserverLoopbackKubeconfigName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: apiserverLoopbackKubeconfigName,
				},
			},
		},
		{
			Name: ServiceSignerCASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ServiceSignerCASecretName,
				},
			},
		},
		{

			Name: apiServerOauthMetadataConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: apiServerOauthMetadataConfigMapName}},
			},
		},
	}
}
