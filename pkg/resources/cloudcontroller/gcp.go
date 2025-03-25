/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	GCPCCMDeploymentName = "gcp-cloud-controller-manager"
)

var (
	gcpResourceRequirements = corev1.ResourceRequirements{
		// 50m comes from https://github.com/kubernetes/cloud-provider-gcp/blob/ccm/v28.2.1/cluster/gce/gci/configure-helper.sh#L3471C10-L3471C10
		// and there is no other limit/request being inserted into the GCP CCM manifest
		// (cf. https://github.com/kubernetes/cloud-provider-gcp/blob/ccm/v28.2.1/deploy/cloud-controller-manager.manifest#L32)
		//
		// All other resource constraints are made up.

		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}
)

// See https://github.com/kubernetes/cloud-provider-gcp/blob/ccm/v28.2.1/cluster/gce/gci/configure-helper.sh#L2271 for
// the source of variables that get injected in the GCP CCM build process.

func gcpDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return GCPCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Spec.Replicas = resources.Int32(1)

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "scratch",
			})

			var err error
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err =
				resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			ccmVersion := GCPCCMVersion(data.Cluster().Spec.Version)

			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)
			dep.Spec.Template.Spec.Volumes = append(
				getVolumes(data.IsKonnectivityEnabled(), true),
				corev1.Volume{
					Name: "cloud-credentials",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.ClusterCloudCredentialsSecretName,
						},
					},
				},

				// define a scratch volume to keep the service account JSON file
				corev1.Volume{
					Name: "scratch",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							// explicitly setting this to memory does not just save
							// disk space, it also make the pod eligible for the cluster-autoscaler;
							// cf. https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/FAQ.md#what-types-of-pods-can-prevent-ca-from-removing-a-node
							Medium: corev1.StorageMediumMemory,
						},
					},
				},
			)

			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			// The service account is base64 in KKP for legacy reasons (meaning the cloud-credentials
			// secret contains "double encoded" data); the CCM takes a plain JSON file and so we inject
			// an init container to decode the base64 file.
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				getGCPInitContainer(data),
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  ccmContainerName,
					Image: registry.Must(data.RewriteImage(resources.RegistryK8S + "/cloud-provider-gcp/cloud-controller-manager:" + ccmVersion)),
					Args: []string{
						"/go-runner",
						"--redirect-stderr=true",
						"/cloud-controller-manager",
						"--v=2",
						fmt.Sprintf("--cloud-config=/etc/kubernetes/cloud/%s", resources.CloudConfigKey),
						"--secure-port=10258",
						"--use-service-account-credentials",
						"--cloud-provider=gce",
						"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
						"--authorization-kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
						"--authentication-kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
						"--allocate-node-cidrs",
						fmt.Sprintf("--cluster-name=%s", data.Cluster().Name),
						fmt.Sprintf("--cluster-cidr=%s", data.Cluster().Spec.ClusterNetwork.Pods.GetIPv4CIDR()),
					},
					Env: append(
						getEnvVars(),
						corev1.EnvVar{
							Name:  "GOOGLE_APPLICATION_CREDENTIALS",
							Value: "/etc/kubermatic/cloud-credentials/serviceAccount.json",
						},
					),
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Port:   intstr.FromInt(10258),
								Scheme: corev1.URISchemeHTTPS,
								Path:   "/healthz",
							},
						},
						InitialDelaySeconds: 15,
						TimeoutSeconds:      15,

						// Kubernetes default values, specified so we do not end up in a reconcile loop

						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
					},
					VolumeMounts: append(
						getVolumeMounts(true),
						corev1.VolumeMount{
							Name:      "scratch",
							MountPath: "/etc/kubermatic/cloud-credentials",
						},
					),
				},
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				ccmContainerName: gcpResourceRequirements.DeepCopy(),
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func getGCPInitContainer(data *resources.TemplateData) corev1.Container {
	return corev1.Container{
		Name:    "decode-sa",
		Image:   registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/util:2.5.0")),
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			"base64 -d /input/serviceAccount > /scratch/serviceAccount.json",
		},
		VolumeMounts: append(
			getVolumeMounts(true),
			corev1.VolumeMount{
				Name:      "cloud-credentials",
				MountPath: "/input",
			},
			corev1.VolumeMount{
				Name:      "scratch",
				MountPath: "/scratch",
			},
		),
	}
}

func GCPCCMVersion(version semver.Semver) string {
	// https://github.com/kubernetes/cloud-provider-gcp/tags
	// Image promotion is known to be laggy, so you can also check the registry contents directly:
	// gcrane ls --json registry.k8s.io/cloud-provider-gcp/cloud-controller-manager | jq -r '.tags[]'

	switch version.MajorMinor() {
	case v128:
		return "v28.2.1"
	case v129:
		return "v29.0.0"
	case v130:
		fallthrough
	case v131:
		fallthrough
	case v132:
		fallthrough
	default:
		return "v30.0.0"
	}
}
