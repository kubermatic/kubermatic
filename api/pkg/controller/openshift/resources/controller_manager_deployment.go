package resources

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	controllerManagerDefaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

const (
	ControllerManagerDeploymentName = "openshift-controller-manager"
)

// DeploymentCreator returns the function to create and update the controller manager deployment
func ControllerManagerDeploymentCreator(ctx context.Context, data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return ControllerManagerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.ControllerManagerDeploymentName
			dep.Labels = resources.BaseAppLabel(ControllerManagerDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(ControllerManagerDeploymentName, nil),
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
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getControllerManagerVolumes()
			podLabels, err := data.GetPodTemplateLabelsWithContext(ctx, ControllerManagerDeploymentName, volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/path":                  "/metrics",
					"prometheus.io/scrape_with_kube_cert": "true",
					"prometheus.io/port":                  "8444",
				},
			}

			// Configure user cluster DNS resolver for this pod.
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Volumes = volumes

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
			}

			controllerManagerMounts := []corev1.VolumeMount{
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
					Name: resources.InternalUserClusterAdminKubeconfigSecretName,
					// We have to mount a Kubeconfig that doesn't use 127.0.0.1 as address
					// here
					MountPath: "/etc/origin/master/loopback-kubeconfig",
					ReadOnly:  true,
				},
				{
					Name:      openshiftControllerMangerConfigMapName,
					MountPath: "/etc/origin/master",
				},
				{
					Name:      ServiceSignerCASecretName,
					MountPath: "/etc/origin/master/service-signer-ca",
				},
				{
					MountPath: "/etc/kubernetes/tls",
					Name:      resources.ApiserverTLSSecretName,
					ReadOnly:  true,
				},
			}
			if data.Cluster().Spec.Cloud.VSphere != nil {
				fakeVMWareUUIDMount := corev1.VolumeMount{
					Name:      resources.CloudConfigConfigMapName,
					SubPath:   cloudconfig.FakeVMWareUUIDKeyName,
					MountPath: "/sys/class/dmi/id/product_serial",
					ReadOnly:  true,
				}
				// Required because of https://github.com/kubernetes/kubernetes/issues/65145
				controllerManagerMounts = append(controllerManagerMounts, fakeVMWareUUIDMount)
			}

			resourceRequirements := controllerManagerDefaultResourceRequirements
			if data.Cluster().Spec.ComponentsOverride.ControllerManager.Resources != nil {
				resourceRequirements = *data.Cluster().Spec.ComponentsOverride.ControllerManager.Resources
			}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:                     ControllerManagerDeploymentName,
					Image:                    data.ImageRegistry(resources.RegistryDocker) + "/openshift/origin-control-plane:v3.11",
					ImagePullPolicy:          corev1.PullIfNotPresent,
					Command:                  []string{"/usr/bin/openshift", "start", "master", "controllers"},
					Args:                     []string{"--config=/etc/origin/master/master-config.yaml", "--listen=https://0.0.0.0:8444"},
					Env:                      getControllerManagerEnvVars(data.Cluster()),
					TerminationMessagePath:   corev1.TerminationMessagePathDefault,
					TerminationMessagePolicy: corev1.TerminationMessageReadFile,
					Resources:                resourceRequirements,
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: getHealthGetAction(),
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 8,
						Handler: corev1.Handler{
							HTTPGet: getHealthGetAction(),
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					VolumeMounts: controllerManagerMounts,
				},
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(ControllerManagerDeploymentName, data.Cluster().Name)

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(ControllerManagerDeploymentName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}
func getControllerManagerVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
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
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.OpenVPNClientCertificatesSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.InternalUserClusterAdminKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.InternalUserClusterAdminKubeconfigSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: openshiftControllerMangerConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: openshiftControllerMangerConfigMapName},
					DefaultMode:          resources.Int32(420),
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
			//TODO: Create a distinct serving cert for the controller manager
			// it is a required config option
			Name: resources.ApiserverTLSSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ApiserverTLSSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}
}

func getControllerManagerEnvVars(cluster *kubermaticv1.Cluster) []corev1.EnvVar {
	var vars []corev1.EnvVar
	if cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: cluster.Spec.Cloud.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: cluster.Spec.Cloud.AWS.SecretAccessKey})
		vars = append(vars, corev1.EnvVar{Name: "AWS_VPC_ID", Value: cluster.Spec.Cloud.AWS.VPCID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_AVAILABILITY_ZONE", Value: cluster.Spec.Cloud.AWS.AvailabilityZone})
	}
	return vars
}

func getHealthGetAction() *corev1.HTTPGetAction {
	return &corev1.HTTPGetAction{
		Path:   "/healthz",
		Scheme: corev1.URISchemeHTTPS,
		Port:   intstr.FromInt(8444),
	}
}
