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
	"path/filepath"
	"strings"

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"
	"k8c.io/kubermatic/v2/pkg/resources/etcd/etcdrunning"
	"k8c.io/kubermatic/v2/pkg/resources/konnectivity"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
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

	gte131, _ = semverlib.NewConstraint(">= 1.31")
	lt135, _  = semverlib.NewConstraint("<= 1.34")
)

const (
	name                 = "apiserver"
	auditLogsSidecarName = "audit-logs"
)

// DeploymentReconciler returns the function to create and update the API server deployment.
func DeploymentReconciler(data *resources.TemplateData, enableOIDCAuthentication bool) reconciling.NamedDeploymentReconcilerFactory {
	enableEncryptionConfiguration := data.Cluster().IsEncryptionEnabled() || data.Cluster().IsEncryptionActive()

	return func() (string, reconciling.DeploymentReconciler) {
		return resources.ApiserverDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			var err error

			baseLabels := resources.BaseAppLabels(resources.ApiserverDeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)
			override := data.Cluster().Spec.ComponentsOverride.Apiserver
			if override.Replicas != nil {
				dep.Spec.Replicas = override.Replicas
			}
			dep.Spec.Template.Spec.Tolerations = override.Tolerations

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			dep.Spec.Template.Spec.ServiceAccountName = rbac.EtcdLauncherServiceAccountName
			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(true)

			auditLogEnabled := data.Cluster().Spec.AuditLogging != nil && data.Cluster().Spec.AuditLogging.Enabled
			auditWebhookBackendEnabled := data.Cluster().Spec.AuditLogging != nil && data.Cluster().Spec.AuditLogging.WebhookBackend != nil

			volumes := getVolumes(data, enableEncryptionConfiguration, auditLogEnabled, auditWebhookBackendEnabled)
			volumeMounts := getVolumeMounts(data, enableEncryptionConfiguration, auditWebhookBackendEnabled)

			version := data.Cluster().Status.Versions.Apiserver.Semver()
			address := data.Cluster().Status.Address

			// these volumes should not block the autoscaler from evicting the pod
			safeToEvictVolumes := []string{resources.AuditLogVolumeName, resources.KonnectivityUDS}

			kubernetes.EnsureLabels(&dep.Spec.Template, map[string]string{
				resources.VersionLabel: version.String(),
			})

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				"prometheus.io/scrape_with_kube_cert":                   "true",
				"prometheus.io/path":                                    "/metrics",
				"prometheus.io/port":                                    fmt.Sprint(address.Port),
				resources.ClusterLastRestartAnnotation:                  data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: strings.Join(safeToEvictVolumes, ","),
			})

			etcdEndpoints := etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName)

			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				etcdrunning.Container(etcdEndpoints, data),
			}

			var konnectivityProxySidecar *corev1.Container
			var openvpnSidecar *corev1.Container
			var dnatControllerSidecar *corev1.Container

			if data.IsKonnectivityEnabled() {
				konnectivityProxySidecar, err = konnectivity.ProxySidecar(data, *dep.Spec.Replicas)
				if err != nil {
					return nil, fmt.Errorf("failed to get konnectivity-proxy sidecar: %w", err)
				}
			} else {
				openvpnSidecar, err = vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
				if err != nil {
					return nil, fmt.Errorf("failed to get openvpn-client sidecar: %w", err)
				}

				dnatControllerSidecar, err = vpnsidecar.DnatControllerContainer(
					data,
					"dnat-controller",
					fmt.Sprintf("https://127.0.0.1:%d", address.Port),
				)
				if err != nil {
					return nil, fmt.Errorf("failed to get dnat-controller sidecar: %w", err)
				}
			}

			flags, err := getApiserverFlags(
				data, etcdEndpoints, enableOIDCAuthentication, auditLogEnabled,
				enableEncryptionConfiguration, auditWebhookBackendEnabled,
			)
			if err != nil {
				return nil, err
			}

			if auditWebhookBackendEnabled && data.Cluster().Spec.AuditLogging.WebhookBackend.AuditWebhookInitialBackoff != "" {
				flags = append(flags, "--audit-webhook-initial-backoff", data.Cluster().Spec.AuditLogging.WebhookBackend.AuditWebhookInitialBackoff)
			}

			envVars, err := GetEnvVars(data)
			if err != nil {
				return nil, err
			}

			apiserverContainer := &corev1.Container{
				Name:    resources.ApiserverDeploymentName,
				Image:   registry.Must(data.RewriteImage(resources.RegistryK8S + "/kube-apiserver:v" + version.String())),
				Command: []string{"/usr/local/bin/kube-apiserver"},
				Env:     envVars,
				Args:    flags,
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: address.Port,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path:   "/healthz",
							Port:   intstr.FromInt(int(address.Port)),
							Scheme: "HTTPS",
						},
					},
					FailureThreshold: 3,
					PeriodSeconds:    5,
					SuccessThreshold: 1,
					TimeoutSeconds:   15,
				},
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path:   "/healthz",
							Port:   intstr.FromInt(int(address.Port)),
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
			}

			var defResourceRequirements map[string]*corev1.ResourceRequirements
			if data.IsKonnectivityEnabled() {
				dep.Spec.Template.Spec.Containers = []corev1.Container{
					*konnectivityProxySidecar,
					*apiserverContainer,
				}
				defResourceRequirements = map[string]*corev1.ResourceRequirements{
					name:                          defaultResourceRequirements.DeepCopy(),
					konnectivityProxySidecar.Name: konnectivityProxySidecar.Resources.DeepCopy(),
				}
			} else {
				dep.Spec.Template.Spec.Containers = []corev1.Container{
					*openvpnSidecar,
					*dnatControllerSidecar,
					*apiserverContainer,
				}

				defResourceRequirements = map[string]*corev1.ResourceRequirements{
					name:                       defaultResourceRequirements.DeepCopy(),
					openvpnSidecar.Name:        openvpnSidecar.Resources.DeepCopy(),
					dnatControllerSidecar.Name: dnatControllerSidecar.Resources.DeepCopy(),
				}
			}

			overrides := resources.GetOverrides(data.Cluster().Spec.ComponentsOverride)

			if auditLogEnabled {
				defResourceRequirements[auditLogsSidecarName] = &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("10Mi"),
						corev1.ResourceCPU:    resource.MustParse("5m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("60Mi"),
						corev1.ResourceCPU:    resource.MustParse("50m"),
					},
				}

				if data.Cluster().Spec.AuditLogging.SidecarSettings != nil && data.Cluster().Spec.AuditLogging.SidecarSettings.Resources != nil {
					overrides[auditLogsSidecarName] = data.Cluster().Spec.AuditLogging.SidecarSettings.Resources
				}

				dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers,
					corev1.Container{
						Name:    auditLogsSidecarName,
						Image:   registry.Must(data.RewriteImage(resources.RegistryDocker + "/fluent/fluent-bit:4.0.0")),
						Command: []string{"/fluent-bit/bin/fluent-bit"},
						Args:    []string{"-c", "/etc/fluent-bit/fluent-bit.conf"},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      resources.AuditLogVolumeName,
								MountPath: "/var/log/kubernetes/audit",
								ReadOnly:  false,
							},
							{
								Name:      resources.FluentBitSecretName,
								MountPath: "/etc/fluent-bit/",
								ReadOnly:  true,
							},
						},
						Env: data.GetAuditLoggingSidecarEnvs(),
					},
				)
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, overrides, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(name, kubermaticv1.AntiAffinityTypePreferred)

			return dep, nil
		}
	}
}

//nolint:gocyclo
func getApiserverFlags(
	data *resources.TemplateData,
	etcdEndpoints []string,
	enableOIDCAuthentication, auditLogEnabled, enableEncryption, auditWebhookEnabled bool,
) ([]string, error) {
	overrideFlags, err := getApiserverOverrideFlags(data)
	if err != nil {
		return nil, fmt.Errorf("could not get components override flags: %w", err)
	}

	cluster := data.Cluster()

	admissionPlugins := sets.New(
		"NamespaceLifecycle",
		"NodeRestriction",
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

	if useEventRateLimitAdmissionPlugin(data) {
		admissionPlugins.Insert(resources.EventRateLimitAdmissionPlugin)
	}

	admissionPlugins.Insert(cluster.Spec.AdmissionPlugins...)

	address := data.Cluster().Status.Address

	serviceAccountKeyFile := filepath.Join("/etc/kubernetes/service-account-key", resources.ServiceAccountKeySecretKey)
	flags := []string{
		"--etcd-servers", strings.Join(etcdEndpoints, ","),
		"--etcd-cafile", "/etc/etcd/pki/client/ca.crt",
		"--etcd-certfile", filepath.Join("/etc/etcd/pki/client", resources.ApiserverEtcdClientCertificateCertSecretKey),
		"--etcd-keyfile", filepath.Join("/etc/etcd/pki/client", resources.ApiserverEtcdClientCertificateKeySecretKey),
		"--storage-backend", "etcd3",
		"--enable-admission-plugins", strings.Join(sets.List(admissionPlugins), ","),
		"--admission-control-config-file", "/etc/kubernetes/adm-control/admission-control.yaml",
		"--external-hostname", address.ExternalName,
		"--token-auth-file", "/etc/kubernetes/tokens/tokens.csv",
		"--enable-bootstrap-token-auth",
		"--service-account-key-file", serviceAccountKeyFile,
		"--service-cluster-ip-range", strings.Join(cluster.Spec.ClusterNetwork.Services.CIDRBlocks, ","),
		"--service-node-port-range", overrideFlags.NodePortRange,
		"--allow-privileged",
		"--audit-log-maxage", "30",
		"--audit-log-maxbackup", "3",
		"--audit-log-maxsize", "100",
		"--audit-log-path", "/var/log/kubernetes/audit/audit.log",
		"--tls-cert-file", "/etc/kubernetes/tls/apiserver-tls.crt",
		"--tls-cipher-suites", strings.Join(resources.GetAllowedTLSCipherSuites(), ","),
		"--tls-private-key-file", "/etc/kubernetes/tls/apiserver-tls.key",
		"--proxy-client-cert-file", "/etc/kubernetes/pki/front-proxy/client/" + resources.ApiserverProxyClientCertificateCertSecretKey,
		"--proxy-client-key-file", "/etc/kubernetes/pki/front-proxy/client/" + resources.ApiserverProxyClientCertificateKeySecretKey,
		"--client-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--kubelet-client-certificate", "/etc/kubernetes/kubelet/kubelet-client.crt",
		"--kubelet-client-key", "/etc/kubernetes/kubelet/kubelet-client.key",
	}

	if !cluster.Spec.IsAuthorizationConfigurationFileEnabled() {
		flags = append(flags, "--authorization-mode", getAuthorizationModesString(data.Cluster().Spec.AuthorizationConfig))
	}

	// the "bring-your-own" provider does not support automatic TLS rotation in kubelets yet,
	// and because of that certs might expire and kube-apiserver cannot validate the connection anymore.
	if cluster.Spec.Cloud.BringYourOwn == nil && cluster.Spec.Cloud.Edge == nil {
		flags = append(flags, "--kubelet-certificate-authority", "/etc/kubernetes/pki/ca/ca.crt")
	}

	flags = append(flags,
		"--requestheader-client-ca-file", "/etc/kubernetes/pki/front-proxy/ca/ca.crt",
		"--requestheader-allowed-names", "apiserver-aggregator",
		"--requestheader-extra-headers-prefix", "X-Remote-Extra-",
		"--requestheader-group-headers", "X-Remote-Group",
		"--requestheader-username-headers", "X-Remote-User",
		"--endpoint-reconciler-type", "none",
		// this can't be passed as two strings as the other parameters
		"--profiling=false",
	)

	// prepend to have advertise-address as first argument and avoid
	// triggering unneeded redeployments.
	flags = append([]string{
		// advertise-address is the external IP under which the apiserver is available.
		// The same address is used for all apiserver replicas.
		"--advertise-address", address.IP,
		// The port on which apiserver is serving.
		// For Nodeport / LoadBalancer expose strategies we use the apiserver-external service NodePort value.
		// For Tunneling expose strategy we use a fixed port.
		"--secure-port", fmt.Sprint(address.Port),
	}, flags...)

	if auditLogEnabled || auditWebhookEnabled {
		flags = append(flags, "--audit-policy-file", "/etc/kubernetes/audit/policy.yaml")
	}

	if auditWebhookEnabled {
		flags = append(flags, "--audit-webhook-config-file", "/etc/kubernetes/audit/webhook/webhook.yaml")
	}
	// enable service account signing key and issuer in Kubernetes 1.20 or when
	// explicitly enabled in the cluster object
	var audiences []string

	issuer := address.URL
	if saConfig := cluster.Spec.ServiceAccount; saConfig != nil {
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

	if data.IsKonnectivityEnabled() {
		audiences = append(audiences, "system:konnectivity-server")
	}

	flags = append(flags,
		"--service-account-issuer", issuer,
		"--service-account-signing-key-file", serviceAccountKeyFile,
		"--api-audiences", strings.Join(audiences, ","),
	)

	flags = append(flags, "--kubelet-preferred-address-types", resources.GetKubeletPreferredAddressTypes(cluster, data.IsKonnectivityEnabled()))

	cloudProviderName := resources.GetKubernetesCloudProviderName(data.Cluster(), resources.ExternalCloudProviderEnabled(data.Cluster()))
	if cloudProviderName != "" && cloudProviderName != resources.CloudProviderExternalFlag {
		flags = append(flags, "--cloud-provider", cloudProviderName)
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}

	oidcSettings := cluster.Spec.OIDC
	if oidcSettings.IssuerURL != "" && oidcSettings.ClientID != "" {
		flags = append(flags,
			"--oidc-ca-file", fmt.Sprintf("/etc/kubernetes/pki/ca-bundle/%s", resources.CABundleConfigMapKey),
			"--oidc-issuer-url", oidcSettings.IssuerURL,
			"--oidc-client-id", oidcSettings.ClientID,
		)

		if oidcSettings.UsernameClaim != "" {
			flags = append(flags, "--oidc-username-claim", oidcSettings.UsernameClaim)
		}
		if oidcSettings.GroupsClaim != "" {
			flags = append(flags, "--oidc-groups-claim", oidcSettings.GroupsClaim)
		}
		if oidcSettings.RequiredClaim != "" {
			flags = append(flags, "--oidc-required-claim", oidcSettings.RequiredClaim)
		}
		if oidcSettings.GroupsPrefix != "" {
			flags = append(flags, "--oidc-groups-prefix", oidcSettings.GroupsPrefix)
		}
		if oidcSettings.UsernamePrefix != "" {
			flags = append(flags, "--oidc-username-prefix", oidcSettings.UsernamePrefix)
		}
	} else if enableOIDCAuthentication {
		flags = append(flags,
			"--oidc-ca-file", fmt.Sprintf("/etc/kubernetes/pki/ca-bundle/%s", resources.CABundleConfigMapKey),
			"--oidc-issuer-url", data.OIDCIssuerURL(),
			"--oidc-client-id", data.OIDCIssuerClientID(),
			"--oidc-username-claim", "email",
			"--oidc-groups-prefix", "oidc:",
			"--oidc-groups-claim", "groups",
		)
	}

	featureGates := data.GetCSIMigrationFeatureGates(cluster.Status.Versions.Apiserver.Semver())

	if data.DRAEnabled() {
		featureGates = append(featureGates, "DynamicResourceAllocation=true")
		flags = append(flags, "--runtime-config=resource.k8s.io/v1beta1=true")
	}

	version := cluster.Status.Versions.Apiserver.Semver()
	if gte131.Check(version) && lt135.Check(version) {
		// enable recommended CEL cost feature gates (Kube 1.31-1.34), as per
		// https://github.com/kubernetes/kubernetes/pull/124675
		// Note: These feature gates were graduated to GA and removed in Kubernetes 1.35
		featureGates = append(featureGates,
			"StrictCostEnforcementForVAP=true",
			"StrictCostEnforcementForWebhooks=true",
		)
	}

	if len(featureGates) > 0 {
		flags = append(flags, "--feature-gates")
		flags = append(flags, strings.Join(featureGates, ","))
	}

	if data.IsKonnectivityEnabled() {
		flags = append(flags, "--egress-selector-config-file",
			"/etc/kubernetes/konnectivity/egress-selector-configuration.yaml")
	}

	if enableEncryption {
		flags = append(flags, "--encryption-provider-config",
			"/etc/kubernetes/encryption-configuration/encryption-configuration.yaml")
	}

	flags = append(flags, generateAuthorizationFlags(cluster)...)

	return flags, nil
}

func generateAuthorizationFlags(cluster *kubermaticv1.Cluster) []string {
	flags := []string{}
	if cluster.Spec.IsWebhookAuthorizationEnabled() {
		flags = append(flags, "--authorization-webhook-config-file", "/etc/kubernetes/authorization-webhook.yaml")
		flags = append(flags, "--authorization-webhook-version", cluster.Spec.GetAuthorizationWebhookVersion())

		// Add cache TTL flags if specified by user (override Kubernetes defaults)
		webhookConfig := cluster.Spec.AuthorizationConfig.AuthorizationWebhookConfiguration
		if webhookConfig.CacheAuthorizedTTL != nil {
			flags = append(flags, "--authorization-webhook-cache-authorized-ttl", webhookConfig.CacheAuthorizedTTL.Duration.String())
		}
		if webhookConfig.CacheUnauthorizedTTL != nil {
			flags = append(flags, "--authorization-webhook-cache-unauthorized-ttl", webhookConfig.CacheUnauthorizedTTL.Duration.String())
		}
	}

	if cluster.Spec.IsAuthorizationConfigurationFileEnabled() {
		flags = append(flags, "--authorization-config", filepath.Join(getAuthorizationConfigurationMountPath(cluster.Spec.AuthorizationConfig), cluster.Spec.AuthorizationConfig.AuthorizationConfigurationFile.SecretKey))
	}

	return flags
}

// getApiserverOverrideFlags creates all settings that may be overridden by cluster specific componentsOverrideSettings
// otherwise global overrides or defaults will be set.
func getApiserverOverrideFlags(data *resources.TemplateData) (kubermaticv1.APIServerSettings, error) {
	settings := kubermaticv1.APIServerSettings{
		NodePortRange: data.ComputedNodePortRange(),
	}

	// endpointReconcilingDisabled section
	settings.EndpointReconcilingDisabled = new(bool)
	if data.Cluster().Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled != nil {
		settings.EndpointReconcilingDisabled = data.Cluster().Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled
	}

	return settings, nil
}

func getVolumeMounts(data *resources.TemplateData, isEncryptionEnabled bool, isAuditWebhookEnabled bool) []corev1.VolumeMount {
	vms := []corev1.VolumeMount{
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
	}

	if data.IsKonnectivityEnabled() {
		vms = append(vms, []corev1.VolumeMount{
			{
				Name:      resources.KonnectivityUDS,
				MountPath: "/etc/kubernetes/konnectivity-server",
			},
			{
				Name:      resources.KonnectivityKubeApiserverEgress,
				MountPath: "/etc/kubernetes/konnectivity",
				ReadOnly:  true,
			},
		}...)
	}

	if isEncryptionEnabled {
		vms = append(vms, corev1.VolumeMount{
			Name:      resources.EncryptionConfigurationSecretName,
			MountPath: "/etc/kubernetes/encryption-configuration",
			ReadOnly:  true,
		})
	}

	if isAuditWebhookEnabled {
		vms = append(vms, corev1.VolumeMount{
			Name:      resources.AuditWebhookVolumeName,
			MountPath: "/etc/kubernetes/audit/webhook",
			ReadOnly:  true,
		})
	}

	if data.Cluster().Spec.IsWebhookAuthorizationEnabled() {
		vms = append(vms, corev1.VolumeMount{
			Name:      resources.AuthorizationWebhookVolumeName,
			MountPath: "/etc/kubernetes/authorization-webhook.yaml",
			SubPath:   data.Cluster().Spec.AuthorizationConfig.AuthorizationWebhookConfiguration.SecretKey,
			ReadOnly:  true,
		})
	}

	if data.Cluster().Spec.IsAuthorizationConfigurationFileEnabled() {
		vms = append(vms, corev1.VolumeMount{
			Name:      resources.AuthorizationConfigurationVolumeName,
			MountPath: getAuthorizationConfigurationMountPath(data.Cluster().Spec.AuthorizationConfig),
			ReadOnly:  true,
		})
	}

	return vms
}

func getVolumes(data *resources.TemplateData, isEncryptionEnabled, isAuditEnabled bool, isAuditWebhookEnabled bool) []corev1.Volume {
	vs := []corev1.Volume{
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
	}

	if data.IsKonnectivityEnabled() {
		vs = append(vs, []corev1.Volume{
			{
				Name: resources.KonnectivityKubeconfigSecretName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  resources.KonnectivityKubeconfigSecretName,
						DefaultMode: intPtr(420),
					},
				},
			},
			{
				Name: resources.KonnectivityProxyTLSSecretName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  resources.KonnectivityProxyTLSSecretName,
						DefaultMode: intPtr(420),
					},
				},
			},
			{
				Name: resources.KonnectivityUDS,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: resources.KonnectivityKubeApiserverEgress,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.KonnectivityKubeApiserverEgress,
						},
						DefaultMode: intPtr(420),
					},
				},
			},
		}...)
	} else {
		vs = append(vs, []corev1.Volume{
			{
				Name: resources.OpenVPNClientCertificatesSecretName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: resources.OpenVPNClientCertificatesSecretName,
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
		}...)
	}

	if isEncryptionEnabled {
		vs = append(vs, corev1.Volume{
			Name: resources.EncryptionConfigurationSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.EncryptionConfigurationSecretName,
				},
			},
		})
	}

	if isAuditEnabled {
		vs = append(vs, corev1.Volume{
			Name: resources.FluentBitSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.FluentBitSecretName,
				},
			},
		})
	}

	if isAuditWebhookEnabled {
		vs = append(vs, corev1.Volume{
			Name: resources.AuditWebhookVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: data.Cluster().Spec.AuditLogging.WebhookBackend.AuditWebhookConfig.Name,
				},
			},
		})
	}

	if data.Cluster().Spec.IsWebhookAuthorizationEnabled() {
		vs = append(vs, corev1.Volume{
			Name: resources.AuthorizationWebhookVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: data.Cluster().Spec.AuthorizationConfig.AuthorizationWebhookConfiguration.SecretName,
				},
			},
		})
	}

	if data.Cluster().Spec.IsAuthorizationConfigurationFileEnabled() {
		vs = append(vs, corev1.Volume{
			Name: resources.AuthorizationConfigurationVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: data.Cluster().Spec.AuthorizationConfig.AuthorizationConfigurationFile.SecretName,
				},
			},
		})
	}

	return vs
}

type kubeAPIServerEnvData interface {
	Cluster() *kubermaticv1.Cluster
	Seed() *kubermaticv1.Seed
}

func GetEnvVars(data kubeAPIServerEnvData) ([]corev1.EnvVar, error) {
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

	return append(vars, resources.GetHTTPProxyEnvVarsFromSeed(data.Seed(), data.Cluster().Status.Address.InternalName)...), nil
}

func getAuthorizationModesString(authorizationConfig *kubermaticv1.AuthorizationConfig) string {
	if authorizationConfig != nil && authorizationConfig.EnabledModes != nil {
		return strings.Join(authorizationConfig.EnabledModes, ",")
	}

	return "Node,RBAC"
}

func getAuthorizationConfigurationMountPath(authorizationConfig *kubermaticv1.AuthorizationConfig) string {
	if authorizationConfig != nil && authorizationConfig.AuthorizationConfigurationFile != nil && len(authorizationConfig.AuthorizationConfigurationFile.SecretMountPath) > 0 {
		return authorizationConfig.AuthorizationConfigurationFile.SecretMountPath
	}

	return "/etc/kubernetes/authorization-configs"
}

func intPtr(n int32) *int32 {
	return &n
}
