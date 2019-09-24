package machinecontroller

import (
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
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
	controllerResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("25m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

const (
	Name = "machine-controller"

	tag = "v1.5.7"

	nodeLocalDNSCacheAddress = "169.254.20.10"
)

type machinecontrollerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
	Cluster() *kubermaticv1.Cluster
	ClusterIPByServiceName(string) (string, error)
	DC() *provider.DatacenterMeta
	NodeLocalDNSCacheEnabled() bool
}

// DeploymentCreator returns the function to create and update the machine controller deployment
func DeploymentCreator(data machinecontrollerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.MachineControllerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.MachineControllerDeploymentName
			dep.Labels = resources.BaseAppLabel(Name, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(Name, nil),
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

			volumes := []corev1.Volume{getKubeconfigVolume()}
			podLabels, err := data.GetPodTemplateLabels(Name, volumes, nil)
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

			clusterDNSIP := nodeLocalDNSCacheAddress
			if !data.NodeLocalDNSCacheEnabled() {
				clusterDNSIP, err = resources.UserClusterDNSResolverIP(data.Cluster())
				if err != nil {
					return nil, err
				}
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:      Name,
					Image:     data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/machine-controller:" + tag,
					Command:   []string{"/usr/local/bin/machine-controller"},
					Args:      getFlags(clusterDNSIP, data.DC().Node),
					Env:       getEnvVars(data),
					Resources: controllerResourceRequirements,
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/ready",
								Port:   intstr.FromInt(8085),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold:    3,
						InitialDelaySeconds: 15,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.MachineControllerKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
					},
				},
			}

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(Name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

func getKubeconfigVolume() corev1.Volume {
	return corev1.Volume{
		Name: resources.MachineControllerKubeconfigSecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: resources.MachineControllerKubeconfigSecretName,
				// We have to make the secret readable for all for now because owner/group cannot be changed.
				// ( upstream proposal: https://github.com/kubernetes/kubernetes/pull/28733 )
				DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
			},
		},
	}
}

func getEnvVars(data machinecontrollerData) []corev1.EnvVar {
	var vars []corev1.EnvVar
	if data.Cluster().Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: data.Cluster().Spec.Cloud.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: data.Cluster().Spec.Cloud.AWS.SecretAccessKey})
	}
	if data.Cluster().Spec.Cloud.Openstack != nil {
		vars = append(vars, corev1.EnvVar{Name: "OS_AUTH_URL", Value: data.DC().Spec.Openstack.AuthURL})
		vars = append(vars, corev1.EnvVar{Name: "OS_USER_NAME", Value: data.Cluster().Spec.Cloud.Openstack.Username})
		vars = append(vars, corev1.EnvVar{Name: "OS_PASSWORD", Value: data.Cluster().Spec.Cloud.Openstack.Password})
		vars = append(vars, corev1.EnvVar{Name: "OS_DOMAIN_NAME", Value: data.Cluster().Spec.Cloud.Openstack.Domain})
		vars = append(vars, corev1.EnvVar{Name: "OS_TENANT_NAME", Value: data.Cluster().Spec.Cloud.Openstack.Tenant})
	}
	if data.Cluster().Spec.Cloud.Hetzner != nil {
		vars = append(vars, corev1.EnvVar{Name: "HZ_TOKEN", Value: data.Cluster().Spec.Cloud.Hetzner.Token})
	}
	if data.Cluster().Spec.Cloud.Digitalocean != nil {
		vars = append(vars, corev1.EnvVar{Name: "DO_TOKEN", Value: data.Cluster().Spec.Cloud.Digitalocean.Token})
	}
	if data.Cluster().Spec.Cloud.VSphere != nil {
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_ADDRESS", Value: data.DC().Spec.VSphere.Endpoint})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_USERNAME", Value: data.Cluster().Spec.Cloud.VSphere.InfraManagementUser.Username})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_PASSWORD", Value: data.Cluster().Spec.Cloud.VSphere.InfraManagementUser.Password})
	}
	if data.Cluster().Spec.Cloud.Packet != nil {
		vars = append(vars, corev1.EnvVar{Name: "PACKET_API_KEY", Value: data.Cluster().Spec.Cloud.Packet.APIKey})
		vars = append(vars, corev1.EnvVar{Name: "PACKET_PROJECT_ID", Value: data.Cluster().Spec.Cloud.Packet.ProjectID})
	}
	if data.Cluster().Spec.Cloud.GCP != nil {
		vars = append(vars, corev1.EnvVar{Name: "GOOGLE_SERVICE_ACCOUNT", Value: data.Cluster().Spec.Cloud.GCP.ServiceAccount})
	}
	return vars
}

func getFlags(clusterDNSIP string, nodeSettings provider.NodeSettings) []string {
	flags := []string{
		"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"-logtostderr",
		"-v", "4",
		"-cluster-dns", clusterDNSIP,
		"-internal-listen-address", "0.0.0.0:8085",
	}
	if len(nodeSettings.InsecureRegistries) > 0 {
		flags = append(flags, "-node-insecure-registries", strings.Join(nodeSettings.InsecureRegistries, ","))
	}
	if nodeSettings.HTTPProxy != "" {
		flags = append(flags, "-node-http-proxy", nodeSettings.HTTPProxy)
	}
	if nodeSettings.NoProxy != "" {
		flags = append(flags, "-node-no-proxy", nodeSettings.NoProxy)
	}
	if nodeSettings.PauseImage != "" {
		flags = append(flags, "-node-pause-image", nodeSettings.PauseImage)
	}
	if nodeSettings.HyperkubeImage != "" {
		flags = append(flags, "-node-hyperkube-image", nodeSettings.HyperkubeImage)
	}

	return flags
}
