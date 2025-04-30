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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
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
	AWSCCMDeploymentName = "aws-cloud-controller-manager"
)

var (
	awsResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("300Mi"),
			corev1.ResourceCPU:    resource.MustParse("200m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}
)

func awsDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return AWSCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Spec.Replicas = resources.Int32(1)

			var err error
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err =
				resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			ccmVersion := AWSCCMVersion(data.Cluster().Spec.Version)

			flags := []string{
				"/bin/aws-cloud-controller-manager",
				"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
				"--cloud-config=/etc/kubernetes/cloud/config",
				"--cloud-provider=aws", "--configure-cloud-routes=false",
				fmt.Sprintf("--cluster-cidr=%s", data.Cluster().Spec.ClusterNetwork.Pods.GetIPv4CIDR()),
			}

			if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] {
				flags = append(flags, "--cluster-name", data.Cluster().Name)
			}

			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)
			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled(), true)
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    ccmContainerName,
					Image:   registry.Must(data.RewriteImage(resources.RegistryK8S + "/provider-aws/cloud-controller-manager:" + ccmVersion)),
					Command: flags,
					Env: append(
						getEnvVars(),
						corev1.EnvVar{
							Name: "AWS_ACCESS_KEY_ID",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: resources.ClusterCloudCredentialsSecretName,
									},
									Key: resources.AWSAccessKeyID,
								},
							},
						},
						corev1.EnvVar{
							Name: "AWS_SECRET_ACCESS_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: resources.ClusterCloudCredentialsSecretName,
									},
									Key: resources.AWSSecretAccessKey,
								},
							},
						},
					),
					VolumeMounts: getVolumeMounts(true),
				},
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				ccmContainerName: awsResourceRequirements.DeepCopy(),
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func AWSCCMVersion(version semver.Semver) string {
	// https://github.com/kubernetes/cloud-provider-aws/tags (releases are not consistently created)
	// gcrane ls --json registry.k8s.io/provider-aws/cloud-controller-manager | jq -r '.tags[]'

	switch version.MajorMinor() {
	case v128:
		return "v1.28.9"
	case v129:
		return "v1.29.7"
	case v130:
		return "v1.30.3"
	case v131:
		return "v1.31.1"
	case v132:
		fallthrough
	default:
		return "v1.32.0"
	}
}
