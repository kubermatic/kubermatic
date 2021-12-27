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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
				corev1.ResourceCPU:    resource.MustParse("2"),
			},
		},
	}
)

const (
	Name = "operating-system-manager"
	Tag  = "v0.2.0"
)

type operatingSystemManagerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	Cluster() *kubermaticv1.Cluster
	ImageRegistry(string) string
	NodeLocalDNSCacheEnabled() bool
	DC() *kubermaticv1.Datacenter
	ComputedNodePortRange() string
}

// DeploymentCreator returns the function to create and update the machine controller deployment
func DeploymentCreator(data operatingSystemManagerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.OperatingSystemManagerDeploymentName, func(in *appsv1.Deployment) (*appsv1.Deployment, error) {
			_, creator := DeploymentCreatorWithoutInitWrapper(data)()
			deployment, err := creator(in)
			if err != nil {
				return nil, err
			}
			// TODO(mq): add osm crds to wait for
			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, deployment.Spec.Template.Spec, sets.NewString(Name), "")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			deployment.Spec.Template.Spec = *wrappedPodSpec

			return deployment, nil
		}
	}
}

// DeploymentCreatorWithoutInitWrapper returns the function to create and update the machine controller deployment without the
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

			volumes := []corev1.Volume{getKubeconfigVolume(), getCABundleVolume()}
			dep.Spec.Template.Spec.Volumes = volumes

			podLabels, err := data.GetPodTemplateLabels(Name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
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

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}

			repository := data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/operating-system-manager"

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

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    Name,
					Image:   repository + ":" + Tag,
					Command: []string{"/usr/local/bin/operating-system-manager"},
					Args:    getFlags(data.DC().Node, cs),
					Env: []corev1.EnvVar{
						{
							Name:  "KUBECONFIG",
							Value: "/etc/kubernetes/kubeconfig/kubeconfig",
						},
					},
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
							Name:      resources.OperatingSystemManagerKubeconfigSecretName,
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
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, controllerResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
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

func getFlags(nodeSettings *kubermaticv1.NodeSettings, cs *clusterSpec) []string {
	flags := []string{
		"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"-cluster-dns", cs.clusterDNSIP,
		"-cluster-name", cs.Name,
		"-external-namespace", fmt.Sprintf("%s-%s", "cluster", cs.Name),
		"-external-cloud-provider", cs.cloudProvider,
	}

	if nodeSettings != nil {
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

	if cs.podCidr != "" {
		flags = append(flags, "-pod-cidr", cs.podCidr)
	}

	if cs.nodePortRange != "" {
		flags = append(flags, "-node-port-range", cs.nodePortRange)
	}

	if cs.containerRuntime != "" {
		flags = append(flags, "-container-runtime", cs.containerRuntime)
	}

	return flags
}
