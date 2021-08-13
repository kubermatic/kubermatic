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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"path"
	"strings"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	backupcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/backup"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/kubermatic/v2/pkg/webhook"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knet "k8s.io/apimachinery/pkg/util/net"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"
)

type controllerRunOptions struct {
	internalAddr            string
	enableLeaderElection    bool
	leaderElectionNamespace string

	externalURL                                      string
	dc                                               string
	workerName                                       string
	versionsFile                                     string
	updatesFile                                      string
	workerCount                                      int
	overwriteRegistry                                string
	nodePortRange                                    string
	nodeAccessNetwork                                string
	kubernetesAddonsPath                             string
	kubernetesAddons                                 kubermaticv1.AddonList
	backupContainerFile                              string
	backupDeleteContainerFile                        string
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
	kubermaticImage                                  string
	etcdLauncherImage                                string
	enableEtcdBackupRestoreController                bool
	dnatControllerImage                              string
	namespace                                        string
	apiServerDefaultReplicas                         int
	apiServerEndpointReconcilingDisabled             bool
	controllerManagerDefaultReplicas                 int
	schedulerDefaultReplicas                         int
	admissionWebhook                                 webhook.Options
	concurrentClusterUpdate                          int
	addonEnforceInterval                             int
	caBundle                                         *certificates.CABundle

	// OIDC configuration
	oidcIssuerURL          string
	oidcIssuerClientID     string
	oidcIssuerClientSecret string

	// Used in the tunneling expose strategy
	tunnelingAgentIP flagopts.IPValue

	featureGates features.FeatureGate

	// MLA configuration
	enableUserClusterMLA  bool
	mlaNamespace          string
	grafanaURL            string
	grafanaHeaderName     string
	grafanaSecret         string
	cortexAlertmanagerURL string
	cortexRulerURL        string
	lokiRulerURL          string

	// Machine Controller configuration
	machineControllerImageTag        string
	machineControllerImageRepository string
}

func newControllerRunOptions() (controllerRunOptions, error) {
	c := controllerRunOptions{
		featureGates: features.FeatureGate{},
		// Default IP used by tunneling agents
		tunnelingAgentIP: flagopts.IPValue{IP: net.ParseIP("192.168.30.10")},
	}

	var (
		rawEtcdDiskSize             string
		caBundleFile                string
		defaultKubernetesAddonsList string
		defaultKubernetesAddonsFile string
	)

	flag.BoolVar(&c.enableLeaderElection, "enable-leader-election", true, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&c.leaderElectionNamespace, "leader-election-namespace", "", "Leader election namespace. In-cluster discovery will be attempted in such case.")
	flag.StringVar(&c.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal server is running on")
	flag.StringVar(&c.externalURL, "external-url", "", "The external url for the apiserver host and the the dc.(Required)")
	flag.StringVar(&c.dc, "datacenter-name", "", "The name of the seed datacenter, the controller is running in. It will be used to build the absolute url for a customer cluster.")
	flag.StringVar(&c.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.StringVar(&c.versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&c.updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.IntVar(&c.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&c.overwriteRegistry, "overwrite-registry", "", "registry to use for all images")
	flag.StringVar(&c.nodePortRange, "nodeport-range", resources.DefaultNodePortRange, "Deprecated: configure defaultComponentSettings on Seed resource. NodePort range to use for new clusters. It must be within the NodePort range of the seed-cluster")
	flag.StringVar(&c.nodeAccessNetwork, "node-access-network", kubermaticv1.DefaultNodeAccessNetwork, "A network which allows direct access to nodes via VPN. Uses CIDR notation.")
	flag.StringVar(&c.kubernetesAddonsPath, "kubernetes-addons-path", "/opt/addons/kubernetes", "Path to addon manifests. Should contain sub-folders for each addon")
	flag.StringVar(&defaultKubernetesAddonsList, "kubernetes-addons-list", "", "Comma separated list of Addons to install into every user-cluster. Mutually exclusive with `--kubernetes-addons-file`")
	flag.StringVar(&defaultKubernetesAddonsFile, "kubernetes-addons-file", "", "File that contains a list of default kubernetes addons. Mutually exclusive with `--kubernetes-addons-list`")
	flag.StringVar(&c.backupContainerFile, "backup-container", "", fmt.Sprintf("[Required] Filepath of a backup container yaml. It must mount a volume named %s from which it reads the etcd backups", backupcontroller.SharedVolumeName))
	flag.StringVar(&c.backupDeleteContainerFile, "backup-delete-container", "", "Filepath of a backup deletion container yaml. It receives the name of the backup to delete in an env variable ($BACKUP_TO_DELETE). If not specified, the backup container must handle deletion.")
	flag.StringVar(&c.cleanupContainerFile, "cleanup-container", "", "(Only required for the old backup controller) Filepath of a cleanup container yaml. The container will be used to cleanup the backup directory for a cluster after it got deleted.")
	flag.StringVar(&c.backupContainerImage, "backup-container-init-image", backupcontroller.DefaultBackupContainerImage, "Docker image to use for the init container in the backup job, must be an etcd v3 image. Only set this if your cluster can not use the public quay.io registry")
	flag.StringVar(&c.backupInterval, "backup-interval", backupcontroller.DefaultBackupInterval, "Interval in which the etcd gets backed up")
	flag.StringVar(&rawEtcdDiskSize, "etcd-disk-size", "5Gi", "Size for the etcd PV's. Only applies to new clusters.")
	flag.StringVar(&c.inClusterPrometheusRulesFile, "in-cluster-prometheus-rules-file", "", "The file containing the custom alerting rules for the prometheus running in the cluster-foo namespaces.")
	flag.BoolVar(&c.inClusterPrometheusDisableDefaultRules, "in-cluster-prometheus-disable-default-rules", false, "A flag indicating whether the default rules for the prometheus running in the cluster-foo namespaces should be deployed.")
	flag.StringVar(&c.dockerPullConfigJSONFile, "docker-pull-config-json-file", "", "The file containing the docker auth config.")
	flag.BoolVar(&c.inClusterPrometheusDisableDefaultScrapingConfigs, "in-cluster-prometheus-disable-default-scraping-configs", false, "A flag indicating whether the default scraping configs for the prometheus running in the cluster-foo namespaces should be deployed.")
	flag.StringVar(&c.inClusterPrometheusScrapingConfigsFile, "in-cluster-prometheus-scraping-configs-file", "", "The file containing the custom scraping configs for the prometheus running in the cluster-foo namespaces.")
	flag.StringVar(&c.monitoringScrapeAnnotationPrefix, "monitoring-scrape-annotation-prefix", "monitoring.kubermatic.io", "The prefix for monitoring annotations in the user cluster. Default: monitoring.kubermatic.io -> monitoring.kubermatic.io/port, monitoring.kubermatic.io/path")
	flag.Var(&c.featureGates, "feature-gates", "A set of key=value pairs that describe feature gates for various features.")
	flag.StringVar(&c.oidcIssuerURL, "oidc-issuer-url", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.StringVar(&c.oidcIssuerClientID, "oidc-issuer-client-id", "", "Issuer client ID")
	flag.StringVar(&c.oidcIssuerClientSecret, "oidc-issuer-client-secret", "", "OpenID client secret")
	flag.StringVar(&c.kubermaticImage, "kubermatic-image", resources.DefaultKubermaticImage, "The location from which to pull the Kubermatic image")
	flag.StringVar(&c.etcdLauncherImage, "etcd-launcher-image", resources.DefaultEtcdLauncherImage, "The location from which to pull the etcd launcher image")
	flag.BoolVar(&c.enableEtcdBackupRestoreController, "enable-etcd-backups-restores", false, "Whether to enable the new etcd backup and restore controllers")
	flag.StringVar(&c.dnatControllerImage, "dnatcontroller-image", resources.DefaultDNATControllerImage, "The location of the dnatcontroller-image")
	flag.StringVar(&c.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources")
	flag.IntVar(&c.apiServerDefaultReplicas, "apiserver-default-replicas", 2, "Deprecated: configure defaultComponentSettings on Seed resource. The default number of replicas for usercluster api servers")
	flag.BoolVar(&c.apiServerEndpointReconcilingDisabled, "apiserver-reconciling-disabled-by-default", false, "Deprecated: configure defaultComponentSettings on Seed resource. Whether to disable reconciling for the apiserver endpoints by default")
	flag.IntVar(&c.controllerManagerDefaultReplicas, "controller-manager-default-replicas", 1, "Deprecated: configure defaultComponentSettings on Seed resource. The default number of replicas for usercluster controller managers")
	flag.IntVar(&c.schedulerDefaultReplicas, "scheduler-default-replicas", 1, "Deprecated: configure defaultComponentSettings on Seed resource. The default number of replicas for usercluster schedulers")
	flag.IntVar(&c.concurrentClusterUpdate, "max-parallel-reconcile", 10, "The default number of resources updates per cluster")
	flag.IntVar(&c.addonEnforceInterval, "addon-enforce-interval", 5, "Check and ensure default usercluster addons are deployed every interval in minutes. Set to 0 to disable.")
	flag.StringVar(&caBundleFile, "ca-bundle", "", "File containing the PEM-encoded CA bundle for all userclusters")
	flag.Var(&c.tunnelingAgentIP, "tunneling-agent-ip", "The address used by the tunneling agents.")
	flag.BoolVar(&c.enableUserClusterMLA, "enable-user-cluster-mla", false, "Enables user cluster MLA (Monitoring, Logging & Alerting) stack in the seed.")
	flag.StringVar(&c.mlaNamespace, "mla-namespace", "mla", "The namespace in which the user cluster MLA stack is running.")
	flag.StringVar(&c.grafanaURL, "grafana-url", "http://grafana.mla.svc.cluster.local", "The URL of Grafana instance which in running for MLA stack.")
	flag.StringVar(&c.grafanaHeaderName, "grafana-header-name", "X-Forwarded-Email", "Grafana Auth Proxy HTTP Header that will contain the username or email")
	flag.StringVar(&c.grafanaSecret, "grafana-secret-name", "mla/grafana", "Grafana secret name in format namespace/secretname, that contains basic auth info")
	flag.StringVar(&c.cortexAlertmanagerURL, "cortex-alertmanager-url", "http://cortex-alertmanager.mla.svc.cluster.local:8080", "The URL of cortex alertmanager which is running for MLA stack.")
	flag.StringVar(&c.cortexRulerURL, "cortex-ruler-url", "http://cortex-ruler.mla.svc.cluster.local:8080", "The URL of cortex ruler which is running for MLA stack.")
	flag.StringVar(&c.lokiRulerURL, "loki-ruler-url", "http://loki-distributed-ruler.mla.svc.cluster.local:3100", "The URL of loki ruler which is running for MLA stack.")
	flag.StringVar(&c.machineControllerImageTag, "machine-controller-image-tag", "", "The Machine Controller image tag.")
	flag.StringVar(&c.machineControllerImageRepository, "machine-controller-image-repository", "", "The Machine Controller image repository.")
	c.admissionWebhook.AddFlags(flag.CommandLine, true)
	addFlags(flag.CommandLine)
	flag.Parse()

	if err := c.admissionWebhook.Validate(); err != nil {
		return c, fmt.Errorf("invalid admission webhook configuration: %v", err)
	}

	etcdDiskSize, err := resource.ParseQuantity(rawEtcdDiskSize)
	if err != nil {
		return c, fmt.Errorf("failed to parse value of flag etcd-disk-size (%q): %v", rawEtcdDiskSize, err)
	}
	c.etcdDiskSize = etcdDiskSize

	if c.overwriteRegistry != "" {
		c.overwriteRegistry = path.Clean(strings.TrimSpace(c.overwriteRegistry))
	}

	c.kubernetesAddons, err = loadAddons(defaultKubernetesAddonsList, defaultKubernetesAddonsFile)
	if err != nil {
		return c, err
	}

	caBundle, err := certificates.NewCABundleFromFile(caBundleFile)
	if err != nil {
		return c, fmt.Errorf("invalid CA bundle file (%q): %v", caBundleFile, err)
	}
	c.caBundle = caBundle

	return c, nil
}

func (o controllerRunOptions) validate() error {
	if o.featureGates.Enabled(features.OpenIDAuthPlugin) {
		if len(o.oidcIssuerURL) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-issuer-url\" flag was not specified", features.OpenIDAuthPlugin)
		}

		if _, err := url.Parse(o.oidcIssuerURL); err != nil {
			return fmt.Errorf("wrong format of \"oidc-issuer-url\" flag, err = %v", err)
		}

		if len(o.oidcIssuerClientID) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-issuer-client-id\" flag was not specified", features.OpenIDAuthPlugin)
		}

		if len(o.oidcIssuerClientSecret) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-issuer-client-secret\" flag was not specified", features.OpenIDAuthPlugin)
		}
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

	if o.monitoringScrapeAnnotationPrefix == "" {
		return fmt.Errorf("monitoring-scrape-annotation-prefix is undefined")
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

	// Validate node-port range
	if _, err := knet.ParsePortRange(o.nodePortRange); err != nil {
		return fmt.Errorf("failed to parse nodePortRange: %v", err)
	}

	// Validate the metrics-server addon is disabled, otherwise it creates conflicts with the resources
	// we create for the metrics-server running in the seed and will render the latter unusable
	for _, addon := range o.kubernetesAddons.Items {
		if addon.Name == "metrics-server" {
			return errors.New("the metrics-server addon must be disabled, it is now deployed inside the seed cluster")
		}
	}

	return nil
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
	versions             kubermatic.Versions
}

func loadAddons(listOpt, fileOpt string) (kubermaticv1.AddonList, error) {
	addonList := kubermaticv1.AddonList{}
	if listOpt != "" && fileOpt != "" {
		return addonList, errors.New("addon-list and addon-path are mutually exclusive")
	}
	if listOpt != "" {
		for _, addonName := range strings.Split(listOpt, ",") {
			labels, err := getAddonDefaultLabels(addonName)
			if err != nil {
				return addonList, fmt.Errorf("failed to get default addon labels: %v", err)
			}
			addonList.Items = append(addonList.Items, kubermaticv1.Addon{ObjectMeta: metav1.ObjectMeta{Name: addonName, Labels: labels}})
		}
	}
	if fileOpt != "" {
		data, err := ioutil.ReadFile(fileOpt)
		if err != nil {
			return addonList, fmt.Errorf("failed to read %q: %v", fileOpt, err)
		}
		if err := yaml.Unmarshal(data, &addonList); err != nil {
			return addonList, fmt.Errorf("failed to parse file from addon-path %q: %v", fileOpt, err)
		}
	}

	return addonList, nil
}

func getAddonDefaultLabels(addonName string) (map[string]string, error) {
	defaultAddonList := kubermaticv1.AddonList{}
	if err := yaml.Unmarshal([]byte(common.DefaultKubernetesAddons), &defaultAddonList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal default addon list: %v", err)
	}
	for _, addon := range defaultAddonList.Items {
		if addon.Name == addonName {
			return addon.Labels, nil
		}
	}
	return nil, nil
}
