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
	"slices"
	"strings"

	semverlib "github.com/Masterminds/semver/v3"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

var (
	controllerResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.OperatingSystemManagerContainerName: {
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
	Tag = "v1.5.0"
)

type operatingSystemManagerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	Cluster() *kubermaticv1.Cluster
	RewriteImage(string) (string, error)
	NodeLocalDNSCacheEnabled() bool
	GetCSIMigrationFeatureGates(version *semverlib.Version) []string
	DC() *kubermaticv1.Datacenter
	ComputedNodePortRange() string
	OperatingSystemManagerImageTag() string
	OperatingSystemManagerImageRepository() string
}

// DeploymentReconciler returns the function to create and update the operating system manager deployment.
func DeploymentReconciler(data operatingSystemManagerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.OperatingSystemManagerDeploymentName, func(in *appsv1.Deployment) (*appsv1.Deployment, error) {
			_, creator := DeploymentReconcilerWithoutInitWrapper(data)()
			deployment, err := creator(in)
			if err != nil {
				return nil, err
			}

			deployment.Spec.Template, err = apiserver.IsRunningWrapper(data, deployment.Spec.Template, sets.New(resources.OperatingSystemManagerContainerName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

			return deployment, nil
		}
	}
}

// DeploymentReconcilerWithoutInitWrapper returns the function to create and update the operating system manager deployment without the
// wrapper that checks for apiserver availabiltiy. This allows to adjust the command.
func DeploymentReconcilerWithoutInitWrapper(data operatingSystemManagerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.OperatingSystemManagerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(resources.OperatingSystemManagerDeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.OperatingSystemManager != nil && data.Cluster().Spec.ComponentsOverride.OperatingSystemManager.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.OperatingSystemManager.Replicas
			}
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			volumes := []corev1.Volume{getKubeconfigVolume()}
			dep.Spec.Template.Spec.Volumes = volumes

			podLabels, err := data.GetPodTemplateLabels(resources.OperatingSystemManagerDeploymentName, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %w", err)
			}

			kubernetes.EnsureLabels(&dep.Spec.Template, podLabels)
			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/path":   "/metrics",
				"prometheus.io/port":   "8080",
			})

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

			cloudProviderName, err := kubermaticv1helper.ClusterCloudProviderName(data.Cluster().Spec.Cloud)
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

			repository := registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/operating-system-manager"))
			if r := data.OperatingSystemManagerImageRepository(); r != "" {
				repository = r
			}
			tag := Tag
			if t := data.OperatingSystemManagerImageTag(); t != "" {
				tag = t
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.OperatingSystemManagerContainerName,
					Image:   repository + ":" + tag,
					Command: []string{"/usr/local/bin/osm-controller"},
					Args:    getFlags(data.DC().Node, cs, data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider], data.GetCSIMigrationFeatureGates(nil), data.Cluster().Spec.ImagePullSecret),
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
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: resources.Bool(false),
						ReadOnlyRootFilesystem:   resources.Bool(true),
						RunAsNonRoot:             resources.Bool(true),
						RunAsUser:                resources.Int64(65534),
						RunAsGroup:               resources.Int64(65534),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								corev1.Capability("ALL"),
							},
						},
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			}

			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, controllerResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			if data.Cluster().Spec.ComponentsOverride.OperatingSystemManager != nil && len(data.Cluster().Spec.ComponentsOverride.OperatingSystemManager.Tolerations) > 0 {
				dep.Spec.Template.Spec.Tolerations = data.Cluster().Spec.ComponentsOverride.OperatingSystemManager.Tolerations
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
		"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"-cluster-dns", cs.clusterDNSIP,
		"-health-probe-address", "0.0.0.0:8085",
		"-metrics-address", "0.0.0.0:8080",
		"-namespace", "kube-system",
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
		ref.SecretKeyRef.Optional = ptr.To(true)

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
		vars = append(vars, corev1.EnvVar{Name: "KUBEVIRT_KUBECONFIG", ValueFrom: refTo(resources.KubeVirtKubeconfig)})
	}

	return resources.SanitizeEnvVars(vars), nil
}

func getContainerdFlags(crid *kubermaticv1.ContainerRuntimeContainerd) []string {
	var (
		registries, flags []string
	)

	// fetch all keys from the map and sort them
	// for stable order.
	for registry := range crid.Registries {
		registries = append(registries, registry)
	}

	slices.Sort(registries)

	for _, registry := range registries {
		for _, endpoint := range crid.Registries[registry].Mirrors {
			flags = append(flags, fmt.Sprintf("-node-containerd-registry-mirrors=%s=%s", registry, endpoint))
		}
	}

	return flags
}
