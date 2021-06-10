/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package machinecontroller

import (
	"fmt"
	"strings"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	controllerResourceRequirements = map[string]*corev1.ResourceRequirements{
		Name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("25m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("2"),
			},
		},
	}
)

const (
	Name = "machine-controller"

	tag = "v1.19.2"

	NodeLocalDNSCacheAddress = "169.254.20.10"
)

type machinecontrollerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	ImageRegistry(string) string
	Cluster() *kubermaticv1.Cluster
	ClusterIPByServiceName(string) (string, error)
	DC() *kubermaticv1.Datacenter
	NodeLocalDNSCacheEnabled() bool
	Seed() *kubermaticv1.Seed
}

// DeploymentCreator returns the function to create and update the machine controller deployment
func DeploymentCreator(data machinecontrollerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.MachineControllerDeploymentName, func(in *appsv1.Deployment) (*appsv1.Deployment, error) {
			_, creator := DeploymentCreatorWithoutInitWrapper(data)()
			deployment, err := creator(in)
			if err != nil {
				return nil, err
			}
			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, deployment.Spec.Template.Spec, sets.NewString(Name), "Machine,cluster.k8s.io/v1alpha1")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			deployment.Spec.Template.Spec = *wrappedPodSpec

			return deployment, nil
		}
	}
}

// DeploymentCreator returns the function to create and update the machine controller deployment without the
// wrapper that checks for apiserver availabiltiy. This allows to adjust the command.
func DeploymentCreatorWithoutInitWrapper(data machinecontrollerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.MachineControllerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.MachineControllerDeploymentName
			dep.Labels = resources.BaseAppLabels(Name, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(Name, nil),
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

			clusterDNSIP := NodeLocalDNSCacheAddress
			if !data.NodeLocalDNSCacheEnabled() {
				clusterDNSIP, err = resources.UserClusterDNSResolverIP(data.Cluster())
				if err != nil {
					return nil, err
				}
			}

			envVars, err := getEnvVars(data)
			if err != nil {
				return nil, err
			}

			externalCloudProvider := data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    Name,
					Image:   data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/machine-controller:" + tag,
					Command: []string{"/usr/local/bin/machine-controller"},
					Args:    getFlags(clusterDNSIP, data.DC().Node, externalCloudProvider),
					Env: append(envVars, corev1.EnvVar{
						Name:  "KUBECONFIG",
						Value: "/etc/kubernetes/kubeconfig/kubeconfig",
					}),
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
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, controllerResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

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
			},
		},
	}
}

func getEnvVars(data machinecontrollerData) ([]corev1.EnvVar, error) {
	credentials, err := resources.GetCredentials(data)
	if err != nil {
		return nil, err
	}

	var vars []corev1.EnvVar
	if data.Cluster().Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: credentials.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: credentials.AWS.SecretAccessKey})
	}
	if data.Cluster().Spec.Cloud.Azure != nil {
		vars = append(vars, corev1.EnvVar{Name: "AZURE_CLIENT_ID", Value: credentials.Azure.ClientID})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_CLIENT_SECRET", Value: credentials.Azure.ClientSecret})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_TENANT_ID", Value: credentials.Azure.TenantID})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_SUBSCRIPTION_ID", Value: credentials.Azure.SubscriptionID})
	}
	if data.Cluster().Spec.Cloud.Openstack != nil {
		vars = append(vars, corev1.EnvVar{Name: "OS_AUTH_URL", Value: data.DC().Spec.Openstack.AuthURL})
		vars = append(vars, corev1.EnvVar{Name: "OS_USER_NAME", Value: credentials.Openstack.Username})
		vars = append(vars, corev1.EnvVar{Name: "OS_PASSWORD", Value: credentials.Openstack.Password})
		vars = append(vars, corev1.EnvVar{Name: "OS_DOMAIN_NAME", Value: credentials.Openstack.Domain})
		vars = append(vars, corev1.EnvVar{Name: "OS_TENANT_NAME", Value: credentials.Openstack.Tenant})
		vars = append(vars, corev1.EnvVar{Name: "OS_TENANT_ID", Value: credentials.Openstack.TenantID})
	}
	if data.Cluster().Spec.Cloud.Hetzner != nil {
		vars = append(vars, corev1.EnvVar{Name: "HZ_TOKEN", Value: credentials.Hetzner.Token})
	}
	if data.Cluster().Spec.Cloud.Digitalocean != nil {
		vars = append(vars, corev1.EnvVar{Name: "DO_TOKEN", Value: credentials.Digitalocean.Token})
	}
	if data.Cluster().Spec.Cloud.VSphere != nil {
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_ADDRESS", Value: data.DC().Spec.VSphere.Endpoint})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_USERNAME", Value: credentials.VSphere.Username})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_PASSWORD", Value: credentials.VSphere.Password})
	}
	if data.Cluster().Spec.Cloud.Packet != nil {
		vars = append(vars, corev1.EnvVar{Name: "PACKET_API_KEY", Value: credentials.Packet.APIKey})
		vars = append(vars, corev1.EnvVar{Name: "PACKET_PROJECT_ID", Value: credentials.Packet.ProjectID})
	}
	if data.Cluster().Spec.Cloud.GCP != nil {
		vars = append(vars, corev1.EnvVar{Name: "GOOGLE_SERVICE_ACCOUNT", Value: credentials.GCP.ServiceAccount})
	}
	if data.Cluster().Spec.Cloud.Kubevirt != nil {
		vars = append(vars, corev1.EnvVar{Name: "KUBEVIRT_KUBECONFIG", Value: credentials.Kubevirt.KubeConfig})
	}
	if data.Cluster().Spec.Cloud.Alibaba != nil {
		vars = append(vars, corev1.EnvVar{Name: "ALIBABA_ACCESS_KEY_ID", Value: credentials.Alibaba.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "ALIBABA_ACCESS_KEY_SECRET", Value: credentials.Alibaba.AccessKeySecret})
	}
	vars = append(vars, resources.GetHTTPProxyEnvVarsFromSeed(data.Seed(), data.Cluster().Address.InternalName)...)
	return vars, nil
}

func getFlags(clusterDNSIP string, nodeSettings *kubermaticv1.NodeSettings, externalCloudProvider bool) []string {
	flags := []string{
		"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"-logtostderr",
		"-v", "4",
		"-cluster-dns", clusterDNSIP,
		"-internal-listen-address", "0.0.0.0:8085",
	}

	if nodeSettings != nil {
		if len(nodeSettings.InsecureRegistries) > 0 {
			flags = append(flags, "-node-insecure-registries", strings.Join(nodeSettings.InsecureRegistries, ","))
		}
		if !nodeSettings.HTTPProxy.Empty() {
			flags = append(flags, "-node-http-proxy", nodeSettings.HTTPProxy.String())
		}
		if !nodeSettings.NoProxy.Empty() {
			flags = append(flags, "-node-no-proxy", nodeSettings.NoProxy.String())
		}
		if nodeSettings.PauseImage != "" {
			flags = append(flags, "-node-pause-image", nodeSettings.PauseImage)
		}

		// TODO(kron4eg): deprecate and remove this
		if nodeSettings.HyperkubeImage != "" {
			flags = append(flags, "-node-hyperkube-image", nodeSettings.HyperkubeImage)
		}
	}

	if externalCloudProvider {
		flags = append(flags, "-external-cloud-provider=true")
	}

	return flags
}
