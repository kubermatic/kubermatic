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

package cloudcontroller

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	VSphereCCMDeploymentName = "vsphere-cloud-controller-manager"
)

var (
	vsphereCPIResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("200m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

func vsphereDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return VSphereCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = VSphereCCMDeploymentName
			dep.Labels = resources.BaseAppLabels(VSphereCCMDeploymentName, nil)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(VSphereCCMDeploymentName, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(VSphereCCMDeploymentName, dep.Spec.Template.Spec.Volumes, map[string]string{
				"component": "cloud-controller-manager",
				"tier":      "control-plane",
			})
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

			version := VSphereCCMVersion(data.Cluster().Status.Versions.ControlPlane)
			container := getVSphereCCMContainer(version, data)

			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)
			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled(), true)
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				container,
			}

			return dep, nil
		}
	}
}

func getVSphereCCMContainer(version string, data *resources.TemplateData) corev1.Container {
	controllerManagerImage := registry.Must(data.RewriteImage(resources.RegistryGCR + "/cloud-provider-vsphere/cpi/release/manager:v" + version))
	c := corev1.Container{
		Name:  ccmContainerName,
		Image: controllerManagerImage,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: ptr.To[int64](1001),
		},
		Command: []string{
			"/bin/vsphere-cloud-controller-manager",
		},
		Args: []string{
			"--v=2",
			"--cloud-provider=vsphere",
			"--cloud-config=/etc/kubernetes/cloud/config",
			"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		},
		Env:          getEnvVars(),
		VolumeMounts: getVolumeMounts(true),
		Resources:    vsphereCPIResourceRequirements,
	}
	if data.Cluster().IsDualStack() {
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "ENABLE_ALPHA_DUAL_STACK",
			Value: "true",
		})
	}
	if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] {
		c.Args = append(c.Args, "--cluster-name", data.Cluster().Name)
	}

	return c
}

func VSphereCCMVersion(version semver.Semver) string {
	// https://github.com/kubernetes/cloud-provider-vsphere/releases
	switch version.MajorMinor() {
	case v126:
		return "1.26.2"
	case v127:
		return "1.27.0"
	case v128:
		return "1.28.0"
	case v129:
		fallthrough
	default:
		return "1.29.0"
	}
}
