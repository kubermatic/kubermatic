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

package usercluster

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.UserClusterControllerContainerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("25m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
		},
	}
)

// userclusterControllerData is the subet of the deploymentData interface
// that is actually required by the usercluster deployment
// This makes importing the deployment elsewhere (openshift controller)
// easier as only have to implement the parts that are actually in use.
type userclusterControllerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetLegacyOverwriteRegistry() string
	RewriteImage(string) (string, error)
	Cluster() *kubermaticv1.Cluster
	NodeLocalDNSCacheEnabled() bool
	GetOpenVPNServerPort() (int32, error)
	GetKonnectivityServerPort() (int32, error)
	GetKonnectivityKeepAliveTime() string
	GetTunnelingAgentIP() string
	GetMLAGatewayPort() (int32, error)
	KubermaticAPIImage() string
	KubermaticDockerTag() string
	GetCloudProviderName() (string, error)
	UserClusterMLAEnabled() bool
	IsKonnectivityEnabled() bool
	DC() *kubermaticv1.Datacenter
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	GetEnvVars() ([]corev1.EnvVar, error)
}

// DeploymentReconciler returns the function to create and update the user cluster controller deployment
//
//nolint:gocyclo
func DeploymentReconciler(data userclusterControllerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.UserClusterControllerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.UserClusterControllerDeploymentName
			dep.Labels = resources.BaseAppLabels(resources.UserClusterControllerDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)

			if data.Cluster().Spec.ComponentsOverride.UserClusterController != nil && data.Cluster().Spec.ComponentsOverride.UserClusterController.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.UserClusterController.Replicas
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(resources.UserClusterControllerDeploymentName, nil),
			}
			dep.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
			dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
				MaxSurge: &intstr.IntOrString{
					Type: intstr.Int,
					// The readiness probe only turns ready if a sync succeeded.
					// That requires that the controller acquires the leader lock, which only happens if the other instance stops
					IntVal: 1,
				},
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes(data)
			podLabels, err := data.GetPodTemplateLabels(resources.UserClusterControllerDeploymentName, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %w", err)
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

			dnsClusterIP, err := resources.UserClusterDNSResolverIP(data.Cluster())
			if err != nil {
				return nil, err
			}

			enableUserSSHKeyAgent := data.Cluster().Spec.EnableUserSSHKeyAgent
			if enableUserSSHKeyAgent == nil {
				enableUserSSHKeyAgent = pointer.Bool(true)
			}

			address := data.Cluster().Status.Address

			args := append([]string{
				"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"-metrics-listen-address", "0.0.0.0:8085",
				"-health-listen-address", "0.0.0.0:8086",
				"-namespace", "$(NAMESPACE)",
				"-cluster-url", address.URL,
				"-cluster-name", data.Cluster().Name,
				"-dns-cluster-ip", dnsClusterIP,
				"-overwrite-registry", data.GetLegacyOverwriteRegistry(),
				"-version", data.Cluster().Status.Versions.ControlPlane.String(),
				"-application-cache", resources.ApplicationCacheMountPath,
				fmt.Sprintf("-enable-ssh-key-agent=%t", *enableUserSSHKeyAgent),
				fmt.Sprintf("-opa-integration=%t", data.Cluster().Spec.OPAIntegration != nil && data.Cluster().Spec.OPAIntegration.Enabled),
				fmt.Sprintf("-ca-bundle=/opt/ca-bundle/%s", resources.CABundleConfigMapKey),
				fmt.Sprintf("-node-local-dns-cache=%t", data.NodeLocalDNSCacheEnabled()),
			}, getNetworkArgs(data)...)

			if email := data.Cluster().Status.UserEmail; email != "" {
				args = append(args, "-owner-email", email)
			}

			if data.Cluster().Spec.DebugLog {
				args = append(args, "-log-debug=true")
			}

			if data.IsKonnectivityEnabled() {
				args = append(args, "-konnectivity-enabled=true")

				kHost := address.ExternalName
				if data.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
					kHost = fmt.Sprintf("%s.%s", resources.KonnectivityProxyServiceName, kHost)
				}
				kPort, err := data.GetKonnectivityServerPort()
				if err != nil {
					return nil, err
				}
				args = append(args, "-konnectivity-server-host", kHost)
				args = append(args, "-konnectivity-server-port", fmt.Sprint(kPort))
				args = append(args, "-konnectivity-keepalive-time", data.GetKonnectivityKeepAliveTime())
			} else {
				openvpnServerPort, err := data.GetOpenVPNServerPort()
				if err != nil {
					return nil, err
				}
				args = append(args, "-openvpn-server-port", fmt.Sprint(openvpnServerPort))
			}

			if data.Cluster().Spec.Features[kubermaticv1.KubeSystemNetworkPolicies] {
				args = append(args, "-enable-network-policies")
			}

			if data.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
				args = append(args, "-tunneling-agent-ip", data.GetTunnelingAgentIP())
				args = append(args, "-kas-secure-port", fmt.Sprint(resources.APIServerSecurePort))
			}

			providerName, err := data.GetCloudProviderName()
			if err != nil {
				return nil, fmt.Errorf("failed to get cloud provider name: %w", err)
			}
			args = append(args, "-cloud-provider-name", providerName)

			if data.Cluster().Spec.Cloud.Nutanix != nil && data.Cluster().Spec.Cloud.Nutanix.CSI != nil {
				args = append(args, "-nutanix-csi-enabled=true")
			}

			if data.Cluster().Spec.UpdateWindow != nil && data.Cluster().Spec.UpdateWindow.Length != "" && data.Cluster().Spec.UpdateWindow.Start != "" {
				args = append(args, "-update-window-start", data.Cluster().Spec.UpdateWindow.Start, "-update-window-length", data.Cluster().Spec.UpdateWindow.Length)
			}

			if data.Cluster().Spec.OPAIntegration != nil && data.Cluster().Spec.OPAIntegration.WebhookTimeoutSeconds != nil {
				args = append(args, "-opa-webhook-timeout", fmt.Sprint(*data.Cluster().Spec.OPAIntegration.WebhookTimeoutSeconds))
			}

			if data.Cluster().Spec.OPAIntegration != nil && data.Cluster().Spec.OPAIntegration.Enabled {
				args = append(args, fmt.Sprintf("-enable-mutation=%t", data.Cluster().Spec.OPAIntegration.ExperimentalEnableMutation))
			}

			if data.Cluster().Spec.Cloud.Kubevirt != nil {
				args = append(args, "-kv-vmi-eviction-controller")
				args = append(args, "-kv-infra-kubeconfig", "/etc/kubernetes/kubevirt/infra-kubeconfig")
			}

			if data.UserClusterMLAEnabled() && data.Cluster().Spec.MLA != nil {
				args = append(args, fmt.Sprintf("-user-cluster-monitoring=%t", data.Cluster().Spec.MLA.MonitoringEnabled))
				args = append(args, fmt.Sprintf("-user-cluster-logging=%t", data.Cluster().Spec.MLA.LoggingEnabled))

				if data.Cluster().Spec.MLA.MonitoringEnabled || data.Cluster().Spec.MLA.LoggingEnabled {
					mlaGatewayPort, err := data.GetMLAGatewayPort()
					if err != nil {
						return nil, err
					}
					mlaEndpoint := net.JoinHostPort(address.ExternalName, fmt.Sprintf("%d", mlaGatewayPort))
					if data.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
						mlaEndpoint = resources.MLAGatewaySNIPrefix + mlaEndpoint
					}
					args = append(args, "-mla-gateway-url", "https://"+mlaEndpoint)
				}
			}

			if kubermaticv1helper.NeedCCMMigration(data.Cluster()) {
				args = append(args, "-ccm-migration")
			}

			if kubermaticv1helper.CCMMigrationCompleted(data.Cluster()) {
				args = append(args, "-ccm-migration-completed")
			}

			labelArgsValue, err := getLabelsArgValue(data.Cluster())
			if err != nil {
				return nil, fmt.Errorf("failed to get label args value: %w", err)
			}
			if labelArgsValue != "" {
				args = append(args, "-node-labels", labelArgsValue)
			}

			if data.Cluster().Spec.ComponentsOverride.UserClusterController != nil && data.Cluster().Spec.ComponentsOverride.UserClusterController.Tolerations != nil {
				dep.Spec.Template.Spec.Tolerations = data.Cluster().Spec.ComponentsOverride.UserClusterController.Tolerations
			}

			envVars, err := data.GetEnvVars()
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.UserClusterControllerContainerName,
					Image:   data.KubermaticAPIImage() + ":" + data.KubermaticDockerTag(),
					Command: []string{"/usr/local/bin/user-cluster-controller-manager"},
					Args:    args,
					Env: append(envVars, corev1.EnvVar{
						Name: "NAMESPACE",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath:  "metadata.namespace",
								APIVersion: "v1",
							},
						},
					}),
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/readyz",
								Port:   intstr.FromInt(8086),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold: 5,
						PeriodSeconds:    5,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					VolumeMounts: getVolumeMounts(data),
				},
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}
			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.New(resources.UserClusterControllerContainerName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

func getVolumes(data userclusterControllerData) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: resources.InternalUserClusterAdminKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
				},
			},
		},
		{
			Name: "ca-bundle",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CABundleConfigMapName,
					},
				},
			},
		},
		{
			Name: resources.ApplicationCacheVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: resources.GetApplicationCacheSize(data.Cluster().Spec.ApplicationSettings),
				},
			},
		},
	}

	if data.Cluster().Spec.Cloud.Kubevirt != nil {
		volumes = append(volumes, corev1.Volume{
			Name: resources.KubeVirtInfraSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.KubeVirtInfraSecretName,
				},
			},
		})
	}

	return volumes
}

func getVolumeMounts(data userclusterControllerData) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{
			Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
		{
			Name:      "ca-bundle",
			MountPath: "/opt/ca-bundle/",
			ReadOnly:  true,
		},
		{
			Name:      resources.ApplicationCacheVolumeName,
			MountPath: resources.ApplicationCacheMountPath,
			ReadOnly:  false,
		},
	}

	if data.Cluster().Spec.Cloud.Kubevirt != nil {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      resources.KubeVirtInfraSecretName,
			MountPath: "/etc/kubernetes/kubevirt",
			ReadOnly:  true,
		})
	}
	return mounts
}

func getNetworkArgs(data userclusterControllerData) []string {
	networkFlags := make([]string, len(data.Cluster().Spec.MachineNetworks)*2)
	i := 0

	for _, n := range data.Cluster().Spec.MachineNetworks {
		networkFlags[i] = "--ipam-controller-network"
		i++
		networkFlags[i] = fmt.Sprintf("%s,%s,%s", n.CIDR, n.Gateway, strings.Join(n.DNSServers, ","))
		i++
	}

	return networkFlags
}

func getLabelsArgValue(cluster *kubermaticv1.Cluster) (string, error) {
	labelsToApply := map[string]string{}
	for key, value := range cluster.Labels {
		if kubermaticv1.ProtectedClusterLabels.Has(key) {
			continue
		}
		labelsToApply[key] = value
	}

	if len(labelsToApply) == 0 {
		return "", nil
	}

	bytes, err := json.Marshal(labelsToApply)
	if err != nil {
		return "", fmt.Errorf("failed to marshal labels: %w", err)
	}
	return string(bytes), nil
}
