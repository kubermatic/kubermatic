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

package apiserver

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/net"
	"path/filepath"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"
	"k8c.io/kubermatic/v2/pkg/resources/etcd/etcdrunning"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("4Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

const (
	name = "apiserver"

	defaultNodePortRange = "30000-32767"
)

func AuditConfigMapCreator() reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.AuditConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{
					"policy.yaml": `apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata
`}
			}
			return cm, nil
		}
	}
}

// DeploymentCreator returns the function to create and update the API server deployment
func DeploymentCreator(data *resources.TemplateData, enableOIDCAuthentication bool) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.ApiserverDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.ApiserverDeploymentName
			dep.Labels = resources.BaseAppLabels(name, nil)

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.Apiserver.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.Apiserver.Replicas
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(name, nil),
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes()
			volumeMounts := getVolumeMounts()

			if enableOIDCAuthentication && len(data.OIDCCAFile()) > 0 {
				volumes = append(volumes, getDexCASecretVolume())
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      resources.DexCASecretName,
					MountPath: "/etc/kubernetes/dex/ca",
					ReadOnly:  true,
				})
			}

			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/scrape_with_kube_cert": "true",
					"prometheus.io/path":                  "/metrics",
					"prometheus.io/port":                  fmt.Sprint(data.Cluster().Address.Port),
				},
			}

			etcdEndpoints := etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName)

			// Configure user cluster DNS resolver for this pod.
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				etcdrunning.Container(etcdEndpoints, data),
			}

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn-client sidecar: %v", err)
			}

			dnatControllerSidecar, err := vpnsidecar.DnatControllerContainer(
				data,
				"dnat-controller",
				fmt.Sprintf("https://127.0.0.1:%d", data.Cluster().Address.Port),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get dnat-controller sidecar: %v", err)
			}
			auditLogEnabled := data.Cluster().Spec.AuditLogging != nil && data.Cluster().Spec.AuditLogging.Enabled
			flags, err := getApiserverFlags(data, etcdEndpoints, enableOIDCAuthentication, auditLogEnabled)
			if err != nil {
				return nil, err
			}

			envVars, err := GetEnvVars(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				*dnatControllerSidecar,
				{
					Name:    resources.ApiserverDeploymentName,
					Image:   data.ImageRegistry(resources.RegistryK8SGCR) + "/kube-apiserver:v" + data.Cluster().Spec.Version.String(),
					Command: []string{"/usr/local/bin/kube-apiserver"},
					Env:     envVars,
					Args:    flags,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: data.Cluster().Address.Port,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(int(data.Cluster().Address.Port)),
								Scheme: "HTTPS",
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    5,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(int(data.Cluster().Address.Port)),
								Scheme: "HTTPS",
							},
						},
						InitialDelaySeconds: 15,
						FailureThreshold:    8,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					VolumeMounts: volumeMounts,
				},
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				name:                       defaultResourceRequirements.DeepCopy(),
				openvpnSidecar.Name:        openvpnSidecar.Resources.DeepCopy(),
				dnatControllerSidecar.Name: dnatControllerSidecar.Resources.DeepCopy(),
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			if data.Cluster().Spec.AuditLogging != nil && data.Cluster().Spec.AuditLogging.Enabled {
				dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers,
					corev1.Container{
						Name:    "audit-logs",
						Image:   "docker.io/fluent/fluent-bit:1.2.2",
						Command: []string{"/fluent-bit/bin/fluent-bit"},
						Args:    []string{"-i", "tail", "-p", "path=/var/log/kubernetes/audit/audit.log", "-p", "db=/var/log/kubernetes/audit/fluentbit.db", "-o", "stdout"},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      resources.AuditLogVolumeName,
								MountPath: "/var/log/kubernetes/audit",
								ReadOnly:  false,
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("10Mi"),
								corev1.ResourceCPU:    resource.MustParse("5m"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("60Mi"),
								corev1.ResourceCPU:    resource.MustParse("50m"),
							},
						},
					},
				)
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(name, data.Cluster().Name)

			return dep, nil
		}
	}
}

func getApiserverFlags(data *resources.TemplateData, etcdEndpoints []string, enableOIDCAuthentication, auditLogEnabled bool) ([]string, error) {
	overrideFlags := getApiserverOverrideFlags(data)

	cluster := data.Cluster()

	admissionPlugins := sets.NewString(
		"NamespaceLifecycle",
		"LimitRanger",
		"ServiceAccount",
		"DefaultStorageClass",
		"DefaultTolerationSeconds",
		"MutatingAdmissionWebhook",
		"ValidatingAdmissionWebhook",
		"Priority",
		"ResourceQuota",
	)
	if cluster.Spec.UsePodSecurityPolicyAdmissionPlugin {
		admissionPlugins.Insert("PodSecurityPolicy")
	}
	if cluster.Spec.UsePodNodeSelectorAdmissionPlugin {
		admissionPlugins.Insert(resources.PodNodeSelectorAdmissionPlugin)
	}

	admissionPlugins.Insert(cluster.Spec.AdmissionPlugins...)

	serviceAccountKeyFile := filepath.Join("/etc/kubernetes/service-account-key", resources.ServiceAccountKeySecretKey)
	flags := []string{
		"--etcd-servers", strings.Join(etcdEndpoints, ","),
		"--etcd-cafile", "/etc/etcd/pki/client/ca.crt",
		"--etcd-certfile", filepath.Join("/etc/etcd/pki/client", resources.ApiserverEtcdClientCertificateCertSecretKey),
		"--etcd-keyfile", filepath.Join("/etc/etcd/pki/client", resources.ApiserverEtcdClientCertificateKeySecretKey),
		"--storage-backend", "etcd3",
		"--enable-admission-plugins", strings.Join(admissionPlugins.List(), ","),
		"--admission-control-config-file", "/etc/kubernetes/adm-control/admission-control.yaml",
		"--authorization-mode", "Node,RBAC",
		"--external-hostname", cluster.Address.ExternalName,
		"--token-auth-file", "/etc/kubernetes/tokens/tokens.csv",
		"--enable-bootstrap-token-auth",
		"--service-account-key-file", serviceAccountKeyFile,
		// There are efforts upstream adding support for multiple cidr's. Until that has landed, we'll take the first entry
		"--service-cluster-ip-range", cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0],
		"--service-node-port-range", overrideFlags.NodePortRange,
		"--allow-privileged",
		"--audit-log-maxage", "30",
		"--audit-log-maxbackup", "3",
		"--audit-log-maxsize", "100",
		"--audit-log-path", "/var/log/kubernetes/audit/audit.log",
		"--tls-cert-file", "/etc/kubernetes/tls/apiserver-tls.crt",
		"--tls-private-key-file", "/etc/kubernetes/tls/apiserver-tls.key",
		"--proxy-client-cert-file", "/etc/kubernetes/pki/front-proxy/client/" + resources.ApiserverProxyClientCertificateCertSecretKey,
		"--proxy-client-key-file", "/etc/kubernetes/pki/front-proxy/client/" + resources.ApiserverProxyClientCertificateKeySecretKey,
		"--client-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--kubelet-client-certificate", "/etc/kubernetes/kubelet/kubelet-client.crt",
		"--kubelet-client-key", "/etc/kubernetes/kubelet/kubelet-client.key",
		"--requestheader-client-ca-file", "/etc/kubernetes/pki/front-proxy/ca/ca.crt",
		"--requestheader-allowed-names", "apiserver-aggregator",
		"--requestheader-extra-headers-prefix", "X-Remote-Extra-",
		"--requestheader-group-headers", "X-Remote-Group",
		"--requestheader-username-headers", "X-Remote-User",
	}

	if cluster.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
		flags = append(flags,
			// The advertise address is used as endpoint address for the kubernetes
			// service in the default namespace of the user cluster.
			"--advertise-address", cluster.Address.IP,
			// The secure port is used as target port for the kubernetes service in
			// the default namespace of the user cluster, we use the NodePort value
			// for being able to access the apiserver from the usercluster side.
			"--secure-port", fmt.Sprint(cluster.Address.Port))
	} else {
		// pre-pend to have advertise-address as first argument and avoid
		// triggering unneeded redeployments.
		flags = append([]string{
			// The advertise address is used as endpoint address for the kubernetes
			// service in the default namespace of the user cluster.
			"--advertise-address", cluster.Address.IP,
			// The secure port is used as target port for the kubernetes service in
			// the default namespace of the user cluster, we use the NodePort value
			// for being able to access the apiserver from the usercluster side.
			"--secure-port", fmt.Sprint(cluster.Address.Port),
			"--kubernetes-service-node-port", fmt.Sprint(cluster.Address.Port),
		}, flags...)
	}

	if auditLogEnabled {
		flags = append(flags, "--audit-policy-file", "/etc/kubernetes/audit/policy.yaml")
	}

	if *overrideFlags.EndpointReconcilingDisabled {
		flags = append(flags, "--endpoint-reconciler-type=none")
	}

	// enable service account signing key and issuer in Kubernetes 1.20 or when
	// explicitly enabled in the cluster object
	saConfig := cluster.Spec.ServiceAccount
	if cluster.Spec.Version.Minor() >= 20 || (saConfig != nil && saConfig.TokenVolumeProjectionEnabled) {
		var audiences []string

		issuer := cluster.Address.URL
		if saConfig != nil {
			if saConfig.Issuer != "" {
				issuer = saConfig.Issuer
			}

			if len(saConfig.APIAudiences) > 0 {
				audiences = saConfig.APIAudiences
			}
		}

		if len(audiences) == 0 {
			audiences = []string{issuer}
		}

		flags = append(flags,
			"--service-account-issuer", issuer,
			"--service-account-signing-key-file", serviceAccountKeyFile,
			"--api-audiences", strings.Join(audiences, ","),
		)
	}

	if cluster.Spec.Cloud.GCP != nil {
		flags = append(flags, "--kubelet-preferred-address-types", "InternalIP")
	} else {
		flags = append(flags, "--kubelet-preferred-address-types", "ExternalIP,InternalIP")
	}

	cloudProviderName := data.GetKubernetesCloudProviderName()
	if cloudProviderName != "" && cloudProviderName != "external" {
		flags = append(flags, "--cloud-provider", cloudProviderName)
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}

	oidcSettings := cluster.Spec.OIDC
	if oidcSettings.IssuerURL != "" && oidcSettings.ClientID != "" {
		flags = append(flags, "--oidc-issuer-url", oidcSettings.IssuerURL)
		flags = append(flags, "--oidc-client-id", oidcSettings.ClientID)
		if oidcSettings.UsernameClaim != "" {
			flags = append(flags, "--oidc-username-claim", oidcSettings.UsernameClaim)
		}
		if oidcSettings.GroupsClaim != "" {
			flags = append(flags, "--oidc-groups-claim", oidcSettings.GroupsClaim)
		}
		if oidcSettings.RequiredClaim != "" {
			flags = append(flags, "--oidc-required-claim", oidcSettings.RequiredClaim)
		}
	} else if enableOIDCAuthentication {
		flags = append(flags,
			"--oidc-issuer-url", data.OIDCIssuerURL(),
			"--oidc-client-id", data.OIDCIssuerClientID(),
			"--oidc-username-claim", "email",
			"--oidc-groups-prefix", "oidc:",
			"--oidc-groups-claim", "groups",
		)
		if len(data.OIDCCAFile()) > 0 {
			flags = append(flags, "--oidc-ca-file", "/etc/kubernetes/dex/ca/caBundle.pem")
		}
	}

	return flags, nil
}

// getApiserverOverrideFlags creates all settings that may be overridden by cluster specific componentsOverrideSettings
// otherwise global overrides or defaults will be set
func getApiserverOverrideFlags(data *resources.TemplateData) (settings kubermaticv1.APIServerSettings) {
	// nodePortRange section
	settings.NodePortRange = data.NodePortRange()
	overrideNodePortRange := data.Cluster().Spec.ComponentsOverride.Apiserver.NodePortRange
	if overrideNodePortRange != "" {
		if _, err := net.ParsePortRange(overrideNodePortRange); err == nil {
			settings.NodePortRange = overrideNodePortRange
		}
	}
	if settings.NodePortRange == "" {
		settings.NodePortRange = defaultNodePortRange
	}

	// endpointReconcilingDisabled section
	settings.EndpointReconcilingDisabled = new(bool)
	if data.Cluster().Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled != nil {
		settings.EndpointReconcilingDisabled = data.Cluster().Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled
	}

	return
}

func getVolumeMounts() []corev1.VolumeMount {
	return append([]corev1.VolumeMount{
		{
			MountPath: "/etc/kubernetes/tls",
			Name:      resources.ApiserverTLSSecretName,
			ReadOnly:  true,
		},
		{
			Name:      resources.TokensSecretName,
			MountPath: "/etc/kubernetes/tokens",
			ReadOnly:  true,
		},
		{
			Name:      resources.KubeletClientCertificatesSecretName,
			MountPath: "/etc/kubernetes/kubelet",
			ReadOnly:  true,
		},
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
			Name:      resources.ApiserverEtcdClientCertificateSecretName,
			MountPath: "/etc/etcd/pki/client",
			ReadOnly:  true,
		},
		{
			Name:      resources.ApiserverFrontProxyClientCertificateSecretName,
			MountPath: "/etc/kubernetes/pki/front-proxy/client",
			ReadOnly:  true,
		},
		{
			Name:      resources.FrontProxyCASecretName,
			MountPath: "/etc/kubernetes/pki/front-proxy/ca",
			ReadOnly:  true,
		},
		{
			Name:      resources.AuditConfigMapName,
			MountPath: "/etc/kubernetes/audit",
			ReadOnly:  true,
		},
		{
			Name:      resources.AuditLogVolumeName,
			MountPath: "/var/log/kubernetes/audit",
			ReadOnly:  false,
		},
		{
			Name:      resources.AdmissionControlConfigMapName,
			MountPath: "/etc/kubernetes/adm-control",
			ReadOnly:  true,
		},
	}, resources.GetHostCACertVolumeMounts()...)
}

func getVolumes() []corev1.Volume {
	return append([]corev1.Volume{
		{
			Name: resources.ApiserverTLSSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ApiserverTLSSecretName,
				},
			},
		},
		{
			Name: resources.TokensSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.TokensSecretName,
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
			Name: resources.KubeletClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.KubeletClientCertificatesSecretName,
				},
			},
		},
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CASecretName,
					Items: []corev1.KeyToPath{
						{
							Path: resources.CACertSecretKey,
							Key:  resources.CACertSecretKey,
						},
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
			Name: resources.ApiserverEtcdClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ApiserverEtcdClientCertificateSecretName,
				},
			},
		},
		{
			Name: resources.ApiserverFrontProxyClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ApiserverFrontProxyClientCertificateSecretName,
				},
			},
		},
		{
			Name: resources.FrontProxyCASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.FrontProxyCASecretName,
				},
			},
		},
		{
			Name: resources.KubeletDnatControllerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.KubeletDnatControllerKubeconfigSecretName,
				},
			},
		},
		{
			Name: resources.AuditConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.AuditConfigMapName,
					},
					Optional: new(bool),
				},
			},
		},
		{
			Name: resources.AuditLogVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: resources.AdmissionControlConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.AdmissionControlConfigMapName,
					},
				},
			},
		},
	}, resources.GetHostCACertVolumes()...)
}

type kubeAPIServerEnvData interface {
	resources.CredentialsData
	Seed() *kubermaticv1.Seed
}

func GetEnvVars(data kubeAPIServerEnvData) ([]corev1.EnvVar, error) {
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
	return append(vars, resources.GetHTTPProxyEnvVarsFromSeed(data.Seed(), data.Cluster().Address.InternalName)...), nil
}

func getDexCASecretVolume() corev1.Volume {
	return corev1.Volume{
		Name: resources.DexCASecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: resources.DexCASecretName,
			},
		},
	}
}
