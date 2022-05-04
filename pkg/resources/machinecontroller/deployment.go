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
	"strconv"
	"strings"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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
	Tag  = "v1.48.0"
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
	GetCSIMigrationFeatureGates() []string
	MachineControllerImageTag() string
	MachineControllerImageRepository() string
}

// DeploymentCreator returns the function to create and update the machine controller deployment.
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
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}
			deployment.Spec.Template.Spec = *wrappedPodSpec

			return deployment, nil
		}
	}
}

// DeploymentCreatorWithoutInitWrapper returns the function to create and update the machine controller deployment without the
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

			volumes := []corev1.Volume{getKubeconfigVolume(), getCABundleVolume()}
			dep.Spec.Template.Spec.Volumes = volumes

			podLabels, err := data.GetPodTemplateLabels(Name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %w", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/path":   "/metrics",
					"prometheus.io/port":   "8080",
				},
			}

			clusterDNSIP := resources.NodeLocalDNSCacheAddress
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

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}

			repository := data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/machine-controller"
			if r := data.MachineControllerImageRepository(); r != "" {
				repository = r
			}
			tag := Tag
			if t := data.MachineControllerImageTag(); t != "" {
				tag = t
			}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    Name,
					Image:   repository + ":" + tag,
					Command: []string{"/usr/local/bin/machine-controller"},
					Args:    getFlags(clusterDNSIP, data.DC().Node, data.Cluster().Spec.ContainerRuntime, data.Cluster().Spec.EnableOperatingSystemManager, data.Cluster().Spec.Features),
					Env: append(envVars, corev1.EnvVar{
						Name:  "PROBER_KUBECONFIG",
						Value: "/etc/kubernetes/kubeconfig/kubeconfig",
					}),
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/readyz",
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
						{
							Name:      resources.CABundleConfigMapName,
							MountPath: "/etc/kubernetes/pki/ca-bundle",
							ReadOnly:  true,
						},
					},
				},
			}

			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, controllerResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
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
		vars = append(vars, corev1.EnvVar{Name: "AWS_ASSUME_ROLE_ARN", Value: credentials.AWS.AssumeRoleARN})
		vars = append(vars, corev1.EnvVar{Name: "AWS_ASSUME_ROLE_EXTERNAL_ID", Value: credentials.AWS.AssumeRoleExternalID})
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
		vars = append(vars, corev1.EnvVar{Name: "OS_PROJECT_NAME", Value: credentials.Openstack.Project})
		vars = append(vars, corev1.EnvVar{Name: "OS_PROJECT_ID", Value: credentials.Openstack.ProjectID})
		vars = append(vars, corev1.EnvVar{Name: "OS_APPLICATION_CREDENTIAL_ID", Value: credentials.Openstack.ApplicationCredentialID})
		vars = append(vars, corev1.EnvVar{Name: "OS_APPLICATION_CREDENTIAL_SECRET", Value: credentials.Openstack.ApplicationCredentialSecret})
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
	if data.Cluster().Spec.Cloud.Anexia != nil {
		vars = append(vars, corev1.EnvVar{Name: "ANEXIA_TOKEN", Value: credentials.Anexia.Token})
	}
	if data.Cluster().Spec.Cloud.Nutanix != nil {
		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_ENDPOINT", Value: data.DC().Spec.Nutanix.Endpoint})
		if port := data.DC().Spec.Nutanix.Port; port != nil {
			vars = append(vars, corev1.EnvVar{Name: "NUTANIX_PORT", Value: strconv.Itoa(int(*port))})
		}
		if data.DC().Spec.Nutanix.AllowInsecure {
			vars = append(vars, corev1.EnvVar{Name: "NUTANIX_INSECURE", Value: "true"})
		}

		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_USERNAME", Value: credentials.Nutanix.Username})
		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_PASSWORD", Value: credentials.Nutanix.Password})
		if proxyURL := credentials.Nutanix.ProxyURL; proxyURL != "" {
			vars = append(vars, corev1.EnvVar{Name: "NUTANIX_PROXY_URL", Value: proxyURL})
		}

		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_CLUSTER_NAME", Value: data.Cluster().Spec.Cloud.Nutanix.ClusterName})
	}
	vars = append(vars, resources.GetHTTPProxyEnvVarsFromSeed(data.Seed(), data.Cluster().Address.InternalName)...)

	vars = resources.SanitizeEnvVars(vars)
	vars = append(vars, corev1.EnvVar{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}})

	return vars, nil
}

func getFlags(clusterDNSIP string, nodeSettings *kubermaticv1.NodeSettings, cri string, enableOperatingSystemManager bool, features map[string]bool) []string {
	flags := []string{
		"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"-logtostderr",
		"-v", "4",
		"-cluster-dns", clusterDNSIP,
		"-health-probe-address", "0.0.0.0:8085",
		"-metrics-address", "0.0.0.0:8080",
		"-ca-bundle", "/etc/kubernetes/pki/ca-bundle/ca-bundle.pem",
		"-node-csr-approver",
	}

	if nodeSettings != nil {
		if len(nodeSettings.InsecureRegistries) > 0 {
			flags = append(flags, "-node-insecure-registries", strings.Join(nodeSettings.InsecureRegistries, ","))
		}
		if len(nodeSettings.RegistryMirrors) > 0 {
			flags = append(flags, "-node-registry-mirrors", strings.Join(nodeSettings.RegistryMirrors, ","))
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
	}

	externalCloudProvider := features[kubermaticv1.ClusterFeatureExternalCloudProvider]
	if externalCloudProvider {
		flags = append(flags, "-node-external-cloud-provider")
	}

	if cri != "" {
		flags = append(flags, "-node-container-runtime", cri)
	}

	// Machine Controller will use OSM for managing machine's provisioning and bootstrapping configurations
	if enableOperatingSystemManager {
		flags = append(flags, "-use-osm")
	}

	return flags
}
