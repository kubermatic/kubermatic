/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

const (
	DigitalOceanCCMDeploymentName = "digitalocean-cloud-controller-manager"
)

var digitalOceanResourceRequirements = corev1.ResourceRequirements{
	Requests: corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse("50Mi"),
		corev1.ResourceCPU:    resource.MustParse("100m"),
	},
	Limits: corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse("512Mi"),
		corev1.ResourceCPU:    resource.MustParse("1"),
	},
}

func digitalOceanDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return DigitalOceanCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Spec.Replicas = resources.Int32(1)

			var err error
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			version := DigitaloceanCCMVersion(data.Cluster().Spec.Version)

			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)
			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled(), true)
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    ccmContainerName,
					Image:   registry.Must(data.RewriteImage(resources.RegistryDocker + "/digitalocean/digitalocean-cloud-controller-manager:" + version)),
					Command: []string{"/bin/digitalocean-cloud-controller-manager"},
					Args:    getDigitalOceanFlags(),
					Env: []corev1.EnvVar{
						{
							Name: "DO_ACCESS_TOKEN",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: resources.ClusterCloudCredentialsSecretName,
									},
									Key: resources.DigitaloceanToken,
								},
							},
						},
						{
							Name:  "REGION",
							Value: data.DC().Spec.Digitalocean.Region,
						},
					},
					VolumeMounts: getVolumeMounts(true),
				},
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				ccmContainerName: digitalOceanResourceRequirements.DeepCopy(),
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}
			return dep, nil
		}
	}
}

func getDigitalOceanFlags() []string {
	return []string{
		"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		"--leader-elect=false",
	}
}

func DigitaloceanCCMVersion(version semver.Semver) string {
	// https://github.com/digitalocean/digitalocean-cloud-controller-manager/releases
	//
	// Note that DO CCM is documented to be
	//
	//     > Because of the fast Kubernetes release cycles, CCM will only support the
	//     > version that is also supported on DigitalOcean Kubernetes product.
	//     > Any other releases will be not officially supported by us.
	//
	// So to be sure, confer with
	// https://docs.digitalocean.com/products/kubernetes/details/supported-releases/
	// and see which Kubernetes release are currently meant to be supported.
	// Whenever a Kubernetes release goes out of DO support, pin the dependency down
	// by replacing the `fallthrough` with a return statement.

	switch version.MajorMinor() {
	case v132: // 6 February 2025 – 27 March 2026
		fallthrough
	case v133: // 16 June 2025 - 27 July 2026
		fallthrough
	case v134: // (not supported yet)
		fallthrough
	default:
		// This should always be the latest version.
		return "v0.1.63"
	}
}
