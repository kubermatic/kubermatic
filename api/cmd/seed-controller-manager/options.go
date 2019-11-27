package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"path"
	"strings"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/backup"
	"github.com/kubermatic/kubermatic/api/pkg/features"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	seedvalidation "github.com/kubermatic/kubermatic/api/pkg/validation/seed"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/net"
	certutil "k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type controllerRunOptions struct {
	kubeconfig   string
	masterURL    string
	internalAddr string

	masterResources                                  string
	externalURL                                      string
	dc                                               string
	dcFile                                           string
	workerName                                       string
	versionsFile                                     string
	updatesFile                                      string
	workerCount                                      int
	overwriteRegistry                                string
	nodePortRange                                    string
	nodeAccessNetwork                                string
	kubernetesAddonsPath                             string
	openshiftAddonsPath                              string
	kubernetesAddonsList                             string
	openshiftAddonsList                              string
	backupContainerFile                              string
	cleanupContainerFile                             string
	backupContainerImage                             string
	backupInterval                                   string
	etcdDiskSize                                     resource.Quantity
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
	monitoringScrapeAnnotationPrefix                 string
	dockerPullConfigJSONFile                         string
	log                                              kubermaticlog.Options
	kubermaticImage                                  string
	dnatControllerImage                              string
	dynamicDatacenters                               bool
	namespace                                        string
	apiServerDefaultReplicas                         int
	apiServerEndpointReconcilingDisabled             bool
	controllerManagerDefaultReplicas                 int
	schedulerDefaultReplicas                         int
	seedValidationHook                               seedvalidation.WebhookOpts
	concurrentClusterUpdate                          int

	// OIDC configuration
	oidcCAFile             string
	oidcIssuerURL          string
	oidcIssuerClientID     string
	oidcIssuerClientSecret string

	featureGates features.FeatureGate
}

func newControllerRunOptions() (controllerRunOptions, error) {
	c := controllerRunOptions{}
	var rawFeatureGates string
	var rawEtcdDiskSize string

	flag.StringVar(&c.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&c.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&c.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal server is running on")
	flag.StringVar(&c.masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&c.externalURL, "external-url", "", "The external url for the apiserver host and the the dc.(Required)")
	flag.StringVar(&c.dc, "datacenter-name", "", "The name of the seed datacenter, the controller is running in. It will be used to build the absolute url for a customer cluster.")
	flag.StringVar(&c.dcFile, "datacenters", "", "The datacenters.yaml file path")
	flag.StringVar(&c.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.StringVar(&c.versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&c.updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.IntVar(&c.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&c.overwriteRegistry, "overwrite-registry", "", "registry to use for all images")
	flag.StringVar(&c.nodePortRange, "nodeport-range", "30000-32767", "NodePort range to use for new clusters. It must be within the NodePort range of the seed-cluster")
	flag.StringVar(&c.nodeAccessNetwork, "node-access-network", "10.254.0.0/16", "A network which allows direct access to nodes via VPN. Uses CIDR notation.")
	flag.StringVar(&c.kubernetesAddonsPath, "kubernetes-addons-path", "/opt/addons/kubernetes", "Path to addon manifests. Should contain sub-folders for each addon")
	flag.StringVar(&c.openshiftAddonsPath, "openshift-addons-path", "/opt/addons/openshift", "Path to addon manifests. Should contain sub-folders for each addon")
	flag.StringVar(&c.kubernetesAddonsList, "kubernetes-addons-list", "canal,csi,dns,kube-proxy,openvpn,rbac,kubelet-configmap,default-storage-class,node-exporter,nodelocal-dns-cache", "Comma separated list of Addons to install into every user-cluster")
	flag.StringVar(&c.openshiftAddonsList, "openshift-addons-list", "openvpn,rbac,crd,network,default-storage-class,registry", "Comma separated list of addons to install into every openshift user cluster")
	flag.StringVar(&c.backupContainerFile, "backup-container", "", fmt.Sprintf("[Required] Filepath of a backup container yaml. It must mount a volume named %s from which it reads the etcd backups", backupcontroller.SharedVolumeName))
	flag.StringVar(&c.cleanupContainerFile, "cleanup-container", "", "[Required] Filepath of a cleanup container yaml. The container will be used to cleanup the backup directory for a cluster after it got deleted.")
	flag.StringVar(&c.backupContainerImage, "backup-container-init-image", backupcontroller.DefaultBackupContainerImage, "Docker image to use for the init container in the backup job, must be an etcd v3 image. Only set this if your cluster can not use the public quay.io registry")
	flag.StringVar(&c.backupInterval, "backup-interval", backupcontroller.DefaultBackupInterval, "Interval in which the etcd gets backed up")
	flag.StringVar(&rawEtcdDiskSize, "etcd-disk-size", "5Gi", "Size for the etcd PV's. Only applies to new clusters.")
	flag.StringVar(&c.inClusterPrometheusRulesFile, "in-cluster-prometheus-rules-file", "", "The file containing the custom alerting rules for the prometheus running in the cluster-foo namespaces.")
	flag.BoolVar(&c.inClusterPrometheusDisableDefaultRules, "in-cluster-prometheus-disable-default-rules", false, "A flag indicating whether the default rules for the prometheus running in the cluster-foo namespaces should be deployed.")
	flag.StringVar(&c.dockerPullConfigJSONFile, "docker-pull-config-json-file", "config.json", "The file containing the docker auth config.")
	flag.BoolVar(&c.inClusterPrometheusDisableDefaultScrapingConfigs, "in-cluster-prometheus-disable-default-scraping-configs", false, "A flag indicating whether the default scraping configs for the prometheus running in the cluster-foo namespaces should be deployed.")
	flag.StringVar(&c.inClusterPrometheusScrapingConfigsFile, "in-cluster-prometheus-scraping-configs-file", "", "The file containing the custom scraping configs for the prometheus running in the cluster-foo namespaces.")
	flag.StringVar(&c.monitoringScrapeAnnotationPrefix, "monitoring-scrape-annotation-prefix", "monitoring.kubermatic.io", "The prefix for monitoring annotations in the user cluster. Default: monitoring.kubermatic.io -> monitoring.kubermatic.io/port, monitoring.kubermatic.io/path")
	flag.StringVar(&rawFeatureGates, "feature-gates", "", "A set of key=value pairs that describe feature gates for various features.")
	flag.StringVar(&c.oidcCAFile, "oidc-ca-file", "", "The path to the certificate for the CA that signed your identity provider’s web certificate.")
	flag.StringVar(&c.oidcIssuerURL, "oidc-issuer-url", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.StringVar(&c.oidcIssuerClientID, "oidc-issuer-client-id", "", "Issuer client ID")
	flag.StringVar(&c.oidcIssuerClientSecret, "oidc-issuer-client-secret", "", "OpenID client secret")
	flag.BoolVar(&c.log.Debug, "log-debug", false, "Enables debug logging")
	flag.StringVar(&c.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())
	flag.StringVar(&c.kubermaticImage, "kubermatic-image", resources.DefaultKubermaticImage, "The location from which to pull the Kubermatic image")
	flag.StringVar(&c.dnatControllerImage, "dnatcontroller-image", resources.DefaultDNATControllerImage, "The location of the dnatcontroller-image")
	flag.BoolVar(&c.dynamicDatacenters, "dynamic-datacenters", false, "Whether to enable dynamic datacenters")
	flag.StringVar(&c.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources")
	flag.IntVar(&c.apiServerDefaultReplicas, "apiserver-default-replicas", 2, "The default number of replicas for usercluster api servers")
	flag.BoolVar(&c.apiServerEndpointReconcilingDisabled, "apiserver-reconciling-disabled-by-default", false, "Whether to disable reconciling for the apiserver endpoints by default")
	flag.IntVar(&c.controllerManagerDefaultReplicas, "controller-manager-default-replicas", 1, "The default number of replicas for usercluster controller managers")
	flag.IntVar(&c.schedulerDefaultReplicas, "scheduler-default-replicas", 1, "The default number of replicas for usercluster schedulers")
	flag.IntVar(&c.concurrentClusterUpdate, "max-parallel-reconcile", 10, "The default number of resources updates per cluster")
	c.seedValidationHook.AddFlags(flag.CommandLine)
	flag.Parse()

	featureGates, err := features.NewFeatures(rawFeatureGates)
	if err != nil {
		return c, err
	}
	c.featureGates = featureGates

	etcdDiskSize, err := resource.ParseQuantity(rawEtcdDiskSize)
	if err != nil {
		return c, fmt.Errorf("failed to parse value of flag etcd-disk-size (%q): %v", rawEtcdDiskSize, err)
	}
	c.etcdDiskSize = etcdDiskSize

	if c.overwriteRegistry != "" {
		c.overwriteRegistry = path.Clean(strings.TrimSpace(c.overwriteRegistry))
	}

	return c, nil
}

func (o controllerRunOptions) validate() error {

	if o.featureGates.Enabled(OpenIDAuthPlugin) {
		if len(o.oidcIssuerURL) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-issuer-url\" flag was not specified", OpenIDAuthPlugin)
		}

		if _, err := url.Parse(o.oidcIssuerURL); err != nil {
			return fmt.Errorf("wrong format of \"oidc-issuer-url\" flag, err = %v", err)
		}

		if len(o.oidcIssuerClientID) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-issuer-client-id\" flag was not specified", OpenIDAuthPlugin)
		}

		if len(o.oidcIssuerClientSecret) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-issuer-client-secret\" flag was not specified", OpenIDAuthPlugin)
		}
	}

	if o.masterResources == "" {
		return fmt.Errorf("master-resources path is undefined")
	}

	if o.externalURL == "" {
		return fmt.Errorf("external-url is undefined")
	}

	if o.dc == "" {
		return fmt.Errorf("datacenter-name is undefined")
	}

	if o.backupContainerFile == "" {
		return fmt.Errorf("backup-container is undefined")
	}

	if o.dockerPullConfigJSONFile == "" {
		return fmt.Errorf("docker-pull-config-json-file is undefined")
	}

	if o.monitoringScrapeAnnotationPrefix == "" {
		return fmt.Errorf("moniotring-scrape-annotation-prefix is undefined")
	}

	if o.apiServerDefaultReplicas < 1 {
		return fmt.Errorf("--apiserver-default-replicas must be > 0 (was %d)", o.apiServerDefaultReplicas)
	}
	if o.controllerManagerDefaultReplicas < 1 {
		return fmt.Errorf("--controller-manager-default-replicas must be > 0 (was %d)", o.controllerManagerDefaultReplicas)
	}
	if o.schedulerDefaultReplicas < 1 {
		return fmt.Errorf("--scheduler-default-replicas must be > 0 (was %d)", o.schedulerDefaultReplicas)
	}
	if o.concurrentClusterUpdate < 1 {
		return fmt.Errorf("--max-parallel-reconcile must be > 0 (was %d)", o.concurrentClusterUpdate)
	}
	// Validate OIDC CA file
	if err := o.validateCABundle(); err != nil {
		return fmt.Errorf("validation CA bundle file failed: %v", err)
	}

	// Validate node-port range
	net.ParsePortRangeOrDie(o.nodePortRange)

	// Validate the metrics-server addon is disabled, otherwise it creates conflicts with the resources
	// we create for the metrics-server running in the seed and will render the latter unusable
	if strings.Contains(o.kubernetesAddonsList, "metrics-server") {
		return errors.New("the metrics-server addon must be disabled, it is now deployed inside the seed cluster")
	}

	if err := o.log.Validate(); err != nil {
		return err
	}

	return nil
}

// validateDexSecretWithCABundle
func (o controllerRunOptions) validateCABundle() error {
	if len(o.oidcCAFile) == 0 {
		return nil
	}

	bytes, err := ioutil.ReadFile(o.oidcCAFile)
	if err != nil {
		return fmt.Errorf("failed to read '%s': %v", o.oidcCAFile, err)
	}

	_, err = certutil.ParseCertsPEM(bytes)
	return err
}

// controllerContext holds all controllerRunOptions plus everything that
// needs to be initialized first
type controllerContext struct {
	ctx                  context.Context
	runOptions           controllerRunOptions
	mgr                  manager.Manager
	clientProvider       *client.Provider
	seedGetter           provider.SeedGetter
	dockerPullConfigJSON []byte
	log                  *zap.SugaredLogger
}
