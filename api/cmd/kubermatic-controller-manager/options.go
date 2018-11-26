package main

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/net"
	kubeinformers "k8s.io/client-go/informers"
	certutil "k8s.io/client-go/util/cert"

	"github.com/kubermatic/kubermatic/api/pkg/features"

	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/backup"
)

// DexCAFileName CA file name
const DexCAFileName = "ca.pem"

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
	addonsPath                                       string
	addonsList                                       string
	backupContainerFile                              string
	cleanupContainerFile                             string
	backupContainerImage                             string
	backupInterval                                   string
	etcdDiskSize                                     string
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
	monitoringScrapeAnnotationPrefix                 string
	dockerPullConfigJSONFile                         string

	// OIDC configuration
	skipOidcTLSVerify   bool
	oidcIssuerURL       string
	oidcIssuerClientID  string
	oidcDexSecretCAName string

	featureGates features.FeatureGate
}

func newControllerRunOptions() (controllerRunOptions, error) {
	c := controllerRunOptions{}
	var rawFeatureGates string

	flag.StringVar(&c.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&c.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&c.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal server is running on")
	flag.StringVar(&c.masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&c.externalURL, "external-url", "", "The external url for the apiserver host and the the dc.(Required)")
	flag.StringVar(&c.dc, "datacenter-name", "", "The name of the seed datacenter, the controller is running in. It will be used to build the absolute url for a customer cluster.")
	flag.StringVar(&c.dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&c.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.StringVar(&c.versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&c.updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.IntVar(&c.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&c.overwriteRegistry, "overwrite-registry", "", "registry to use for all images")
	flag.StringVar(&c.nodePortRange, "nodeport-range", "30000-32767", "NodePort range to use for new clusters. It must be within the NodePort range of the seed-cluster")
	flag.StringVar(&c.nodeAccessNetwork, "node-access-network", "10.254.0.0/16", "A network which allows direct access to nodes via VPN. Uses CIDR notation.")
	flag.StringVar(&c.addonsPath, "addons-path", "/opt/addons", "Path to addon manifests. Should contain sub-folders for each addon")
	flag.StringVar(&c.addonsList, "addons-list", "canal,dashboard,dns,kube-proxy,openvpn,rbac,kubelet-configmap,default-storage-class", "Comma separated list of Addons to install into every user-cluster")
	flag.StringVar(&c.backupContainerFile, "backup-container", "", fmt.Sprintf("[Required] Filepath of a backup container yaml. It must mount a volume named %s from which it reads the etcd backups", backupcontroller.SharedVolumeName))
	flag.StringVar(&c.cleanupContainerFile, "cleanup-container", "", "[Required] Filepath of a cleanup container yaml. The container will be used to cleanup the backup directory for a cluster after it got deleted.")
	flag.StringVar(&c.backupContainerImage, "backup-container-init-image", backupcontroller.DefaultBackupContainerImage, "Docker image to use for the init container in the backup job, must be an etcd v3 image. Only set this if your cluster can not use the public quay.io registry")
	flag.StringVar(&c.backupInterval, "backup-interval", backupcontroller.DefaultBackupInterval, "Interval in which the etcd gets backed up")
	flag.StringVar(&c.etcdDiskSize, "etcd-disk-size", "5Gi", "Size for the etcd PV's. Only applies to new clusters.")
	flag.StringVar(&c.inClusterPrometheusRulesFile, "in-cluster-prometheus-rules-file", "", "The file containing the custom alerting rules for the prometheus running in the cluster-foo namespaces.")
	flag.BoolVar(&c.inClusterPrometheusDisableDefaultRules, "in-cluster-prometheus-disable-default-rules", false, "A flag indicating whether the default rules for the prometheus running in the cluster-foo namespaces should be deployed.")
	flag.StringVar(&c.dockerPullConfigJSONFile, "docker-pull-config-json-file", "config.json", "The file containing the docker auth config.")
	flag.BoolVar(&c.inClusterPrometheusDisableDefaultScrapingConfigs, "in-cluster-prometheus-disable-default-scraping-configs", false, "A flag indicating whether the default scraping configs for the prometheus running in the cluster-foo namespaces should be deployed.")
	flag.StringVar(&c.inClusterPrometheusScrapingConfigsFile, "in-cluster-prometheus-scraping-configs-file", "", "The file containing the custom scraping configs for the prometheus running in the cluster-foo namespaces.")
	flag.StringVar(&c.monitoringScrapeAnnotationPrefix, "monitoring-scrape-annotation-prefix", "monitoring.kubermatic.io", "The prefix for monitoring annotations in the user cluster. Default: monitoring.kubermatic.io -> monitoring.kubermatic.io/port, monitoring.kubermatic.io/path")
	flag.StringVar(&rawFeatureGates, "feature-gates", "", "A set of key=value pairs that describe feature gates for various features.")
	flag.BoolVar(&c.skipOidcTLSVerify, "skip-oidc-tls-verify", false, "The flag used to skip TLS verification against dex.")
	flag.StringVar(&c.oidcIssuerURL, "oidc-issuer-url", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.StringVar(&c.oidcIssuerClientID, "oidc-issuer-client-id", "", "Issuer client ID")
	flag.StringVar(&c.oidcDexSecretCAName, "oidc-dex-secret-ca-name", "oauth/dex-ca", "The secret name which contains CA bundle for TLS verification against dex. This flag is optional. The default value is oauth/dex-ca.")
	flag.Parse()

	featureGates, err := features.NewFeatures(rawFeatureGates)
	if err != nil {
		return c, err
	}
	c.featureGates = featureGates
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

		if !o.skipOidcTLSVerify {
			if len(o.oidcDexSecretCAName) == 0 {
				return fmt.Errorf("%s feature with TLS verification is enabled but \"oidc-dex-secret-ca-name\" flag was not specified", OpenIDAuthPlugin)
			}

			splitResult := strings.Split(o.oidcDexSecretCAName, "/")
			if len(splitResult) != 2 {
				return fmt.Errorf("wrong \"oidc-dex-secret-ca-name\" field format . The format should be \"namespace/secret-name\" but was %s: ", o.oidcDexSecretCAName)
			}
		}

	} else {
		// don't pass OpenID issuer flags if OpenIDAuthPlugin disabled
		if len(o.oidcIssuerURL) > 0 {
			return fmt.Errorf("%s feature is disabled but \"oidc-issuer-url\" flag was specified, please remove it", OpenIDAuthPlugin)
		}

		if len(o.oidcIssuerClientID) > 0 {
			return fmt.Errorf("%s feature is disabled but \"oidc-issuer-client-id\" flag was specified, please remove it", OpenIDAuthPlugin)
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

	// Validate etcd disk size
	resource.MustParse(o.etcdDiskSize)

	// Validate node-port range
	net.ParsePortRangeOrDie(o.nodePortRange)

	// dcFile, versionFile, updatesFile are required by cluster controller
	// the following code ensures that the files are available and fails fast if not.
	_, err := provider.LoadDatacentersMeta(o.dcFile)
	if err != nil {
		return fmt.Errorf("failed to load datacenter yaml %q: %v", o.dcFile, err)
	}
	return nil
}

func (o controllerRunOptions) validateDexSecretWithCABundle(client kubernetes.Interface) error {

	if o.featureGates.Enabled(OpenIDAuthPlugin) && !o.skipOidcTLSVerify {

		splitResult := strings.Split(o.oidcDexSecretCAName, "/")

		// Retrieve elements.
		namespace := splitResult[0]
		secretName := splitResult[1]

		rawSecret, err := client.CoreV1().Secrets(namespace).Get(secretName, metaV1.GetOptions{})
		if err != nil {
			return fmt.Errorf("%s feature with TLS verification is enabled but secret with name %s from namespace %s containing CA bundle cannot be find", OpenIDAuthPlugin, secretName, namespace)
		}

		_, err = certutil.ParseCertsPEM(rawSecret.Data[DexCAFileName])
		if err != nil {
			return fmt.Errorf("%s feature with TLS verification is enabled but secret got an invalid cert: %v", OpenIDAuthPlugin, err)
		}
	}

	return nil
}

type controllerContext struct {
	runOptions                controllerRunOptions
	stopCh                    <-chan struct{}
	kubeClient                kubernetes.Interface
	kubermaticClient          kubermaticclientset.Interface
	kubermaticInformerFactory kubermaticinformers.SharedInformerFactory
	kubeInformerFactory       kubeinformers.SharedInformerFactory
}
