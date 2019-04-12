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

			apiserverIsRunningContainer, err := apiserver.IsRunningInitContainer(data)
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{*apiserverIsRunningContainer}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            name,
					Image:           data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/api:" + resources.KUBERMATICCOMMIT,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/usr/local/bin/user-cluster-controller-manager"},
					Args: append([]string{
						"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"-metrics-listen-address", "0.0.0.0:8085",
						"-health-listen-address", "0.0.0.0:8086",
						"-namespace", "$(NAMESPACE)",
						"-ca-cert", "/etc/kubernetes/pki/ca/ca.crt",
						"-cluster-url", data.Cluster().Address.URL,
						"-openvpn-server-port", fmt.Sprint(openvpnServerPort),
						fmt.Sprintf("-openshift=%t", openshift),
						fmt.Sprintf("-openvpn-ca-cert-file=%s/%s", openvpnCAMountDir, resources.OpenVPNCACertKey),
						fmt.Sprintf("-openvpn-ca-key-file=%s/%s", openvpnCAMountDir, resources.OpenVPNCAKeyKey),
						"-logtostderr",
						"-v", "2",
					}, getNetworkArgs(data)...),
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
					TerminationMessagePath:   corev1.TerminationMessagePathDefault,
					TerminationMessagePolicy: corev1.TerminationMessageReadFile,
					Resources:                defaultResourceRequirements,
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
					// We have to make the secret readable for all for now because owner/group cannot be changed.
					// ( upstream proposal: https://github.com/kubernetes/kubernetes/pull/28733 )
					DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
				},
			},
		},
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CASecretName,
					DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
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
					SecretName:  resources.OpenVPNCASecretName,
					DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
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
