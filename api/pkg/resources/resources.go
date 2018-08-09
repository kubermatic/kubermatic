package resources

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	corev1lister "k8s.io/client-go/listers/core/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

const (
	//ApiserverDeploymentName is the name for the apiserver deployment
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
	//EtcdClientServiceName is the name for the etcd service for clients (ClusterIP)
	EtcdClientServiceName = "etcd-client"
	//OpenVPNServerServiceName is the name for the openvpn server service
	OpenVPNServerServiceName = "openvpn-server"

	//AdminKubeconfigSecretName is the name for the secret containing the private ca key
	AdminKubeconfigSecretName = "admin-kubeconfig"
	//SchedulerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the scheduler
	SchedulerKubeconfigSecretName = "scheduler-kubeconfig"
	//KubeStateMetricsKubeconfigSecretName is the name for the secret containing the kubeconfig used by kube-state-metrics
	KubeStateMetricsKubeconfigSecretName = "kube-state-metrics-kubeconfig"
	//MachineControllerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the scheduler
	MachineControllerKubeconfigSecretName = "machinecontroller-kubeconfig"
	//ControllerManagerKubeconfigSecretName is the name for the secret containing the kubeconfig used by the scheduler
	ControllerManagerKubeconfigSecretName = "controllermanager-kubeconfig"

	//CASecretName is the name for the secret containing the root ca key
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
	//ApiserverProxyClientCertificateSecretName is the name for the secret containing the apiserver's client certificate for proxy auth
	ApiserverProxyClientCertificateSecretName = "apiserver-proxy-client-certificate"

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

	// RegistryKubernetesGCR defines the kubernetes docker registry at google
	RegistryKubernetesGCR = "gcr.io"
	// RegistryDocker defines the default docker.io registry
	RegistryDocker = "docker.io"
	// RegistryQuay defines the image registry from coreos/redhat - quay
	RegistryQuay = "quay.io"
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
	// AdminKubeconfigSecretKey admin-kubeconfig
	AdminKubeconfigSecretKey = "admin-kubeconfig"
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
type ConfigMapCreator = func(data *TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error)

// SecretCreator defines an interface to create/update Secret's
type SecretCreator = func(data *TemplateData, existing *corev1.Secret) (*corev1.Secret, error)

// StatefulSetCreator defines an interface to create/update StatefulSet
type StatefulSetCreator = func(data *TemplateData, existing *appsv1.StatefulSet) (*appsv1.StatefulSet, error)

// ServiceCreator defines an interface to create/update Services
type ServiceCreator = func(data *TemplateData, existing *corev1.Service) (*corev1.Service, error)

// RoleCreator defines an interface to create/update RBAC Roles
type RoleCreator = func(data *TemplateData, existing *rbacv1.Role) (*rbacv1.Role, error)

// RoleBindingCreator defines an interface to create/update RBAC RoleBinding's
type RoleBindingCreator = func(data *TemplateData, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)

// ClusterRoleCreator defines an interface to create/update RBAC ClusterRoles
type ClusterRoleCreator = func(data *TemplateData, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error)

// ClusterRoleBindingCreator defines an interface to create/update RBAC ClusterRoleBinding's
type ClusterRoleBindingCreator = func(data *TemplateData, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error)

// DeploymentCreator defines an interface to create/update Deployment's
type DeploymentCreator = func(data *TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error)

// PodDisruptionBudgetCreator defines an interface to create/update PodDisruptionBudgets's
type PodDisruptionBudgetCreator = func(data *TemplateData, existing *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error)

// TemplateData is a group of data required for template generation
type TemplateData struct {
	Cluster           *kubermaticv1.Cluster
	DC                *provider.DatacenterMeta
	SecretLister      corev1lister.SecretLister
	ConfigMapLister   corev1lister.ConfigMapLister
	ServiceLister     corev1lister.ServiceLister
	OverwriteRegistry string
	NodePortRange     string
	NodeAccessNetwork string
	EtcdDiskSize      resource.Quantity
}

// GetClusterRef returns a instance of a OwnerReference for the Cluster in the TemplateData
func (d *TemplateData) GetClusterRef() metav1.OwnerReference {
	return GetClusterRef(d.Cluster)
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

// NewTemplateData returns an instance of TemplateData
func NewTemplateData(
	cluster *kubermaticv1.Cluster,
	dc *provider.DatacenterMeta,
	secretLister corev1lister.SecretLister,
	configMapLister corev1lister.ConfigMapLister,
	serviceLister corev1lister.ServiceLister,
	overwriteRegistry string,
	nodePortRange string,
	nodeAccessNetwork string,
	etcdDiskSize resource.Quantity) *TemplateData {
	return &TemplateData{
		Cluster:           cluster,
		DC:                dc,
		ConfigMapLister:   configMapLister,
		SecretLister:      secretLister,
		ServiceLister:     serviceLister,
		OverwriteRegistry: overwriteRegistry,
		NodePortRange:     nodePortRange,
		NodeAccessNetwork: nodeAccessNetwork,
		EtcdDiskSize:      etcdDiskSize,
	}
}

// ClusterIPByServiceName returns the ClusterIP as string for the
// Service specified by `name`. Service lookup happens within
// `Cluster.Status.NamespaceName`. When ClusterIP fails to parse
// as valid IP address, an error is returned.
func (d *TemplateData) ClusterIPByServiceName(name string) (string, error) {
	service, err := d.ServiceLister.Services(d.Cluster.Status.NamespaceName).Get(name)
	if err != nil {
		return "", fmt.Errorf("could not get service %s from lister for cluster %s: %v", name, d.Cluster.Name, err)
	}
	if net.ParseIP(service.Spec.ClusterIP) == nil {
		return "", fmt.Errorf("service %s in cluster %s has no valid cluster ip (\"%s\"): %v", name, d.Cluster.Name, service.Spec.ClusterIP, err)
	}
	return service.Spec.ClusterIP, nil
}

// SecretRevision returns the resource version of the secret specified by name. A empty string will be returned in case of an error
func (d *TemplateData) SecretRevision(name string) (string, error) {
	secret, err := d.SecretLister.Secrets(d.Cluster.Status.NamespaceName).Get(name)
	if err != nil {
		return "", fmt.Errorf("could not get secret %s from lister for cluster %s: %v", name, d.Cluster.Name, err)
	}
	return secret.ResourceVersion, nil
}

// ConfigMapRevision returns the resource version of the configmap specified by name. A empty string will be returned in case of an error
func (d *TemplateData) ConfigMapRevision(name string) (string, error) {
	cm, err := d.ConfigMapLister.ConfigMaps(d.Cluster.Status.NamespaceName).Get(name)
	if err != nil {
		return "", fmt.Errorf("could not get configmap %s from lister for cluster %s: %v", name, d.Cluster.Name, err)
	}
	return cm.ResourceVersion, nil
}

// ProviderName returns the name of the clusters providerName
func (d *TemplateData) ProviderName() string {
	p, err := provider.ClusterCloudProviderName(d.Cluster.Spec.Cloud)
	if err != nil {
		glog.V(0).Infof("could not identify cloud provider: %v", err)
	}
	return p
}

// GetApiserverExternalNodePort returns the nodeport of the external apiserver service
func (d *TemplateData) GetApiserverExternalNodePort() (int32, error) {
	s, err := d.ServiceLister.Services(d.Cluster.Status.NamespaceName).Get(ApiserverExternalServiceName)
	if err != nil {

		return 0, fmt.Errorf("failed to get NodePort for external apiserver service: %v", err)

	}
	return s.Spec.Ports[0].NodePort, nil
}

// InClusterApiserverAddress takes the ClusterIP and node-port of the external/secure apiserver service
// and returns them joined by a `:`.
// Service lookup happens within `Cluster.Status.NamespaceName`.
func (d *TemplateData) InClusterApiserverAddress() (string, error) {
	service, err := d.ServiceLister.Services(d.Cluster.Status.NamespaceName).Get(ApiserverExternalServiceName)
	if err != nil {
		return "", fmt.Errorf("could not get service %s from lister for cluster %s: %v", ApiserverExternalServiceName, d.Cluster.Name, err)
	}
	if net.ParseIP(service.Spec.ClusterIP) == nil {
		return "", fmt.Errorf("service %s in cluster %s has no valid cluster ip (\"%s\"): %v", ApiserverExternalServiceName, d.Cluster.Name, service.Spec.ClusterIP, err)
	}
	return fmt.Sprintf("%s:%d", service.Spec.ClusterIP, service.Spec.Ports[0].NodePort), nil
}

// ImageRegistry returns the image registry to use or the passed in default if no override is specified
func (d *TemplateData) ImageRegistry(defaultRegistry string) string {
	if d.OverwriteRegistry != "" {
		return d.OverwriteRegistry
	}
	return defaultRegistry
}

// GetClusterCA returns the root CA of the cluster
func (d *TemplateData) GetClusterCA() (*triple.KeyPair, error) {
	return GetClusterCAFromLister(d.Cluster, d.SecretLister)
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

// UserClusterDNSPolicyAndConfig returns a DNSPolicy and DNSConfig to configure Pods to use user cluster DNS
func UserClusterDNSPolicyAndConfig(d *TemplateData) (corev1.DNSPolicy, *corev1.PodDNSConfig, error) {
	// DNSNone indicates that the pod should use empty DNS settings. DNS
	// parameters such as nameservers and search paths should be defined via
	// DNSConfig.
	dnsConfigOptionNdots := "5"
	dnsConfigResolverIP, err := d.ClusterIPByServiceName(DNSResolverServiceName)
	if err != nil {
		return corev1.DNSNone, nil, err
	}
	if len(d.Cluster.Spec.ClusterNetwork.DNSDomain) == 0 {
		return corev1.DNSNone, nil, fmt.Errorf("invalid (empty) DNSDomain in ClusterNetwork spec for cluster %s", d.Cluster.Name)
	}
	return corev1.DNSNone, &corev1.PodDNSConfig{
		Nameservers: []string{dnsConfigResolverIP},
		Searches: []string{
			fmt.Sprintf("kube-system.svc.%s", d.Cluster.Spec.ClusterNetwork.DNSDomain),
			fmt.Sprintf("svc.%s", d.Cluster.Spec.ClusterNetwork.DNSDomain),
			d.Cluster.Spec.ClusterNetwork.DNSDomain,
		},
		Options: []corev1.PodDNSConfigOption{
			{
				Name:  "ndots",
				Value: &dnsConfigOptionNdots,
			},
		},
	}, nil
}

// GetPodTemplateLabels returns a set of labels for a Pod including the revisions of depending secrets and configmaps.
// This will force pods being restarted as soon as one of the secrets/configmaps get updated.
func (d *TemplateData) GetPodTemplateLabels(name string, volumes []corev1.Volume, additionalLabels map[string]string) (map[string]string, error) {
	podLabels := BaseAppLabel(name, additionalLabels)

	for _, v := range volumes {
		if v.VolumeSource.Secret != nil {
			revision, err := d.SecretRevision(v.VolumeSource.Secret.SecretName)
			if err != nil {
				return nil, err
			}
			podLabels[fmt.Sprintf("%s-secret-revision", v.VolumeSource.Secret.SecretName)] = revision
		}
		if v.VolumeSource.ConfigMap != nil {
			revision, err := d.ConfigMapRevision(v.VolumeSource.ConfigMap.Name)
			if err != nil {
				return nil, err
			}
			podLabels[fmt.Sprintf("%s-configmap-revision", v.VolumeSource.ConfigMap.Name)] = revision
		}
	}

	return podLabels, nil
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

// CertWillExpireSoon returns if the certificate will expire in the next 30 days
func CertWillExpireSoon(cert *x509.Certificate) bool {
	return time.Until(cert.NotAfter) < minimumCertValidity30d
}

// IsServerCertificateValidForAllOf validates if the given data is present in the given server certificate
func IsServerCertificateValidForAllOf(cert *x509.Certificate, commonName, svcName, svcNamespace, dnsDomain string, ips, hostnames []string) bool {
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
	wantIPs := sets.NewString(ips...)

	if !wantIPs.Equal(certIPs) {
		return false
	}

	wantDNSNames := sets.NewString(svcName, svcName+"."+svcNamespace, svcName+"."+svcNamespace+".svc", svcName+"."+svcNamespace+".svc."+dnsDomain)
	wantDNSNames.Insert(hostnames...)
	certDNSNames := sets.NewString(cert.DNSNames...)

	return wantDNSNames.Equal(certDNSNames)
}

// IsClientCertificateValidForAllOf validates if the given data is present in the given client certificate
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

// GetClusterCAFromLister returns the root CA of the cluster from the lister
func GetClusterCAFromLister(cluster *kubermaticv1.Cluster, lister corev1lister.SecretLister) (*triple.KeyPair, error) {
	caCertSecret, err := lister.Secrets(cluster.Status.NamespaceName).Get(CASecretName)
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
