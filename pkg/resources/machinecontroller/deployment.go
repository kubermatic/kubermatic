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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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
	// TODO: update to tagged machine-controller version.
	Tag = "e7e4ba8eb535086374108ed235c2e33bab7f4292"
)

type machinecontrollerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	RewriteImage(string) (string, error)
	Cluster() *kubermaticv1.Cluster
	ClusterIPByServiceName(string) (string, error)
	DC() *kubermaticv1.Datacenter
	NodeLocalDNSCacheEnabled() bool
	Seed() *kubermaticv1.Seed
	GetCSIMigrationFeatureGates() []string
	MachineControllerImageTag() string
	MachineControllerImageRepository() string
	GetEnvVars() ([]corev1.EnvVar, error)
}

// DeploymentReconciler returns the function to create and update the machine controller deployment.
func DeploymentReconciler(data machinecontrollerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.MachineControllerDeploymentName, func(in *appsv1.Deployment) (*appsv1.Deployment, error) {
			_, creator := DeploymentReconcilerWithoutInitWrapper(data)()
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

// DeploymentReconcilerWithoutInitWrapper returns the function to create and update the machine controller deployment without the
// wrapper that checks for apiserver availabiltiy. This allows to adjust the command.
func DeploymentReconcilerWithoutInitWrapper(data machinecontrollerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
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

			envVars, err := data.GetEnvVars()
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}

			repository := registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/machine-controller"))
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
					Args:    getFlags(clusterDNSIP, data.DC().Node, data.Cluster().Spec.ContainerRuntime, data.Cluster().Spec.ImagePullSecret, data.Cluster().Spec.IsOperatingSystemManagerEnabled(), data.Cluster().Spec.Features),
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

func getFlags(clusterDNSIP string, nodeSettings *kubermaticv1.NodeSettings, cri string, imagePullSecret *corev1.SecretReference, enableOperatingSystemManager bool, features map[string]bool) []string {
	flags := []string{
		"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
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

	if imagePullSecret != nil {
		flags = append(flags, "-node-registry-credentials-secret", fmt.Sprintf("%s/%s", imagePullSecret.Namespace, imagePullSecret.Name))
	}

	// Machine Controller will use OSM for managing machine's provisioning and bootstrapping configurations
	if enableOperatingSystemManager {
		flags = append(flags, "-use-osm")
	}

	return flags
}
