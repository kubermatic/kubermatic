package resources

import (
	"errors"
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
	Cluster                                          *kubermaticv1.Cluster
	DC                                               *provider.DatacenterMeta
	SeedDC                                           string
	SecretLister                                     corev1lister.SecretLister
	ConfigMapLister                                  corev1lister.ConfigMapLister
	ServiceLister                                    corev1lister.ServiceLister
	OverwriteRegistry                                string
	NodePortRange                                    string
	NodeAccessNetwork                                string
	EtcdDiskSize                                     resource.Quantity
	InClusterPrometheusRulesFile                     string
	InClusterPrometheusDisableDefaultRules           bool
	InClusterPrometheusDisableDefaultScrapingConfigs bool
	InClusterPrometheusScrapingConfigsFile           string
	DockerPullConfigJSON                             []byte
}

// UserClusterData groups data required for resource creation in user-cluster
type UserClusterData struct {
}

// RoleDataProvider provides data about roles
type RoleDataProvider interface {
	GetClusterRef() metav1.OwnerReference
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
		Cluster:                                cluster,
		DC:                                     dc,
		SeedDC:                                 seedDatacenter,
		ConfigMapLister:                        configMapLister,
		SecretLister:                           secretLister,
		ServiceLister:                          serviceLister,
		OverwriteRegistry:                      overwriteRegistry,
		NodePortRange:                          nodePortRange,
		NodeAccessNetwork:                      nodeAccessNetwork,
		EtcdDiskSize:                           etcdDiskSize,
		InClusterPrometheusRulesFile:           inClusterPrometheusRulesFile,
		InClusterPrometheusDisableDefaultRules: inClusterPrometheusDisableDefaultRules,
		InClusterPrometheusDisableDefaultScrapingConfigs: inClusterPrometheusDisableDefaultScrapingConfigs,
		InClusterPrometheusScrapingConfigsFile:           inClusterPrometheusScrapingConfigsFile,
		DockerPullConfigJSON:                             dockerPullConfigJSON,
	}
}

// GetClusterRef returns a instance of a OwnerReference for the Cluster in the TemplateData
func (d *TemplateData) GetClusterRef() metav1.OwnerReference {
	return GetClusterRef(d.Cluster)
}

// ExternalIP returns the external facing IP or an error if no IP exists
func (d *TemplateData) ExternalIP() (*net.IP, error) {
	ip := net.ParseIP(d.Cluster.Address.IP)
	if ip == nil {
		return nil, fmt.Errorf("failed to create a net.IP object from the external cluster IP '%s'", d.Cluster.Address.IP)
	}
	return &ip, nil
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

// InClusterApiserverURL takes the ClusterIP and node-port of the external/secure apiserver service
// and returns them joined by a `:`.
// Service lookup happens within `Cluster.Status.NamespaceName`.
func (d *TemplateData) InClusterApiserverURL() (*url.URL, error) {
	service, err := d.ServiceLister.Services(d.Cluster.Status.NamespaceName).Get(ApiserverExternalServiceName)
	if err != nil {
		return nil, fmt.Errorf("could not get service %s from lister for cluster %s: %v", ApiserverExternalServiceName, d.Cluster.Name, err)
	}

	if len(service.Spec.Ports) != 1 {
		return nil, errors.New("apiserver service does not have exactly one port")
	}

	return url.Parse(fmt.Sprintf("https://%s.%s.svc.cluster.local:%d", ApiserverExternalServiceName, d.Cluster.Status.NamespaceName, service.Spec.Ports[0].NodePort))
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
	return GetClusterRootCA(d.Cluster, d.SecretLister)
}

// GetFrontProxyCA returns the root CA of the cluster
func (d *TemplateData) GetFrontProxyCA() (*triple.KeyPair, error) {
	return GetClusterFrontProxyCA(d.Cluster, d.SecretLister)
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

// GetPodTemplateLabels returns a set of labels for a Pod including the revisions of depending secrets and configmaps.
// This will force pods being restarted as soon as one of the secrets/configmaps get updated.
func (d *TemplateData) GetPodTemplateLabels(appName string, volumes []corev1.Volume, additionalLabels map[string]string) (map[string]string, error) {
	podLabels := AppClusterLabel(appName, d.Cluster.Name, additionalLabels)

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

// GetClusterRef returns a instance of a OwnerReference for the Cluster in the TemplateData
func (d *UserClusterData) GetClusterRef() metav1.OwnerReference {
	panic("GetClusterRef not implemented for UserClusterData")
}
