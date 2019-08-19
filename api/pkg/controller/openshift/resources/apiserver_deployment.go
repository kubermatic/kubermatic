package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
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
	ApiserverDeploymentName = "apiserver-openshift"
	// TODO: Change this to `apiserver-openshift`. Requires to mange the service
	// with this controller first as we need to adapt its label selector
	legacyAppLabelValue = "apiserver"
)

// DeploymentCreator returns the function to create and update the API server deployment
func APIDeploymentCreator(ctx context.Context, data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return ApiserverDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {

			dep.Name = ApiserverDeploymentName
			dep.Labels = resources.BaseAppLabel(legacyAppLabelValue, nil)

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.Apiserver.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.Apiserver.Replicas
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(legacyAppLabelValue, nil),
			}
			dep.Spec.Strategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
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

			resourceRequirements := apiServerDefaultResourceRequirements.DeepCopy()
			if data.Cluster().Spec.ComponentsOverride.Apiserver.Resources != nil {
				resourceRequirements = data.Cluster().Spec.ComponentsOverride.Apiserver.Resources
			}

			envVars, err := getAPIServerEnvVars(data)
			if err != nil {
				return nil, err
			}

			image, err := openshiftKubeAPIServerImage(data.Cluster().Spec.Version.String())
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				*dnatControllerSidecar,
				{
					Name:      ApiserverDeploymentName,
					Image:     image,
					Command:   []string{"hypershift", "openshift-kube-apiserver"},
					Args:      []string{"--config=/etc/origin/master/master-config.yaml"},
					Env:       envVars,
					Resources: *resourceRequirements,
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

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(ApiserverDeploymentName, data.Cluster().Name)

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
			Name:      resources.DexCASecretName,
			MountPath: "/etc/kubernetes/dex/ca",
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
					SecretName:  resources.ApiserverTLSSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.OpenVPNClientCertificatesSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.KubeletClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.KubeletClientCertificatesSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
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
					SecretName:  resources.ServiceAccountKeySecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
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
					DefaultMode: resources.Int32(420),
				},
			},
		},
		{
			Name: resources.ApiserverEtcdClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ApiserverEtcdClientCertificateSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.ApiserverFrontProxyClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ApiserverFrontProxyClientCertificateSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.FrontProxyCASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.FrontProxyCASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.KubeletDnatControllerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.KubeletDnatControllerKubeconfigSecretName,
					DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
				},
			},
		},
		{
			Name: openshiftKubeAPIServerConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: openshiftKubeAPIServerConfigMapName},
					DefaultMode:          resources.Int32(420),
				},
			},
		},
		{
			Name: apiserverLoopbackKubeconfigName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  apiserverLoopbackKubeconfigName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: ServiceSignerCASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  ServiceSignerCASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.DexCASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.DexCASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}
}

func getAPIServerEnvVars(data resources.CredentialsData) ([]corev1.EnvVar, error) {
	credentials, err := resources.GetCredentials(data)
	if err != nil {
		return nil, err
	}
	cluster := data.Cluster()

	var vars []corev1.EnvVar
	if cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: credentials.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: credentials.AWS.SecretAccessKey})
		vars = append(vars, corev1.EnvVar{Name: "AWS_VPC_ID", Value: cluster.Spec.Cloud.AWS.VPCID})
	}
	return vars, nil
}

func openshiftKubeAPIServerImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case "4.1.9":
		return "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:86255c4efe6bbc141a0f41444f863bbd5cd832ffca21d2b737a4f9c225ed00ad", nil
	default:
		return "", fmt.Errorf("no image available for openshift version %q", openshiftVersion)
	}
}
