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

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var controllerResourceRequirements = map[string]*corev1.ResourceRequirements{
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

const (
	Name = "machine-controller"
	Tag  = "v1.64.1"
)

type machinecontrollerData interface {
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	RewriteImage(string) (string, error)
	Cluster() *kubermaticv1.Cluster
	ClusterIPByServiceName(string) (string, error)
	DC() *kubermaticv1.Datacenter
	NodeLocalDNSCacheEnabled() bool
	Seed() *kubermaticv1.Seed
	GetCSIMigrationFeatureGates(version *semverlib.Version) []string
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
			deployment.Spec.Template, err = apiserver.IsRunningWrapper(data, deployment.Spec.Template, sets.New(Name), "cluster.k8s.io,v1alpha1,machines,kube-system")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

			return deployment, nil
		}
	}
}

// DeploymentReconcilerWithoutInitWrapper returns the function to create and update the machine controller deployment without the
// wrapper that checks for apiserver availability. This allows to adjust the command.
func DeploymentReconcilerWithoutInitWrapper(data machinecontrollerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.MachineControllerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(Name, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				"prometheus.io/scrape":                 "true",
				"prometheus.io/path":                   "/metrics",
				"prometheus.io/port":                   "8080",
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "temp",
			})

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

			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsNonRoot: resources.Bool(true),
				RunAsUser:    resources.Int64(65534),
				RunAsGroup:   resources.Int64(65534),
				FSGroup:      resources.Int64(65534),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    Name,
					Image:   repository + ":" + tag,
					Command: []string{"/usr/local/bin/machine-controller"},
					Args:    getFlags(data.Cluster().Spec.Features),
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
						{
							Name:      "temp",
							MountPath: "/tmp",
						},
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: resources.Bool(false),
						ReadOnlyRootFilesystem:   resources.Bool(true),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								corev1.Capability("ALL"),
							},
						},
					},
				},
			}

			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				getKubeconfigVolume(),
				getCABundleVolume(),
				{
					Name: "temp",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}

			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, controllerResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func getFlags(features map[string]bool) []string {
	flags := []string{
		"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"-health-probe-address", "0.0.0.0:8085",
		"-metrics-address", "0.0.0.0:8080",
		"-ca-bundle", "/etc/kubernetes/pki/ca-bundle/ca-bundle.pem",
		"-node-csr-approver",
	}

	externalCloudProvider := features[kubermaticv1.ClusterFeatureExternalCloudProvider]
	if externalCloudProvider {
		flags = append(flags, "-node-external-cloud-provider")
	}

	return flags
}
