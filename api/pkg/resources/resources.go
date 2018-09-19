package resources

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"

	"github.com/go-test/deep"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	admissionv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	corev1lister "k8s.io/client-go/listers/core/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

// KUBERMATICCOMMIT is a magic variable containing the git commit hash of the current (as in currently executing) kubermatic api. It gets feeded by Makefile as a ldflag.
var KUBERMATICCOMMIT string

const (
	// KubermaticNamespaceName specifies the name of the kubermatic namespace
	KubermaticNamespaceName = "kubermatic"
	// ApiserverDeploymentName is the name of the apiserver deployment
	ApiserverDeploymentName = "apiserver"
	//ControllerManagerDeploymentName is the name for the controller manager deployment
	ControllerManagerDeploymentName = "controller-manager"
	//SchedulerDeploymentName is the name for the scheduler deployment
	SchedulerDeploymentName = "scheduler"
	//MachineControllerDeploymentName is the name for the machine-controller deployment
	MachineControllerDeploymentName = "machine-controller"
	//OpenVPNServerDeploymentName is the name for the openvpn server deployment
	OpenVPNServerDeploymentName = "openvpn-server"
	//DNSResolverDeploymentName is the name of the dns resolver deployment
	DNSResolverDeploymentName = "dns-resolver"
	//DNSResolverConfigMapName is the name of the dns resolvers configmap
	DNSResolverConfigMapName = "dns-resolver"
	//DNSResolverServiceName is the name of the dns resolvers service
	DNSResolverServiceName = "dns-resolver"
	//KubeStateMetricsDeploymentName is the name for the kube-state-metrics deployment
	KubeStateMetricsDeploymentName = "kube-state-metrics"

	//PrometheusStatefulSetName is the name for the prometheus StatefulSet
	PrometheusStatefulSetName = "prometheus"
	//EtcdStatefulSetName is the name for the etcd StatefulSet
	EtcdStatefulSetName = "etcd"

	//ApiserverExternalServiceName is the name for the external apiserver service
	ApiserverExternalServiceName = "apiserver-external"
	//ApiserverInternalServiceName is the name for the internal apiserver service
	ApiserverInternalServiceName = "apiserver"
	//PrometheusServiceName is the name for the prometheus service
	PrometheusServiceName = "prometheus"
	//EtcdServiceName is the name for the etcd service
	EtcdServiceName = "etcd"
	//EtcdDefragCronJobName is the name for the defrag cronjob deployment
	EtcdDefragCronJobName = "etcd-defragger"
	//OpenVPNServerServiceName is the name for the openvpn server service
	OpenVPNServerServiceName = "openvpn-server"

	//AdminKubeconfigSecretName is the name for the secret containing the private ca key
	AdminKubeconfigSecretName = "admin-kubeconfig"
	//SchedulerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the scheduler
	SchedulerKubeconfigSecretName = "scheduler-kubeconfig"
	//KubeletDnatControllerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the kubeletdnatcontroller
	KubeletDnatControllerKubeconfigSecretName = "kubeletdnatcontroller-kubeconfig"
	//KubeStateMetricsKubeconfigSecretName is the name for the secret containing the kubeconfig used by kube-state-metrics
	KubeStateMetricsKubeconfigSecretName = "kube-state-metrics-kubeconfig"
	//ControllerManagerKubeconfigSecretName is the name of the secret containing the kubeconfig used by controller manager
	ControllerManagerKubeconfigSecretName = "controllermanager-kubeconfig"
	//MachineControllerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the machinecontroller
	MachineControllerKubeconfigSecretName = "machinecontroller-kubeconfig"
	//IPAMControllerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the ipam controller
	IPAMControllerKubeconfigSecretName = "ipamcontroller-kubeconfig"

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

	//CloudConfigConfigMapName is the name for the configmap containing the cloud-config
	CloudConfigConfigMapName = "cloud-config"
	//OpenVPNClientConfigsConfigMapName is the name for the ConfigMap containing the OpenVPN client config used within the user cluster
	OpenVPNClientConfigsConfigMapName = "openvpn-client-configs"
	//OpenVPNClientConfigConfigMapName is the name for the ConfigMap containing the OpenVPN client config used by the client inside the user cluster
	OpenVPNClientConfigConfigMapName = "openvpn-client-config"
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
	//ControllerManagerCertUsername is the name of the user coming from kubeconfig cert
	ControllerManagerCertUsername = "system:kube-controller-manager"
	//SchedulerCertUsername is the name of the user coming from kubeconfig cert
	SchedulerCertUsername = "system:kube-scheduler"
	//KubeletDnatControllerCertUsername is the name of the user coming from kubeconfig cert
	KubeletDnatControllerCertUsername = "kubermatic:kubeletdnat-controller"
	//IPAMControllerCertUsername is the name of the user coming from kubeconfig cert
	IPAMControllerCertUsername = "kubermatic:ipam-controller"

	// MachineIPAMInitializerConfigurationName is the name of the initializerconfiguration used for setting up static ips for machines
	MachineIPAMInitializerConfigurationName = "ipam-initializer"
	// MachineIPAMInitializerName is the name of the initializer used for setting up static ips for machines
	MachineIPAMInitializerName = "ipam.kubermatic.io"
	// IPAMControllerDeploymentName is the name of the ipam controller's deployment
	IPAMControllerDeploymentName = "ipam-controller"

	// IPAMControllerRoleName is the name for the IPAMController roles
	IPAMControllerRoleName = "ipam-controller"
	// IPAMControllerRoleBindingName is the name for the IPAMController rolebinding
	IPAMControllerRoleBindingName = "ipam-controller"
	// IPAMControllerClusterRoleName is the name for the IPAMController cluster role
	IPAMControllerClusterRoleName = "system:kubermatic-ipam-controller"
	// IPAMControllerClusterRoleBindingName is the name for the IPAMController clusterrolebinding
	IPAMControllerClusterRoleBindingName = "system:kubermatic-ipam-controller"

	// KubeletDnatControllerClusterRoleName is the name for the KubeletDnatController cluster role
	KubeletDnatControllerClusterRoleName = "system:kubermatic-kubeletdnat-controller"
	// KubeletDnatControllerClusterRoleBindingName is the name for the KubeletDnatController clusterrolebinding
	KubeletDnatControllerClusterRoleBindingName = "system:kubermatic-kubeletdnat-controller"

	//MachineControllerRoleName is the name for the MachineController roles
	MachineControllerRoleName = "machine-controller"
	//MachineControllerRoleBindingName is the name for the MachineController rolebinding
	MachineControllerRoleBindingName = "machine-controller"
	//MachineControllerClusterRoleName is the name for the MachineController cluster role
	MachineControllerClusterRoleName = "system:kubermatic-machine-controller"
	//KubeStateMetricsClusterRoleName is the name for the KubeStateMetrics cluster role
	KubeStateMetricsClusterRoleName = "system:kubermatic-kube-state-metrics"
	//MachineControllerClusterRoleBindingName is the name for the MachineController clusterrolebinding
	MachineControllerClusterRoleBindingName = "system:kubermatic-machine-controller"
	//KubeStateMetricsClusterRoleBindingName is the name for the KubeStateMetrics clusterrolebinding
	KubeStateMetricsClusterRoleBindingName = "system:kubermatic-kube-state-metrics"
	//ControllerManagerRoleBindingName is the name of the controller-manager's rolebindings
	ControllerManagerRoleBindingName = "kubermatic:controller-manager"
	//ControllerManagerClusterRoleBindingName is the name of the controller-manager's clusterrolebindings
	ControllerManagerClusterRoleBindingName = "kubermatic:controller-manager"

	// EtcdPodDisruptionBudgetName is the name of the PDB for the etcd statefulset
	EtcdPodDisruptionBudgetName = "etcd"

	// DefaultOwnerReadOnlyMode represents file mode with read permission for owner only
	DefaultOwnerReadOnlyMode = 0400

	// DefaultAllReadOnlyMode represents file mode with read permissions for all
	DefaultAllReadOnlyMode = 0444

	// AppLabelKey defines the label key app which should be used within resources
	AppLabelKey = "app"

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
	// KubeconfigSecretKey kubeconfig
	KubeconfigSecretKey = "kubeconfig"
	// TokensSecretKey tokens.csv
	TokensSecretKey = "tokens.csv"
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
)

const (
	minimumCertValidity30d = 30 * 24 * time.Hour
)

// ConfigMapCreator defines an interface to create/update ConfigMap's
type ConfigMapCreator = func(data ConfigMapDataProvider, existing *corev1.ConfigMap) (*corev1.ConfigMap, error)

// SecretCreator defines an interface to create/update Secret's
type SecretCreator = func(data SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error)

// StatefulSetCreator defines an interface to create/update StatefulSet
type StatefulSetCreator = func(data StatefulSetDataProvider, existing *appsv1.StatefulSet) (*appsv1.StatefulSet, error)

// ServiceCreator defines an interface to create/update Services
type ServiceCreator = func(data ServiceDataProvider, existing *corev1.Service) (*corev1.Service, error)

// ServiceAccountCreator defines an interface to create/update ServiceAccounts
type ServiceAccountCreator = func(data ServiceAccountDataProvider, existing *corev1.ServiceAccount) (*corev1.ServiceAccount, error)

// RoleCreator defines an interface to create/update RBAC Roles
type RoleCreator = func(data RoleDataProvider, existing *rbacv1.Role) (*rbacv1.Role, error)

// RoleBindingCreator defines an interface to create/update RBAC RoleBinding's
type RoleBindingCreator = func(data RoleBindingDataProvider, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)

// ClusterRoleCreator defines an interface to create/update RBAC ClusterRoles
type ClusterRoleCreator = func(data ClusterRoleDataProvider, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error)

// ClusterRoleBindingCreator defines an interface to create/update RBAC ClusterRoleBinding's
type ClusterRoleBindingCreator = func(data ClusterRoleBindingDataProvider, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error)

// DeploymentCreator defines an interface to create/update Deployment's
type DeploymentCreator = func(data DeploymentDataProvider, existing *appsv1.Deployment) (*appsv1.Deployment, error)

// InitializerConfigurationCreator defines an interface to create/update InitializerConfigurations
type InitializerConfigurationCreator = func(data *TemplateData, existing *admissionv1alpha1.InitializerConfiguration) (*admissionv1alpha1.InitializerConfiguration, error)

// PodDisruptionBudgetCreator defines an interface to create/update PodDisruptionBudgets
type PodDisruptionBudgetCreator = func(data *TemplateData, existing *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error)

// CronJobCreator defines an interface to create/update CronJobs
type CronJobCreator = func(data *TemplateData, existing *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error)

// CRDCreateor defines an interface to create/update CustomRessourceDefinitions
type CRDCreateor = func(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error)

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

// UserClusterDNSPolicyAndConfig returns a DNSPolicy and DNSConfig to configure Pods to use user cluster DNS
func UserClusterDNSPolicyAndConfig(d DeploymentDataProvider) (corev1.DNSPolicy, *corev1.PodDNSConfig, error) {
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
func IsServerCertificateValidForAllOf(cert *x509.Certificate, commonName string, altNames certutil.AltNames) bool {
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

	return wantDNSNames.Equal(certDNSNames)
}

// IsClientCertificateValidForAllOf validates if the given data matches exactly the given client certificate
// (It also returns true if all given data is in the cert, but the cert has more organizations)
func IsClientCertificateValidForAllOf(cert *x509.Certificate, commonName string, organizations []string) bool {
	if CertWillExpireSoon(cert) {
		return false
	}

	if cert.Subject.CommonName != commonName {
		return false
	}

	wantOrganizations := sets.NewString(organizations...)
	certOrganizations := sets.NewString(cert.Subject.Organization...)

	return wantOrganizations.Equal(certOrganizations)
}

// getClusterCAFromLister returns the CA of the cluster from the lister
func getClusterCAFromLister(name string, cluster *kubermaticv1.Cluster, lister corev1lister.SecretLister) (*triple.KeyPair, error) {
	caCertSecret, err := lister.Secrets(cluster.Status.NamespaceName).Get(name)
	if err != nil {
		return nil, fmt.Errorf("unable to check if a CA cert already exists: %v", err)
	}

	certs, err := certutil.ParseCertsPEM(caCertSecret.Data[CACertSecretKey])
	if err != nil {
		return nil, fmt.Errorf("got an invalid cert from the CA secret %s: %v", CASecretName, err)
	}

	key, err := certutil.ParsePrivateKeyPEM(caCertSecret.Data[CAKeySecretKey])
	if err != nil {
		return nil, fmt.Errorf("got an invalid private key from the CA secret %s: %v", CASecretName, err)
	}

	return &triple.KeyPair{
		Cert: certs[0],
		Key:  key.(*rsa.PrivateKey),
	}, nil
}

// GetClusterRootCA returns the root CA of the cluster from the lister
func GetClusterRootCA(cluster *kubermaticv1.Cluster, lister corev1lister.SecretLister) (*triple.KeyPair, error) {
	return getClusterCAFromLister(CASecretName, cluster, lister)
}

// GetClusterFrontProxyCA returns the frontproxy CA of the cluster from the lister
func GetClusterFrontProxyCA(cluster *kubermaticv1.Cluster, lister corev1lister.SecretLister) (*triple.KeyPair, error) {
	return getClusterCAFromLister(FrontProxyCASecretName, cluster, lister)
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

// DeepEqual compares both objects for equality
func DeepEqual(a, b metav1.Object) bool {
	//TODO: Check why equality.Semantic.DeepEqual returns a different result than deep.Equal
	// Reproducible by changing the code to use equality.Semantic.DeepEqual & create a cluster.
	// The ensureDeployments & ensureStatefulSets function in the cluster controller will update the resources on each sync
	diff := deep.Equal(a, b)
	if diff == nil {
		return true
	}

	glog.V(8).Infof("Object %T %s/%s differs from the one, generated: %v", a, a.GetNamespace(), a.GetName(), diff)
	return false
}
