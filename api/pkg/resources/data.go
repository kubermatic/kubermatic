package resources

import (
	"fmt"
	"net"
	"net/url"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/cert/triple"
)

// TemplateData is a group of data required for template generation
type TemplateData struct {
	cluster                                          *kubermaticv1.Cluster
	dC                                               *provider.DatacenterMeta
	SeedDC                                           string
	SecretLister                                     corev1lister.SecretLister
	configMapLister                                  corev1lister.ConfigMapLister
	serviceLister                                    corev1lister.ServiceLister
	OverwriteRegistry                                string
	nodePortRange                                    string
	nodeAccessNetwork                                string
	etcdDiskSize                                     resource.Quantity
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
	DockerPullConfigJSON                             []byte
}

// TemplateData returns data for templating
func (d *TemplateData) TemplateData() interface{} {
	return d
}

// UserClusterData groups data required for resource creation in user-cluster
type UserClusterData struct {
	configMapLister corev1lister.ConfigMapLister
	serviceLister   corev1lister.ServiceLister
}

// RoleDataProvider provides data
type RoleDataProvider interface {
	GetClusterRef() metav1.OwnerReference
}

// RoleBindingDataProvider provides data
type RoleBindingDataProvider interface {
	GetClusterRef() metav1.OwnerReference
	Cluster() *kubermaticv1.Cluster
}

// ClusterRoleDataProvider provides data
type ClusterRoleDataProvider interface {
	GetClusterRef() metav1.OwnerReference
}

// ClusterRoleBindingDataProvider provides data
type ClusterRoleBindingDataProvider interface {
	GetClusterRef() metav1.OwnerReference
}

// ServiceAccountDataProvider provides data
type ServiceAccountDataProvider interface {
	GetClusterRef() metav1.OwnerReference
}

// ConfigMapDataProvider provides data
type ConfigMapDataProvider interface {
	GetClusterRef() metav1.OwnerReference
	Cluster() *kubermaticv1.Cluster
	TemplateData() interface{}
	ServiceLister() corev1lister.ServiceLister
	NodeAccessNetwork() string
	InClusterPrometheusRulesFile() string
	InClusterPrometheusScrapingConfigsFile() string
	InClusterPrometheusDisableDefaultRules() bool
	InClusterPrometheusDisableDefaultScrapingConfigs() bool
}

// SecretDataProvider provides data
type SecretDataProvider interface {
	GetClusterRef() metav1.OwnerReference
	InClusterApiserverURL() (*url.URL, error)
	GetFrontProxyCA() (*triple.KeyPair, error)
	GetRootCA() (*triple.KeyPair, error)
	GetOpenVPNCA() (*ECDSAKeyPair, error)
	ExternalIP() (*net.IP, error)
	Cluster() *kubermaticv1.Cluster
}

// ServiceDataProvider provides data
type ServiceDataProvider interface {
	GetClusterRef() metav1.OwnerReference
	Cluster() *kubermaticv1.Cluster
}

// DeploymentDataProvider provides data
type DeploymentDataProvider interface {
	GetClusterRef() metav1.OwnerReference
	ClusterIPByServiceName(name string) (string, error)
	GetApiserverExternalNodePort() (int32, error)
	EtcdDiskSize() resource.Quantity
	NodeAccessNetwork() string
	NodePortRange() string
	ServiceLister() corev1lister.ServiceLister
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	InClusterApiserverURL() (*url.URL, error)
	ImageRegistry(string) string
	Cluster() *kubermaticv1.Cluster
	DC() *provider.DatacenterMeta
}

// StatefulSetDataProvider provides data
type StatefulSetDataProvider interface {
	DeploymentDataProvider
}

// NewTemplateData returns an instance of TemplateData
func NewTemplateData(
	cluster *kubermaticv1.Cluster,
	dc *provider.DatacenterMeta,
	seedDatacenter string,
	secretLister corev1lister.SecretLister,
	configMapLister corev1lister.ConfigMapLister,
	serviceLister corev1lister.ServiceLister,
	overwriteRegistry string,
	nodePortRange string,
	nodeAccessNetwork string,
	etcdDiskSize resource.Quantity,
	inClusterPrometheusRulesFile string,
	inClusterPrometheusDisableDefaultRules bool,
	inClusterPrometheusDisableDefaultScrapingConfigs bool,
	inClusterPrometheusScrapingConfigsFile string,
	dockerPullConfigJSON []byte) *TemplateData {
	return &TemplateData{
		cluster:                                cluster,
		dC:                                     dc,
		SeedDC:                                 seedDatacenter,
		configMapLister:                        configMapLister,
		SecretLister:                           secretLister,
		serviceLister:                          serviceLister,
		OverwriteRegistry:                      overwriteRegistry,
		nodePortRange:                          nodePortRange,
		nodeAccessNetwork:                      nodeAccessNetwork,
		etcdDiskSize:                           etcdDiskSize,
		inClusterPrometheusRulesFile:           inClusterPrometheusRulesFile,
		inClusterPrometheusDisableDefaultRules: inClusterPrometheusDisableDefaultRules,
		inClusterPrometheusDisableDefaultScrapingConfigs: inClusterPrometheusDisableDefaultScrapingConfigs,
		inClusterPrometheusScrapingConfigsFile:           inClusterPrometheusScrapingConfigsFile,
		DockerPullConfigJSON:                             dockerPullConfigJSON,
	}
}

// Cluster returns the cluster
func (d *TemplateData) Cluster() *kubermaticv1.Cluster {
	return d.cluster
}

// DC returns the dC
func (d *TemplateData) DC() *provider.DatacenterMeta {
	return d.dC
}

// ServiceLister returns the serviceLister
func (d *TemplateData) ServiceLister() corev1lister.ServiceLister {
	return d.serviceLister
}

// ConfigMapLister returns the configMapLister
func (d *TemplateData) ConfigMapLister() corev1lister.ConfigMapLister {
	return d.configMapLister
}

// EtcdDiskSize returns the etcd disk size
func (d *TemplateData) EtcdDiskSize() resource.Quantity {
	return d.etcdDiskSize
}

// InClusterPrometheusRulesFile returns inClusterPrometheusRulesFile
func (d *TemplateData) InClusterPrometheusRulesFile() string {
	return d.inClusterPrometheusRulesFile
}

// InClusterPrometheusDisableDefaultRules returns whether to disable default rules
func (d *TemplateData) InClusterPrometheusDisableDefaultRules() bool {
	return d.inClusterPrometheusDisableDefaultRules
}

// InClusterPrometheusDisableDefaultScrapingConfigs returns whether to disable default scrape configs
func (d *TemplateData) InClusterPrometheusDisableDefaultScrapingConfigs() bool {
	return d.inClusterPrometheusDisableDefaultScrapingConfigs
}

// InClusterPrometheusScrapingConfigsFile returns inClusterPrometheusScrapingConfigsFile
func (d *TemplateData) InClusterPrometheusScrapingConfigsFile() string {
	return d.inClusterPrometheusScrapingConfigsFile
}

// NodeAccessNetwork returns the node access network
func (d *TemplateData) NodeAccessNetwork() string {
	return d.nodeAccessNetwork
}

// NodePortRange returns the node access network
func (d *TemplateData) NodePortRange() string {
	return d.nodePortRange
}

// NewUserClusterData returns an instance of UserClusterData
func NewUserClusterData(
	configMapLister corev1lister.ConfigMapLister,
	serviceLister corev1lister.ServiceLister) *UserClusterData {
	return &UserClusterData{
		configMapLister: configMapLister,
		serviceLister:   serviceLister,
	}
}

// GetClusterRef returns a instance of a OwnerReference for the Cluster in the TemplateData
func (d *TemplateData) GetClusterRef() metav1.OwnerReference {
	return GetClusterRef(d.cluster)
}

// ExternalIP returns the external facing IP or an error if no IP exists
func (d *TemplateData) ExternalIP() (*net.IP, error) {
	return GetClusterExternalIP(d.cluster)
}

// ClusterIPByServiceName returns the ClusterIP as string for the
// Service specified by `name`. Service lookup happens within
// `Cluster.Status.NamespaceName`. When ClusterIP fails to parse
// as valid IP address, an error is returned.
func (d *TemplateData) ClusterIPByServiceName(name string) (string, error) {
	service, err := d.serviceLister.Services(d.cluster.Status.NamespaceName).Get(name)
	if err != nil {
		return "", fmt.Errorf("could not get service %s from lister for cluster %s: %v", name, d.cluster.Name, err)
	}
	if net.ParseIP(service.Spec.ClusterIP) == nil {
		return "", fmt.Errorf("service %s in cluster %s has no valid cluster ip (\"%s\"): %v", name, d.cluster.Name, service.Spec.ClusterIP, err)
	}
	return service.Spec.ClusterIP, nil
}

// ProviderName returns the name of the clusters providerName
func (d *TemplateData) ProviderName() string {
	p, err := provider.ClusterCloudProviderName(d.cluster.Spec.Cloud)
	if err != nil {
		glog.V(0).Infof("could not identify cloud provider: %v", err)
	}
	return p
}

// GetApiserverExternalNodePort returns the nodeport of the external apiserver service
func (d *TemplateData) GetApiserverExternalNodePort() (int32, error) {
	s, err := d.serviceLister.Services(d.cluster.Status.NamespaceName).Get(ApiserverExternalServiceName)
	if err != nil {

		return 0, fmt.Errorf("failed to get NodePort for external apiserver service: %v", err)

	}
	return s.Spec.Ports[0].NodePort, nil
}

// InClusterApiserverURL takes the ClusterIP and node-port of the external/secure apiserver service
// and returns them joined by a `:`.
// Service lookup happens within `Cluster.Status.NamespaceName`.
func (d *TemplateData) InClusterApiserverURL() (*url.URL, error) {
	return GetClusterApiserverURL(d.cluster, d.serviceLister)
}

// ImageRegistry returns the image registry to use or the passed in default if no override is specified
func (d *TemplateData) ImageRegistry(defaultRegistry string) string {
	if d.OverwriteRegistry != "" {
		return d.OverwriteRegistry
	}
	return defaultRegistry
}

// GetRootCA returns the root CA of the cluster
func (d *TemplateData) GetRootCA() (*triple.KeyPair, error) {
	return GetClusterRootCA(d.cluster, d.SecretLister)
}

// GetFrontProxyCA returns the root CA for the front proxy
func (d *TemplateData) GetFrontProxyCA() (*triple.KeyPair, error) {
	return GetClusterFrontProxyCA(d.cluster, d.SecretLister)
}

// GetOpenVPNCA returns the root ca for the OpenVPN
func (d *TemplateData) GetOpenVPNCA() (*ECDSAKeyPair, error) {
	return GetOpenVPNCA(d.cluster, d.SecretLister)
}

// SecretRevision returns the resource version of the secret specified by name. A empty string will be returned in case of an error
func (d *TemplateData) SecretRevision(name string) (string, error) {
	secret, err := d.SecretLister.Secrets(d.cluster.Status.NamespaceName).Get(name)
	if err != nil {
		return "", fmt.Errorf("could not get secret %s from lister for cluster %s: %v", name, d.cluster.Name, err)
	}
	return secret.ResourceVersion, nil
}

// ConfigMapRevision returns the resource version of the configmap specified by name. A empty string will be returned in case of an error
func (d *TemplateData) ConfigMapRevision(name string) (string, error) {
	cm, err := d.ConfigMapLister().ConfigMaps(d.cluster.Status.NamespaceName).Get(name)
	if err != nil {
		return "", fmt.Errorf("could not get configmap %s from lister for cluster %s: %v", name, d.cluster.Name, err)
	}
	return cm.ResourceVersion, nil
}

// GetPodTemplateLabels returns a set of labels for a Pod including the revisions of depending secrets and configmaps.
// This will force pods being restarted as soon as one of the secrets/configmaps get updated.
func (d *TemplateData) GetPodTemplateLabels(appName string, volumes []corev1.Volume, additionalLabels map[string]string) (map[string]string, error) {
	podLabels := AppClusterLabel(appName, d.cluster.Name, additionalLabels)

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

// ServiceLister returns the serviceLister
func (d *UserClusterData) ServiceLister() corev1lister.ServiceLister {
	return d.serviceLister
}

// ConfigMapLister returns the configMapLister
func (d *UserClusterData) ConfigMapLister() corev1lister.ConfigMapLister {
	return d.configMapLister
}

// GetClusterRef panics
func (d *UserClusterData) GetClusterRef() metav1.OwnerReference {
	panic("GetClusterRef not implemented for UserClusterData")
}

// ImageRegistry panics
func (d *UserClusterData) ImageRegistry(defaultRegistry string) string {
	panic("ImageRegistry not implemented for UserClusterData")
}

// GetPodTemplateLabels panics
func (d *UserClusterData) GetPodTemplateLabels(_ string, _ []corev1.Volume, _ map[string]string) (map[string]string, error) {
	panic("GetPodTemplateLabels not implemented for UserClusterData")
}

// NodeAccessNetwork panics
func (d *UserClusterData) NodeAccessNetwork() string {
	panic("NodeAccessNetwork not implemented for UserClusterData")
}

// GetApiserverExternalNodePort panics
func (d *UserClusterData) GetApiserverExternalNodePort() (int32, error) {
	panic("GetApiserverExternalNodePort not implemented for UserClusterData")
}

// ClusterIPByServiceName panics
func (d *UserClusterData) ClusterIPByServiceName(name string) (string, error) {
	panic("ClusterIPByServiceName not implemented for UserClusterData")
}

// InClusterApiserverURL panics
func (d *UserClusterData) InClusterApiserverURL() (*url.URL, error) {
	panic("InClusterApiserverURL not implemented for UserClusterData")
}

// GetFrontProxyCA panics
func (d *UserClusterData) GetFrontProxyCA() (*triple.KeyPair, error) {
	panic("GetFrontProxyCA not implemented for UserClusterData")
}

// ExternalIP panics
func (d *UserClusterData) ExternalIP() (*net.IP, error) {
	panic("ExternalIP not implemented for UserClusterData")
}

// TemplateData returns data for templating
func (d *UserClusterData) TemplateData() interface{} {
	return d
}

// InClusterPrometheusRulesFile panics
func (d *UserClusterData) InClusterPrometheusRulesFile() string {
	panic("InClusterPrometheusRulesFile not implemented for UserClusterData")
}

// InClusterPrometheusScrapingConfigsFile returns inClusterPrometheusScrapingConfigsFile
func (d *UserClusterData) InClusterPrometheusScrapingConfigsFile() string {
	panic("InClusterPrometheusScrapingConfigsFile not implemented for UserClusterData")
}

// InClusterPrometheusDisableDefaultRules panics
func (d *UserClusterData) InClusterPrometheusDisableDefaultRules() bool {
	panic("InClusterPrometheusDisableDefaultRules not implemented for UserClusterData")
}

// InClusterPrometheusDisableDefaultScrapingConfigs panics
func (d *UserClusterData) InClusterPrometheusDisableDefaultScrapingConfigs() bool {
	panic("InClusterPrometheusDisableDefaultScrapingConfigs not implemented for UserClusterData")
}

// Cluster panics
func (d *UserClusterData) Cluster() *kubermaticv1.Cluster {
	panic("Cluster not implemented for UserClusterData")
}
