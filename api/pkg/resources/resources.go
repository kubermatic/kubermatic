package resources

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	corev1lister "k8s.io/client-go/listers/core/v1"
	certutil "k8s.io/client-go/util/cert"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// KUBERMATICCOMMIT is a magic variable containing the git commit hash of the current (as in currently executing) kubermatic api. It gets feeded by Makefile as a ldflag.
var KUBERMATICCOMMIT string

// KUBERMATICGITTAG is a magic variable containing the output of `git describe` for the current (as in currently executing) kubermatic api. It gets fed by Makefile as an ldflag.
var KUBERMATICGITTAG = "manual_build"

const (
	// ApiserverDeploymentName is the name of the apiserver deployment
	ApiserverDeploymentName = "apiserver"
	//ControllerManagerDeploymentName is the name for the controller manager deployment
	ControllerManagerDeploymentName = "controller-manager"
	//SchedulerDeploymentName is the name for the scheduler deployment
	SchedulerDeploymentName = "scheduler"
	//MachineControllerDeploymentName is the name for the machine-controller deployment
	MachineControllerDeploymentName = "machine-controller"
	// MachineControllerWebhookDeploymentName is the name for the machine-controller webhook deployment
	MachineControllerWebhookDeploymentName = "machine-controller-webhook"
	//MetricsServerDeploymentName is the name for the metrics-server deployment
	MetricsServerDeploymentName = "metrics-server"
	//OpenVPNServerDeploymentName is the name for the openvpn server deployment
	OpenVPNServerDeploymentName = "openvpn-server"
	//DNSResolverDeploymentName is the name of the dns resolver deployment
	DNSResolverDeploymentName = "dns-resolver"
	//DNSResolverConfigMapName is the name of the dns resolvers configmap
	DNSResolverConfigMapName = "dns-resolver"
	//DNSResolverServiceName is the name of the dns resolvers service
	DNSResolverServiceName = "dns-resolver"
	//DNSResolverPodDisruptionBudetName is the name of the dns resolvers pdb
	DNSResolverPodDisruptionBudetName = "dns-resolver"
	//DNSResolverVPAName is the name of the dns resolvers VerticalPodAutoscaler
	KubeStateMetricsDeploymentName = "kube-state-metrics"
	// UserClusterControllerDeploymentName is the name of the usercluster-controller deployment
	UserClusterControllerDeploymentName = "usercluster-controller"

	//PrometheusStatefulSetName is the name for the prometheus StatefulSet
	PrometheusStatefulSetName = "prometheus"
	//EtcdStatefulSetName is the name for the etcd StatefulSet
	EtcdStatefulSetName = "etcd"

	//ApiserverExternalServiceName is the name for the external apiserver service
	ApiserverExternalServiceName = "apiserver-external"
	//ApiserverInternalServiceName is the name for the internal apiserver service
	ApiserverInternalServiceName = "apiserver"
	// MetricsServerServiceName is the name for the metrics-server service
	MetricsServerServiceName = "metrics-server"
	// MetricsServerExternalNameServiceName is the name for the metrics-server service inside the user cluster
	MetricsServerExternalNameServiceName = "metrics-server"
	//EtcdServiceName is the name for the etcd service
	EtcdServiceName = "etcd"
	//EtcdDefragCronJobName is the name for the defrag cronjob deployment
	EtcdDefragCronJobName = "etcd-defragger"
	//OpenVPNServerServiceName is the name for the openvpn server service
	OpenVPNServerServiceName = "openvpn-server"
	//MachineControllerWebhookServiceName is the name of the machine-controller webhook service
	MachineControllerWebhookServiceName = "machine-controller-webhook"

	// MetricsServerAPIServiceName is the name for the metrics-server APIService
	MetricsServerAPIServiceName = "v1beta1.metrics.k8s.io"

	//AdminKubeconfigSecretName is the name for the secret containing the private ca key
	AdminKubeconfigSecretName = "admin-kubeconfig"
	//SchedulerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the scheduler
	SchedulerKubeconfigSecretName = "scheduler-kubeconfig"
	//KubeletDnatControllerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the kubeletdnatcontroller
	KubeletDnatControllerKubeconfigSecretName = "kubeletdnatcontroller-kubeconfig"
	//KubeStateMetricsKubeconfigSecretName is the name for the secret containing the kubeconfig used by kube-state-metrics
	KubeStateMetricsKubeconfigSecretName = "kube-state-metrics-kubeconfig"
	//MetricsServerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the metrics-server
	MetricsServerKubeconfigSecretName = "metrics-server"
	//ControllerManagerKubeconfigSecretName is the name of the secret containing the kubeconfig used by controller manager
	ControllerManagerKubeconfigSecretName = "controllermanager-kubeconfig"
	//MachineControllerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the machinecontroller
	MachineControllerKubeconfigSecretName = "machinecontroller-kubeconfig"
	//MachineControllerWebhookServingCertSecretName is the name for the secret containing the serving cert for the
	//machine-controller webhook
	MachineControllerWebhookServingCertSecretName = "machinecontroller-webhook-serving-cert"
	//MachineControllerWebhookServingCertCertKeyName is the name for the key that contains the cert
	MachineControllerWebhookServingCertCertKeyName = "cert.pem"
	//MachineControllerWebhookServingCertKeyKeyName is the name for the key that contains the key
	MachineControllerWebhookServingCertKeyKeyName = "key.pem"
	//PrometheusApiserverClientCertificateSecretName is the name for the secret containing the client certificate used by prometheus to access the apiserver
	PrometheusApiserverClientCertificateSecretName = "prometheus-apiserver-certificate"

	// ImagePullSecretName specifies the name of the dockercfg secret used to access the private repo.
	ImagePullSecretName = "dockercfg"

	//FrontProxyCASecretName is the name for the secret containing the front proxy ca
	FrontProxyCASecretName = "front-proxy-ca"
	//CASecretName is the name for the secret containing the root ca
	CASecretName = "ca"
	//ApiserverTLSSecretName is the name for the secrets required for the apiserver tls
	ApiserverTLSSecretName = "apiserver-tls"
	//KubeletClientCertificatesSecretName is the name for the secret containing the kubelet client certificates
	KubeletClientCertificatesSecretName = "kubelet-client-certificates"
	//ServiceAccountKeySecretName is the name for the secret containing the service account key
	ServiceAccountKeySecretName = "service-account-key"
	//TokensSecretName is the name for the secret containing the user tokens
	TokensSecretName = "tokens"
	// OpenVPNCASecretName is the name of the secret that contains the OpenVPN CA
	OpenVPNCASecretName = "openvpn-ca"
	//OpenVPNServerCertificatesSecretName is the name for the secret containing the openvpn server certificates
	OpenVPNServerCertificatesSecretName = "openvpn-server-certificates"
	//OpenVPNClientCertificatesSecretName is the name for the secret containing the openvpn client certificates
	OpenVPNClientCertificatesSecretName = "openvpn-client-certificates"
	//EtcdTLSCertificateSecretName is the name for the secret containing the etcd tls certificate used for transport security
	EtcdTLSCertificateSecretName = "etcd-tls-certificate"
	//ApiserverEtcdClientCertificateSecretName is the name for the secret containing the client certificate used by the apiserver for authenticating against etcd
	ApiserverEtcdClientCertificateSecretName = "apiserver-etcd-client-certificate"
	//ApiserverFrontProxyClientCertificateSecretName is the name for the secret containing the apiserver's client certificate for proxy auth
	ApiserverFrontProxyClientCertificateSecretName = "apiserver-proxy-client-certificate"
	// DexCASecretName is the name of the secret that contains the Dex CA bundle
	DexCASecretName = "dex-ca"
	// DexCAFileName is the name of Dex CA bundle file
	DexCAFileName = "caBundle.pem"

	//CloudConfigConfigMapName is the name for the configmap containing the cloud-config
	CloudConfigConfigMapName = "cloud-config"
	//OpenVPNClientConfigsConfigMapName is the name for the ConfigMap containing the OpenVPN client config used within the user cluster
	OpenVPNClientConfigsConfigMapName = "openvpn-client-configs"
	//OpenVPNClientConfigConfigMapName is the name for the ConfigMap containing the OpenVPN client config used by the client inside the user cluster
	OpenVPNClientConfigConfigMapName = "openvpn-client-config"
	//ClusterInfoConfigMapName is the name for the ConfigMap containing the cluster-info used by the bootstrap token machanism
	ClusterInfoConfigMapName = "cluster-info"
	//PrometheusConfigConfigMapName is the name for the configmap containing the prometheus config
	PrometheusConfigConfigMapName = "prometheus"

	//PrometheusServiceAccountName is the name for the Prometheus serviceaccount
	PrometheusServiceAccountName = "prometheus"

	//PrometheusRoleName is the name for the Prometheus role
	PrometheusRoleName = "prometheus"

	//PrometheusRoleBindingName is the name for the Prometheus rolebinding
	PrometheusRoleBindingName = "prometheus"

	//MachineControllerCertUsername is the name of the user coming from kubeconfig cert
	MachineControllerCertUsername = "machine-controller"
	//KubeStateMetricsCertUsername is the name of the user coming from kubeconfig cert
	KubeStateMetricsCertUsername = "kube-state-metrics"
	//MetricsServerCertUsername is the name of the user coming from kubeconfig cert
	MetricsServerCertUsername = "metrics-server"
	//ControllerManagerCertUsername is the name of the user coming from kubeconfig cert
	ControllerManagerCertUsername = "system:kube-controller-manager"
	//SchedulerCertUsername is the name of the user coming from kubeconfig cert
	SchedulerCertUsername = "system:kube-scheduler"
	//KubeletDnatControllerCertUsername is the name of the user coming from kubeconfig cert
	KubeletDnatControllerCertUsername = "kubermatic:kubeletdnat-controller"
	// PrometheusCertUsername is the name of the user coming from kubeconfig cert
	PrometheusCertUsername = "prometheus"

	// KubeletDnatControllerClusterRoleName is the name for the KubeletDnatController cluster role
	KubeletDnatControllerClusterRoleName = "system:kubermatic-kubeletdnat-controller"
	// KubeletDnatControllerClusterRoleBindingName is the name for the KubeletDnatController clusterrolebinding
	KubeletDnatControllerClusterRoleBindingName = "system:kubermatic-kubeletdnat-controller"

	//ClusterInfoReaderRoleName is the name for the role which allows reading the cluster-info ConfigMap
	ClusterInfoReaderRoleName = "cluster-info"
	//MachineControllerRoleName is the name for the MachineController roles
	MachineControllerRoleName = "machine-controller"
	//MachineControllerRoleBindingName is the name for the MachineController rolebinding
	MachineControllerRoleBindingName = "machine-controller"
	//ClusterInfoAnonymousRoleBindingName is the name for the RoleBinding giving access to the cluster-info ConfigMap to anonymous users
	ClusterInfoAnonymousRoleBindingName = "cluster-info"
	//MetricsServerAuthReaderRoleName is the name for the metrics server role
	MetricsServerAuthReaderRoleName = "metrics-server-auth-reader"
	//MachineControllerClusterRoleName is the name for the MachineController cluster role
	MachineControllerClusterRoleName = "system:kubermatic-machine-controller"
	//KubeStateMetricsClusterRoleName is the name for the KubeStateMetrics cluster role
	KubeStateMetricsClusterRoleName = "system:kubermatic-kube-state-metrics"
	//MetricsServerClusterRoleName is the name for the metrics server cluster role
	MetricsServerClusterRoleName = "system:metrics-server"
	//PrometheusClusterRoleName is the name for the Prometheus cluster role
	PrometheusClusterRoleName = "external-prometheus"
	//MachineControllerClusterRoleBindingName is the name for the MachineController ClusterRoleBinding
	MachineControllerClusterRoleBindingName = "system:kubermatic-machine-controller"
	//KubeStateMetricsClusterRoleBindingName is the name for the KubeStateMetrics ClusterRoleBinding
	KubeStateMetricsClusterRoleBindingName = "system:kubermatic-kube-state-metrics"
	//PrometheusClusterRoleBindingName is the name for the Prometheus ClusterRoleBinding
	PrometheusClusterRoleBindingName = "system:external-prometheus"
	//MetricsServerResourceReaderClusterRoleBindingName is the name for the metrics server ClusterRoleBinding
	MetricsServerResourceReaderClusterRoleBindingName = "system:metrics-server"

	// EtcdPodDisruptionBudgetName is the name of the PDB for the etcd StatefulSet
	EtcdPodDisruptionBudgetName = "etcd"
	// ApiserverPodDisruptionBudgetName is the name of the PDB for the apiserver deployment
	ApiserverPodDisruptionBudgetName = "apiserver"
	// MetricsServerPodDisruptionBudgetName is the name of the PDB for the metrics-server deployment
	MetricsServerPodDisruptionBudgetName = "metrics-server"

	// DefaultOwnerReadOnlyMode represents file mode with read permission for owner only
	DefaultOwnerReadOnlyMode = 0400

	// DefaultAllReadOnlyMode represents file mode with read permissions for all
	DefaultAllReadOnlyMode = 0444

	// AppLabelKey defines the label key app which should be used within resources
	AppLabelKey = "app"
	// ClusterLabelKey defines the label key for the cluster name
	ClusterLabelKey = "cluster"

	// EtcdClusterSize defines the size of the etcd to use
	EtcdClusterSize = 3

	// RegistryGCR defines the kubernetes docker registry at google
	RegistryGCR = "gcr.io"
	// RegistryDocker defines the default docker.io registry
	RegistryDocker = "docker.io"
	// RegistryQuay defines the image registry from coreos/redhat - quay
	RegistryQuay = "quay.io"

	// TopologyKeyHostname defines the topology key for the node hostname
	TopologyKeyHostname = "kubernetes.io/hostname"

	// MachineCRDName defines the CRD name for machine objects
	MachineCRDName = "machines.cluster.k8s.io"
	// MachineSetCRDName defines the CRD name for machineset objects
	MachineSetCRDName = "machinesets.cluster.k8s.io"
	// MachineDeploymentCRDName defines the CRD name for machinedeployment objects
	MachineDeploymentCRDName = "machinedeployments.cluster.k8s.io"
	// ClusterCRDName defines the CRD name for cluster objects
	ClusterCRDName = "clusters.cluster.k8s.io"

	// MachineControllerMutatingWebhookConfigurationName is the name of the machine-controllers mutating webhook
	// configuration
	MachineControllerMutatingWebhookConfigurationName = "machine-controller.kubermatic.io"

	// InternalUserClusterAdminKubeconfigSecretName is the name of the secret containing an admin kubeconfig that can only be used from
	// within the seed cluster
	InternalUserClusterAdminKubeconfigSecretName = "internal-admin-kubeconfig"
	// UserClusterControllerCertUsername is the name of the user coming from kubeconfig cert
	InternalUserClusterAdminKubeconfigCertUsername = "kubermatic-controllers"
)

const (
	// CAKeySecretKey ca.key
	CAKeySecretKey = "ca.key"
	// CACertSecretKey ca.crt
	CACertSecretKey = "ca.crt"
	// ApiserverTLSKeySecretKey apiserver-tls.key
	ApiserverTLSKeySecretKey = "apiserver-tls.key"
	// ApiserverTLSCertSecretKey apiserver-tls.crt
	ApiserverTLSCertSecretKey = "apiserver-tls.crt"
	// KubeletClientKeySecretKey kubelet-client.key
	KubeletClientKeySecretKey = "kubelet-client.key"
	// KubeletClientCertSecretKey kubelet-client.crt
	KubeletClientCertSecretKey = "kubelet-client.crt" // FIXME confusing naming: s/CertSecretKey/CertSecretName/
	// ServiceAccountKeySecretKey sa.key
	ServiceAccountKeySecretKey = "sa.key"
	// ServiceAccountKeyPublicKey is the public key for the service account signer key
	ServiceAccountKeyPublicKey = "sa.pub"
	// KubeconfigSecretKey kubeconfig
	KubeconfigSecretKey = "kubeconfig"
	// TokensSecretKey tokens.csv
	TokensSecretKey = "tokens.csv"
	// OpenVPNCACertKey cert.pem, must match CACertSecretKey, otherwise getClusterCAFromLister doesnt work as it has
	// the key hardcoded
	OpenVPNCACertKey = CACertSecretKey
	// OpenVPNCAKeyKey key.pem, must match CAKeySecretKey, otherwise getClusterCAFromLister doesnt work as it has
	// the key hardcoded
	OpenVPNCAKeyKey = CAKeySecretKey
	// OpenVPNServerKeySecretKey server.key
	OpenVPNServerKeySecretKey = "server.key"
	// OpenVPNServerCertSecretKey server.crt
	OpenVPNServerCertSecretKey = "server.crt"
	// OpenVPNInternalClientKeySecretKey client.key
	OpenVPNInternalClientKeySecretKey = "client.key"
	// OpenVPNInternalClientCertSecretKey client.crt
	OpenVPNInternalClientCertSecretKey = "client.crt"
	// EtcdTLSCertSecretKey etcd-tls.crt
	EtcdTLSCertSecretKey = "etcd-tls.crt"
	// EtcdTLSKeySecretKey etcd-tls.key
	EtcdTLSKeySecretKey = "etcd-tls.key"

	// KubeconfigDefaultContextKey is the context key used for all kubeconfigs
	KubeconfigDefaultContextKey = "default"

	// ApiserverEtcdClientCertificateCertSecretKey apiserver-etcd-client.crt
	ApiserverEtcdClientCertificateCertSecretKey = "apiserver-etcd-client.crt"
	// ApiserverEtcdClientCertificateKeySecretKey apiserver-etcd-client.key
	ApiserverEtcdClientCertificateKeySecretKey = "apiserver-etcd-client.key"

	// ApiserverProxyClientCertificateCertSecretKey apiserver-proxy-client.crt
	ApiserverProxyClientCertificateCertSecretKey = "apiserver-proxy-client.crt"
	// ApiserverProxyClientCertificateKeySecretKey apiserver-proxy-client.key
	ApiserverProxyClientCertificateKeySecretKey = "apiserver-proxy-client.key"

	// BackupEtcdClientCertificateCertSecretKey backup-etcd-client.crt
	BackupEtcdClientCertificateCertSecretKey = "backup-etcd-client.crt"
	// BackupEtcdClientCertificateKeySecretKey backup-etcd-client.key
	BackupEtcdClientCertificateKeySecretKey = "backup-etcd-client.key"

	// PrometheusClientCertificateCertSecretKey prometheus-client.crt
	PrometheusClientCertificateCertSecretKey = "prometheus-client.crt"
	// PrometheusClientCertificateKeySecretKey prometheus-client.key
	PrometheusClientCertificateKeySecretKey = "prometheus-client.key"
)

const (
	minimumCertValidity30d = 30 * 24 * time.Hour
)

// ECDSAKeyPair is a ECDSA x509 certifcate and private key
type ECDSAKeyPair struct {
	Key  *ecdsa.PrivateKey
	Cert *x509.Certificate
}

// CRDCreateor defines an interface to create/update CustomRessourceDefinitions
type CRDCreateor = func(version semver.Semver, existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error)

// APIServiceCreator defines an interface to create/update APIService's
type APIServiceCreator = func(existing *apiregistrationv1beta1.APIService) (*apiregistrationv1beta1.APIService, error)

// GetClusterExternalIP returns a net.IP for the given Cluster
func GetClusterExternalIP(cluster *kubermaticv1.Cluster) (*net.IP, error) {
	ip := net.ParseIP(cluster.Address.IP)
	if ip == nil {
		return nil, fmt.Errorf("failed to create a net.IP object from the external cluster IP '%s'", cluster.Address.IP)
	}
	return &ip, nil
}

// GetClusterRef returns a metav1.OwnerReference for the given Cluster
func GetClusterRef(cluster *kubermaticv1.Cluster) metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))
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
	ip, _, err := net.ParseCIDR(block)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster dns ip for cluster `%s`: %v'", block, err)
	}
	ip[len(ip)-1] = ip[len(ip)-1] + 10
	return ip.String(), nil
}

// InClusterApiserverIP returns the first usable IP of the service cidr.
// Its the in cluster IP for the apiserver
func InClusterApiserverIP(cluster *kubermaticv1.Cluster) (*net.IP, error) {
	if len(cluster.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		return nil, errors.New("no service cidr defined")
	}

	block := cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0]
	ip, _, err := net.ParseCIDR(block)
	if err != nil {
		return nil, fmt.Errorf("invalid service cidr %s", block)
	}
	ip[len(ip)-1] = ip[len(ip)-1] + 1
	return &ip, nil
}

type userClusterDNSPolicyAndConfigData interface {
	Cluster() *kubermaticv1.Cluster
	ClusterIPByServiceName(name string) (string, error)
}

// UserClusterDNSPolicyAndConfig returns a DNSPolicy and DNSConfig to configure Pods to use user cluster DNS
func UserClusterDNSPolicyAndConfig(d userClusterDNSPolicyAndConfigData) (corev1.DNSPolicy, *corev1.PodDNSConfig, error) {
	// DNSNone indicates that the pod should use empty DNS settings. DNS
	// parameters such as nameservers and search paths should be defined via
	// DNSConfig.
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

// BaseAppLabel returns the minimum required labels
func BaseAppLabel(name string, additionalLabels map[string]string) map[string]string {
	labels := map[string]string{
		AppLabelKey: name,
	}
	for k, v := range additionalLabels {
		labels[k] = v
	}
	return labels
}

// AppClusterLabel returns the base app label + the cluster label. Additional labels can be included as well
func AppClusterLabel(appName, clusterName string, additionalLabels map[string]string) map[string]string {
	podLabels := BaseAppLabel(appName, additionalLabels)
	podLabels["cluster"] = clusterName

	return podLabels
}

// CertWillExpireSoon returns if the certificate will expire in the next 30 days
func CertWillExpireSoon(cert *x509.Certificate) bool {
	return time.Until(cert.NotAfter) < minimumCertValidity30d
}

// IsServerCertificateValidForAllOf validates if the given data is present in the given server certificate
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

	certIPs := sets.NewString(getIPStrings(cert.IPAddresses)...)
	wantIPs := sets.NewString(getIPStrings(altNames.IPs)...)

	if !wantIPs.Equal(certIPs) {
		return false
	}

	wantDNSNames := sets.NewString(altNames.DNSNames...)
	certDNSNames := sets.NewString(cert.DNSNames...)

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
		glog.Errorf("Certificate verification for CN %s failed due to: %v", commonName, err)
		return false
	}

	return true
}

// IsClientCertificateValidForAllOf validates if the given data matches exactly the given client certificate
// (It also returns true if all given data is in the cert, but the cert has more organizations)
func IsClientCertificateValidForAllOf(cert *x509.Certificate, commonName string, organizations []string, ca *x509.Certificate) bool {
	if CertWillExpireSoon(cert) {
		return false
	}

	if cert.Subject.CommonName != commonName {
		return false
	}

	wantOrganizations := sets.NewString(organizations...)
	certOrganizations := sets.NewString(cert.Subject.Organization...)

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
		glog.Errorf("Certificate verification for CN %s failed due to: %v", commonName, err)
		return false
	}

	return true
}

func getECDSAClusterCAFromLister(ctx context.Context, name string, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) (*ECDSAKeyPair, error) {
	cert, key, err := getClusterCAFromLister(ctx, name, cluster, client)
	if err != nil {
		return nil, err
	}
	ecdsaKey, isECDSAKey := key.(*ecdsa.PrivateKey)
	if !isECDSAKey {
		return nil, errors.New("key is not a ECDSA key")
	}
	return &ECDSAKeyPair{Cert: cert, Key: ecdsaKey}, nil
}

func getRSAClusterCAFromLister(ctx context.Context, name string, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) (*triple.KeyPair, error) {
	cert, key, err := getClusterCAFromLister(ctx, name, cluster, client)
	if err != nil {
		return nil, err
	}
	rsaKey, isRSAKey := key.(*rsa.PrivateKey)
	if !isRSAKey {
		return nil, errors.New("key is not a RSA key")
	}
	return &triple.KeyPair{Cert: cert, Key: rsaKey}, nil
}

// getClusterCAFromLister returns the CA of the cluster from the lister
func getClusterCAFromLister(ctx context.Context, name string, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) (*x509.Certificate, interface{}, error) {
	caSecret := &corev1.Secret{}
	caSecretKey := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: name}
	if err := client.Get(ctx, caSecretKey, caSecret); err != nil {
		return nil, nil, fmt.Errorf("unable to check if a CA cert already exists: %v", err)
	}

	certs, err := certutil.ParseCertsPEM(caSecret.Data[CACertSecretKey])
	if err != nil {
		return nil, nil, fmt.Errorf("got an invalid cert from the CA secret %s: %v", caSecretKey, err)
	}

	if len(certs) != 1 {
		return nil, nil, fmt.Errorf("did not find exactly one but %v certificates in the CA secret", len(certs))
	}

	key, err := certutil.ParsePrivateKeyPEM(caSecret.Data[CAKeySecretKey])
	if err != nil {
		return nil, nil, fmt.Errorf("got an invalid private key from the CA secret %s: %v", caSecretKey, err)
	}

	return certs[0], key, nil
}

// GetDexCAFromFile returns the Dex CA from the lister
func GetDexCAFromFile(caBundleFilePath string) ([]*x509.Certificate, error) {

	f, err := os.Open(caBundleFilePath)
	if err != nil {
		return nil, fmt.Errorf("got an invalid CA bundle file %v", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			glog.Fatal(err)
		}
	}()

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	dexCACerts, err := certutil.ParseCertsPEM(bytes)
	if err != nil {
		return nil, fmt.Errorf("got an invalid cert: %v", err)
	}

	return dexCACerts, nil
}

// GetClusterRootCA returns the root CA of the cluster from the lister
func GetClusterRootCA(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) (*triple.KeyPair, error) {
	return getRSAClusterCAFromLister(ctx, CASecretName, cluster, client)
}

// GetClusterFrontProxyCA returns the frontproxy CA of the cluster from the lister
func GetClusterFrontProxyCA(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) (*triple.KeyPair, error) {
	return getRSAClusterCAFromLister(ctx, FrontProxyCASecretName, cluster, client)
}

// GetOpenVPNCA returns the OpenVPN CA of the cluster from the lister
func GetOpenVPNCA(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) (*ECDSAKeyPair, error) {
	return getECDSAClusterCAFromLister(ctx, OpenVPNCASecretName, cluster, client)
}

// ClusterIPForService returns the cluster ip for the given service
func ClusterIPForService(name, namespace string, serviceLister corev1lister.ServiceLister) (*net.IP, error) {
	service, err := serviceLister.Services(namespace).Get(name)
	if err != nil {
		return nil, fmt.Errorf("could not get service %s/%s from lister: %v", namespace, name, err)
	}

	if service.Spec.ClusterIP == "" {
		return nil, errors.New("service has no ClusterIP")
	}

	ip := net.ParseIP(service.Spec.ClusterIP)
	if ip == nil {
		return nil, fmt.Errorf("service %s/%s has no valid cluster ip (\"%s\"): %v", namespace, name, service.Spec.ClusterIP, err)
	}

	return &ip, nil
}

// GetAbsoluteServiceDNSName returns the absolute DNS name for the given service and the given cluster. Absolute means a trailing dot will be appended to the DNS name
func GetAbsoluteServiceDNSName(service, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local.", service, namespace)
}

// SecretRevision returns the resource version of the Secret specified by name.
func SecretRevision(ctx context.Context, key types.NamespacedName, client ctrlruntimeclient.Client) (string, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, key, secret); err != nil {
		return "", fmt.Errorf("could not get Secret %s: %v", key, err)
	}
	return secret.ResourceVersion, nil
}

// ConfigMapRevision returns the resource version of the ConfigMap specified by name.
func ConfigMapRevision(ctx context.Context, key types.NamespacedName, client ctrlruntimeclient.Client) (string, error) {
	cm := &corev1.ConfigMap{}
	if err := client.Get(ctx, key, cm); err != nil {
		return "", fmt.Errorf("could not get ConfigMap %s: %v", key, err)
	}
	return cm.ResourceVersion, nil
}

// GetPodTemplateLabels returns a set of labels for a Pod including the revisions of depending secrets and configmaps.
// This will force pods being restarted as soon as one of the secrets/configmaps get updated.
func GetPodTemplateLabels(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	appName, clusterName, namespace string,
	volumes []corev1.Volume,
	additionalLabels map[string]string,
) (map[string]string, error) {
	podLabels := AppClusterLabel(appName, clusterName, additionalLabels)

	for _, v := range volumes {
		if v.VolumeSource.Secret != nil {
			key := types.NamespacedName{Namespace: namespace, Name: v.VolumeSource.Secret.SecretName}
			revision, err := SecretRevision(ctx, key, client)
			if err != nil {
				return nil, err
			}
			podLabels[fmt.Sprintf("%s-secret-revision", v.VolumeSource.Secret.SecretName)] = revision
		}
		if v.VolumeSource.ConfigMap != nil {
			key := types.NamespacedName{Namespace: namespace, Name: v.VolumeSource.ConfigMap.Name}
			revision, err := ConfigMapRevision(ctx, key, client)
			if err != nil {
				return nil, err
			}
			podLabels[fmt.Sprintf("%s-configmap-revision", v.VolumeSource.ConfigMap.Name)] = revision
		}
	}

	return podLabels, nil
}
