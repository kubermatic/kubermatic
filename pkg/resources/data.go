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
	"crypto/x509"
	"fmt"
	"net"
	"strings"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TemplateData is a group of data required for template generation
type TemplateData struct {
	ctx                                              context.Context
	client                                           ctrlruntimeclient.Client
	cluster                                          *kubermaticv1.Cluster
	dc                                               *kubermaticv1.Datacenter
	seed                                             *kubermaticv1.Seed
	OverwriteRegistry                                string
	nodePortRange                                    string
	nodeAccessNetwork                                string
	etcdDiskSize                                     resource.Quantity
	monitoringScrapeAnnotationPrefix                 string
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
	oidcCAFile                                       string
	oidcIssuerURL                                    string
	oidcIssuerClientID                               string
	nodeLocalDNSCacheEnabled                         bool
	kubermaticImage                                  string
	etcdLauncherImage                                string
	dnatControllerImage                              string
	supportsFailureDomainZoneAntiAffinity            bool
}

// NewTemplateData returns an instance of TemplateData
func NewTemplateData(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	cluster *kubermaticv1.Cluster,
	dc *kubermaticv1.Datacenter,
	seed *kubermaticv1.Seed,
	overwriteRegistry string,
	nodePortRange string,
	nodeAccessNetwork string,
	etcdDiskSize resource.Quantity,
	monitoringScrapeAnnotationPrefix string,
	inClusterPrometheusRulesFile string,
	inClusterPrometheusDisableDefaultRules bool,
	inClusterPrometheusDisableDefaultScrapingConfigs bool,
	inClusterPrometheusScrapingConfigsFile string,
	oidcCAFile string,
	oidcURL string,
	oidcIssuerClientID string,
	nodeLocalDNSCacheEnabled bool,
	kubermaticImage string,
	etcdLauncherImage string,
	dnatControllerImage string,
	supportsFailureDomainZoneAntiAffinity bool) *TemplateData {
	return &TemplateData{
		ctx:                                    ctx,
		client:                                 client,
		cluster:                                cluster,
		dc:                                     dc,
		seed:                                   seed,
		OverwriteRegistry:                      overwriteRegistry,
		nodePortRange:                          nodePortRange,
		nodeAccessNetwork:                      nodeAccessNetwork,
		etcdDiskSize:                           etcdDiskSize,
		monitoringScrapeAnnotationPrefix:       monitoringScrapeAnnotationPrefix,
		inClusterPrometheusRulesFile:           inClusterPrometheusRulesFile,
		inClusterPrometheusDisableDefaultRules: inClusterPrometheusDisableDefaultRules,
		inClusterPrometheusDisableDefaultScrapingConfigs: inClusterPrometheusDisableDefaultScrapingConfigs,
		inClusterPrometheusScrapingConfigsFile:           inClusterPrometheusScrapingConfigsFile,
		oidcCAFile:                                       oidcCAFile,
		oidcIssuerURL:                                    oidcURL,
		oidcIssuerClientID:                               oidcIssuerClientID,
		nodeLocalDNSCacheEnabled:                         nodeLocalDNSCacheEnabled,
		kubermaticImage:                                  kubermaticImage,
		etcdLauncherImage:                                etcdLauncherImage,
		dnatControllerImage:                              dnatControllerImage,
		supportsFailureDomainZoneAntiAffinity:            supportsFailureDomainZoneAntiAffinity,
	}
}

// GetViewerToken returns the viewer token
func (d *TemplateData) GetViewerToken() (string, error) {
	viewerTokenSecret := &corev1.Secret{}
	if err := d.client.Get(d.ctx, ctrlruntimeclient.ObjectKey{Name: ViewerTokenSecretName, Namespace: d.cluster.Status.NamespaceName}, viewerTokenSecret); err != nil {
		return "", err
	}
	return string(viewerTokenSecret.Data[ViewerTokenSecretKey]), nil
}

// GetDexCA returns the chain of public certificates of the Dex
func (d *TemplateData) GetDexCA() ([]*x509.Certificate, error) {
	return GetDexCAFromFile(d.oidcCAFile)
}

// OIDCCAFile return CA file
func (d *TemplateData) OIDCCAFile() string {
	return d.oidcCAFile
}

// OIDCIssuerURL returns URL of the OpenID token issuer
func (d *TemplateData) OIDCIssuerURL() string {
	return d.oidcIssuerURL
}

// OIDCIssuerClientID return the issuer client ID
func (d *TemplateData) OIDCIssuerClientID() string {
	return d.oidcIssuerClientID
}

// Cluster returns the cluster
func (d *TemplateData) Cluster() *kubermaticv1.Cluster {
	return d.cluster
}

// ClusterVersion returns version of the cluster
func (d *TemplateData) ClusterVersion() string {
	return d.cluster.Spec.Version.String()
}

// DC returns the dc
func (d *TemplateData) DC() *kubermaticv1.Datacenter {
	return d.dc
}

// EtcdDiskSize returns the etcd disk size
func (d *TemplateData) EtcdDiskSize() resource.Quantity {
	return d.etcdDiskSize
}

// EtcdLauncherImage returns the etcd launcher image and tag.
func (d *TemplateData) EtcdLauncherImage() (string, string) {
	imageSplit := strings.Split(d.etcdLauncherImage, "/")
	var registry, imageWithoutRegistry string
	if len(imageSplit) != 3 {
		registry = RegistryDocker
		imageWithoutRegistry = strings.Join(imageSplit, "/")
	} else {
		registry = imageSplit[0]
		imageWithoutRegistry = strings.Join(imageSplit[1:], "/")
	}
	s := strings.Split(imageWithoutRegistry, ":")
	if len(s) == 2 {
		return fmt.Sprint(d.ImageRegistry(registry), "/", s[0]), s[1]
	}
	return fmt.Sprint(d.ImageRegistry(registry), "/", imageWithoutRegistry), ""
}

// MonitoringScrapeAnnotationPrefix returns the scrape annotation prefix
func (d *TemplateData) MonitoringScrapeAnnotationPrefix() string {
	return strings.NewReplacer(".", "_", "/", "").Replace(d.monitoringScrapeAnnotationPrefix)
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
	service := &corev1.Service{}
	key := types.NamespacedName{Namespace: d.cluster.Status.NamespaceName, Name: name}
	if err := d.client.Get(d.ctx, key, service); err != nil {
		return "", fmt.Errorf("could not get service %s: %v", key, err)
	}

	if net.ParseIP(service.Spec.ClusterIP) == nil {
		return "", fmt.Errorf("service %s has no valid cluster ip (\"%s\")", key, service.Spec.ClusterIP)
	}
	return service.Spec.ClusterIP, nil
}

// ProviderName returns the name of the clusters providerName
func (d *TemplateData) ProviderName() string {
	p, err := provider.ClusterCloudProviderName(d.cluster.Spec.Cloud)
	if err != nil {
		klog.Errorf("could not identify cloud provider: %v", err)
	}
	return p
}

// ImageRegistry returns the image registry to use or the passed in default if no override is specified
func (d *TemplateData) ImageRegistry(registry string) string {
	if d.OverwriteRegistry != "" {
		return d.OverwriteRegistry
	}
	return registry
}

// GetRootCA returns the root CA of the cluster
func (d *TemplateData) GetRootCA() (*triple.KeyPair, error) {
	return GetClusterRootCA(d.ctx, d.cluster.Status.NamespaceName, d.client)
}

// GetFrontProxyCA returns the root CA for the front proxy
func (d *TemplateData) GetFrontProxyCA() (*triple.KeyPair, error) {
	return GetClusterFrontProxyCA(d.ctx, d.cluster.Status.NamespaceName, d.client)
}

// GetOpenVPNCA returns the root ca for the OpenVPN
func (d *TemplateData) GetOpenVPNCA() (*ECDSAKeyPair, error) {
	return GetOpenVPNCA(d.ctx, d.cluster.Status.NamespaceName, d.client)
}

// GetPodTemplateLabels returns a set of labels for a Pod including the revisions of depending secrets and configmaps.
// This will force pods being restarted as soon as one of the secrets/configmaps get updated.
func (d *TemplateData) GetPodTemplateLabels(appName string, volumes []corev1.Volume, additionalLabels map[string]string) (map[string]string, error) {
	return GetPodTemplateLabels(d.ctx, d.client, appName, d.cluster.Name, d.cluster.Status.NamespaceName, volumes, additionalLabels)
}

// GetApiserverExternalNodePort returns the nodeport of the external apiserver service
func (d *TemplateData) GetOpenVPNServerPort() (int32, error) {
	service := &corev1.Service{}
	key := types.NamespacedName{Namespace: d.cluster.Status.NamespaceName, Name: OpenVPNServerServiceName}
	if err := d.client.Get(d.ctx, key, service); err != nil {
		return 0, fmt.Errorf("failed to get NodePort for openvpn server service: %v", err)
	}

	return service.Spec.Ports[0].NodePort, nil
}

func (d *TemplateData) NodeLocalDNSCacheEnabled() bool {
	return d.nodeLocalDNSCacheEnabled
}

func (d *TemplateData) KubermaticAPIImage() string {
	apiImageSplit := strings.Split(d.kubermaticImage, "/")
	var registry, imageWithoutRegistry string
	if len(apiImageSplit) != 3 {
		registry = RegistryDocker
		imageWithoutRegistry = strings.Join(apiImageSplit, "/")
	} else {
		registry = apiImageSplit[0]
		imageWithoutRegistry = strings.Join(apiImageSplit[1:], "/")
	}
	return d.ImageRegistry(registry) + "/" + imageWithoutRegistry
}

func (d *TemplateData) DNATControllerImage() string {
	dnatControllerImageSplit := strings.Split(d.dnatControllerImage, "/")
	var registry, imageWithoutRegistry string
	if len(dnatControllerImageSplit) != 3 {
		registry = RegistryDocker
		imageWithoutRegistry = strings.Join(dnatControllerImageSplit, "/")
	} else {
		registry = dnatControllerImageSplit[0]
		imageWithoutRegistry = strings.Join(dnatControllerImageSplit[1:], "/")
	}
	return d.ImageRegistry(registry) + "/" + imageWithoutRegistry
}

func (d *TemplateData) SupportsFailureDomainZoneAntiAffinity() bool {
	return d.supportsFailureDomainZoneAntiAffinity
}

func (d *TemplateData) GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return provider.SecretKeySelectorValueFuncFactory(d.ctx, d.client)(configVar, key)
}

func (d *TemplateData) GetKubernetesCloudProviderName() string {
	return GetKubernetesCloudProviderName(d.Cluster())
}

func (d *TemplateData) CloudCredentialSecretTemplate() ([]byte, error) {
	// This is needed for Openshift only
	return nil, nil
}

func GetKubernetesCloudProviderName(cluster *kubermaticv1.Cluster) string {
	switch {
	case cluster.Spec.Cloud.AWS != nil:
		return "aws"
	case cluster.Spec.Cloud.VSphere != nil:
		return "vsphere"
	case cluster.Spec.Cloud.Azure != nil:
		return "azure"
	case cluster.Spec.Cloud.GCP != nil:
		return "gce"
	case cluster.Spec.Cloud.Openstack != nil:
		if cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
			return "external"
		}
		return "openstack"
	default:
		return ""
	}
}

func (d *TemplateData) Seed() *kubermaticv1.Seed {
	return d.seed
}
