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

package controllermanager

import (
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

const (
	name = "controller-manager"
)

// DeploymentCreator returns the function to create and update the controller manager deployment
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.ControllerManagerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.ControllerManagerDeploymentName
			dep.Labels = resources.BaseAppLabels(name, nil)

			flags, err := getFlags(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(name, nil),
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes()
			volumeMounts := getVolumeMounts()

			if data.Cluster().Spec.Cloud.GCP != nil {
				serviceAccountVolume := corev1.Volume{
					Name: resources.GoogleServiceAccountVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.GoogleServiceAccountSecretName,
						},
					},
				}
				volumes = append(volumes, serviceAccountVolume)
			}

			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/path":                  "/metrics",
					"prometheus.io/scrape_with_kube_cert": "true",
					"prometheus.io/port":                  "10257",
				},
			}

			// Configure user cluster DNS resolver for this pod.
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Volumes = volumes

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
			}

			if data.Cluster().Spec.Cloud.VSphere != nil {
				fakeVMWareUUIDMount := corev1.VolumeMount{
					Name:      resources.CloudConfigConfigMapName,
					SubPath:   cloudconfig.FakeVMWareUUIDKeyName,
					MountPath: "/sys/class/dmi/id/product_serial",
					ReadOnly:  true,
				}
				// Required because of https://github.com/kubernetes/kubernetes/issues/65145
				volumeMounts = append(volumeMounts, fakeVMWareUUIDMount)
			}
			if data.Cluster().Spec.Cloud.GCP != nil {
				serviceAccountMount := corev1.VolumeMount{
					Name:      resources.GoogleServiceAccountVolumeName,
					MountPath: "/etc/gcp",
					ReadOnly:  true,
				}
				volumeMounts = append(volumeMounts, serviceAccountMount)
			}

			envVars, err := GetEnvVars(data)
			if err != nil {
				return nil, err
			}

			healthAction := &corev1.HTTPGetAction{
				Path:   "/healthz",
				Scheme: corev1.URISchemeHTTPS,
				Port:   intstr.FromInt(10257),
			}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:    resources.ControllerManagerDeploymentName,
					Image:   data.ImageRegistry(resources.RegistryK8SGCR) + "/kube-controller-manager:v" + data.Cluster().Spec.Version.String(),
					Command: []string{"/usr/local/bin/kube-controller-manager"},
					Args:    flags,
					Env:     envVars,
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: healthAction,
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 8,
						Handler: corev1.Handler{
							HTTPGet: healthAction,
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					VolumeMounts: volumeMounts,
				},
			}
			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				name:                defaultResourceRequirements.DeepCopy(),
				openvpnSidecar.Name: openvpnSidecar.Resources.DeepCopy(),
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(name, data.Cluster().Name)

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

func getFlags(data *resources.TemplateData) ([]string, error) {
	controllers := []string{"*", "bootstrapsigner", "tokencleaner"}
	// If CCM migration is enabled and all kubeletes have not been migrated yet
	// disable the cloud controllers.
	// If in-tree cloud providers are deactivated (KCMCloudControllersDeactivated is true),
	// we don't want to disable any controllers, because those clusters are already using
	// the external CCM (newly-created OpenStack clusters).
	if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] &&
		!kubermaticv1helper.ClusterConditionHasStatus(data.Cluster(), kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted, corev1.ConditionTrue) &&
		!data.KCMCloudControllersDeactivated(true) {
		controllers = append(controllers, "-cloud-node-lifecycle", "-route", "-service")
	}
	flags := []string{
		"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"--service-account-private-key-file", "/etc/kubernetes/service-account-key/sa.key",
		"--root-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-cert-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-key-file", "/etc/kubernetes/pki/ca/ca.key",
		"--cluster-cidr", data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		"--allocate-node-cidrs",
		"--controllers", strings.Join(controllers, ","),
		"--use-service-account-credentials",
	}

	featureGates := []string{"RotateKubeletClientCertificate=true",
		"RotateKubeletServerCertificate=true"}
	featureGates = append(featureGates, data.GetCSIMigrationFeatureGates()...)

	flags = append(flags, "--feature-gates")
	flags = append(flags, strings.Join(featureGates, ","))

	cloudProviderName := resources.GetKubernetesCloudProviderName(data.Cluster(),
		resources.ExternalCloudProviderEnabled(data.Cluster()))
	if cloudProviderName != "" && cloudProviderName != "external" {
		flags = append(flags, "--cloud-provider", cloudProviderName)
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
		if cloudProviderName == "azure" && data.Cluster().Spec.Version.Semver().Minor() >= 15 {
			// Required so multiple clusters using the same resource group can allocate public IPs.
			// Ref: https://github.com/kubernetes/kubernetes/pull/77630
			flags = append(flags, "--cluster-name", data.Cluster().Name)
		}
	}

	if val := CloudRoutesFlagVal(data.Cluster().Spec.Cloud); val != nil {
		flags = append(flags, fmt.Sprintf("--configure-cloud-routes=%t", *val))
	}

	// New flag in v1.12 which gets used to perform permission checks for tokens
	flags = append(flags, "--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
	// New flag in v1.12 which gets used to perform permission checks for certs
	flags = append(flags, "--client-ca-file", "/etc/kubernetes/pki/ca/ca.crt")

	// With 1.13 we're using the secure port for scraping metrics as the insecure port got marked deprecated
	flags = append(flags, "--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
	flags = append(flags, "--authorization-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
	// We're going to use the https endpoints for scraping the metrics starting from 1.12. Thus we can deactivate the http endpoint
	flags = append(flags, "--port", "0")

	return flags, nil
}

func getVolumeMounts() []corev1.VolumeMount {
	return append([]corev1.VolumeMount{
		{
			Name:      resources.CASecretName,
			MountPath: "/etc/kubernetes/pki/ca",
			ReadOnly:  true,
		},
		{
			Name:      resources.ServiceAccountKeySecretName,
			MountPath: "/etc/kubernetes/service-account-key",
			ReadOnly:  true,
		},
		{
			Name:      resources.CloudConfigConfigMapName,
			MountPath: "/etc/kubernetes/cloud",
			ReadOnly:  true,
		},
		{
			Name:      resources.ControllerManagerKubeconfigSecretName,
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
	},
		resources.GetHostCACertVolumeMounts()...)
}

func getVolumes() []corev1.Volume {
	return append([]corev1.Volume{
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CASecretName,
				},
			},
		},
		{
			Name: resources.ServiceAccountKeySecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ServiceAccountKeySecretName,
				},
			},
		},
		{
			Name: resources.CloudConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CloudConfigConfigMapName,
					},
				},
			},
		},
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		},
		{
			Name: resources.ControllerManagerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ControllerManagerKubeconfigSecretName,
				},
			},
		},
	}, resources.GetHostCACertVolumes()...)
}

type kubeControllerManagerEnvData interface {
	resources.CredentialsData
	Seed() *kubermaticv1.Seed
}

func GetEnvVars(data kubeControllerManagerEnvData) ([]corev1.EnvVar, error) {
	credentials, err := resources.GetCredentials(data)
	if err != nil {
		return nil, err
	}
	cluster := data.Cluster()

	var vars []corev1.EnvVar
	if cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: credentials.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: credentials.AWS.SecretAccessKey})
		vars = append(vars, corev1.EnvVar{Name: "AWS_VPC_ID", Value: cluster.Spec.Cloud.AWS.VPCID})
	}
	if cluster.Spec.Cloud.GCP != nil {
		vars = append(vars, corev1.EnvVar{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/etc/gcp/serviceAccount"})
	}
	vars = append(vars, resources.GetHTTPProxyEnvVarsFromSeed(data.Seed(), data.Cluster().Address.InternalName)...)
	return vars, nil
}

func CloudRoutesFlagVal(cloudSpec kubermaticv1.CloudSpec) *bool {
	if cloudSpec.AWS != nil {
		return utilpointer.BoolPtr(false)
	}
	if cloudSpec.Openstack != nil {
		return utilpointer.BoolPtr(false)
	}
	if cloudSpec.VSphere != nil {
		return utilpointer.BoolPtr(false)
	}
	if cloudSpec.Azure != nil {
		return utilpointer.BoolPtr(false)
	}
	if cloudSpec.GCP != nil {
		return utilpointer.BoolPtr(true)
	}
	return nil
}
