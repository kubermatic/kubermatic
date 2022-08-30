/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package operatingsystemmanager

import (
	"fmt"
	"strings"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

var (
	controllerResourceRequirements = map[string]*corev1.ResourceRequirements{
		Name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("1"),
			},
		},
	}
)

const (
	Name = "operating-system-manager"
	Tag  = "v1.0.0"
)

type operatingSystemManagerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	Cluster() *kubermaticv1.Cluster
	ImageRegistry(string) string
	NodeLocalDNSCacheEnabled() bool
	GetCSIMigrationFeatureGates() []string
	DC() *kubermaticv1.Datacenter
	ComputedNodePortRange() string
	OperatingSystemManagerImageTag() string
	OperatingSystemManagerImageRepository() string
}

// DeploymentCreator returns the function to create and update the operating system manager deployment.
func DeploymentCreator(data operatingSystemManagerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.OperatingSystemManagerDeploymentName, func(in *appsv1.Deployment) (*appsv1.Deployment, error) {
			_, creator := DeploymentCreatorWithoutInitWrapper(data)()
			deployment, err := creator(in)
			if err != nil {
				return nil, err
			}

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, deployment.Spec.Template.Spec, sets.NewString(Name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}
			deployment.Spec.Template.Spec = *wrappedPodSpec

			return deployment, nil
		}
	}
}

// DeploymentCreatorWithoutInitWrapper returns the function to create and update the operating system manager deployment without the
// wrapper that checks for apiserver availabiltiy. This allows to adjust the command.
func DeploymentCreatorWithoutInitWrapper(data operatingSystemManagerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.OperatingSystemManagerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.OperatingSystemManagerDeploymentName
			dep.Labels = resources.BaseAppLabels(Name, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(Name, nil),
			}

			volumes := []corev1.Volume{getKubeconfigVolume()}
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

			cloudProviderName, err := provider.ClusterCloudProviderName(data.Cluster().Spec.Cloud)
			if err != nil {
				return nil, err
			}

			var podCidr string
			if len(data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks) > 0 {
				podCidr = data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0]
			}

			cs := &clusterSpec{
				Name:             data.Cluster().Name,
				clusterDNSIP:     clusterDNSIP,
				containerRuntime: data.Cluster().Spec.ContainerRuntime,
				cloudProvider:    cloudProviderName,
				podCidr:          podCidr,
				nodePortRange:    data.ComputedNodePortRange(),
			}

			repository := data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/operating-system-manager"
			if r := data.OperatingSystemManagerImageRepository(); r != "" {
				repository = r
			}
			tag := Tag
			if t := data.OperatingSystemManagerImageTag(); t != "" {
				tag = t
			}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    Name,
					Image:   repository + ":" + tag,
					Command: []string{"/usr/local/bin/osm-controller"},
					Args:    getFlags(data.DC().Node, cs, data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider], data.GetCSIMigrationFeatureGates(), data.Cluster().Spec.ImagePullSecret),
					Env:     envVars,
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
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
					ReadinessProbe: &corev1.Probe{
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
							Name:      resources.OperatingSystemManagerKubeconfigSecretName,
							MountPath: "/etc/kubernetes/worker-kubeconfig",
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

type clusterSpec struct {
	Name             string
	clusterDNSIP     string
	containerRuntime string
	cloudProvider    string
	nodePortRange    string
	podCidr          string
}

func getFlags(nodeSettings *kubermaticv1.NodeSettings, cs *clusterSpec, externalCloudProvider bool, csiMigrationFeatureGates []string, imagePullSecret *corev1.SecretReference) []string {
	flags := []string{
		"-worker-cluster-kubeconfig", "/etc/kubernetes/worker-kubeconfig/kubeconfig",
		"-cluster-dns", cs.clusterDNSIP,
		"-health-probe-address", "0.0.0.0:8085",
		"-metrics-address", "0.0.0.0:8080",
		"-namespace", fmt.Sprintf("%s-%s", "cluster", cs.Name),
	}

	if externalCloudProvider {
		flags = append(flags, "-external-cloud-provider")
	}

	if nodeSettings != nil {
		if len(nodeSettings.InsecureRegistries) > 0 {
			flags = append(flags, "-node-insecure-registries", strings.Join(nodeSettings.InsecureRegistries, ","))
		}
		if nodeSettings.ContainerdRegistryMirrors != nil {
			flags = append(flags, getContainerdFlags(nodeSettings.ContainerdRegistryMirrors)...)
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
			flags = append(flags, "-pause-image", nodeSettings.PauseImage)
		}
	}

	if len(csiMigrationFeatureGates) > 0 {
		flags = append(flags, "-node-kubelet-feature-gates", strings.Join(csiMigrationFeatureGates, ","))
	}

	if imagePullSecret != nil {
		flags = append(flags, "-node-registry-credentials-secret", fmt.Sprintf("%s/%s", imagePullSecret.Namespace, imagePullSecret.Name))
	}

	if cs.containerRuntime != "" {
		flags = append(flags, "-container-runtime", cs.containerRuntime)
	}

	return flags
}

func getEnvVars(data operatingSystemManagerData) ([]corev1.EnvVar, error) {
	refTo := func(key string) *corev1.EnvVarSource {
		return &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: resources.ClusterCloudCredentialsSecretName,
				},
				Key: key,
			},
		}
	}

	optionalRefTo := func(key string) *corev1.EnvVarSource {
		ref := refTo(key)
		ref.SecretKeyRef.Optional = pointer.Bool(true)

		return ref
	}

	var vars []corev1.EnvVar
	if data.Cluster().Spec.Cloud.Azure != nil {
		vars = append(vars, corev1.EnvVar{Name: "AZURE_CLIENT_ID", ValueFrom: refTo(resources.AzureClientID)})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_CLIENT_SECRET", ValueFrom: refTo(resources.AzureClientSecret)})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_TENANT_ID", ValueFrom: refTo(resources.AzureTenantID)})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_SUBSCRIPTION_ID", ValueFrom: refTo(resources.AzureSubscriptionID)})
	}
	if data.Cluster().Spec.Cloud.Openstack != nil {
		vars = append(vars, corev1.EnvVar{Name: "OS_AUTH_URL", Value: data.DC().Spec.Openstack.AuthURL})
		vars = append(vars, corev1.EnvVar{Name: "OS_USER_NAME", ValueFrom: refTo(resources.OpenstackUsername)})
		vars = append(vars, corev1.EnvVar{Name: "OS_PASSWORD", ValueFrom: refTo(resources.OpenstackPassword)})
		vars = append(vars, corev1.EnvVar{Name: "OS_DOMAIN_NAME", ValueFrom: refTo(resources.OpenstackDomain)})
		vars = append(vars, corev1.EnvVar{Name: "OS_PROJECT_NAME", ValueFrom: optionalRefTo(resources.OpenstackProject)})
		vars = append(vars, corev1.EnvVar{Name: "OS_PROJECT_ID", ValueFrom: optionalRefTo(resources.OpenstackProjectID)})
		vars = append(vars, corev1.EnvVar{Name: "OS_APPLICATION_CREDENTIAL_ID", ValueFrom: optionalRefTo(resources.OpenstackApplicationCredentialID)})
		vars = append(vars, corev1.EnvVar{Name: "OS_APPLICATION_CREDENTIAL_SECRET", ValueFrom: optionalRefTo(resources.OpenstackApplicationCredentialSecret)})
	}
	if data.Cluster().Spec.Cloud.VSphere != nil {
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_ADDRESS", Value: data.DC().Spec.VSphere.Endpoint})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_USERNAME", ValueFrom: refTo(resources.VsphereUsername)})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_PASSWORD", ValueFrom: refTo(resources.VspherePassword)})
	}
	if data.Cluster().Spec.Cloud.GCP != nil {
		vars = append(vars, corev1.EnvVar{Name: "GOOGLE_SERVICE_ACCOUNT", ValueFrom: refTo(resources.GCPServiceAccount)})
	}
	if data.Cluster().Spec.Cloud.Kubevirt != nil {
		vars = append(vars, corev1.EnvVar{Name: "KUBEVIRT_KUBECONFIG", ValueFrom: refTo(resources.KubevirtKubeConfig)})
	}

	return resources.SanitizeEnvVars(vars), nil
}

func getContainerdFlags(crid *kubermaticv1.ContainerRuntimeContainerd) []string {
	var flags []string
	for registry, mirror := range crid.Registries {
		var flag string
		for _, endpoint := range mirror.Mirrors {
			flag = fmt.Sprintf("-node-containerd-registry-mirrors=%s=%s", registry, endpoint)
			flags = append(flags, flag)
		}
	}

	return flags
}
