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
	"strconv"
	"strings"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	httpproberapi "k8c.io/kubermatic/v2/cmd/http-prober/api"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	providerconfig "k8c.io/machine-controller/pkg/providerconfig/types"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubenetutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/sets"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CloudProviderExternalFlag = "external"
)

type CABundle interface {
	CertPool() *x509.CertPool
	String() string
}

// TemplateData is a group of data required for template generation.
type TemplateData struct {
	ctx                              context.Context
	client                           ctrlruntimeclient.Client
	cluster                          *kubermaticv1.Cluster
	dc                               *kubermaticv1.Datacenter
	seed                             *kubermaticv1.Seed
	config                           *kubermaticv1.KubermaticConfiguration
	OverwriteRegistry                string
	nodePortRange                    string
	nodeAccessNetwork                string
	etcdDiskSize                     resource.Quantity
	oidcIssuerURL                    string
	oidcIssuerClientID               string
	kubermaticImage                  string
	dnatControllerImage              string
	networkIntfMgrImage              string
	machineControllerImageTag        string
	machineControllerImageRepository string
	backupSchedule                   time.Duration
	versions                         kubermatic.Versions
	caBundle                         CABundle
	clusterBackupStorageLocation     *kubermaticv1.ClusterBackupStorageLocation
	apiServerAlternateNames          *certutil.AltNames

	supportsFailureDomainZoneAntiAffinity bool

	userClusterMLAEnabled bool
	isKonnectivityEnabled bool

	tunnelingAgentIP string

	etcdLauncherImage         string
	etcdBackupStoreContainer  *corev1.Container
	etcdBackupDeleteContainer *corev1.Container
	etcdBackupDestination     *kubermaticv1.BackupDestination
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

func (td *TemplateDataBuilder) WithKubermaticConfiguration(cfg *kubermaticv1.KubermaticConfiguration) *TemplateDataBuilder {
	td.data.config = cfg
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

func (td *TemplateDataBuilder) WithUserClusterMLAEnabled(enabled bool) *TemplateDataBuilder {
	td.data.userClusterMLAEnabled = enabled
	return td
}

func (td *TemplateDataBuilder) WithKonnectivityEnabled(enabled bool) *TemplateDataBuilder {
	td.data.isKonnectivityEnabled = enabled
	return td
}

func (td *TemplateDataBuilder) WithCABundle(bundle CABundle) *TemplateDataBuilder {
	td.data.caBundle = bundle
	return td
}

func (td *TemplateDataBuilder) WithClusterBackupStorageLocation(loc *kubermaticv1.ClusterBackupStorageLocation) *TemplateDataBuilder {
	td.data.clusterBackupStorageLocation = loc
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

func (td *TemplateDataBuilder) WithKubermaticImage(image string) *TemplateDataBuilder {
	td.data.kubermaticImage = image
	return td
}

func (td *TemplateDataBuilder) WithEtcdLauncherImage(image string) *TemplateDataBuilder {
	td.data.etcdLauncherImage = image
	return td
}

func (td *TemplateDataBuilder) WithEtcdBackupStoreContainer(container *corev1.Container, isCustom bool) *TemplateDataBuilder {
	if !isCustom {
		container.Image = registry.Must(td.data.RewriteImage(container.Image))
	}
	td.data.etcdBackupStoreContainer = container
	return td
}

func (td *TemplateDataBuilder) WithEtcdBackupDeleteContainer(container *corev1.Container, isCustom bool) *TemplateDataBuilder {
	if !isCustom {
		container.Image = registry.Must(td.data.RewriteImage(container.Image))
	}
	td.data.etcdBackupDeleteContainer = container
	return td
}

func (td *TemplateDataBuilder) WithEtcdBackupDestination(destination *kubermaticv1.BackupDestination) *TemplateDataBuilder {
	td.data.etcdBackupDestination = destination
	return td
}

func (td *TemplateDataBuilder) WithDnatControllerImage(image string) *TemplateDataBuilder {
	td.data.dnatControllerImage = image
	return td
}

func (td *TemplateDataBuilder) WithNetworkIntfMgrImage(image string) *TemplateDataBuilder {
	td.data.networkIntfMgrImage = image
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

func (td *TemplateDataBuilder) WithMachineControllerImageTag(tag string) *TemplateDataBuilder {
	td.data.machineControllerImageTag = tag
	return td
}

func (td *TemplateDataBuilder) WithMachineControllerImageRepository(repository string) *TemplateDataBuilder {
	td.data.machineControllerImageRepository = repository
	return td
}

func (td *TemplateDataBuilder) WithTunnelingAgentIP(tunnelingAgentIP string) *TemplateDataBuilder {
	td.data.tunnelingAgentIP = tunnelingAgentIP
	return td
}

func (td *TemplateDataBuilder) WithAPIServerAlternateNames(altNames *certutil.AltNames) *TemplateDataBuilder {
	td.data.apiServerAlternateNames = altNames
	return td
}

func (td TemplateDataBuilder) Build() *TemplateData {
	// TODO: Add validation
	return &td.data
}

// GetViewerToken returns the viewer token.
func (d *TemplateData) GetViewerToken() (string, error) {
	viewerTokenSecret := &corev1.Secret{}
	if err := d.client.Get(d.ctx, ctrlruntimeclient.ObjectKey{Name: ViewerTokenSecretName, Namespace: d.cluster.Status.NamespaceName}, viewerTokenSecret); err != nil {
		return "", err
	}
	return string(viewerTokenSecret.Data[ViewerTokenSecretKey]), nil
}

// CABundle returns the set of CA certificates that should be used
// for all outgoing communication.
func (d *TemplateData) CABundle() CABundle {
	return d.caBundle
}

// OIDCIssuerURL returns URL of the OpenID token issuer.
func (d *TemplateData) OIDCIssuerURL() string {
	return d.oidcIssuerURL
}

// OIDCIssuerClientID return the issuer client ID.
func (d *TemplateData) OIDCIssuerClientID() string {
	return d.oidcIssuerClientID
}

// Cluster returns the cluster.
func (d *TemplateData) Cluster() *kubermaticv1.Cluster {
	return d.cluster
}

// DC returns the dc.
func (d *TemplateData) DC() *kubermaticv1.Datacenter {
	return d.dc
}

// EtcdDiskSize returns the etcd disk size.
func (d *TemplateData) EtcdDiskSize() resource.Quantity {
	return d.etcdDiskSize
}

func (d *TemplateData) EtcdLauncherImage() string {
	return registry.Must(d.RewriteImage(d.etcdLauncherImage))
}

func (d *TemplateData) EtcdBackupStoreContainer() *corev1.Container {
	return d.etcdBackupStoreContainer
}

func (d *TemplateData) EtcdBackupDeleteContainer() *corev1.Container {
	return d.etcdBackupDeleteContainer
}

func (d *TemplateData) EtcdBackupDestination() *kubermaticv1.BackupDestination {
	return d.etcdBackupDestination
}

func (d *TemplateData) EtcdLauncherTag() string {
	return d.versions.Kubermatic
}

func (d *TemplateData) NodePortProxyTag() string {
	return d.versions.Kubermatic
}

// UserClusterMLAEnabled returns userClusterMLAEnabled.
func (d *TemplateData) UserClusterMLAEnabled() bool {
	return d.userClusterMLAEnabled
}

// IsKonnectivityEnabled returns isKonnectivityEnabled.
func (d *TemplateData) IsKonnectivityEnabled() bool {
	return d.isKonnectivityEnabled
}

// NodeAccessNetwork returns the node access network.
func (d *TemplateData) NodeAccessNetwork() string {
	return d.nodeAccessNetwork
}

// NodePortRange returns the node access network.
func (d *TemplateData) NodePortRange() string {
	return d.nodePortRange
}

// NodePorts returns low and high NodePorts from NodePortRange().
func (d *TemplateData) NodePorts() (int, int) {
	portrange, err := kubenetutil.ParsePortRange(d.ComputedNodePortRange())
	if err != nil {
		portrange, _ = kubenetutil.ParsePortRange(DefaultNodePortRange)
	}

	return portrange.Base, portrange.Base + portrange.Size - 1
}

// ComputedNodePortRange is NodePortRange() with defaulting and ComponentsOverride logic.
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

func (d *TemplateData) GetClusterBackupStorageLocation() *kubermaticv1.ClusterBackupStorageLocation {
	return d.clusterBackupStorageLocation
}

// GetClusterRef returns a instance of a OwnerReference for the Cluster in the TemplateData.
func (d *TemplateData) GetClusterRef() metav1.OwnerReference {
	return GetClusterRef(d.cluster)
}

// ExternalIP returns the external facing IP or an error if no IP exists.
func (d *TemplateData) ExternalIP() (*net.IP, error) {
	return GetClusterExternalIP(d.cluster)
}

func (d *TemplateData) MachineControllerImageTag() string {
	return d.machineControllerImageTag
}

func (d *TemplateData) MachineControllerImageRepository() string {
	return d.machineControllerImageRepository
}

func (d *TemplateData) OperatingSystemManagerImageTag() string {
	return d.config.Spec.UserCluster.OperatingSystemManager.ImageTag
}

func (d *TemplateData) OperatingSystemManagerImageRepository() string {
	return d.config.Spec.UserCluster.OperatingSystemManager.ImageRepository
}

// ClusterIPByServiceName returns the ClusterIP as string for the
// Service specified by `name`. Service lookup happens within
// `Cluster.Status.NamespaceName`. When ClusterIP fails to parse
// as valid IP address, an error is returned.
func (d *TemplateData) ClusterIPByServiceName(name string) (string, error) {
	service := &corev1.Service{}
	key := types.NamespacedName{Namespace: d.cluster.Status.NamespaceName, Name: name}
	if err := d.client.Get(d.ctx, key, service); err != nil {
		return "", fmt.Errorf("could not get service %s: %w", key, err)
	}

	if net.ParseIP(service.Spec.ClusterIP) == nil {
		return "", fmt.Errorf("service %s has no valid cluster ip (\"%s\")", key, service.Spec.ClusterIP)
	}
	return service.Spec.ClusterIP, nil
}

// ProviderName returns the name of the clusters providerName.
func (d *TemplateData) ProviderName() string {
	p, err := kubermaticv1helper.ClusterCloudProviderName(d.cluster.Spec.Cloud)
	if err != nil {
		kubermaticlog.Logger.Errorw("could not identify cloud provider", zap.Error(err))
	}
	return p
}

// GetLegacyOverwriteRegistry should not be used by new code, rather the
// ImageRewriter() should be used instead.
func (d *TemplateData) GetLegacyOverwriteRegistry() string {
	return d.OverwriteRegistry
}

// ImageRewriter returns a Docker image rewriter.
func (d *TemplateData) ImageRewriter() registry.ImageRewriter {
	return registry.GetImageRewriterFunc(d.OverwriteRegistry)
}

// RewriteImage rewrites a Docker image to apply a custom registry if specified.
func (d *TemplateData) RewriteImage(image string) (string, error) {
	return d.ImageRewriter()(image)
}

// GetRootCA returns the root CA of the cluster.
func (d *TemplateData) GetRootCA() (*triple.KeyPair, error) {
	return GetClusterRootCA(d.ctx, d.cluster.Status.NamespaceName, d.client)
}

// GetFrontProxyCA returns the root CA for the front proxy.
func (d *TemplateData) GetFrontProxyCA() (*triple.KeyPair, error) {
	return GetClusterFrontProxyCA(d.ctx, d.cluster.Status.NamespaceName, d.client)
}

// GetOpenVPNCA returns the root ca for the OpenVPN.
func (d *TemplateData) GetOpenVPNCA() (*ECDSAKeyPair, error) {
	return GetOpenVPNCA(d.ctx, d.cluster.Status.NamespaceName, d.client)
}

// GetMLAGatewayCA returns the root CA for the MLA Gateway.
func (d *TemplateData) GetMLAGatewayCA() (*ECDSAKeyPair, error) {
	return GetMLAGatewayCA(d.ctx, d.cluster.Status.NamespaceName, d.client)
}

// GetOpenVPNServerPort returns the nodeport of the external apiserver service.
func (d *TemplateData) GetOpenVPNServerPort() (int32, error) {
	// When using tunneling expose strategy the port is fixed
	if d.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
		return 1194, nil
	}
	service := &corev1.Service{}
	key := types.NamespacedName{Namespace: d.cluster.Status.NamespaceName, Name: OpenVPNServerServiceName}
	if err := d.client.Get(d.ctx, key, service); err != nil {
		return 0, fmt.Errorf("failed to get NodePort for openvpn server service: %w", err)
	}

	return service.Spec.Ports[0].NodePort, nil
}

// GetAPIServerAlternateNames returns the alternate names for the apiserver certificate from the
// corresponding services in the cluster namespace.
func (d *TemplateData) GetAPIServerAlternateNames() *certutil.AltNames {
	return d.apiServerAlternateNames
}

// GetKonnectivityServerPort returns the nodeport of the external Konnectivity Server service.
func (d *TemplateData) GetKonnectivityServerPort() (int32, error) {
	// When using tunneling expose strategy the port is fixed and equal to apiserver port
	if d.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
		return d.Cluster().Status.Address.Port, nil
	}
	service := &corev1.Service{}
	key := types.NamespacedName{Namespace: d.cluster.Status.NamespaceName, Name: KonnectivityProxyServiceName}
	if err := d.client.Get(d.ctx, key, service); err != nil {
		return 0, fmt.Errorf("failed to get NodePort for Konnectivity Server service: %w", err)
	}

	return service.Spec.Ports[0].NodePort, nil
}

func (d *TemplateData) GetKonnectivityKeepAliveTime() string {
	if t := d.Cluster().Spec.ComponentsOverride.KonnectivityProxy.KeepaliveTime; t != "" {
		return t
	}
	return kubermaticv1.DefaultKonnectivityKeepaliveTime
}

func (d *TemplateData) GetTunnelingAgentIP() string {
	if ip := d.Cluster().Spec.ClusterNetwork.TunnelingAgentIP; ip != "" {
		return ip
	}
	if d.tunnelingAgentIP != "" {
		return d.tunnelingAgentIP
	}
	return DefaultTunnelingAgentIP
}

// GetMLAGatewayPort returns the NodePort of the external MLA Gateway service.
func (d *TemplateData) GetMLAGatewayPort() (int32, error) {
	// When using tunneling expose strategy the port is fixed and equal to apiserver port
	if d.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
		return d.Cluster().Status.Address.Port, nil
	}
	service := &corev1.Service{}
	key := types.NamespacedName{Namespace: d.cluster.Status.NamespaceName, Name: MLAGatewayExternalServiceName}
	if err := d.client.Get(d.ctx, key, service); err != nil {
		return 0, fmt.Errorf("failed to get NodePort for MLA Gateway service: %w", err)
	}

	return service.Spec.Ports[0].NodePort, nil
}

func (d *TemplateData) NodeLocalDNSCacheEnabled() bool {
	// NOTE: even if NodeLocalDNSCacheEnabled is nil, we assume it is enabled (backward compatibility for already existing clusters)
	return d.Cluster().Spec.ClusterNetwork.NodeLocalDNSCacheEnabled == nil || *d.Cluster().Spec.ClusterNetwork.NodeLocalDNSCacheEnabled
}

func (d *TemplateData) KubermaticAPIImage() string {
	return registry.Must(d.RewriteImage(d.kubermaticImage))
}

func (d *TemplateData) KubermaticDockerTag() string {
	return d.versions.Kubermatic
}

func (d *TemplateData) DNATControllerImage() string {
	return registry.Must(d.RewriteImage(d.dnatControllerImage))
}

func (d *TemplateData) NetworkIntfMgrImage() string {
	return registry.Must(d.RewriteImage(d.networkIntfMgrImage))
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

func (d *TemplateData) GetSecretKeyValue(ref *corev1.SecretKeySelector) ([]byte, error) {
	secret := corev1.Secret{}
	if err := d.client.Get(d.ctx, ctrlruntimeclient.ObjectKey{Name: ref.Name, Namespace: d.cluster.Status.NamespaceName}, &secret); err != nil {
		return nil, err
	}

	val, ok := secret.Data[ref.Key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret", ref.Key)
	}

	return val, nil
}

func (d *TemplateData) GetCloudProviderName() (string, error) {
	return kubermaticv1helper.ClusterCloudProviderName(d.Cluster().Spec.Cloud)
}

func (d *TemplateData) GetCSIMigrationFeatureGates(version *semverlib.Version) []string {
	return GetCSIMigrationFeatureGates(d.Cluster(), version)
}

// KCMCloudControllersDeactivated return true if the KCM is ready and the
// cloud-controllers are disabled.
// * There is no 'cloud-provider' flag.
// * The cloud controllers are disabled.
// This is used to avoid deploying the CCM before the in-tree cloud controllers
// have been deactivated.
func (d *TemplateData) KCMCloudControllersDeactivated() bool {
	logger := kubermaticlog.Logger

	kcm := appsv1.Deployment{}
	if err := d.client.Get(d.ctx, ctrlruntimeclient.ObjectKey{Name: ControllerManagerDeploymentName, Namespace: d.cluster.Status.NamespaceName}, &kcm); err != nil {
		logger.Errorw("could not get kcm deployment", zap.Error(err))
		return false
	}

	ready, _ := kubernetes.IsDeploymentRolloutComplete(&kcm, 0)
	logger.Debugw("controller-manager deployment rollout status", "ready", ready)

	if c := getContainer(&kcm, ControllerManagerDeploymentName); c != nil {
		if ok, cmd := UnwrapCommand(*c); ok {
			logger.Debugw("controller-manager command", "args", cmd.Args)

			// If no --cloud-provider flag is provided in-tree cloud provider is disabled.
			if ok, val := getArgValue(cmd.Args, "--cloud-provider"); !ok || val == CloudProviderExternalFlag {
				logger.Debug("in-tree cloud provider disabled in controller-manager deployment")
				return ready
			}

			// Otherwise cloud countrollers could have been explicitly disabled
			if ok, val := getArgValue(cmd.Args, "--controllers"); ok {
				controllers := strings.Split(val, ",")
				logger.Debugw("cloud controllers disabled in controller-manager deployment", "controllers", controllers)
				return ready && sets.New(controllers...).HasAll("-cloud-node-lifecycle", "-route", "-service")
			}
		}
	}

	return false
}

func UnwrapCommand(container corev1.Container) (found bool, command httpproberapi.Command) {
	for i, arg := range container.Args {
		kubermaticlog.Logger.Debugw("unwrap command processing argument", "arg", arg)
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
		kubermaticlog.Logger.Debugw("processing argument", "arg", arg)
		if arg == argName {
			kubermaticlog.Logger.Debugw("found argument", "name", argName)
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
		if cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
			return CloudProviderExternalFlag
		}
		return "aws"
	case cluster.Spec.Cloud.VSphere != nil:
		if cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
			return CloudProviderExternalFlag
		}
		return "vsphere"
	case cluster.Spec.Cloud.Azure != nil:
		if externalCloudProvider {
			return CloudProviderExternalFlag
		}
		return "azure"
	case cluster.Spec.Cloud.GCP != nil:
		if cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
			return CloudProviderExternalFlag
		}
		return "gce"
	case cluster.Spec.Cloud.Openstack != nil:
		if externalCloudProvider {
			return CloudProviderExternalFlag
		}
		return "openstack"
	case cluster.Spec.Cloud.Hetzner != nil:
		if cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
			return CloudProviderExternalFlag
		}
		return ""
	case cluster.Spec.Cloud.Digitalocean != nil:
		if cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
			return CloudProviderExternalFlag
		}
		return "digitalocean"
	default:
		return ""
	}
}

func ExternalCloudProviderEnabled(cluster *kubermaticv1.Cluster) bool {
	// If we are migrating from in-tree cloud provider to CSI driver, we
	// should not disable the in-tree cloud provider until all kubelets are
	// migrated, otherwise we won't be able to use the volume API.
	hasCSIMigrationCompletedCond := cluster.Status.Conditions[kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted].Status == corev1.ConditionTrue

	return cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] &&
		(hasCSIMigrationCompletedCond || !metav1.HasAnnotation(cluster.ObjectMeta, kubermaticv1.CSIMigrationNeededAnnotation))
}

func GetCSIMigrationFeatureGates(cluster *kubermaticv1.Cluster, version *semverlib.Version) []string {
	var featureFlags []string

	if metav1.HasAnnotation(cluster.ObjectMeta, kubermaticv1.CSIMigrationNeededAnnotation) {
		// The CSIMigrationNeededAnnotation is removed when all kubelets have
		// been migrated. Both of these feature gates have already been removed in Kubernetes 1.30+.
		migrationCompleted := cluster.Status.Conditions[kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted].Status == corev1.ConditionTrue

		if migrationCompleted && cluster.Spec.Version.Semver().Minor() < 30 {
			if cluster.Spec.Cloud.Openstack != nil {
				featureFlags = append(featureFlags, "InTreePluginOpenStackUnregister=true")
			}
			if cluster.Spec.Cloud.VSphere != nil {
				featureFlags = append(featureFlags, "InTreePluginvSphereUnregister=true")
			}
		}
	}

	return featureFlags
}

func (d *TemplateData) Seed() *kubermaticv1.Seed {
	return d.seed
}

func (d *TemplateData) KubermaticConfiguration() *kubermaticv1.KubermaticConfiguration {
	return d.config
}

func (data *TemplateData) GetEnvVars() ([]corev1.EnvVar, error) {
	cluster := data.Cluster()
	dc := data.DC()

	refTo := func(key string) *corev1.EnvVarSource {
		return &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: ClusterCloudCredentialsSecretName,
				},
				Key: key,
			},
		}
	}

	optionalRefTo := func(key string) *corev1.EnvVarSource {
		ref := refTo(key)
		ref.SecretKeyRef.Optional = ptr.To(true)

		return ref
	}

	var vars []corev1.EnvVar
	if cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", ValueFrom: refTo(AWSAccessKeyID)})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", ValueFrom: refTo(AWSSecretAccessKey)})
		vars = append(vars, corev1.EnvVar{Name: "AWS_ASSUME_ROLE_ARN", Value: cluster.Spec.Cloud.AWS.AssumeRoleARN})
		vars = append(vars, corev1.EnvVar{Name: "AWS_ASSUME_ROLE_EXTERNAL_ID", Value: cluster.Spec.Cloud.AWS.AssumeRoleExternalID})
	}
	if cluster.Spec.Cloud.Azure != nil {
		vars = append(vars, corev1.EnvVar{Name: "AZURE_CLIENT_ID", ValueFrom: refTo(AzureClientID)})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_CLIENT_SECRET", ValueFrom: refTo(AzureClientSecret)})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_TENANT_ID", ValueFrom: refTo(AzureTenantID)})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_SUBSCRIPTION_ID", ValueFrom: refTo(AzureSubscriptionID)})
	}
	if cluster.Spec.Cloud.Openstack != nil {
		vars = append(vars, corev1.EnvVar{Name: "OS_AUTH_URL", Value: dc.Spec.Openstack.AuthURL})
		vars = append(vars, corev1.EnvVar{Name: "OS_USER_NAME", ValueFrom: refTo(OpenstackUsername)})
		vars = append(vars, corev1.EnvVar{Name: "OS_PASSWORD", ValueFrom: refTo(OpenstackPassword)})
		vars = append(vars, corev1.EnvVar{Name: "OS_DOMAIN_NAME", ValueFrom: refTo(OpenstackDomain)})
		vars = append(vars, corev1.EnvVar{Name: "OS_PROJECT_NAME", ValueFrom: optionalRefTo(OpenstackProject)})
		vars = append(vars, corev1.EnvVar{Name: "OS_PROJECT_ID", ValueFrom: optionalRefTo(OpenstackProjectID)})
		vars = append(vars, corev1.EnvVar{Name: "OS_APPLICATION_CREDENTIAL_ID", ValueFrom: optionalRefTo(OpenstackApplicationCredentialID)})
		vars = append(vars, corev1.EnvVar{Name: "OS_APPLICATION_CREDENTIAL_SECRET", ValueFrom: optionalRefTo(OpenstackApplicationCredentialSecret)})
	}
	if cluster.Spec.Cloud.Hetzner != nil {
		vars = append(vars, corev1.EnvVar{Name: "HZ_TOKEN", ValueFrom: refTo(HetznerToken)})
	}
	if cluster.Spec.Cloud.Digitalocean != nil {
		vars = append(vars, corev1.EnvVar{Name: "DO_TOKEN", ValueFrom: refTo(DigitaloceanToken)})
	}
	if cluster.Spec.Cloud.VSphere != nil {
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(data.ctx, data.client)
		spec := cluster.Spec.Cloud.VSphere
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_ADDRESS", Value: dc.Spec.VSphere.Endpoint})
		if val, _ := secretKeySelector(spec.CredentialsReference, VsphereInfraManagementUserUsername); val != "" {
			vars = append(vars, corev1.EnvVar{Name: "VSPHERE_USERNAME", ValueFrom: refTo(VsphereInfraManagementUserUsername)})
		} else {
			vars = append(vars, corev1.EnvVar{Name: "VSPHERE_USERNAME", ValueFrom: refTo(VsphereUsername)})
		}

		if val, _ := secretKeySelector(spec.CredentialsReference, VsphereInfraManagementUserPassword); val != "" {
			vars = append(vars, corev1.EnvVar{Name: "VSPHERE_PASSWORD", ValueFrom: refTo(VsphereInfraManagementUserPassword)})
		} else {
			vars = append(vars, corev1.EnvVar{Name: "VSPHERE_PASSWORD", ValueFrom: refTo(VspherePassword)})
		}
	}
	if cluster.Spec.Cloud.Baremetal != nil {
		if cluster.Spec.Cloud.Baremetal.Tinkerbell != nil {
			vars = append(vars, corev1.EnvVar{Name: "TINK_KUBECONFIG", ValueFrom: refTo(TinkerbellKubeconfig)})
		}
	}
	if cluster.Spec.Cloud.Packet != nil {
		vars = append(vars, corev1.EnvVar{Name: "METAL_AUTH_TOKEN", ValueFrom: refTo(PacketAPIKey)})
		vars = append(vars, corev1.EnvVar{Name: "METAL_PROJECT_ID", ValueFrom: refTo(PacketProjectID)})
	}
	if cluster.Spec.Cloud.GCP != nil {
		vars = append(vars, corev1.EnvVar{Name: "GOOGLE_SERVICE_ACCOUNT", ValueFrom: refTo(GCPServiceAccount)})
	}
	if cluster.Spec.Cloud.Kubevirt != nil {
		vars = append(vars, corev1.EnvVar{Name: "KUBEVIRT_KUBECONFIG", ValueFrom: refTo(KubeVirtKubeconfig)})
	}
	if cluster.Spec.Cloud.Alibaba != nil {
		vars = append(vars, corev1.EnvVar{Name: "ALIBABA_ACCESS_KEY_ID", ValueFrom: refTo(AlibabaAccessKeyID)})
		vars = append(vars, corev1.EnvVar{Name: "ALIBABA_ACCESS_KEY_SECRET", ValueFrom: refTo(AlibabaAccessKeySecret)})
	}
	if cluster.Spec.Cloud.Anexia != nil {
		vars = append(vars, corev1.EnvVar{Name: "ANEXIA_TOKEN", ValueFrom: refTo(AnexiaToken)})
	}
	if cluster.Spec.Cloud.Nutanix != nil {
		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_CLUSTER_NAME", Value: cluster.Spec.Cloud.Nutanix.ClusterName})
		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_ENDPOINT", Value: dc.Spec.Nutanix.Endpoint})

		if port := dc.Spec.Nutanix.Port; port != nil {
			vars = append(vars, corev1.EnvVar{Name: "NUTANIX_PORT", Value: strconv.Itoa(int(*port))})
		}
		if dc.Spec.Nutanix.AllowInsecure {
			vars = append(vars, corev1.EnvVar{Name: "NUTANIX_INSECURE", Value: "true"})
		}

		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_USERNAME", ValueFrom: refTo(NutanixUsername)})
		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_PASSWORD", ValueFrom: refTo(NutanixPassword)})
		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_PROXY_URL", ValueFrom: optionalRefTo(NutanixProxyURL)}) // proxy URL can be empty
	}
	if cluster.Spec.Cloud.VMwareCloudDirector != nil {
		vars = append(vars, corev1.EnvVar{Name: "VCD_URL", Value: dc.Spec.VMwareCloudDirector.URL})
		vars = append(vars, corev1.EnvVar{Name: "VCD_USER", ValueFrom: refTo(VMwareCloudDirectorUsername)})
		vars = append(vars, corev1.EnvVar{Name: "VCD_PASSWORD", ValueFrom: refTo(VMwareCloudDirectorPassword)})
		vars = append(vars, corev1.EnvVar{Name: "VCD_ORG", ValueFrom: refTo(VMwareCloudDirectorOrganization)})
		vars = append(vars, corev1.EnvVar{Name: "VCD_VDC", ValueFrom: refTo(VMwareCloudDirectorVDC)})
		vars = append(vars, corev1.EnvVar{Name: "VCD_API_TOKEN", ValueFrom: optionalRefTo(VMwareCloudDirectorAPIToken)})

		if dc.Spec.VMwareCloudDirector.AllowInsecure {
			vars = append(vars, corev1.EnvVar{Name: "VCD_ALLOW_UNVERIFIED_SSL", Value: "true"})
		}
	}
	vars = append(vars, GetHTTPProxyEnvVarsFromSeed(data.Seed(), cluster.Status.Address.InternalName)...)

	vars = SanitizeEnvVars(vars)
	if cluster.Spec.Cloud.Kubevirt != nil && dc.Spec.Kubevirt != nil && dc.Spec.Kubevirt.NamespacedMode != nil && dc.Spec.Kubevirt.NamespacedMode.Enabled {
		vars = append(vars, corev1.EnvVar{Name: "POD_NAMESPACE", Value: dc.Spec.Kubevirt.NamespacedMode.Namespace})
	} else {
		vars = append(vars, corev1.EnvVar{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}})
	}

	return vars, nil
}
