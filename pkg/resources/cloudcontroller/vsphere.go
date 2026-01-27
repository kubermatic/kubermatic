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
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
			kubernetes.EnsureLabels(&dep.Spec.Template, map[string]string{
				"component": "cloud-controller-manager",
				"tier":      "control-plane",
			})

			var err error
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err =
				resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			container := getVSphereCCMContainer(data)

			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)
			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled(), true)
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				container,
			}

			return dep, nil
		}
	}
}

func getVSphereCCMContainer(data *resources.TemplateData) corev1.Container {
	clusterVersion := data.Cluster().Status.Versions.ControlPlane
	version := VSphereCCMVersion(clusterVersion)
	repository := ccmRepository(clusterVersion)

	controllerManagerImage := registry.Must(data.RewriteImage(repository + ":v" + version))
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

var registryCutoff = semver.NewSemverOrDie("1.28.0")

func ccmRepository(version semver.Semver) string {
	// See https://github.com/kubermatic/kubermatic/issues/13719 for why we have mirrored pre-1.28 images.
	if version.LessThan(registryCutoff) {
		return resources.RegistryQuay + "/kubermatic/mirror/cloud-provider-vsphere/ccm"
	}

	return resources.RegistryK8S + "/cloud-pv-vsphere/cloud-provider-vsphere"
}

func VSphereCCMVersion(version semver.Semver) string {
	// https://github.com/kubernetes/cloud-provider-vsphere/releases
	// gcrane ls --json registry.k8s.io/cloud-pv-vsphere/cloud-provider-vsphere | jq -r '.tags[]'

	switch version.MajorMinor() {
	case v131:
		return "1.31.0"
	case v132:
		return "1.32.2"
	case v133:
		fallthrough
	case v134:
		fallthrough
	case v135:
		fallthrough
	default:
		return "1.35.0"
	}
}
