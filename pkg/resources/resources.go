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

package resources

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/util/s3"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	corev1lister "k8s.io/client-go/listers/core/v1"
	certutil "k8s.io/client-go/util/cert"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ApiServer secure port.
	APIServerSecurePort = 6443

	NodeLocalDNSCacheAddress = "169.254.20.10"
)

const (
	// ApiserverDeploymentName is the name of the apiserver deployment.
	ApiserverDeploymentName = "apiserver"
	// ControllerManagerDeploymentName is the name for the controller manager deployment.
	ControllerManagerDeploymentName = "controller-manager"
	// SchedulerDeploymentName is the name for the scheduler deployment.
	SchedulerDeploymentName = "scheduler"
	// OperatingSystemManagerDeploymentName is the name for the operating-system-manager deployment.
	OperatingSystemManagerDeploymentName = "operating-system-manager"
	// OperatingSystemManagerContainerName is the name for the container created within the operating-system-manager deployment.
	OperatingSystemManagerContainerName = "operating-system-manager"
	// OperatingSystemManagerWebhookDeploymentName is the name for the operating-system-manager webhook deployment.
	OperatingSystemManagerWebhookDeploymentName = "operating-system-manager-webhook"
	// OperatingSystemManagerWebhookServiceName is the name for the operating-system-manager webhook service.
	OperatingSystemManagerWebhookServiceName = "operating-system-manager-webhook"
	// MachineControllerDeploymentName is the name for the machine-controller deployment.
	MachineControllerDeploymentName = "machine-controller"
	// MachineControllerWebhookDeploymentName is the name for the machine-controller webhook deployment.
	MachineControllerWebhookDeploymentName = "machine-controller-webhook"
	// MetricsServerDeploymentName is the name for the metrics-server deployment.
	MetricsServerDeploymentName = "metrics-server"
	// OpenVPNServerDeploymentName is the name for the openvpn server deployment.
	OpenVPNServerDeploymentName = "openvpn-server"
	// DNSResolverDeploymentName is the name of the dns resolver deployment.
	DNSResolverDeploymentName = "dns-resolver"
	// DNSResolverConfigMapName is the name of the dns resolvers configmap.
	DNSResolverConfigMapName = "dns-resolver"
	// DNSResolverServiceName is the name of the dns resolvers service.
	DNSResolverServiceName = "dns-resolver"
	// DNSResolverPodDisruptionBudetName is the name of the dns resolvers pdb.
	DNSResolverPodDisruptionBudetName = "dns-resolver"
	// KubeStateMetricsDeploymentName is the name of the kube-state-metrics deployment.
	KubeStateMetricsDeploymentName = "kube-state-metrics"
	// UserClusterControllerDeploymentName is the name of the usercluster-controller deployment.
	UserClusterControllerDeploymentName = "usercluster-controller"
	// UserClusterControllerContainerName is the name of the container within the usercluster-controller deployment.
	UserClusterControllerContainerName = "usercluster-controller"
	// KubernetesDashboardDeploymentName is the name of the Kubernetes Dashboard deployment.
	KubernetesDashboardDeploymentName = "kubernetes-dashboard"
	// KubeLBDeploymentName is the name of the KubeLB deployment.
	KubeLBDeploymentName = "kubelb-ccm"
	// MetricsScraperDeploymentName is the name of dashboard-metrics-scraper deployment.
	MetricsScraperDeploymentName = "dashboard-metrics-scraper"
	// MetricsScraperServiceName is the name of dashboard-metrics-scraper service.
	MetricsScraperServiceName = "dashboard-metrics-scraper"
	// PrometheusStatefulSetName is the name for the prometheus StatefulSet.
	PrometheusStatefulSetName = "prometheus"
	// EtcdStatefulSetName is the name for the etcd StatefulSet.
	EtcdStatefulSetName = "etcd"
	// EtcdDefaultBackupConfigName is the name for the default (preinstalled) EtcdBackupConfig of a cluster.
	EtcdDefaultBackupConfigName = "default-backups"
	// EtcdTLSEnabledAnnotation is the annotation assigned to etcd Pods that run with a TLS peer endpoint.
	EtcdTLSEnabledAnnotation = "etcd.kubermatic.k8c.io/tls-peer-enabled"
	// EncryptionConfigurationSecretName is the name of secret storing the API server's EncryptionConfiguration.
	EncryptionConfigurationSecretName = "apiserver-encryption-configuration"
	// EncryptionConfigurationKeyName is the name of the secret key that is used to store the configuration file for encryption-at-rest.
	EncryptionConfigurationKeyName = "encryption-configuration.yaml"
	// NodePortProxyEnvoyDeploymentName is the name of the nodeport-proxy deployment in the user cluster.
	NodePortProxyEnvoyDeploymentName = "nodeport-proxy-envoy"
	// NodePortProxyEnvoyContainerName is the name of the envoy container in the nodeport-proxy deployment.
	NodePortProxyEnvoyContainerName = "envoy"

	// ApiserverServiceName is the name for the apiserver service.
	ApiserverServiceName = "apiserver-external"
	// FrontLoadBalancerServiceName is the name of the LoadBalancer service that fronts everything
	// when using exposeStrategy "LoadBalancer".
	FrontLoadBalancerServiceName = "front-loadbalancer"
	// MetricsServerServiceName is the name for the metrics-server service.
	MetricsServerServiceName = "metrics-server"
	// MetricsServerExternalNameServiceName is the name for the metrics-server service inside the user cluster.
	MetricsServerExternalNameServiceName = "metrics-server"
	// EtcdServiceName is the name for the etcd service.
	EtcdServiceName = "etcd"
	// EtcdDefragCronJobName is the name for the defrag cronjob deployment.
	EtcdDefragCronJobName = "etcd-defragger"
	// OpenVPNServerServiceName is the name for the openvpn server service.
	OpenVPNServerServiceName = "openvpn-server"
	// MachineControllerWebhookServiceName is the name of the machine-controller webhook service.
	MachineControllerWebhookServiceName = "machine-controller-webhook"
	// MetricsServerAPIServiceName is the name for the metrics-server APIService.
	MetricsServerAPIServiceName = "v1beta1.metrics.k8s.io"

	// AdminKubeconfigSecretName is the name for the secret containing the private ca key.
	AdminKubeconfigSecretName = "admin-kubeconfig"
	// ViewerKubeconfigSecretName is the name for the secret containing the viewer kubeconfig.
	ViewerKubeconfigSecretName = "viewer-kubeconfig"
	// SchedulerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the scheduler.
	SchedulerKubeconfigSecretName = "scheduler-kubeconfig"
	// KubeLBCCMCertUsername is the name of the user coming from kubeconfig cert.
	KubeLBCCMCertUsername = "kubermatic:kubelb-ccm"

	// KubeletDnatControllerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the kubeletdnatcontroller.
	KubeletDnatControllerKubeconfigSecretName = "kubeletdnatcontroller-kubeconfig"
	// KubeStateMetricsKubeconfigSecretName is the name for the secret containing the kubeconfig used by kube-state-metrics.
	KubeStateMetricsKubeconfigSecretName = "kube-state-metrics-kubeconfig"
	// MetricsServerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the metrics-server.
	MetricsServerKubeconfigSecretName = "metrics-server"
	// ControllerManagerKubeconfigSecretName is the name of the secret containing the kubeconfig used by controller manager.
	ControllerManagerKubeconfigSecretName = "controllermanager-kubeconfig"
	// OperatingSystemManagerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the osm.
	OperatingSystemManagerKubeconfigSecretName = "operatingsystemmanager-kubeconfig"
	// OperatingSystemManagerWebhookKubeconfigSecretName is the name for the secret containing the kubeconfig used by the osm webhook.
	OperatingSystemManagerWebhookKubeconfigSecretName = "operatingsystemmanager-webhook-kubeconfig"
	// MachineControllerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the machinecontroller.
	MachineControllerKubeconfigSecretName = "machinecontroller-kubeconfig"
	// CloudControllerManagerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the external cloud provider.
	CloudControllerManagerKubeconfigSecretName = "cloud-controller-manager-kubeconfig"
	// MachineControllerWebhookServingCertSecretName is the name for the secret containing the serving cert for the
	// machine-controller webhook.
	MachineControllerWebhookServingCertSecretName = "machinecontroller-webhook-serving-cert"
	// MachineControllerWebhookServingCertCertKeyName is the name for the key that contains the cert.
	MachineControllerWebhookServingCertCertKeyName = "cert.pem"
	// MachineControllerWebhookServingCertKeyKeyName is the name for the key that contains the key.
	MachineControllerWebhookServingCertKeyKeyName = "key.pem"
	// OperatingSystemManagerWebhookServingCertSecretName is the name for the operating-system-manager webhook TLS server certificate secret.
	OperatingSystemManagerWebhookServingCertSecretName = "operating-system-manager-webhook-serving-cert"
	// OperatingSystemManagerWebhookServingCertCertKeyName is the name for the key that contains the cert.
	OperatingSystemManagerWebhookServingCertCertKeyName = "tls.crt"
	// OperatingSystemManagerWebhookServingCertKeyKeyName is the name for the key that contains the private key.
	OperatingSystemManagerWebhookServingCertKeyKeyName = "tls.key"
	// PrometheusApiserverClientCertificateSecretName is the name for the secret containing the client certificate used by prometheus to access the apiserver.
	PrometheusApiserverClientCertificateSecretName = "prometheus-apiserver-certificate"
	// KubernetesDashboardKubeconfigSecretName is the name of the kubeconfig secret user for Kubernetes Dashboard.
	KubernetesDashboardKubeconfigSecretName = "kubernetes-dashboard-kubeconfig"
	// WEBTerminalKubeconfigSecretName is the name of the kubeconfig secret user for WEB terminal tools pod.
	WEBTerminalKubeconfigSecretName = "web-terminal-kubeconfig"

	// ImagePullSecretName specifies the name of the dockercfg secret used to access the private repo.
	ImagePullSecretName = "dockercfg"

	// FrontProxyCASecretName is the name for the secret containing the front proxy ca.
	FrontProxyCASecretName = "front-proxy-ca"
	// CASecretName is the name for the secret containing the root ca.
	CASecretName = "ca"
	// ApiserverTLSSecretName is the name for the secrets required for the apiserver tls.
	ApiserverTLSSecretName = "apiserver-tls"
	// KubeletClientCertificatesSecretName is the name for the secret containing the kubelet client certificates.
	KubeletClientCertificatesSecretName = "kubelet-client-certificates"
	// ServiceAccountKeySecretName is the name for the secret containing the service account key.
	ServiceAccountKeySecretName = "service-account-key"
	// TokensSecretName is the name for the secret containing the user tokens.
	TokensSecretName = "tokens"
	// ViewerTokenSecretName is the name for the secret containing the viewer token.
	ViewerTokenSecretName = "viewer-token"
	// OpenVPNCASecretName is the name of the secret that contains the OpenVPN CA.
	OpenVPNCASecretName = "openvpn-ca"
	// OpenVPNServerCertificatesSecretName is the name for the secret containing the openvpn server certificates.
	OpenVPNServerCertificatesSecretName = "openvpn-server-certificates"
	// OpenVPNClientCertificatesSecretName is the name for the secret containing the openvpn client certificates.
	OpenVPNClientCertificatesSecretName = "openvpn-client-certificates"
	// CloudConfigSecretName is the name for the secret containing the cloud-config inside the user cluster.
	CloudConfigSecretName = "cloud-config"
	// CSICloudConfigSecretName is the name for the secret containing the cloud-config used by the csi driver inside the user cluster.
	CSICloudConfigSecretName = "cloud-config-csi"
	// EtcdTLSCertificateSecretName is the name for the secret containing the etcd tls certificate used for transport security.
	EtcdTLSCertificateSecretName = "etcd-tls-certificate"
	// ApiserverEtcdClientCertificateSecretName is the name for the secret containing the client certificate used by the apiserver for authenticating against etcd.
	ApiserverEtcdClientCertificateSecretName = "apiserver-etcd-client-certificate"
	// ApiserverFrontProxyClientCertificateSecretName is the name for the secret containing the apiserver's client certificate for proxy auth.
	ApiserverFrontProxyClientCertificateSecretName = "apiserver-proxy-client-certificate"
	// GoogleServiceAccountSecretName is the name of the secret that contains the Google Service Account.
	GoogleServiceAccountSecretName = "google-service-account"
	// GoogleServiceAccountVolumeName is the name of the volume containing the Google Service Account secret.
	GoogleServiceAccountVolumeName = "google-service-account-volume"
	// AuditLogVolumeName is the name of the volume that hold the audit log of the apiserver.
	AuditLogVolumeName = "audit-log"
	// KubernetesDashboardKeyHolderSecretName is the name of the secret that contains JWE token encryption key
	// used by the Kubernetes Dashboard.
	KubernetesDashboardKeyHolderSecretName = "kubernetes-dashboard-key-holder"
	// KubernetesDashboardCsrfTokenSecretName is the name of the secret that contains CSRF token used by
	// the Kubernetes Dashboard.
	KubernetesDashboardCsrfTokenSecretName = "kubernetes-dashboard-csrf"

	// CABundleConfigMapName is the name for the configmap that contains the CA bundle for all usercluster components.
	CABundleConfigMapName = "ca-bundle"
	// CABundleConfigMapKey is the key under which a ConfigMap must contain a PEM-encoded collection of certificates.
	CABundleConfigMapKey = "ca-bundle.pem"

	// CloudConfigSeedSecretName is the name for the secret containing the cloud-config inside the usercluster namespace
	// on the seed cluster. Not to be confused with CloudConfigSecretName, which is the copy of this Secret inside the
	// usercluster.
	CloudConfigSeedSecretName = "cloud-config"
	// CloudConfigKey is the key under which the cloud-config in the cloud-config Secret can be found.
	CloudConfigKey = "config"
	// OpenVPNClientConfigsConfigMapName is the name for the ConfigMap containing the OpenVPN client config used within the user cluster.
	OpenVPNClientConfigsConfigMapName = "openvpn-client-configs"
	// OpenVPNClientConfigConfigMapName is the name for the ConfigMap containing the OpenVPN client config used by the client inside the user cluster.
	OpenVPNClientConfigConfigMapName = "openvpn-client-config"
	// ClusterInfoConfigMapName is the name for the ConfigMap containing the cluster-info used by the bootstrap token mechanism.
	ClusterInfoConfigMapName = "cluster-info"
	// PrometheusConfigConfigMapName is the name for the configmap containing the prometheus config.
	PrometheusConfigConfigMapName = "prometheus"
	// AuditConfigMapName is the name for the configmap that contains the content of the file that will be passed to the apiserver with the flag "--audit-policy-file".
	AuditConfigMapName = "audit-config"

	// FluentBitSecretName is the name of the secret that contains the fluent-bit configuration mounted
	// into kube-apisever and used by the "audit-logs" sidecar to ship audit logs.
	FluentBitSecretName = "audit-logs-fluentbit"

	// AuditWebhookVolumeName is the name of the volume that contains the audit webhook configuration mounted into kube-apisever.
	AuditWebhookVolumeName = "audit-webhook-backend"

	// AdmissionControlConfigMapName is the name for the configmap that contains the Admission Controller config file.
	AdmissionControlConfigMapName = "adm-control"

	// PrometheusServiceAccountName is the name for the Prometheus serviceaccount.
	PrometheusServiceAccountName = "prometheus"

	// PrometheusRoleName is the name for the Prometheus role.
	PrometheusRoleName = "prometheus"

	// PrometheusRoleBindingName is the name for the Prometheus rolebinding.
	PrometheusRoleBindingName = "prometheus"

	// CloudControllerManagerRoleBindingName is the name for the cloud controller manager rolebinding.
	CloudControllerManagerRoleBindingName = "cloud-controller-manager"

	// DefaultServiceAccountName is the name of Kubernetes default service accounts.
	DefaultServiceAccountName = "default"

	// OperatingSystemManagerCertUsername is the name of the user coming from kubeconfig cert.
	OperatingSystemManagerCertUsername = "operating-system-manager"
	// OperatingSystemManagerWebhookCertUsername is the name of the user coming from the kubeconfig cert.
	OperatingSystemManagerWebhookCertUsername = "operating-system-manager-webhook"
	// MachineControllerCertUsername is the name of the user coming from kubeconfig cert.
	MachineControllerCertUsername = "machine-controller"
	// KubeStateMetricsCertUsername is the name of the user coming from kubeconfig cert.
	KubeStateMetricsCertUsername = "kube-state-metrics"
	// MetricsServerCertUsername is the name of the user coming from kubeconfig cert.
	MetricsServerCertUsername = "metrics-server"
	// MetricsServerServiceAccountName is the name of the metrics server service account.
	MetricsServerServiceAccountName = "metrics-server"
	// ControllerManagerCertUsername is the name of the user coming from kubeconfig cert.
	ControllerManagerCertUsername = "system:kube-controller-manager"
	// CloudControllerManagerCertUsername is the name of the user coming from kubeconfig cert.
	CloudControllerManagerCertUsername = "system:cloud-controller-manager"
	// SchedulerCertUsername is the name of the user coming from kubeconfig cert.
	SchedulerCertUsername = "system:kube-scheduler"
	// KubeletDnatControllerCertUsername is the name of the user coming from kubeconfig cert.
	KubeletDnatControllerCertUsername = "kubermatic:kubeletdnat-controller"
	// PrometheusCertUsername is the name of the user coming from kubeconfig cert.
	PrometheusCertUsername = "prometheus"

	// KubernetesDashboardCertUsername is the name of the user coming from kubeconfig cert.
	KubernetesDashboardCertUsername = "kubermatic:kubernetes-dashboard"
	// MetricsScraperServiceAccountUsername is the name of the user coming from kubeconfig cert.
	MetricsScraperServiceAccountUsername = "dashboard-metrics-scraper"

	// KubeletDnatControllerClusterRoleName is the name for the KubeletDnatController cluster role.
	KubeletDnatControllerClusterRoleName = "system:kubermatic-kubeletdnat-controller"
	// KubeletDnatControllerClusterRoleBindingName is the name for the KubeletDnatController clusterrolebinding.
	KubeletDnatControllerClusterRoleBindingName = "system:kubermatic-kubeletdnat-controller"

	// ClusterInfoReaderRoleName is the name for the role which allows reading the cluster-info ConfigMap.
	ClusterInfoReaderRoleName = "cluster-info"
	// MachineControllerRoleName is the name for the MachineController roles.
	MachineControllerRoleName = "machine-controller"
	// OperatingSystemManagerRoleName is the name for the OperatingSystemManager roles.
	OperatingSystemManagerRoleName = "operating-system-manager"
	// MachineControllerRoleBindingName is the name for the MachineController rolebinding.
	MachineControllerRoleBindingName = "machine-controller"
	// OperatingSystemManagerRoleBindingName is the name for the OperatingSystemManager rolebinding.
	OperatingSystemManagerRoleBindingName = "operating-system-manager"
	// ClusterInfoAnonymousRoleBindingName is the name for the RoleBinding giving access to the cluster-info ConfigMap to anonymous users.
	ClusterInfoAnonymousRoleBindingName = "cluster-info"
	// MetricsServerAuthReaderRoleName is the name for the metrics server role.
	MetricsServerAuthReaderRoleName = "metrics-server-auth-reader"
	// MachineControllerClusterRoleName is the name for the MachineController cluster role.
	MachineControllerClusterRoleName = "system:kubermatic-machine-controller"
	// OperatingSystemManagerClusterRoleName is the name for the OperatingSystemManager cluster role.
	OperatingSystemManagerClusterRoleName = "system:kubermatic-operating-system-manager"
	// KubeStateMetricsClusterRoleName is the name for the KubeStateMetrics cluster role.
	KubeStateMetricsClusterRoleName = "system:kubermatic-kube-state-metrics"
	// MetricsServerClusterRoleName is the name for the metrics server cluster role.
	MetricsServerClusterRoleName = "system:metrics-server"
	// PrometheusClusterRoleName is the name for the Prometheus cluster role.
	PrometheusClusterRoleName = "external-prometheus"
	// MachineControllerClusterRoleBindingName is the name for the MachineController ClusterRoleBinding.
	MachineControllerClusterRoleBindingName = "system:kubermatic-machine-controller"
	// OperatingSystemManagerClusterRoleBindingName is the name for the OperatingSystemManager ClusterRoleBinding.
	OperatingSystemManagerClusterRoleBindingName = "system:kubermatic-operating-system-manager"
	// KubeStateMetricsClusterRoleBindingName is the name for the KubeStateMetrics ClusterRoleBinding.
	KubeStateMetricsClusterRoleBindingName = "system:kubermatic-kube-state-metrics"
	// PrometheusClusterRoleBindingName is the name for the Prometheus ClusterRoleBinding.
	PrometheusClusterRoleBindingName = "system:external-prometheus"
	// MetricsServerResourceReaderClusterRoleBindingName is the name for the metrics server ClusterRoleBinding.
	MetricsServerResourceReaderClusterRoleBindingName = "system:metrics-server"
	// KubernetesDashboardRoleName is the name of the role for the Kubernetes Dashboard.
	KubernetesDashboardRoleName = "system:kubernetes-dashboard"
	// KubernetesDashboardRoleBindingName is the name of the role binding for the Kubernetes Dashboard.
	KubernetesDashboardRoleBindingName = "system:kubernetes-dashboard"
	// MetricsScraperClusterRoleName is the name of the role for the dashboard-metrics-scraper.
	MetricsScraperClusterRoleName = "system:dashboard-metrics-scraper"
	// MetricsScraperClusterRoleBindingName is the name of the role binding for the dashboard-metrics-scraper.
	MetricsScraperClusterRoleBindingName = "system:dashboard-metrics-scraper"

	// EtcdPodDisruptionBudgetName is the name of the PDB for the etcd StatefulSet.
	EtcdPodDisruptionBudgetName = "etcd"
	// ApiserverPodDisruptionBudgetName is the name of the PDB for the apiserver deployment.
	ApiserverPodDisruptionBudgetName = "apiserver"
	// MetricsServerPodDisruptionBudgetName is the name of the PDB for the metrics-server deployment.
	MetricsServerPodDisruptionBudgetName = "metrics-server"

	// KubermaticNamespace is the main kubermatic namespace.
	KubermaticNamespace = "kubermatic"
	// KubermaticWebhookServiceName is the name of the kuberamtic webhook service in seed cluster.
	KubermaticWebhookServiceName = "kubermatic-webhook"
	// GatekeeperControllerDeploymentName is the name of the gatekeeper controller deployment.
	GatekeeperControllerDeploymentName = "gatekeeper-controller-manager"
	// GatekeeperAuditDeploymentName is the name of the gatekeeper audit deployment.
	GatekeeperAuditDeploymentName = "gatekeeper-audit"
	// GatekeeperWebhookServiceName is the name of the gatekeeper webhook service.
	GatekeeperWebhookServiceName = "gatekeeper-webhook-service"
	// GatekeeperWebhookServerCertSecretName is the name of the gatekeeper webhook cert secret name.
	GatekeeperWebhookServerCertSecretName = "gatekeeper-webhook-server-cert"
	// GatekeeperPodDisruptionBudgetName is the name of the PDB for the gatekeeper controller manager.
	GatekeeperPodDisruptionBudgetName = "gatekeeper-controller-manager"
	// GatekeeperRoleName is the name for the Gatekeeper role.
	GatekeeperRoleName = "gatekeeper-manager-role"
	// GatekeeperRoleBindingName is the name for the Gatekeeper rolebinding.
	GatekeeperRoleBindingName = "gatekeeper-manager-rolebinding"
	// GatekeeperServiceAccountName is the name for the Gatekeeper service account.
	GatekeeperServiceAccountName = "gatekeeper-admin"
	// GatekeeperNamespace is the main gatkeeper namespace where the gatekeeper config is stored.
	GatekeeperNamespace = "gatekeeper-system"
	// ExperimentalEnableMutation enables gatekeeper to validate created kubernetes resources and also modify them based on defined mutation policies.
	ExperimentalEnableMutation = false
	// AuditMatchKindOnly enables gatekeeper to only audit resources in OPA cache.
	AuditMatchKindOnly = false
	// ConstraintViolationsLimit defines the maximum number of audit violations reported on a constraint.
	ConstraintViolationsLimit = 20
	// GatekeeperExemptNamespaceLabel label key for exempting namespaces from Gatekeeper checks.
	GatekeeperExemptNamespaceLabel = "admission.gatekeeper.sh/ignore"
	// ClusterCloudCredentialsSecretName is the name the Secret in the cluster namespace that contains
	// the cloud provider credentials. This Secret is a copy of the credentials secret from the KKP
	// namespace (which has a dynamic name).
	ClusterCloudCredentialsSecretName = "cloud-credentials"

	// CloudInitSettingsNamespace are used in order to reach, authenticate and be authorized by the api server, to fetch
	// the machine  provisioning cloud-init.
	CloudInitSettingsNamespace = "cloud-init-settings"
	// DefaultOwnerReadOnlyMode represents file mode with read permission for owner only.
	DefaultOwnerReadOnlyMode = 0400

	// DefaultAllReadOnlyMode represents file mode with read permissions for all.
	DefaultAllReadOnlyMode = 0444

	// AppLabelKey defines the label key app which should be used within resources.
	AppLabelKey = "app"
	// ClusterLabelKey defines the label key for the cluster name.
	ClusterLabelKey = "cluster"
	// VersionLabel is the label containing the application's version.
	VersionLabel = "app.kubernetes.io/version"

	// EtcdClusterSize defines the size of the etcd to use.
	EtcdClusterSize = 3

	// RegistryK8S defines the (new) official registry hosted by the Kubernetes project.
	RegistryK8S = "registry.k8s.io"
	// RegistryDocker defines the default docker.io registry.
	RegistryDocker = "docker.io"
	// RegistryQuay defines the image registry from coreos/redhat - quay.
	RegistryQuay = "quay.io"

	// TopologyKeyHostname defines the topology key for the node hostname.
	TopologyKeyHostname = "kubernetes.io/hostname"
	// TopologyKeyZone defines the topology key for the node's cloud provider zone.
	TopologyKeyZone = "topology.kubernetes.io/zone"

	// ClusterAutoscalerSafeToEvictVolumesAnnotation is an annotation that contains a comma-separated
	// list of hostPath/emptyDir volumes that should not block the pod from being evicted by the
	// cluster-autoscaler.
	// See https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/FAQ.md#what-types-of-pods-can-prevent-ca-from-removing-a-node
	// for more information.
	ClusterAutoscalerSafeToEvictVolumesAnnotation = "cluster-autoscaler.kubernetes.io/safe-to-evict-local-volumes"

	// MachineCRDName defines the CRD name for machine objects.
	MachineCRDName = "machines.cluster.k8s.io"
	// MachineSetCRDName defines the CRD name for machineset objects.
	MachineSetCRDName = "machinesets.cluster.k8s.io"
	// MachineDeploymentCRDName defines the CRD name for machinedeployment objects.
	MachineDeploymentCRDName = "machinedeployments.cluster.k8s.io"

	// MachineControllerMutatingWebhookConfigurationName is the name of the machine-controllers mutating webhook
	// configuration.
	MachineControllerMutatingWebhookConfigurationName = "machine-controller.kubermatic.io"

	// OperatingSystemManagerMutatingWebhookConfigurationName is the name of OSM's mutating webhook configuration.
	OperatingSystemManagerMutatingWebhookConfigurationName = "operating-system-manager.kubermatic.io"
	// OperatingSystemManagerOperatingSystemProfileCRDName defines the CRD name for OSM operatingSysatemProfile objects.
	OperatingSystemManagerOperatingSystemProfileCRDName = "operatingsystemprofiles.operatingsystemmanager.k8c.io"
	// OperatingSystemManagerOperatingSystemConfigCRDName defines the CRD name for OSM operatingSystemConfig objects.
	OperatingSystemManagerOperatingSystemConfigCRDName = "operatingsystemconfigs.operatingsystemmanager.k8c.io"
	// OperatingSystemManagerValidatingWebhookConfigurationName is the name of OSM's validating webhook configuration.
	OperatingSystemManagerValidatingWebhookConfigurationName = "operating-system-manager.kubermatic.io"

	// GatekeeperValidatingWebhookConfigurationName is the name of the gatekeeper validating webhook
	// configuration.
	GatekeeperValidatingWebhookConfigurationName = "gatekeeper-validating-webhook-configuration"
	GatekeeperMutatingWebhookConfigurationName   = "gatekeeper-mutating-webhook-configuration"
	// InternalUserClusterAdminKubeconfigSecretName is the name of the secret containing an admin kubeconfig that can only be used from
	// within the seed cluster.
	InternalUserClusterAdminKubeconfigSecretName = "internal-admin-kubeconfig"
	// InternalUserClusterAdminKubeconfigCertUsername is the name of the user coming from kubeconfig cert.
	InternalUserClusterAdminKubeconfigCertUsername = "kubermatic-controllers"

	// IPVSProxyMode defines the ipvs kube-proxy mode.
	IPVSProxyMode = "ipvs"
	// IPTablesProxyMode defines the iptables kube-proxy mode.
	IPTablesProxyMode = "iptables"
	// EBPFProxyMode defines the eBPF proxy mode (disables kube-proxy and requires CNI support).
	EBPFProxyMode = "ebpf"

	// PodNodeSelectorAdmissionPlugin defines PodNodeSelector admission plugin.
	PodNodeSelectorAdmissionPlugin = "PodNodeSelector"

	// EventRateLimitAdmissionPlugin defines the EventRateLimit admission plugin.
	EventRateLimitAdmissionPlugin = "EventRateLimit"

	// KubeVirtInfraSecretName is the name for the secret containing the kubeconfig of the kubevirt infra cluster.
	KubeVirtInfraSecretName = "cloud-controller-manager-infra-kubeconfig"
	// KubeVirtInfraSecretKey infra kubeconfig.
	KubeVirtInfraSecretKey = "infra-kubeconfig"
	// KubeVirtCSISecretName is the name for the secret containing the kubeconfig of the kubevirt infra cluster for the CSI controller.
	KubeVirtCSISecretName = "csi-infra-kubeconfig"
	// KubeVirtCSISecretKey is the key in the previous secret.
	KubeVirtCSISecretKey = "kubeconfig"
	// KubeVirtCSINamespaceKey  is the key name of the field containing the infra cluster namespace in the CSI ConfigMap.
	KubeVirtCSINamespaceKey = "infraClusterNamespace"
	// KubeVirtCSIClusterLabelKey  is the key name of the field containing the infra cluster labels in the CSI ConfigMap.
	KubeVirtCSIClusterLabelKey = "infraClusterLabels"
	// KubeVirtCSIConfigMapName is the name of the configmap for the CSI controller.
	KubeVirtCSIConfigMapName = "csi-driver-config"
	// KubeVirtCSIControllerName is the name of the deployment of the CSI controller.
	KubeVirtCSIControllerName = "csi-controller"
	// KubeVirtCSIServiceAccountName is the name of the service account of the CSI controller.
	KubeVirtCSIServiceAccountName = "kubevirt-csi"
	// KubeVirtCSIClusterRoleName is the name of the deployment of the CSI controller.
	KubeVirtCSIClusterRoleName = "kubevirt-csi-controller"
	// KubeVirtCSIRoleBindingName is the name of the deployment of the CSI controller.
	KubeVirtCSIRoleBindingName = "csi-controller"

	// VMwareCloudDirectorCSIControllerName is the name of the deployment of the CSI controller.
	VMwareCloudDirectorCSIControllerName = "csi-controller"
	// VMwareCloudDirectorCSISecretName is the name for the secret containing the credentials for VMware Cloud Director.
	VMwareCloudDirectorCSISecretName = "vcloud-basic-auth"
	// VMwareCloudDirectorCSIConfigmapName is the name for the configmap containing the configmap for VMware Cloud Director CSI driver.
	VMwareCloudDirectorCSIConfigmapName = "vcloud-csi-configmap"
	// VMwareCloudDirectorCSIServiceAccountName is the name of the service account of the CSI controller.
	VMwareCloudDirectorCSIServiceAccountName = "vcloud-csi"
	// VMwareCloudDirectorCSICertUsername is the name of the user coming from kubeconfig cert.
	VMwareCloudDirectorCSICertUsername = "kubermatic:vcloud-csi"
	// VMwareCloudDirectorCSIKubeconfigSecretName is the name for the secret containing the kubeconfig used by the osm.
	VMwareCloudDirectorCSIKubeconfigSecretName = "vcloud-csi-kubeconfig"
	// DefaultNodePortRange is a Kubernetes cluster's default nodeport range.
	DefaultNodePortRange = "30000-32767"

	// ClusterLastRestartAnnotation is an optional annotation on Cluster objects that is meant to contain
	// a UNIX timestamp (or similar) value to trigger cluster control plane restarts. The value of this
	// annotation is copied into control plane components.
	ClusterLastRestartAnnotation = "kubermatic.k8c.io/last-restart"
)

const (
	// CAKeySecretKey ca.key.
	CAKeySecretKey = "ca.key"
	// CACertSecretKey ca.crt.
	CACertSecretKey = "ca.crt"
	// ApiserverTLSKeySecretKey apiserver-tls.key.
	ApiserverTLSKeySecretKey = "apiserver-tls.key"
	// ApiserverTLSCertSecretKey apiserver-tls.crt.
	ApiserverTLSCertSecretKey = "apiserver-tls.crt"
	// KubeletClientKeySecretKey kubelet-client.key.
	KubeletClientKeySecretKey = "kubelet-client.key"
	// KubeletClientCertSecretKey kubelet-client.crt.
	KubeletClientCertSecretKey = "kubelet-client.crt"
	// ServiceAccountKeySecretKey sa.key.
	ServiceAccountKeySecretKey = "sa.key"
	// ServiceAccountKeyPublicKey is the public key for the service account signer key.
	ServiceAccountKeyPublicKey = "sa.pub"
	// KubeconfigSecretKey kubeconfig.
	KubeconfigSecretKey = "kubeconfig"
	// TokensSecretKey tokens.csv.
	TokensSecretKey = "tokens.csv"
	// ViewerTokenSecretKey viewersToken.
	ViewerTokenSecretKey = "viewerToken"
	// OpenVPNCACertKey cert.pem, must match CACertSecretKey, otherwise getClusterCAFromLister doesn't work as it has
	// the key hardcoded.
	OpenVPNCACertKey = CACertSecretKey
	// OpenVPNCAKeyKey key.pem, must match CAKeySecretKey, otherwise getClusterCAFromLister doesn't work as it has
	// the key hardcoded.
	OpenVPNCAKeyKey = CAKeySecretKey
	// OpenVPNServerKeySecretKey server.key.
	OpenVPNServerKeySecretKey = "server.key"
	// OpenVPNServerCertSecretKey server.crt.
	OpenVPNServerCertSecretKey = "server.crt"
	// OpenVPNInternalClientKeySecretKey client.key.
	OpenVPNInternalClientKeySecretKey = "client.key"
	// OpenVPNInternalClientCertSecretKey client.crt.
	OpenVPNInternalClientCertSecretKey = "client.crt"
	// EtcdTLSCertSecretKey etcd-tls.crt.
	EtcdTLSCertSecretKey = "etcd-tls.crt"
	// EtcdTLSKeySecretKey etcd-tls.key.
	EtcdTLSKeySecretKey = "etcd-tls.key"

	EtcdBackupAndRestoreS3AccessKeyIDKey        = "ACCESS_KEY_ID"
	EtcdBackupAndRestoreS3SecretKeyAccessKeyKey = "SECRET_ACCESS_KEY"

	EtcdRestoreS3BucketNameKey    = "BUCKET_NAME"
	EtcdRestoreS3EndpointKey      = "ENDPOINT"
	EtcdRestoreDefaultS3SEndpoint = "s3.amazonaws.com"

	// ApiserverEtcdClientCertificateCertSecretKey apiserver-etcd-client.crt.
	ApiserverEtcdClientCertificateCertSecretKey = "apiserver-etcd-client.crt"
	// ApiserverEtcdClientCertificateKeySecretKey apiserver-etcd-client.key.
	ApiserverEtcdClientCertificateKeySecretKey = "apiserver-etcd-client.key"

	// ApiserverProxyClientCertificateCertSecretKey apiserver-proxy-client.crt.
	ApiserverProxyClientCertificateCertSecretKey = "apiserver-proxy-client.crt"
	// ApiserverProxyClientCertificateKeySecretKey apiserver-proxy-client.key.
	ApiserverProxyClientCertificateKeySecretKey = "apiserver-proxy-client.key"

	// BackupEtcdClientCertificateCertSecretKey backup-etcd-client.crt.
	BackupEtcdClientCertificateCertSecretKey = "backup-etcd-client.crt"
	// BackupEtcdClientCertificateKeySecretKey backup-etcd-client.key.
	BackupEtcdClientCertificateKeySecretKey = "backup-etcd-client.key"

	// PrometheusClientCertificateCertSecretKey prometheus-client.crt.
	PrometheusClientCertificateCertSecretKey = "prometheus-client.crt"
	// PrometheusClientCertificateKeySecretKey prometheus-client.key.
	PrometheusClientCertificateKeySecretKey = "prometheus-client.key"

	// ServingCertSecretKey is the secret key for a generic serving cert.
	ServingCertSecretKey = "serving.crt"
	// ServingCertKeySecretKey is the secret key for the key of a generic serving cert.
	ServingCertKeySecretKey = "serving.key"

	// AuthorizationWebhookVolumeName is the name for the authorization-webhook config mounted volume.
	AuthorizationWebhookVolumeName = "authorization-webhook"
	// AuthorizationConfigurationVolumeName is the name for the authorization-configuration mounted volume.
	AuthorizationConfigurationVolumeName = "authorization-configuration"

	// CloudConfigSecretKey is the secret key for cloud-config.
	CloudConfigSecretKey = "config"
	// NutanixCSIConfigSecretKey is the secret key for nutanix csi secret.
	NutanixCSIConfigSecretKey = "key"
	// NutanixCSIConfigSecretName is the secret key for nutanix csi secret.
	NutanixCSIConfigSecretName = "ntnx-secret"

	// VMwareCloudDirectorCSIConfigConfigMapKey is the key for VMware Cloud Director CSI configmap.
	VMwareCloudDirectorCSIConfigConfigMapKey = "vcloud-csi-config.yaml"
	// VMwareCloudDirectorCSIConfigConfigMapName is the name for VMware Cloud Director CSI configmap.
	VMwareCloudDirectorCSIConfigConfigMapName = "vcloud-csi-configmap"
)

const (
	minimumCertValidity30d = 30 * 24 * time.Hour
)

const (
	ExternalClusterKubeconfigPrefix = "kubeconfig-external-cluster"
	// KubeOneNamespacePrefix is the kubeone namespace prefix.
	KubeOneNamespacePrefix = "kubeone"
	// CredentialPrefix is the prefix used for the secrets containing cloud provider crednentials.
	CredentialPrefix = "credential"
	// KubeOne secret prefixes.
	// don't change this as these prefixes are used for rbac generation.
	KubeOneSSHSecretPrefix      = "ssh-kubeone-external-cluster"
	KubeOneManifestSecretPrefix = "manifest-kubeone-external-cluster"
	// KubOne ConfigMap name.
	KubeOneScriptConfigMapName = "kubeone"
	// KubeOne secret keys.
	KubeOneManifest            = "manifest"
	KubeOneSSHPrivateKey       = "id_rsa"
	KubeOneSSHPassphrase       = "passphrase"
	ContainerRuntimeDocker     = "docker"
	ContainerRuntimeContainerd = "containerd"
	// KubeOne natively-supported providers.
	KubeOneAWS                 = "aws"
	KubeOneGCP                 = "gcp"
	KubeOneAzure               = "azure"
	KubeOneDigitalOcean        = "digitalocean"
	KubeOneHetzner             = "hetzner"
	KubeOneNutanix             = "nutanix"
	KubeOneVMwareCloudDirector = "vmwareCloudDirector"
	KubeOneOpenStack           = "openstack"
	KubeOneVSphere             = "vsphere"
	KubeOneImage               = "quay.io/kubermatic/kubeone"
	KubeOneImageTag            = "v1.7.2"
	KubeOneScript              = `
#!/usr/bin/env bash

eval ` + "`" + "ssh-agent" + "`" + ` > /dev/null
printf "#!/bin/sh\necho $PASSPHRASE" > script_returning_pass
chmod +x script_returning_pass
DISPLAY=1 SSH_ASKPASS="./script_returning_pass" ssh-add ~/.ssh/id_rsa > /dev/null 2> /dev/null
rm ${SSH_ASKPASS} -f
			`
)

const (
	AWSAccessKeyID          = "accessKeyId"
	AWSSecretAccessKey      = "secretAccessKey"
	AWSAssumeRoleARN        = "assumeRoleARN"
	AWSAssumeRoleExternalID = "assumeRoleExternalID"

	AzureTenantID       = "tenantID"
	AzureSubscriptionID = "subscriptionID"
	AzureClientID       = "clientID"
	AzureClientSecret   = "clientSecret"

	DigitaloceanToken = "token"

	GCPServiceAccount = "serviceAccount"

	HetznerToken = "token"

	OpenstackUsername                    = "username"
	OpenstackPassword                    = "password"
	OpenstackTenant                      = "tenant"
	OpenstackTenantID                    = "tenantID"
	OpenstackProject                     = "project"
	OpenstackProjectID                   = "projectID"
	OpenstackDomain                      = "domain"
	OpenstackApplicationCredentialID     = "applicationCredentialID"
	OpenstackApplicationCredentialSecret = "applicationCredentialSecret"
	OpenstackToken                       = "token"
	// Below OpenStack constant is added for KubeOne Clusters.
	OpenstackAuthURL = "authURL"
	OpenstackRegion  = "region"

	KubeVirtKubeconfig = "kubeConfig"

	VsphereUsername                    = "username"
	VspherePassword                    = "password"
	VsphereInfraManagementUserUsername = "infraManagementUserUsername"
	VsphereInfraManagementUserPassword = "infraManagementUserPassword"
	// Below VSphere constant is added for KubeOne Clusters.
	VsphereServer = "server"

	AlibabaAccessKeyID     = "accessKeyId"
	AlibabaAccessKeySecret = "accessKeySecret"

	AnexiaToken = "token"

	NutanixUsername    = "username"
	NutanixPassword    = "password"
	NutanixCSIUsername = "csiUsername"
	NutanixCSIPassword = "csiPassword"
	NutanixProxyURL    = "proxyURL"
	// Below Nutanix constant are added for KubeOne Clusters.
	NutanixCSIEndpoint   = "csiEndpoint"
	NutanixClusterName   = "clusterName"
	NutanixAllowInsecure = "allowInsecure"
	NutanixEndpoint      = "endpoint"
	NutanixPort          = "port"

	// VMware Cloud Director provider constants.
	VMwareCloudDirectorUsername     = "username"
	VMwareCloudDirectorAPIToken     = "apiToken"
	VMwareCloudDirectorPassword     = "password"
	VMwareCloudDirectorOrganization = "organization"
	VMwareCloudDirectorVDC          = "vdc"
	VMwareCloudDirectorURL          = "url"

	ServiceAccountTokenType       = "kubernetes.io/service-account-token"
	ServiceAccountTokenAnnotation = "kubernetes.io/service-account.name"

	UserSSHKeys = "usersshkeys"

	// This Constant is used in GetBaremetalCredentials() to get the Tinkerbell kubeconfig.
	TinkerbellKubeconfig = "kubeConfig"
)

const (
	CoreDNSClusterRoleName         = "system:coredns"
	CoreDNSClusterRoleBindingName  = "system:coredns"
	CoreDNSServiceAccountName      = "coredns"
	CoreDNSServiceName             = "kube-dns"
	CoreDNSConfigMapName           = "coredns"
	CoreDNSDeploymentName          = "coredns"
	CoreDNSPodDisruptionBudgetName = "coredns"
)

const (
	EnvoyAgentConfigMapName                    = "envoy-agent"
	EnvoyAgentConfigFileName                   = "envoy.yaml"
	EnvoyAgentDaemonSetName                    = "envoy-agent"
	EnvoyAgentCreateInterfaceInitContainerName = "create-dummy-interface"
	EnvoyAgentAssignAddressContainerName       = "assign-address"
	EnvoyAgentDeviceSetupImage                 = "kubermatic/network-interface-manager"
	// Default tunneling agent IP address.
	DefaultTunnelingAgentIP = "100.64.30.10"
)

const (
	NodeLocalDNSServiceAccountName  = "node-local-dns"
	NodeLocalDNSConfigMapName       = "node-local-dns"
	NodeLocalDNSDaemonSetName       = "node-local-dns"
	DefaultNodeLocalDNSCacheEnabled = true
)

const (
	TokenBlacklist = "token-blacklist"
)

const (
	ExternalClusterIsImported         = "is-imported"
	ExternalClusterIsImportedTrue     = "true"
	ExternalClusterIsImportedFalse    = "false"
	ExternalClusterKubeconfig         = "kubeconfig"
	ExternalEKSClusterAccessKeyID     = "accessKeyId"
	ExternalEKSClusterSecretAccessKey = "secretAccessKey"
	ExternalGKEClusterSeriveAccount   = "serviceAccount"
	GKEUnspecifiedReleaseChannel      = "UNSPECIFIED"
	GKERapidReleaseChannel            = "RAPID"
	GKERegularReleaseChannel          = "REGULAR"
	GKEStableReleaseChannel           = "STABLE"
	ExternalAKSClusterTenantID        = "tenantID"
	ExternalAKSClusterSubscriptionID  = "subscriptionID"
	ExternalAKSClusterClientID        = "clientID"
	ExternalAKSClusterClientSecret    = "clientSecret"
	AKSNodepoolNameLabel              = "kubernetes.azure.com/agentpool"
	EKSNodeGroupNameLabel             = "eks.amazonaws.com/nodegroup"
	GKENodepoolNameLabel              = "cloud.google.com/gke-nodepool"
)

type AKSState string

const (
	CreatingAKSState  AKSState = "Creating"
	RunningAKSState   AKSState = "Running"
	StartingAKSState  AKSState = "Starting"
	StoppingAKSState  AKSState = "Stopping"
	SucceededAKSState AKSState = "Succeeded"
	StoppedAKSState   AKSState = "Stopped"
	FailedAKSState    AKSState = "Failed"
	DeletingAKSState  AKSState = "Deleting"
	UpgradingAKSState AKSState = "Upgrading"
)

type AKSMDState string

const (
	CreatingAKSMDState  AKSMDState = "Creating"
	SucceededAKSMDState AKSMDState = "Succeeded"
	RunningAKSMDState   AKSMDState = "Running"
	StoppedAKSMDState   AKSMDState = "Stopped"
	FailedAKSMDState    AKSMDState = "Failed"
	DeletingAKSMDState  AKSMDState = "Deleting"
	UpgradingAKSMDState AKSMDState = "Upgrading"
	UpdatingAKSMDState  AKSMDState = "Updating"
	ScalingAKSMDState   AKSMDState = "Scaling"
	StartingAKSMDState  AKSMDState = "Starting"
)

type EKSState string

const (
	CreatingEKSState EKSState = "CREATING"
	PendingEKSState  EKSState = "PENDING"
	ActiveEKSState   EKSState = "ACTIVE"
	UpdatingEKSState EKSState = "UPDATING"
	DeletingEKSState EKSState = "DELETING"
	FailedEKSState   EKSState = "FAILED"
)

type EKSMDState string

const (
	CreatingEKSMDState     EKSMDState = "CREATING"
	ActiveEKSMDState       EKSMDState = "ACTIVE"
	UpdatingEKSMDState     EKSMDState = "UPDATING"
	DeletingEKSMDState     EKSMDState = "DELETING"
	CreateFailedEKSMDState EKSMDState = "CREATE_FAILED"
	DeleteFailedEKSMDState EKSMDState = "DELETE_FAILED"
	DegradedEKSMDState     EKSMDState = "DEGRADED"
)

type GKEState string

const (
	ProvisioningGKEState GKEState = "PROVISIONING"
	RunningGKEState      GKEState = "RUNNING"
	ReconcilingGKEState  GKEState = "RECONCILING"
	StoppingGKEState     GKEState = "STOPPING"
	ErrorGKEState        GKEState = "ERROR"
	DegradedGKEState     GKEState = "DEGRADED"
	UnspecifiedGKEState  GKEState = "STATUS_UNSPECIFIED"
)

type GKEMDState string

const (
	ProvisioningGKEMDState     GKEMDState = "PROVISIONING"
	RunningGKEMDState          GKEMDState = "RUNNING"
	ReconcilingGKEMDState      GKEMDState = "RECONCILING"
	StoppingGKEMDState         GKEMDState = "STOPPING"
	ErrorGKEMDState            GKEMDState = "ERROR"
	RunningWithErrorGKEMDState GKEMDState = "RUNNING_WITH_ERROR"
	UnspecifiedGKEMDState      GKEMDState = "STATUS_UNSPECIFIED"
)

const (
	EtcdTrustedCAFile = "/etc/etcd/pki/ca/ca.crt"
	EtcdCertFile      = "/etc/etcd/pki/tls/etcd-tls.crt"
	EtcdKeyFile       = "/etc/etcd/pki/tls/etcd-tls.key"

	EtcdPeerCertFile = "/etc/etcd/pki/tls/etcd-tls.crt"
	EtcdPeerKeyFile  = "/etc/etcd/pki/tls/etcd-tls.key"

	EtcdClientCertFile = "/etc/etcd/pki/client/apiserver-etcd-client.crt"
	EtcdClientKeyFile  = "/etc/etcd/pki/client/apiserver-etcd-client.key"
)

const (
	// CSIMigrationWebhookName is the name of the csi-migration webhook service.
	CSIMigrationWebhookName = "csi-migration-webhook"
	// CSIMigrationWebhookSecretName defines the name of the secret containing the certificates for the csi-migration admission webhook.
	CSIMigrationWebhookSecretName = "csi-migration-webhook-certs"

	// CSIMigrationWebhookConfig is the name for the key that contains the webhook config.
	CSIMigrationWebhookConfig = "webhook.config"
	// CSIMigrationWebhookPort is the port used by the CSI-migration webhook.
	CSIMigrationWebhookPort = 8443
	// VsphereCSIMigrationWebhookConfigurationWebhookName is the webhook's name in the vSphere CSI_migration WebhookConfiguration.
	VsphereCSIMigrationWebhookConfigurationWebhookName = "validation.csi.vsphere.vmware.com"

	// CSISnapshotValidationWebhookConfigurationName part of kubernetes-csi external-snapshotter validation webhook.
	CSISnapshotValidationWebhookConfigurationName = "validation-webhook.snapshot.storage.k8s.io"
	// CSISnapshotValidationWebhookName part of kubernetes-csi external-snapshotter validation webhook.
	CSISnapshotValidationWebhookName = "snapshot-validation-service"

	CSISnapshotWebhookSecretName = "csi-snapshot-webhook-certs"
	// CSIWebhookServingCertCertKeyName is the name for the key that contains the cert.
	CSIWebhookServingCertCertKeyName = "cert.pem"
	// CSIWebhookServingCertKeyKeyName is the name for the key that contains the key.
	CSIWebhookServingCertKeyKeyName = "key.pem"
)

const (
	// KubeLBCCMKubeconfigSecretName is the name for the secret containing the kubeconfig used by the kubelb CCM.
	KubeLBCCMKubeconfigSecretName = "kubelb-ccm-kubeconfig"
	// KubeLBManagerKubeconfigSecretName is the name for the secret containing the kubeconfig for the kubelb management cluster used by the kubelb CCM.
	KubeLBManagerKubeconfigSecretName = "kubelb-manager-kubeconfig"
	// KubeLBAppName is the name of the kubelb app.
	KubeLBAppName = "kubelb-ccm"
)

const (
	UserClusterMLANamespace = "mla-system"
	MLAComponentName        = "mla"

	MLALoggingAgentServiceAccountName     = "mla-logging-agent"
	MLALoggingAgentClusterRoleName        = "system:mla:mla-logging-agent"
	MLALoggingAgentClusterRoleBindingName = "system:mla:mla-logging-agent"
	MLALoggingAgentSecretName             = "mla-logging-agent"
	MLALoggingAgentDaemonSetName          = "mla-logging-agent"

	MLAMonitoringAgentConfigMapName          = "mla-monitoring-agent"
	MLAMonitoringAgentServiceAccountName     = "mla-monitoring-agent"
	MLAMonitoringAgentClusterRoleName        = "system:mla:mla-monitoring-agent"
	MLAMonitoringAgentClusterRoleBindingName = "system:mla:mla-monitoring-agent"
	MLAMonitoringAgentDeploymentName         = "mla-monitoring-agent"

	// MLAGatewayExternalServiceName is the name for the MLA Gateway external service.
	MLAGatewayExternalServiceName = "mla-gateway-ext"
	// MLAGatewaySNIPrefix is the URL prefix which identifies the MLA Gateway endpoint in the external URL if SNI expose strategy is used.
	MLAGatewaySNIPrefix = "mla-gateway."

	// MLAGatewayCASecretName is the name for the secret containing the MLA Gateway CA certificates.
	MLAGatewayCASecretName = "mla-gateway-ca"
	MLAGatewayCACertKey    = CACertSecretKey
	MLAGatewayCAKeyKey     = CAKeySecretKey

	// MLAGatewayCertificatesSecretName is the name for the secret containing the MLA Gateway certificates.
	MLAGatewayCertificatesSecretName = "mla-gateway-certificates"
	MLAGatewayKeySecretKey           = "gateway.key"
	MLAGatewayCertSecretKey          = "gateway.crt"

	// MLAMonitoringAgentCertificatesSecretName is the name for the secret containing the Monitoring Agent (grafana-agent) client certificates.
	MLAMonitoringAgentCertificatesSecretName = "monitoring-agent-certificates"
	MLAMonitoringAgentCertificateCommonName  = "grafana-agent"
	MLAMonitoringAgentClientKeySecretKey     = "client.key"
	MLAMonitoringAgentClientCertSecretKey    = "client.crt"
	MLAMonitoringAgentClientCertMountPath    = "/etc/ssl/mla"

	// MLALoggingAgentCertificatesSecretName is the name for the secret containing the Logging Agent client certificates.
	MLALoggingAgentCertificatesSecretName = "logging-agent-certificates"
	MLALoggingAgentCertificateCommonName  = "logging-agent"
	MLALoggingAgentClientKeySecretKey     = "client.key"
	MLALoggingAgentClientCertSecretKey    = "client.crt"
	MLALoggingAgentClientCertMountPath    = "/etc/ssl/mla"

	AlertmanagerName                    = "alertmanager"
	DefaultAlertmanagerConfigSecretName = "alertmanager"
	AlertmanagerConfigSecretKey         = "alertmanager.yaml"
	DefaultAlertmanagerConfig           = `
template_files: {}
alertmanager_config: |
  route:
    receiver: 'null'
  receivers:
    - name: 'null'
`

	// MLAAdminSettingsName specifies a fixed name of the MLA admin settings custom resource in the cluster namespace.
	MLAAdminSettingsName = "mla-admin-settings"

	// Konnectivity.
	KonnectivityDeploymentName             = "konnectivity-agent"
	KonnectivityClusterRoleBindingName     = "system:konnectivity-server"
	KonnectivityClusterRoleBindingUsername = "system:konnectivity-server"
	KonnectivityServiceAccountName         = "system-konnectivity-agent"
	KonnectivityAgentContainer             = "konnectivity-agent"
	KonnectivityServerContainer            = "konnectivity-server"
	KonnectivityAgentToken                 = "system-konnectivity-agent-token"
	KonnectivityProxyServiceName           = "konnectivity-server"
	KonnectivityProxyTLSSecretName         = "konnectivityproxy-tls"
	KonnectivityKubeconfigSecretName       = "konnectivity-kubeconfig"
	KonnectivityKubeconfigUsername         = "system:konnectivity-server"
	KonnectivityServerConf                 = "kubeconfig"
	KonnectivityKubeApiserverEgress        = "kube-apiserver-egress"
	KonnectivityUDS                        = "konnectivity-uds"
	KonnectivityPodDisruptionBudgetName    = "konnectivity-agent"
)

const (
	// Legacy Prometheus resource names, used only for cleanup/migration purposes.
	UserClusterLegacyPrometheusConfigMapName          = "prometheus"
	UserClusterLegacyPrometheusServiceAccountName     = "prometheus"
	UserClusterLegacyPrometheusClusterRoleName        = "system:mla:prometheus"
	UserClusterLegacyPrometheusClusterRoleBindingName = "system:mla:prometheus"
	UserClusterLegacyPrometheusDeploymentName         = "prometheus"
	UserClusterLegacyPrometheusCertificatesSecretName = "prometheus-certificates"

	// Legacy Promtail resource names, used only for cleanup/migration purposes.
	UserClusterLegacyPromtailServiceAccountName     = "promtail"
	UserClusterLegacyPromtailClusterRoleName        = "system:mla:promtail"
	UserClusterLegacyPromtailClusterRoleBindingName = "system:mla:promtail"
	UserClusterLegacyPromtailSecretName             = "promtail"
	UserClusterLegacyPromtailDaemonSetName          = "promtail"
	UserClusterLegacyPromtailCertificatesSecretName = "promtail-certificates"
)

const (
	NetworkPolicyDefaultDenyAllEgress               = "default-deny-all-egress"
	NetworkPolicyEtcdAllow                          = "etcd-allow"
	NetworkPolicyDNSAllow                           = "dns-allow"
	NetworkPolicyOpenVPNServerAllow                 = "openvpn-server-allow"
	NetworkPolicyMachineControllerWebhookAllow      = "machine-controller-webhook-allow"
	NetworkPolicyUserClusterWebhookAllow            = "usercluster-webhook-allow"
	NetworkPolicyOperatingSystemManagerWebhookAllow = "operating-system-manager-webhook-allow"
	NetworkPolicyMetricsServerAllow                 = "metrics-server-allow"
	NetworkPolicyClusterExternalAddrAllow           = "cluster-external-addr-allow"
	NetworkPolicyOIDCIssuerAllow                    = "oidc-issuer-allow"
	NetworkPolicySeedApiserverAllow                 = "seed-apiserver-allow"
	NetworkPolicyApiserverInternalAllow             = "apiserver-internal-allow"
	NetworkPolicyKyvernoWebhookAllow                = "kyverno-webhook-allow"
)

const (
	UserClusterWebhookDeploymentName        = "usercluster-webhook"
	UserClusterWebhookServiceName           = "usercluster-webhook"
	UserClusterWebhookServingCertSecretName = "usercluster-webhook-serving-cert"
	UserClusterWebhookSeedListenPort        = 443
	UserClusterWebhookUserListenPort        = 6443
)

const (
	// DefaultClusterPodsCIDRIPv4 is the default network range from which IPv4 POD networks are allocated.
	DefaultClusterPodsCIDRIPv4 = "172.25.0.0/16"
	// DefaultClusterPodsCIDRIPv4KubeVirt is the default network range from which IPv4 POD networks are allocated for KubeVirt clusters.
	DefaultClusterPodsCIDRIPv4KubeVirt = "172.26.0.0/16"
	// DefaultClusterPodsCIDRIPv6 is the default network range from which IPv6 POD networks are allocated.
	DefaultClusterPodsCIDRIPv6 = "fd01::/48"

	// DefaultClusterServicesCIDRIPv4 is the default network range from which IPv4 service VIPs are allocated.
	DefaultClusterServicesCIDRIPv4 = "10.240.16.0/20"
	// DefaultClusterServicesCIDRIPv4KubeVirt is the default network range from which IPv4 service VIPs are allocated for KubeVirt clusters.
	DefaultClusterServicesCIDRIPv4KubeVirt = "10.241.0.0/20"
	// DefaultClusterServicesCIDRIPv6 is the default network range from which IPv6 service VIPs are allocated.
	DefaultClusterServicesCIDRIPv6 = "fd02::/108"

	// DefaultNodeCIDRMaskSizeIPv4 is the default mask size used to address the nodes within provided IPv4 Pods CIDR.
	DefaultNodeCIDRMaskSizeIPv4 = 24
	// DefaultNodeCIDRMaskSizeIPv6 is the default mask size used to address the nodes within provided IPv6 Pods CIDR.
	DefaultNodeCIDRMaskSizeIPv6 = 64
)

const (
	// IPv4MatchAnyCIDR is the CIDR used for matching with any IPv4 address.
	IPv4MatchAnyCIDR = "0.0.0.0/0"
	// IPv6MatchAnyCIDR is the CIDR used for matching with any IPv6 address.
	IPv6MatchAnyCIDR = "::/0"
)

const (
	ApplicationCacheVolumeName = "applications-cache"
	ApplicationCacheMountPath  = "/applications-cache"
)

const (
	ClusterBackupUsername           = "velero"
	ClusterBackupServiceAccountName = "velero"
	ClusterBackupNamespaceName      = "velero"
)

var DefaultApplicationCacheSize = resource.MustParse("300Mi")

// GetApplicationCacheSize return the application cache size if defined, otherwise fallback to the default size.
func GetApplicationCacheSize(appSettings *kubermaticv1.ApplicationSettings) *resource.Quantity {
	if appSettings == nil || appSettings.CacheSize == nil {
		return &DefaultApplicationCacheSize
	}
	return appSettings.CacheSize
}

// List of allowed TLS cipher suites.
var allowedTLSCipherSuites = []string{
	// TLS 1.3 cipher suites
	"TLS_AES_128_GCM_SHA256",
	"TLS_AES_256_GCM_SHA384",
	"TLS_CHACHA20_POLY1305_SHA256",
	// TLS 1.0 - 1.2 cipher suites
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
}

// ECDSAKeyPair is a ECDSA x509 certificate and private key.
type ECDSAKeyPair struct {
	Key  *ecdsa.PrivateKey
	Cert *x509.Certificate
}

// Requirements are how much resources are needed by containers in the pod.
type Requirements struct {
	Name     string                       `json:"name,omitempty"`
	Requires *corev1.ResourceRequirements `json:"requires,omitempty"`
}

// GetAllowedTLSCipherSuites returns a list of allowed TLS cipher suites.
func GetAllowedTLSCipherSuites() []string {
	return allowedTLSCipherSuites
}

// GetClusterExternalIP returns a net.IP for the given Cluster.
func GetClusterExternalIP(cluster *kubermaticv1.Cluster) (*net.IP, error) {
	address := cluster.Status.Address

	ip := net.ParseIP(address.IP)
	if ip == nil {
		return nil, fmt.Errorf("failed to create a net.IP object from the external cluster IP '%s'", address.IP)
	}
	return &ip, nil
}

// GetClusterRef returns a metav1.OwnerReference for the given Cluster.
func GetClusterRef(cluster *kubermaticv1.Cluster) metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))
}

// GetProjectRef returns a metav1.OwnerReference for the given Project.
func GetProjectRef(project *kubermaticv1.Project) metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(project, gv.WithKind(kubermaticv1.ProjectKindName))
}

// GetEtcdRestoreRef returns a metav1.OwnerReference for the given EtcdRestore.
func GetEtcdRestoreRef(restore *kubermaticv1.EtcdRestore) metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(restore, gv.WithKind(kubermaticv1.EtcdRestoreKindName))
}

// Int32 returns a pointer to the int32 value passed in.
func Int32(v int32) *int32 {
	return &v
}

// Int64 returns a pointer to the int64 value passed in.
func Int64(v int64) *int64 {
	return &v
}

// Bool returns a pointer to the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

// UserClusterDNSResolverIP returns the 9th usable IP address
// from the first Service CIDR block from ClusterNetwork spec.
// This is by convention the IP address of the DNS resolver.
// Returns "" on error.
func UserClusterDNSResolverIP(cluster *kubermaticv1.Cluster) (string, error) {
	if len(cluster.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		return "", fmt.Errorf("failed to get cluster dns ip for cluster `%s`: empty CIDRBlocks", cluster.Name)
	}
	block := cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0]
	_, ipnet, err := net.ParseCIDR(block)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster dns ip for cluster `%s`: %w", block, err)
	}
	ip := ipnet.IP
	ip[len(ip)-1] = ip[len(ip)-1] + 10
	return ip.String(), nil
}

// InClusterApiserverIP returns the first usable IP of the service cidr.
// Its the in cluster IP for the apiserver.
func InClusterApiserverIP(cluster *kubermaticv1.Cluster) (*net.IP, error) {
	if len(cluster.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		return nil, errors.New("no service cidr defined")
	}

	block := cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0]
	_, ipnet, err := net.ParseCIDR(block)
	if err != nil {
		return nil, fmt.Errorf("invalid service cidr %s", block)
	}
	ip := ipnet.IP
	ip[len(ip)-1] = ip[len(ip)-1] + 1
	return &ip, nil
}

type userClusterDNSPolicyAndConfigData interface {
	Cluster() *kubermaticv1.Cluster
	ClusterIPByServiceName(name string) (string, error)
	IsKonnectivityEnabled() bool
}

// UserClusterDNSPolicyAndConfig returns a DNSPolicy and DNSConfig to configure Pods to use user cluster DNS.
func UserClusterDNSPolicyAndConfig(d userClusterDNSPolicyAndConfigData) (corev1.DNSPolicy, *corev1.PodDNSConfig, error) {
	if d.IsKonnectivityEnabled() {
		// custom DNS resolver in not needed in Konnectivity setup
		return corev1.DNSClusterFirst, nil, nil
	}
	// If Konnectivity is NOT enabled, we deploy a custom DNS resolver
	// for the user cluster in Seed. To use it, we set the DNS policy to DNSNone
	// and set the custom DNS resolver's CLusterIP in the DNSConfig.
	dnsConfigOptionNdots := "5"
	dnsConfigResolverIP, err := d.ClusterIPByServiceName(DNSResolverServiceName)
	if err != nil {
		return corev1.DNSNone, nil, err
	}
	if len(d.Cluster().Spec.ClusterNetwork.DNSDomain) == 0 {
		return corev1.DNSNone, nil, fmt.Errorf("invalid (empty) DNSDomain in ClusterNetwork spec for cluster %s", d.Cluster().Name)
	}
	return corev1.DNSNone, &corev1.PodDNSConfig{
		Nameservers: []string{dnsConfigResolverIP},
		Searches: []string{
			fmt.Sprintf("kube-system.svc.%s", d.Cluster().Spec.ClusterNetwork.DNSDomain),
			fmt.Sprintf("svc.%s", d.Cluster().Spec.ClusterNetwork.DNSDomain),
			d.Cluster().Spec.ClusterNetwork.DNSDomain,
		},
		Options: []corev1.PodDNSConfigOption{
			{
				Name:  "ndots",
				Value: &dnsConfigOptionNdots,
			},
		},
	}, nil
}

// BaseAppLabels returns the minimum required labels.
func BaseAppLabels(appName string, additionalLabels map[string]string) map[string]string {
	labels := map[string]string{
		AppLabelKey: appName,
	}
	for k, v := range additionalLabels {
		labels[k] = v
	}
	return labels
}

// CertWillExpireSoon returns if the certificate will expire in the next 30 days.
func CertWillExpireSoon(cert *x509.Certificate) bool {
	return time.Until(cert.NotAfter) < minimumCertValidity30d
}

// IsServerCertificateValidForAllOf validates if the given data is present in the given server certificate.
func IsServerCertificateValidForAllOf(cert *x509.Certificate, commonName string, altNames certutil.AltNames, ca *x509.Certificate) bool {
	if CertWillExpireSoon(cert) {
		return false
	}

	getIPStrings := func(inIps []net.IP) []string {
		s := make([]string, len(inIps))
		for i, ip := range inIps {
			s[i] = ip.String()
		}
		return s
	}

	if cert.Subject.CommonName != commonName {
		return false
	}

	certIPs := sets.New(getIPStrings(cert.IPAddresses)...)
	wantIPs := sets.New(getIPStrings(altNames.IPs)...)

	if !wantIPs.Equal(certIPs) {
		return false
	}

	wantDNSNames := sets.New(altNames.DNSNames...)
	certDNSNames := sets.New(cert.DNSNames...)

	if !wantDNSNames.Equal(certDNSNames) {
		return false
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(ca)
	verifyOptions := x509.VerifyOptions{
		Roots:     certPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if _, err := cert.Verify(verifyOptions); err != nil {
		kubermaticlog.Logger.Errorw("certificate verification failed", "cn", commonName, zap.Error(err))
		return false
	}

	return true
}

// IsClientCertificateValidForAllOf validates if the given data matches exactly the given client certificate
// (It also returns true if all given data is in the cert, but the cert has more organizations).
func IsClientCertificateValidForAllOf(cert *x509.Certificate, commonName string, organizations []string, ca *x509.Certificate) bool {
	if CertWillExpireSoon(cert) {
		return false
	}

	if cert.Subject.CommonName != commonName {
		return false
	}

	wantOrganizations := sets.New(organizations...)
	certOrganizations := sets.New(cert.Subject.Organization...)

	if !wantOrganizations.Equal(certOrganizations) {
		return false
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(ca)
	verifyOptions := x509.VerifyOptions{
		Roots:     certPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if _, err := cert.Verify(verifyOptions); err != nil {
		kubermaticlog.Logger.Errorw("certificate verification failed", "cn", commonName, zap.Error(err))
		return false
	}

	return true
}

func getECDSAClusterCAFromLister(ctx context.Context, namespace, name string, client ctrlruntimeclient.Client) (*ECDSAKeyPair, error) {
	cert, key, err := getClusterCAFromLister(ctx, namespace, name, client)
	if err != nil {
		return nil, err
	}
	ecdsaKey, isECDSAKey := key.(*ecdsa.PrivateKey)
	if !isECDSAKey {
		return nil, errors.New("key is not a ECDSA key")
	}
	return &ECDSAKeyPair{Cert: cert, Key: ecdsaKey}, nil
}

func getRSAClusterCAFromLister(ctx context.Context, namespace, name string, client ctrlruntimeclient.Client) (*triple.KeyPair, error) {
	cert, key, err := getClusterCAFromLister(ctx, namespace, name, client)
	if err != nil {
		return nil, err
	}
	rsaKey, isRSAKey := key.(*rsa.PrivateKey)
	if !isRSAKey {
		return nil, errors.New("key is not a RSA key")
	}
	return &triple.KeyPair{Cert: cert, Key: rsaKey}, nil
}

// getClusterCAFromLister returns the CA of the cluster from the lister.
func getClusterCAFromLister(ctx context.Context, namespace, name string, client ctrlruntimeclient.Client) (*x509.Certificate, interface{}, error) {
	caSecret := &corev1.Secret{}
	caSecretKey := types.NamespacedName{Namespace: namespace, Name: name}
	if err := client.Get(ctx, caSecretKey, caSecret); err != nil {
		return nil, nil, fmt.Errorf("unable to check if a CA cert already exists: %w", err)
	}

	certs, err := certutil.ParseCertsPEM(caSecret.Data[CACertSecretKey])
	if err != nil {
		return nil, nil, fmt.Errorf("got an invalid cert from the CA secret %s: %w", caSecretKey, err)
	}

	if len(certs) != 1 {
		return nil, nil, fmt.Errorf("did not find exactly one but %v certificates in the CA secret", len(certs))
	}

	key, err := triple.ParsePrivateKeyPEM(caSecret.Data[CAKeySecretKey])
	if err != nil {
		return nil, nil, fmt.Errorf("got an invalid private key from the CA secret %s: %w", caSecretKey, err)
	}

	return certs[0], key, nil
}

// GetCABundleFromFile returns the CA bundle from a file.
func GetCABundleFromFile(file string) ([]*x509.Certificate, error) {
	rawData, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", file, err)
	}

	caCerts, err := certutil.ParseCertsPEM(rawData)
	if err != nil {
		return nil, fmt.Errorf("got an invalid cert: %w", err)
	}

	return caCerts, nil
}

// GetClusterRootCA returns the root CA of the cluster from the lister.
func GetClusterRootCA(ctx context.Context, namespace string, client ctrlruntimeclient.Client) (*triple.KeyPair, error) {
	return getRSAClusterCAFromLister(ctx, namespace, CASecretName, client)
}

// GetClusterFrontProxyCA returns the frontproxy CA of the cluster from the lister.
func GetClusterFrontProxyCA(ctx context.Context, namespace string, client ctrlruntimeclient.Client) (*triple.KeyPair, error) {
	return getRSAClusterCAFromLister(ctx, namespace, FrontProxyCASecretName, client)
}

// GetOpenVPNCA returns the OpenVPN CA of the cluster from the lister.
func GetOpenVPNCA(ctx context.Context, namespace string, client ctrlruntimeclient.Client) (*ECDSAKeyPair, error) {
	return getECDSAClusterCAFromLister(ctx, namespace, OpenVPNCASecretName, client)
}

// GetMLAGatewayCA returns the MLA Gateway CA of the cluster from the lister.
func GetMLAGatewayCA(ctx context.Context, namespace string, client ctrlruntimeclient.Client) (*ECDSAKeyPair, error) {
	return getECDSAClusterCAFromLister(ctx, namespace, MLAGatewayCASecretName, client)
}

// ClusterIPForService returns the cluster ip for the given service.
func ClusterIPForService(name, namespace string, serviceLister corev1lister.ServiceLister) (*net.IP, error) {
	service, err := serviceLister.Services(namespace).Get(name)
	if err != nil {
		return nil, fmt.Errorf("could not get service %s/%s from lister: %w", namespace, name, err)
	}

	if service.Spec.ClusterIP == "" {
		return nil, errors.New("service has no ClusterIP")
	}

	ip := net.ParseIP(service.Spec.ClusterIP)
	if ip == nil {
		return nil, fmt.Errorf("service %s/%s has no valid cluster ip (\"%s\"): %w", namespace, name, service.Spec.ClusterIP, err)
	}

	return &ip, nil
}

// GetAbsoluteServiceDNSName returns the absolute DNS name for the given service and the given cluster. Absolute means a trailing dot will be appended to the DNS name.
func GetAbsoluteServiceDNSName(service, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local.", service, namespace)
}

func GetHTTPProxyEnvVarsFromSeed(seed *kubermaticv1.Seed, inClusterAPIServerURL string) []corev1.EnvVar {
	if seed.Spec.ProxySettings.Empty() {
		return nil
	}
	var envVars []corev1.EnvVar

	if !seed.Spec.ProxySettings.HTTPProxy.Empty() {
		value := seed.Spec.ProxySettings.HTTPProxy.String()
		envVars = []corev1.EnvVar{
			{
				Name:  "HTTP_PROXY",
				Value: value,
			},
			{
				Name:  "HTTPS_PROXY",
				Value: value,
			},
			{
				Name:  "http_proxy",
				Value: value,
			},
			{
				Name:  "https_proxy",
				Value: value,
			},
		}
	}

	noProxyValue := inClusterAPIServerURL
	if !seed.Spec.ProxySettings.NoProxy.Empty() {
		noProxyValue += "," + seed.Spec.ProxySettings.NoProxy.String()
	}
	envVars = append(envVars,
		corev1.EnvVar{Name: "NO_PROXY", Value: noProxyValue},
		corev1.EnvVar{Name: "no_proxy", Value: noProxyValue},
	)

	return envVars
}

// SetResourceRequirements sets resource requirements on provided slice of containers.
// The highest priority has requirements provided using overrides, then requirements provided by the vpa-updater
// (if VPA is enabled), and at the end provided default requirements for a given resource.
func SetResourceRequirements(containers []corev1.Container, defaultRequirements, overrides map[string]*corev1.ResourceRequirements, annotations map[string]string) error {
	// do not accidentally modify the map of default requirements
	requirements := map[string]*corev1.ResourceRequirements{}
	for k, v := range defaultRequirements {
		requirements[k] = v.DeepCopy()
	}

	val, ok := annotations[kubermaticv1.UpdatedByVPALabelKey]
	if ok && val != "" {
		var req []Requirements
		err := json.Unmarshal([]byte(val), &req)
		if err != nil {
			return fmt.Errorf("failed to unmarshal resource requirements provided by vpa from annotation %s: %w", kubermaticv1.UpdatedByVPALabelKey, err)
		}
		for _, r := range req {
			requirements[r.Name] = r.Requires
		}
	}
	for k, v := range overrides {
		defaultRequirement := defaultRequirements[k]
		if v.Requests == nil && defaultRequirement != nil {
			v.Requests = defaultRequirement.Requests
		}
		if v.Limits == nil && defaultRequirement != nil {
			v.Limits = defaultRequirement.Limits
		}

		requirements[k] = v.DeepCopy()
	}

	for i := range containers {
		if requirements[containers[i].Name] != nil {
			containers[i].Resources = *requirements[containers[i].Name]
		}
	}

	return nil
}

func GetOverrides(componentSettings kubermaticv1.ComponentSettings) map[string]*corev1.ResourceRequirements {
	r := map[string]*corev1.ResourceRequirements{}
	if componentSettings.Apiserver.Resources != nil {
		r[ApiserverDeploymentName] = componentSettings.Apiserver.Resources.DeepCopy()
	}
	if componentSettings.KonnectivityProxy.Resources != nil {
		r[KonnectivityServerContainer] = componentSettings.KonnectivityProxy.Resources.DeepCopy()
		r[KonnectivityAgentContainer] = componentSettings.KonnectivityProxy.Resources.DeepCopy()
	}
	if componentSettings.ControllerManager.Resources != nil {
		r[ControllerManagerDeploymentName] = componentSettings.ControllerManager.Resources.DeepCopy()
	}
	if componentSettings.Scheduler.Resources != nil {
		r[SchedulerDeploymentName] = componentSettings.Scheduler.Resources.DeepCopy()
	}
	if componentSettings.Etcd.Resources != nil {
		r[EtcdStatefulSetName] = componentSettings.Etcd.Resources.DeepCopy()
	}
	if componentSettings.Prometheus.Resources != nil {
		r[PrometheusStatefulSetName] = componentSettings.Prometheus.Resources.DeepCopy()
	}
	if componentSettings.NodePortProxyEnvoy.Resources.Requests != nil ||
		componentSettings.NodePortProxyEnvoy.Resources.Limits != nil {
		r[NodePortProxyEnvoyContainerName] = componentSettings.NodePortProxyEnvoy.Resources.DeepCopy()
	}
	if componentSettings.UserClusterController != nil && componentSettings.UserClusterController.Resources != nil {
		r[UserClusterControllerContainerName] = componentSettings.UserClusterController.Resources.DeepCopy()
	}
	if componentSettings.OperatingSystemManager != nil && componentSettings.OperatingSystemManager.Resources != nil {
		r[OperatingSystemManagerContainerName] = componentSettings.OperatingSystemManager.Resources.DeepCopy()
	}
	if componentSettings.CoreDNS != nil && componentSettings.CoreDNS.Resources != nil {
		r[CoreDNSDeploymentName] = componentSettings.CoreDNS.Resources.DeepCopy()
	}
	if componentSettings.KubeStateMetrics != nil && componentSettings.KubeStateMetrics.Resources != nil {
		r[KubeStateMetricsDeploymentName] = componentSettings.KubeStateMetrics.Resources.DeepCopy()
	}

	return r
}

// SupportsFailureDomainZoneAntiAffinity checks if there are any nodes with the
// TopologyKeyZone label.
func SupportsFailureDomainZoneAntiAffinity(ctx context.Context, client ctrlruntimeclient.Client) (bool, error) {
	selector, err := labels.Parse(TopologyKeyZone)
	if err != nil {
		return false, fmt.Errorf("failed to parse selector: %w", err)
	}
	opts := &ctrlruntimeclient.ListOptions{
		LabelSelector: selector,
		Raw: &metav1.ListOptions{
			Limit: 1,
		},
	}

	nodeList := &corev1.NodeList{}
	if err := client.List(ctx, nodeList, opts); err != nil {
		return false, fmt.Errorf("failed to list nodes having the %s label: %w", TopologyKeyZone, err)
	}

	return len(nodeList.Items) != 0, nil
}

// BackupCABundleConfigMapName returns the name of the ConfigMap in the kube-system namespace
// that holds the CA bundle for a given cluster. As the CA bundle technically can be different
// per usercluster, this is not a constant.
func BackupCABundleConfigMapName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s-ca-bundle", cluster.Name)
}

// GetEtcdRestoreS3Client returns an S3 client for downloading the backup for a given EtcdRestore.
// If the EtcdRestore doesn't reference a secret containing the credentials and endpoint and bucket name data,
// one can optionally be created from a well-known secret and configmap in kube-system, or from a specified backup destination.
func GetEtcdRestoreS3Client(ctx context.Context, restore *kubermaticv1.EtcdRestore, createSecretIfMissing bool, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster,
	destination *kubermaticv1.BackupDestination) (*minio.Client, string, error) {
	secretData := make(map[string]string)

	if restore.Spec.BackupDownloadCredentialsSecret != "" {
		secret := &corev1.Secret{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: restore.Spec.BackupDownloadCredentialsSecret}, secret); err != nil {
			return nil, "", fmt.Errorf("failed to get BackupDownloadCredentialsSecret credentials secret %v: %w", restore.Spec.BackupDownloadCredentialsSecret, err)
		}

		for k, v := range secret.Data {
			secretData[k] = string(v)
		}
	} else {
		if !createSecretIfMissing {
			return nil, "", fmt.Errorf("BackupDownloadCredentialsSecret not set")
		}

		credsSecret := &corev1.Secret{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: destination.Credentials.Namespace, Name: destination.Credentials.Name}, credsSecret); err != nil {
			return nil, "", fmt.Errorf("failed to get s3 credentials secret %v/%v: %w", destination.Credentials.Namespace, destination.Credentials.Name, err)
		}
		for k, v := range credsSecret.Data {
			secretData[k] = string(v)
		}
		secretData[EtcdRestoreS3BucketNameKey] = destination.BucketName
		secretData[EtcdRestoreS3EndpointKey] = destination.Endpoint

		creator := func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}
			for k, v := range secretData {
				se.Data[k] = []byte(v)
			}
			return se, nil
		}

		wrappedCreator := reconciling.SecretObjectWrapper(creator)
		wrappedCreator = reconciling.OwnerRefWrapper(GetEtcdRestoreRef(restore))(wrappedCreator)

		secretName := fmt.Sprintf("%s-backupdownload-%s", restore.Name, rand.String(10))

		if err := reconciling.EnsureNamedObject(
			ctx,
			types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: secretName},
			wrappedCreator, client, &corev1.Secret{}, false); err != nil {
			return nil, "", fmt.Errorf("failed to ensure Secret %s: %w", secretName, err)
		}

		oldRestore := restore.DeepCopy()
		restore.Spec.BackupDownloadCredentialsSecret = secretName
		if err := client.Patch(ctx, restore, ctrlruntimeclient.MergeFrom(oldRestore)); err != nil {
			return nil, "", fmt.Errorf("failed to write etcdrestore.backupDownloadCredentialsSecret: %w", err)
		}
	}

	accessKeyID := secretData[EtcdBackupAndRestoreS3AccessKeyIDKey]
	secretAccessKey := secretData[EtcdBackupAndRestoreS3SecretKeyAccessKeyKey]
	bucketName := secretData[EtcdRestoreS3BucketNameKey]
	endpoint := secretData[EtcdRestoreS3EndpointKey]

	if bucketName == "" {
		return nil, "", fmt.Errorf("s3 bucket name not set")
	}
	if endpoint == "" {
		endpoint = EtcdRestoreDefaultS3SEndpoint
	}

	caBundleConfigMap := &corev1.ConfigMap{}
	caBundleKey := types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: BackupCABundleConfigMapName(cluster)}
	if err := client.Get(ctx, caBundleKey, caBundleConfigMap); err != nil {
		return nil, "", fmt.Errorf("failed to get CA bundle ConfigMap: %w", err)
	}
	bundle, ok := caBundleConfigMap.Data[CABundleConfigMapKey]
	if !ok {
		return nil, "", fmt.Errorf("ConfigMap does not contain key %q", CABundleConfigMapKey)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(bundle)) {
		return nil, "", errors.New("CA bundle does not contain any valid certificates")
	}

	s3Client, err := s3.NewClient(endpoint, accessKeyID, secretAccessKey, pool)
	if err != nil {
		return nil, "", fmt.Errorf("error creating S3 client: %w", err)
	}
	s3Client.SetAppInfo("kubermatic", "v0.2")

	return s3Client, bucketName, nil
}

// GetClusterNodeCIDRMaskSizeIPv4 returns effective mask size used to address the nodes within provided IPv4 Pods CIDR.
func GetClusterNodeCIDRMaskSizeIPv4(cluster *kubermaticv1.Cluster) int32 {
	if cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4 != nil {
		return *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4
	}
	return DefaultNodeCIDRMaskSizeIPv4
}

// GetClusterNodeCIDRMaskSizeIPv6 returns effective mask size used to address the nodes within provided IPv6 Pods CIDR.
func GetClusterNodeCIDRMaskSizeIPv6(cluster *kubermaticv1.Cluster) int32 {
	if cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6 != nil {
		return *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6
	}
	return DefaultNodeCIDRMaskSizeIPv6
}

// GetNodePortsAllowedIPRanges returns effective CIDR range to be used for NodePort services for the given cluster
// and provided allowed IP ranges coming from provider-specific API.
func GetNodePortsAllowedIPRanges(cluster *kubermaticv1.Cluster, allowedIPRanges *kubermaticv1.NetworkRanges, allowedIPRange string, seedAllowedIPRanges *kubermaticv1.NetworkRanges) (res kubermaticv1.NetworkRanges) {
	if allowedIPRanges != nil {
		res.CIDRBlocks = allowedIPRanges.CIDRBlocks
	}
	if allowedIPRange != "" && !containsString(allowedIPRanges.CIDRBlocks, allowedIPRange) {
		res.CIDRBlocks = append(res.CIDRBlocks, allowedIPRange)
	}

	if len(res.CIDRBlocks) == 0 {
		if seedAllowedIPRanges != nil && len(seedAllowedIPRanges.CIDRBlocks) > 0 {
			res.CIDRBlocks = seedAllowedIPRanges.CIDRBlocks
			return res
		}
		if cluster.IsIPv4Only() || cluster.IsDualStack() {
			res.CIDRBlocks = append(res.CIDRBlocks, IPv4MatchAnyCIDR)
		}
		if cluster.IsIPv6Only() || cluster.IsDualStack() {
			res.CIDRBlocks = append(res.CIDRBlocks, IPv6MatchAnyCIDR)
		}
	}
	return
}

// GetDefaultPodCIDRIPv4 returns the default IPv4 pod CIDR for the given provider.
func GetDefaultPodCIDRIPv4(provider kubermaticv1.ProviderType) string {
	if provider == kubermaticv1.KubevirtCloudProvider {
		// KubeVirt cluster can be provisioned on top of k8s cluster created by KKP
		// thus we have to avoid network collision
		return DefaultClusterPodsCIDRIPv4KubeVirt
	}
	return DefaultClusterPodsCIDRIPv4
}

// GetDefaultServicesCIDRIPv4 returns the default IPv4 services CIDR for the given provider.
func GetDefaultServicesCIDRIPv4(provider kubermaticv1.ProviderType) string {
	if provider == kubermaticv1.KubevirtCloudProvider {
		// KubeVirt cluster can be provisioned on top of k8s cluster created by KKP
		// thus we have to avoid network collision
		return DefaultClusterServicesCIDRIPv4KubeVirt
	}
	return DefaultClusterServicesCIDRIPv4
}

// GetDefaultProxyMode returns the default proxy mode for the given provider.
func GetDefaultProxyMode(provider kubermaticv1.ProviderType) string {
	if provider == kubermaticv1.HetznerCloudProvider {
		// IPVS causes issues with Hetzner's LoadBalancers, which should
		// be addressed via https://github.com/kubernetes/enhancements/pull/1392
		return IPTablesProxyMode
	}
	return IPVSProxyMode
}

// GetKubeletPreferredAddressTypes returns the preferred address types in the correct order to be used when
// contacting kubelet from the control plane.
func GetKubeletPreferredAddressTypes(cluster *kubermaticv1.Cluster, isKonnectivityEnabled bool) string {
	if cluster.Spec.Cloud.GCP != nil {
		return "InternalIP"
	}
	if cluster.IsDualStack() && cluster.Spec.Cloud.Hetzner != nil {
		// Due to https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/305
		// InternalIP needs to be preferred over ExternalIP in dual-stack Hetzner clusters
		return "InternalIP,ExternalIP"
	}
	if isKonnectivityEnabled {
		// KAS tries to connect to kubelet via konnectivity-agent in the user-cluster.
		// This request fails because of security policies disallow external traffic to the node.
		// So we prefer InternalIP for contacting kubelet when konnectivity is enabled.
		// Refer: https://github.com/kubermatic/kubermatic/pull/7504#discussion_r700992387
		return "InternalIP,ExternalIP"
	}
	return "ExternalIP,InternalIP"
}

// ConvertGBToBytes converts Gigabytes (GB using decimal) to Bytes.
// Takes a non-negative number of GB.
// Returns the number of bytes and a boolean indicating if overflow occurred.
func ConvertGBToBytes(gb uint64) (bytes uint64, overflow bool) {
	const GB uint64 = 1 << 30

	if GB > 0 && gb > math.MaxUint64/GB {
		return 0, true
	}

	return gb * GB, false
}

func containsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
