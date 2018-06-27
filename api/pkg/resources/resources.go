package resources

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
)

const (
	//AddonManagerDeploymentName is the name for the addon-manager deployment
	AddonManagerDeploymentName = "addon-manager"
	//ApiserverDeploymenName is the name for the apiserver deployment
	ApiserverDeploymenName = "apiserver"
	//ControllerManagerDeploymentName is the name for the controller manager deployment
	ControllerManagerDeploymentName = "controller-manager"
	//SchedulerDeploymentName is the name for the scheduler deployment
	SchedulerDeploymentName = "scheduler"
	//MachineControllerDeploymentName is the name for the machine-controller deployment
	MachineControllerDeploymentName = "machine-controller"
	//OpenVPNServerDeploymentName is the name for the openvpn server deployment
	OpenVPNServerDeploymentName = "openvpn-server"

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
	//CAKeySecretName is the name for the secret containing the private ca key
	CAKeySecretName = "ca-key"
	//CACertSecretName is the name for the secret containing the ca.crt
	CACertSecretName = "ca-cert"
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

	//CloudConfigConfigMapName is the name for the configmap containing the cloud-config
	CloudConfigConfigMapName = "cloud-config"
	//OpenVPNClientConfigConfigMapName is the name for the configmap containing the openvpn client config used within the user cluster
	OpenVPNClientConfigConfigMapName = "openvpn-client-configs"
	//PrometheusConfigConfigMapName is the name for the configmap containing the prometheus config
	PrometheusConfigConfigMapName = "prometheus"

	//PrometheusServiceAccountName is the name for the Prometheus serviceaccount
	PrometheusServiceAccountName = "prometheus"

	//PrometheusRoleName is the name for the Prometheus role
	PrometheusRoleName = "prometheus"

	//PrometheusRoleBindingName is the name for the Prometheus rolebinding
	PrometheusRoleBindingName = "prometheus"

	// DefaultOwnerReadOnlyMode represents file mode 0400 in decimal
	DefaultOwnerReadOnlyMode = 256

	// AppLabelKey defines the label key app which should be used within resources
	AppLabelKey = "app"

	// EtcdClusterSize defines the size of the etcd to use
	EtcdClusterSize = 3
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
	KubeletClientCertSecretKey = "kubelet-client.crt"
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

// ClusterRoleBindingCreator defines an interface to create/update RBAC ClusterRoleBinding's
type ClusterRoleBindingCreator = func(data *TemplateData, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error)

// DeploymentCreator defines an interface to create/update Deployment's
type DeploymentCreator = func(data *TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error)

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
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(d.Cluster, gv.WithKind("Cluster"))
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

// ImageRegistry returns the image registry to use or the passed in default if no override is specified
func (d *TemplateData) ImageRegistry(defaultRegistry string) string {
	if d.OverwriteRegistry != "" {
		return d.OverwriteRegistry
	}
	return defaultRegistry
}

// GetLabels returns default labels every resource should have
func GetLabels(app string) map[string]string {
	return map[string]string{AppLabelKey: app}
}
