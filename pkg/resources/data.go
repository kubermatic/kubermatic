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
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	httpproberapi "k8c.io/kubermatic/v2/cmd/http-prober/api"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubenetutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	cloudProviderExternalFlag = "external"
)

type CABundle interface {
	CertPool() *x509.CertPool
	String() string
}

// TemplateData is a group of data required for template generation
type TemplateData struct {
	ctx                      context.Context
	client                   ctrlruntimeclient.Client
	cluster                  *kubermaticv1.Cluster
	dc                       *kubermaticv1.Datacenter
	seed                     *kubermaticv1.Seed
	OverwriteRegistry        string
	nodePortRange            string
	nodeAccessNetwork        string
	etcdDiskSize             resource.Quantity
	oidcIssuerURL            string
	oidcIssuerClientID       string
	nodeLocalDNSCacheEnabled bool
	kubermaticImage          string
	etcdLauncherImage        string
	dnatControllerImage      string
	backupSchedule           time.Duration
	versions                 kubermatic.Versions
	caBundle                 CABundle

	supportsFailureDomainZoneAntiAffinity bool

	monitoringScrapeAnnotationPrefix                 string
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
}

type TemplateDataBuilder struct {
	data TemplateData
}

func NewTemplateDataBuilder() *TemplateDataBuilder {
	return &TemplateDataBuilder{}
}

func (td *TemplateDataBuilder) WithContext(ctx context.Context) *TemplateDataBuilder {
	td.data.ctx = ctx
	return td
}

func (td *TemplateDataBuilder) WithClient(client ctrlruntimeclient.Client) *TemplateDataBuilder {
	td.data.client = client
	return td
}

func (td *TemplateDataBuilder) WithCluster(cluster *kubermaticv1.Cluster) *TemplateDataBuilder {
	td.data.cluster = cluster
	return td
}

func (td *TemplateDataBuilder) WithDatacenter(dc *kubermaticv1.Datacenter) *TemplateDataBuilder {
	td.data.dc = dc
	return td
}

func (td *TemplateDataBuilder) WithSeed(s *kubermaticv1.Seed) *TemplateDataBuilder {
	td.data.seed = s
	return td
}

func (td *TemplateDataBuilder) WithOverwriteRegistry(overwriteRegistry string) *TemplateDataBuilder {
	td.data.OverwriteRegistry = overwriteRegistry
	return td
}

func (td *TemplateDataBuilder) WithNodePortRange(npRange string) *TemplateDataBuilder {
	td.data.nodePortRange = npRange
	return td
}

func (td *TemplateDataBuilder) WithNodeAccessNetwork(nodeAccessNetwork string) *TemplateDataBuilder {
	td.data.nodeAccessNetwork = nodeAccessNetwork
	return td
}

func (td *TemplateDataBuilder) WithEtcdDiskSize(etcdDiskSize resource.Quantity) *TemplateDataBuilder {
	td.data.etcdDiskSize = etcdDiskSize
	return td
}

func (td *TemplateDataBuilder) WithMonitoringScrapeAnnotationPrefix(prefix string) *TemplateDataBuilder {
	td.data.monitoringScrapeAnnotationPrefix = prefix
	return td
}

func (td *TemplateDataBuilder) WithInClusterPrometheusRulesFile(file string) *TemplateDataBuilder {
	td.data.inClusterPrometheusRulesFile = file
	return td
}

func (td *TemplateDataBuilder) WithInClusterPrometheusDefaultRulesDisabled(disabled bool) *TemplateDataBuilder {
	td.data.inClusterPrometheusDisableDefaultRules = disabled
	return td
}

func (td *TemplateDataBuilder) WithInClusterPrometheusDefaultScrapingConfigsDisabled(disabled bool) *TemplateDataBuilder {
	td.data.inClusterPrometheusDisableDefaultScrapingConfigs = disabled
	return td
}

func (td *TemplateDataBuilder) WithInClusterPrometheusScrapingConfigsFile(file string) *TemplateDataBuilder {
	td.data.inClusterPrometheusScrapingConfigsFile = file
	return td
}

func (td *TemplateDataBuilder) WithCABundle(bundle CABundle) *TemplateDataBuilder {
	td.data.caBundle = bundle
	return td
}

func (td *TemplateDataBuilder) WithOIDCIssuerURL(url string) *TemplateDataBuilder {
	td.data.oidcIssuerURL = url
	return td
}

func (td *TemplateDataBuilder) WithOIDCIssuerClientID(clientID string) *TemplateDataBuilder {
	td.data.oidcIssuerClientID = clientID
	return td
}

func (td *TemplateDataBuilder) WithNodeLocalDNSCacheEnabled(enabled bool) *TemplateDataBuilder {
	td.data.nodeLocalDNSCacheEnabled = enabled
	return td
}

func (td *TemplateDataBuilder) WithKubermaticImage(image string) *TemplateDataBuilder {
	td.data.kubermaticImage = image
	return td
}

func (td *TemplateDataBuilder) WithEtcdLauncherImage(image string) *TemplateDataBuilder {
	td.data.etcdLauncherImage = image
	return td
}

func (td *TemplateDataBuilder) WithDnatControllerImage(image string) *TemplateDataBuilder {
	td.data.dnatControllerImage = image
	return td
}

func (td *TemplateDataBuilder) WithVersions(v kubermatic.Versions) *TemplateDataBuilder {
	td.data.versions = v
	return td
}

func (td *TemplateDataBuilder) WithFailureDomainZoneAntiaffinity(enabled bool) *TemplateDataBuilder {
	td.data.supportsFailureDomainZoneAntiAffinity = enabled
	return td
}

func (td *TemplateDataBuilder) WithBackupPeriod(backupPeriod time.Duration) *TemplateDataBuilder {
	td.data.backupSchedule = backupPeriod
	return td
}

func (td TemplateDataBuilder) Build() *TemplateData {
	//TODO(irozzo): Add validation
	return &td.data
}

// NewTemplateData returns an instance of TemplateData
// Deprecated: Use NewTemplateDataBuilder instead.
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
	oidcURL string,
	oidcIssuerClientID string,
	nodeLocalDNSCacheEnabled bool,
	kubermaticImage string,
	etcdLauncherImage string,
	dnatControllerImage string,
	backupSchedule time.Duration,
	supportsFailureDomainZoneAntiAffinity bool,
	versions kubermatic.Versions,
	caBundle CABundle) *TemplateData {
	return &TemplateData{
		ctx:                      ctx,
		client:                   client,
		cluster:                  cluster,
		dc:                       dc,
		seed:                     seed,
		OverwriteRegistry:        overwriteRegistry,
		nodePortRange:            nodePortRange,
		nodeAccessNetwork:        nodeAccessNetwork,
		etcdDiskSize:             etcdDiskSize,
		oidcIssuerURL:            oidcURL,
		oidcIssuerClientID:       oidcIssuerClientID,
		nodeLocalDNSCacheEnabled: nodeLocalDNSCacheEnabled,
		kubermaticImage:          kubermaticImage,
		etcdLauncherImage:        etcdLauncherImage,
		dnatControllerImage:      dnatControllerImage,
		backupSchedule:           backupSchedule,
		versions:                 versions,
		caBundle:                 caBundle,

		supportsFailureDomainZoneAntiAffinity: supportsFailureDomainZoneAntiAffinity,

		monitoringScrapeAnnotationPrefix:                 monitoringScrapeAnnotationPrefix,
		inClusterPrometheusRulesFile:                     inClusterPrometheusRulesFile,
		inClusterPrometheusDisableDefaultRules:           inClusterPrometheusDisableDefaultRules,
		inClusterPrometheusDisableDefaultScrapingConfigs: inClusterPrometheusDisableDefaultScrapingConfigs,
		inClusterPrometheusScrapingConfigsFile:           inClusterPrometheusScrapingConfigsFile,
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

// GetCABundle returns the set of CA certificates that should be used
// for all outgoing communication.
func (d *TemplateData) CABundle() CABundle {
	return d.caBundle
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

func (d *TemplateData) EtcdLauncherImage() string {
	imageSplit := strings.Split(d.etcdLauncherImage, "/")
	var registry, imageWithoutRegistry string
	if len(imageSplit) != 3 {
		registry = RegistryDocker
		imageWithoutRegistry = strings.Join(imageSplit, "/")
	} else {
		registry = imageSplit[0]
		imageWithoutRegistry = strings.Join(imageSplit[1:], "/")
	}
	return d.ImageRegistry(registry) + "/" + imageWithoutRegistry
}

func (d *TemplateData) EtcdLauncherTag() string {
	return d.versions.Kubermatic
}

func (d *TemplateData) NodePortProxyTag() string {
	return d.versions.Kubermatic
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

// NodePorts returns low and high NodePorts from NodePortRange()
func (d *TemplateData) NodePorts() (int, int) {
	portrange, err := kubenetutil.ParsePortRange(d.ComputedNodePortRange())
	if err != nil {
		portrange, _ = kubenetutil.ParsePortRange(DefaultNodePortRange)
	}

	return portrange.Base, portrange.Base + portrange.Size - 1
}

// ComputedNodePortRange is NodePortRange() with defaulting and ComponentsOverride logic
func (d *TemplateData) ComputedNodePortRange() string {
	nodePortRange := d.NodePortRange()

	if nodePortRange == "" {
		nodePortRange = DefaultNodePortRange
	}

	if cluster := d.Cluster(); cluster != nil {
		if npr := cluster.Spec.ComponentsOverride.Apiserver.NodePortRange; npr != "" {
			nodePortRange = npr
		}
	}

	return nodePortRange
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
func (d *TemplateData) ImageRegistry(defaultRegistry string) string {
	if d.OverwriteRegistry != "" {
		return d.OverwriteRegistry
	}
	return defaultRegistry
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
	// When using tunneling expose strategy the port is fixed
	if d.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
		return 1194, nil
	}
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

func (d *TemplateData) KubermaticDockerTag() string {
	return d.versions.Kubermatic
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

func (d *TemplateData) BackupSchedule() time.Duration {
	return d.backupSchedule
}

func (d *TemplateData) DNATControllerTag() string {
	return d.versions.Kubermatic
}

func (d *TemplateData) SupportsFailureDomainZoneAntiAffinity() bool {
	return d.supportsFailureDomainZoneAntiAffinity
}

func (d *TemplateData) GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return provider.SecretKeySelectorValueFuncFactory(d.ctx, d.client)(configVar, key)
}

func (d *TemplateData) GetKubernetesCloudProviderName() string {
	return GetKubernetesCloudProviderName(d.Cluster(),
		d.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider])
}

func (d *TemplateData) GetCSIMigrationFeatureGates() []string {
	return GetCSIMigrationFeatureGates(d.Cluster())
}

// KCMCloudControllersDeactivated return true if the KCM is ready and the
// cloud-controllers are disabled.
// * There is no 'cloud-provider' flag.
// * The cloud controllers are disabled.
// This is used to avoid deploing the CCM before the in-tree cloud controllers
// have been deactivated.
func (d *TemplateData) KCMCloudControllersDeactivated() bool {
	kcm := appsv1.Deployment{}
	if err := d.client.Get(d.ctx, ctrlruntimeclient.ObjectKey{Name: ControllerManagerDeploymentName, Namespace: d.cluster.Status.NamespaceName}, &kcm); err != nil {
		klog.Errorf("could not get kcm deployment: %v", err)
		return false
	}
	ready, _ := kubernetes.IsDeploymentRolloutComplete(&kcm, 0)
	klog.V(4).Infof("controller-manager deployment rollout complete: %t", ready)
	if c := getContainer(&kcm, ControllerManagerDeploymentName); c != nil {
		if ok, cmd := UnwrapCommand(*c); ok {
			klog.V(4).Infof("controller-manager command %v %d", cmd.Args, len(cmd.Args))
			// If no --cloud-provider flag is provided in-tree cloud provider
			// is disabled.
			if ok, val := getArgValue(cmd.Args, "--cloud-provider"); !ok || val == cloudProviderExternalFlag {
				klog.V(4).Info("in-tree cloud provider disabled in controller-manager deployment")
				return ready
			}

			// Otherwise cloud countrollers could have been explicitly disabled
			if ok, val := getArgValue(cmd.Args, "--controllers"); ok {
				controllers := strings.Split(val, ",")
				klog.V(4).Infof("cloud controllers disabled in controller-manager deployment %s", controllers)
				return ready && sets.NewString(controllers...).HasAll("-cloud-node-lifecycle", "-route", "-service")
			}
		}
	}

	return false
}

func UnwrapCommand(container corev1.Container) (found bool, command httpproberapi.Command) {
	for i, arg := range container.Args {
		klog.V(4).Infof("unwrap command processing arg: %s", arg)
		if arg == "-command" && i < len(container.Args)-1 {
			if err := json.Unmarshal([]byte(container.Args[i+1]), &command); err != nil {
				return
			}
			return true, command
		}
	}
	return
}

func getArgValue(args []string, argName string) (bool, string) {
	for i, arg := range args {
		klog.V(4).Infof("processing arg %s", arg)
		if arg == argName {
			klog.V(4).Infof("found argument %s", argName)
			if i >= len(args)-1 {
				return false, ""
			}
			return true, args[i+1]
		}
	}
	return false, ""
}

func getContainer(d *appsv1.Deployment, containerName string) *corev1.Container {
	for _, c := range d.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			return &c
		}
	}
	return nil
}

func GetKubernetesCloudProviderName(cluster *kubermaticv1.Cluster, externalCloudProvider bool) string {
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
		if externalCloudProvider {
			return cloudProviderExternalFlag
		}
		return "openstack"
	case cluster.Spec.Cloud.Hetzner != nil:
		if cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
			return cloudProviderExternalFlag
		}
		return ""
	default:
		return ""
	}
}

func ExternalCloudProviderEnabled(cluster *kubermaticv1.Cluster) bool {
	// If we are migrating from in-tree cloud provider to CSI driver, we
	// should not disable the in-tree cloud provider until all kubelets are
	// migrated, otherwise we won't be able to use the volume API.
	return cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] &&
		(kubermaticv1helper.ClusterConditionHasStatus(cluster, kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted, corev1.ConditionTrue) ||
			!metav1.HasAnnotation(cluster.ObjectMeta, kubermaticv1.CSIMigrationNeededAnnotation))
}

func GetCSIMigrationFeatureGates(cluster *kubermaticv1.Cluster) []string {
	var featureFlags []string
	if metav1.HasAnnotation(cluster.ObjectMeta, kubermaticv1.CSIMigrationNeededAnnotation) {
		// The following feature gates are always enabled when the
		// 'externalCloudProvider' feature is activated.
		if cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
			featureFlags = append(featureFlags, "CSIMigration=true", "CSIMigrationOpenStack=true", "ExpandCSIVolumes=true")
		}
		// The CSIMigrationNeededAnnotation is removed when all kubelets have
		// been migrated.
		if kubermaticv1helper.ClusterConditionHasStatus(cluster, kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted, corev1.ConditionTrue) {
			featureFlags = append(featureFlags, "CSIMigrationOpenStackComplete=true")
		}
	}
	return featureFlags
}

func (d *TemplateData) Seed() *kubermaticv1.Seed {
	return d.seed
}
