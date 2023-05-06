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

package cloudcontroller

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	OpenstackCCMDeploymentName = "openstack-cloud-controller-manager"
)

var (
	osResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

func openStackDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return OpenstackCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = OpenstackCCMDeploymentName
			dep.Labels = resources.BaseAppLabels(OpenstackCCMDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(OpenstackCCMDeploymentName, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(OpenstackCCMDeploymentName, dep.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err =
				resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			ccmImage, err := getOpenStackCCMImage(data.Cluster().Status.Versions.ControlPlane)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.AutomountServiceAccountToken = pointer.Bool(false)
			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled(), true)
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:         ccmContainerName,
					Image:        registry.Must(data.RewriteImage(ccmImage)),
					Command:      []string{"/bin/openstack-cloud-controller-manager"},
					Args:         getOSFlags(data),
					Env:          getEnvVars(),
					VolumeMounts: getVolumeMounts(true),
					SecurityContext: &corev1.SecurityContext{
						RunAsUser: pointer.Int64(1001),
					},
				},
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				ccmContainerName: osResourceRequirements.DeepCopy(),
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func getOSFlags(data *resources.TemplateData) []string {
	flags := []string{
		"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		"--v=1",
		"--cloud-config=/etc/kubernetes/cloud/config",
		"--cloud-provider=openstack",
	}
	if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] {
		flags = append(flags, "--cluster-name", data.Cluster().Name)
	}
	return flags
}

func getOpenStackCCMImage(version semver.Semver) (string, error) {
	switch version.MajorMinor() {
	case v123:
		return resources.RegistryDocker + "/k8scloudprovider/openstack-cloud-controller-manager:v1.23.4", nil
	case v124:
		return resources.RegistryK8S + "/provider-os/openstack-cloud-controller-manager:v1.24.6", nil
	case v125:
		return resources.RegistryK8S + "/provider-os/openstack-cloud-controller-manager:v1.25.5", nil
	case v126:
		return resources.RegistryK8S + "/provider-os/openstack-cloud-controller-manager:v1.26.2", nil
	default:
		return "", fmt.Errorf("%v is not yet supported", version)
	}
}
