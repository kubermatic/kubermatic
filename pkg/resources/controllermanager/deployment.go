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

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

var defaultResourceRequirements = corev1.ResourceRequirements{
	Requests: corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse("100Mi"),
		corev1.ResourceCPU:    resource.MustParse("100m"),
	},
	Limits: corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse("2Gi"),
		corev1.ResourceCPU:    resource.MustParse("2"),
	},
}

const (
	name = "controller-manager"
)

// DeploymentReconciler returns the function to create and update the controller manager deployment.
func DeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.ControllerManagerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(name, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			version := data.Cluster().Status.Versions.ControllerManager.Semver()

			flags, err := getFlags(data, version)
			if err != nil {
				return nil, err
			}

			dep.Spec.Replicas = resources.Int32(1)
			override := data.Cluster().Spec.ComponentsOverride.ControllerManager
			if override.Replicas != nil {
				dep.Spec.Replicas = override.Replicas
			}
			dep.Spec.Template.Spec.Tolerations = override.Tolerations

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes(data.IsKonnectivityEnabled())
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

			kubernetes.EnsureLabels(&dep.Spec.Template, map[string]string{
				resources.VersionLabel: version.String(),
			})
			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				"prometheus.io/path":                   "/metrics",
				"prometheus.io/scrape_with_kube_cert":  "true",
				"prometheus.io/port":                   "10257",
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
			})

			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Volumes = volumes

			if data.Cluster().Spec.Cloud.VSphere != nil {
				fakeVMWareUUIDMount := corev1.VolumeMount{
					Name:      resources.CloudConfigSeedSecretName,
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
				{
					Name:    resources.ControllerManagerDeploymentName,
					Image:   registry.Must(data.RewriteImage(resources.RegistryK8S + "/kube-controller-manager:v" + version.String())),
					Command: []string{"/usr/local/bin/kube-controller-manager"},
					Args:    flags,
					Env:     envVars,
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: healthAction,
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 8,
						ProbeHandler: corev1.ProbeHandler{
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
				name: defaultResourceRequirements.DeepCopy(),
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(name, kubermaticv1.AntiAffinityTypePreferred)

			dep.Spec.Template, err = apiserver.IsRunningWrapper(data, dep.Spec.Template, sets.New(name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

			return dep, nil
		}
	}
}

func getFlags(data *resources.TemplateData, version *semverlib.Version) ([]string, error) {
	cluster := data.Cluster()
	controllers := []string{"*", "bootstrapsigner", "tokencleaner"}

	// If CCM migration is enabled and all kubeletes have not been migrated yet
	// disable the cloud controllers.
	hasCSIMigrationCompleted := cluster.Status.Conditions[kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted].Status == corev1.ConditionTrue

	if metav1.HasAnnotation(cluster.ObjectMeta, kubermaticv1.CCMMigrationNeededAnnotation) && !hasCSIMigrationCompleted {
		controllers = append(controllers, "-cloud-node-lifecycle", "-route", "-service")
	}

	flags := []string{
		"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"--service-account-private-key-file", "/etc/kubernetes/service-account-key/sa.key",
		"--root-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-cert-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-key-file", "/etc/kubernetes/pki/ca/ca.key",
		"--controllers", strings.Join(controllers, ","),
		"--use-service-account-credentials",
		// this can't be passed as two strings as the other parameters
		"--profiling=false",
	}

	// Cilium uses its own node IPAM, use --allocate-node-cidrs and related flags only for other CNIs
	if cluster.Spec.CNIPlugin.Type != kubermaticv1.CNIPluginTypeCilium {
		flags = append(flags, "--allocate-node-cidrs")
		flags = append(flags, "--cluster-cidr", strings.Join(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks, ","))
		flags = append(flags, "--service-cluster-ip-range", strings.Join(cluster.Spec.ClusterNetwork.Services.CIDRBlocks, ","))
		if cluster.IsDualStack() {
			if cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4 != nil {
				flags = append(flags, fmt.Sprintf("--node-cidr-mask-size-ipv4=%d", *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4))
			}
			if cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6 != nil {
				flags = append(flags, fmt.Sprintf("--node-cidr-mask-size-ipv6=%d", *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6))
			}
		} else {
			if cluster.IsIPv4Only() && cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4 != nil {
				flags = append(flags, fmt.Sprintf("--node-cidr-mask-size=%d", *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4))
			}
			if cluster.IsIPv6Only() && cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6 != nil {
				flags = append(flags, fmt.Sprintf("--node-cidr-mask-size=%d", *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6))
			}
		}
		if val := CloudRoutesFlagVal(cluster.Spec.Cloud); val != nil {
			flags = append(flags, fmt.Sprintf("--configure-cloud-routes=%t", *val))
		}
	}

	featureGates := []string{"RotateKubeletServerCertificate=true"}
	featureGates = append(featureGates, data.GetCSIMigrationFeatureGates(cluster.Status.Versions.ControllerManager.Semver())...)
	if data.DRAEnabled() {
		featureGates = append(featureGates, "DynamicResourceAllocation=true")
	}

	flags = append(flags, "--feature-gates")
	flags = append(flags, strings.Join(featureGates, ","))

	cloudProviderName := resources.GetKubernetesCloudProviderName(cluster,
		resources.ExternalCloudProviderEnabled(cluster))
	if cloudProviderName != "" && cloudProviderName != "external" {
		flags = append(flags, "--cloud-provider", cloudProviderName)
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")

		constraint115, err := semverlib.NewConstraint(">= 1.15.0")
		if err != nil {
			return nil, err
		}

		if cloudProviderName == "azure" && constraint115.Check(version) {
			// Required so multiple clusters using the same resource group can allocate public IPs.
			// Ref: https://github.com/kubernetes/kubernetes/pull/77630
			flags = append(flags, "--cluster-name", cluster.Name)
		}
	}

	// New flag in v1.12 which gets used to perform permission checks for tokens
	flags = append(flags, "--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
	// New flag in v1.12 which gets used to perform permission checks for certs
	flags = append(flags, "--client-ca-file", "/etc/kubernetes/pki/ca/ca.crt")

	// With 1.13 we're using the secure port for scraping metrics as the insecure port got marked deprecated
	flags = append(flags, "--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
	flags = append(flags, "--authorization-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")

	// Apply leader election settings
	if lds := cluster.Spec.ComponentsOverride.ControllerManager.LeaseDurationSeconds; lds != nil {
		flags = append(flags, "--leader-elect-lease-duration", fmt.Sprintf("%ds", *lds))
	}
	if rds := cluster.Spec.ComponentsOverride.ControllerManager.LeaderElectionSettings.DeepCopy().RenewDeadlineSeconds; rds != nil {
		flags = append(flags, "--leader-elect-renew-deadline", fmt.Sprintf("%ds", *rds))
	}
	if rps := cluster.Spec.ComponentsOverride.ControllerManager.LeaderElectionSettings.DeepCopy().RetryPeriodSeconds; rps != nil {
		flags = append(flags, "--leader-elect-retry-period", fmt.Sprintf("%ds", *rps))
	}

	return flags, nil
}

func getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      resources.CASecretName,
			MountPath: "/etc/kubernetes/pki/ca",
			ReadOnly:  true,
		},
		{
			Name:      resources.CABundleConfigMapName,
			MountPath: "/etc/kubernetes/pki/ca-bundle",
			ReadOnly:  true,
		},
		{
			Name:      resources.ServiceAccountKeySecretName,
			MountPath: "/etc/kubernetes/service-account-key",
			ReadOnly:  true,
		},
		{
			Name:      resources.CloudConfigSeedSecretName,
			MountPath: "/etc/kubernetes/cloud",
			ReadOnly:  true,
		},
		{
			Name:      resources.ControllerManagerKubeconfigSecretName,
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
	}
}

func getVolumes(isKonnectivityEnabled bool) []corev1.Volume {
	vs := []corev1.Volume{
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CASecretName,
				},
			},
		},
		{
			Name: resources.CABundleConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CABundleConfigMapName,
					},
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
			Name: resources.CloudConfigSeedSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CloudConfigSeedSecretName,
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
	}

	return vs
}

type kubeControllerManagerEnvData interface {
	Cluster() *kubermaticv1.Cluster
	Seed() *kubermaticv1.Seed
}

func GetEnvVars(data kubeControllerManagerEnvData) ([]corev1.EnvVar, error) {
	cluster := data.Cluster()

	vars := []corev1.EnvVar{
		{
			Name:  "SSL_CERT_FILE",
			Value: "/etc/kubernetes/pki/ca-bundle/ca-bundle.pem",
		},
	}

	refTo := func(key string) *corev1.EnvVarSource {
		return &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: resources.ClusterCloudCredentialsSecretName,
				},
				Key: key,
			},
		}
	}

	if cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", ValueFrom: refTo(resources.AWSAccessKeyID)})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", ValueFrom: refTo(resources.AWSSecretAccessKey)})
		vars = append(vars, corev1.EnvVar{Name: "AWS_VPC_ID", Value: cluster.Spec.Cloud.AWS.VPCID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_ASSUME_ROLE_ARN", Value: cluster.Spec.Cloud.AWS.AssumeRoleARN})
		vars = append(vars, corev1.EnvVar{Name: "AWS_ASSUME_ROLE_EXTERNAL_ID", Value: cluster.Spec.Cloud.AWS.AssumeRoleExternalID})
	}

	if cluster.Spec.Cloud.GCP != nil {
		vars = append(vars, corev1.EnvVar{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/etc/gcp/serviceAccount"})
	}

	vars = append(vars, resources.GetHTTPProxyEnvVarsFromSeed(data.Seed(), data.Cluster().Status.Address.InternalName)...)
	return vars, nil
}

func CloudRoutesFlagVal(cloudSpec kubermaticv1.CloudSpec) *bool {
	if cloudSpec.AWS != nil {
		return ptr.To(false)
	}
	if cloudSpec.Openstack != nil {
		return ptr.To(false)
	}
	if cloudSpec.VSphere != nil {
		return ptr.To(false)
	}
	if cloudSpec.Azure != nil {
		return ptr.To(false)
	}
	if cloudSpec.GCP != nil {
		return ptr.To(true)
	}
	return nil
}
