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

package clusterautoscaler

import (
	"fmt"

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
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.ClusterAutoscalerDeploymentName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("25m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
		},
	}
)

type clusterautoscalerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
	Cluster() *kubermaticv1.Cluster
}

// DeploymentCreator returns the function to create and update the cluster-autoscaler deployment
func DeploymentCreator(data clusterautoscalerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.ClusterAutoscalerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			tag := getTag(data.Cluster())
			if tag == "" {
				return nil, fmt.Errorf("no matching autoscaler tag found for version %d", data.Cluster().Spec.Version.Semver().Minor())
			}

			dep.Name = resources.ClusterAutoscalerDeploymentName
			dep.Labels = resources.BaseAppLabels(resources.ClusterAutoscalerDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(resources.ClusterAutoscalerDeploymentName, nil),
			}

			volumes := []corev1.Volume{
				{
					Name: resources.ClusterAutoscalerKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.ClusterAutoscalerKubeconfigSecretName,
						},
					},
				},
			}
			podLabels, err := data.GetPodTemplateLabels(resources.ClusterAutoscalerDeploymentName,
				volumes, nil)
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

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.ClusterAutoscalerDeploymentName,
					Image:   data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/kubernetes-cluster-autoscaler:" + tag,
					Command: []string{"/cluster-autoscaler"},
					Args: []string{
						"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--leader-elect-resource-lock", "configmaps",
						// PercentageUsed threshold. If the current utilization of a node is above this, the CA will never
						// scale it down. Default is 0.5. Increased, because otherwise small nodes never get scaled down
						// because the DS pods on them alone manage to get the utilization above the 0.5 threshold.
						"--scale-down-utilization-threshold", "0.7",
						// For debugging you can add the following to increase verbosity and make scale down kick in without
						// delay:
						// -v=4 --scale-down-delay-after-failure=1s --scale-down-delay-after-add=1s
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health-check",
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
							Name:      resources.ClusterAutoscalerKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
					},
				},
			}

			// This likely won't be enough for bigger clusters, see https://github.com/kubermatic/kubermatic/issues/3568
			// for details on how we want to fix this: https://github.com/kubermatic/kubermatic/issues/3568
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(resources.ClusterAutoscalerDeploymentName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

// getTag returns the correct tag for the cluster version. We need to have a distinct CA
// version for each Kubernetes version, because the CA imports the scheduler code and the
// behaviour of that imported code has to match with what the actual scheduler does
func getTag(cluster *kubermaticv1.Cluster) string {
	switch cluster.Spec.Version.Semver().Minor() {
	case 14:
		return "fe5bee817ad9d37c8ce5e473af201c2f3fdf5b94-1"
	}

	return ""
}
