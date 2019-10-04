package usercluster

import (
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("25m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

const (
	name              = "usercluster-controller"
	openvpnCAMountDir = "/etc/kubernetes/pki/openvpn"
)

// userclusterControllerData is the subet of the deploymentData interface
// that is actually required by the usercluster deployment
// This makes importing the the deployment elsewhere (openshift controller)
// easier as only have to implement the parts that are actually in use
type userclusterControllerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
	Cluster() *kubermaticv1.Cluster
	GetOpenVPNServerPort() (int32, error)
	KubermaticAPIImage() string
	GetKubernetesCloudProviderName() string
	CloudCredentialSecretTemplate() ([]byte, error)
}

// DeploymentCreator returns the function to create and update the user cluster controller deployment
func DeploymentCreator(data userclusterControllerData, openshift bool) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.UserClusterControllerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.UserClusterControllerDeploymentName
			dep.Labels = resources.BaseAppLabel(name, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(name, nil),
			}
			dep.Spec.Strategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
			dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
				MaxSurge: &intstr.IntOrString{
					Type: intstr.Int,
					// The readiness probe only turns ready if a sync succeeded.
					// That requires that the controller acquires the leader lock, which only happens if the other instance stops
					IntVal: 1,
				},
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			openvpnServerPort, err := data.GetOpenVPNServerPort()
			if err != nil {
				return nil, err
			}

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/path":   "/metrics",
					"prometheus.io/port":   "8085",
				},
			}

			dep.Spec.Template.Spec.Volumes = volumes

			args := append([]string{
				"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"-metrics-listen-address", "0.0.0.0:8085",
				"-health-listen-address", "0.0.0.0:8086",
				"-namespace", "$(NAMESPACE)",
				"-ca-cert", "/etc/kubernetes/pki/ca/ca.crt",
				"-cluster-url", data.Cluster().Address.URL,
				"-openvpn-server-port", fmt.Sprint(openvpnServerPort),
				"-overwrite-registry", data.ImageRegistry(""),
				fmt.Sprintf("-openshift=%t", openshift),
				"-version", data.Cluster().Spec.Version.String(),
				"-cloud-provider-name", data.GetKubernetesCloudProviderName(),
				fmt.Sprintf("-openvpn-ca-cert-file=%s/%s", openvpnCAMountDir, resources.OpenVPNCACertKey),
				fmt.Sprintf("-openvpn-ca-key-file=%s/%s", openvpnCAMountDir, resources.OpenVPNCAKeyKey),
			}, getNetworkArgs(data)...)

			cloudCredentialSecretTemplate, err := data.CloudCredentialSecretTemplate()
			if err != nil {
				return nil, fmt.Errorf("failed to get cloud-credential-secret-template: %v", err)
			}
			if cloudCredentialSecretTemplate != nil {
				args = append(args, "-cloud-credential-secret-template", string(cloudCredentialSecretTemplate))
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    name,
					Image:   data.KubermaticAPIImage() + ":" + resources.KUBERMATICCOMMIT,
					Command: []string{"/usr/local/bin/user-cluster-controller-manager"},
					Args:    args,
					Env: []corev1.EnvVar{
						{
							Name: "NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath:  "metadata.namespace",
									APIVersion: "v1",
								},
							},
						},
					},
					Resources: defaultResourceRequirements,
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/ready",
								Port:   intstr.FromInt(8086),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold: 5,
						PeriodSeconds:    5,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      resources.CASecretName,
							MountPath: "/etc/kubernetes/pki/ca",
							ReadOnly:  true,
						},
						{
							Name:      resources.OpenVPNCASecretName,
							MountPath: openvpnCAMountDir,
							ReadOnly:  true,
						},
					},
				},
			}

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.InternalUserClusterAdminKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
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
			Name: resources.OpenVPNCASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNCASecretName,
				},
			},
		},
	}
}

func getNetworkArgs(data userclusterControllerData) []string {
	networkFlags := make([]string, len(data.Cluster().Spec.MachineNetworks)*2)
	i := 0

	for _, n := range data.Cluster().Spec.MachineNetworks {
		networkFlags[i] = "--ipam-controller-network"
		i++
		networkFlags[i] = fmt.Sprintf("%s,%s,%s", n.CIDR, n.Gateway, strings.Join(n.DNSServers, ","))
		i++
	}

	return networkFlags
}
