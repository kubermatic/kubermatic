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
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/validation"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knet "k8s.io/apimachinery/pkg/util/net"
	certutil "k8s.io/client-go/util/cert"
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
	openshiftAddonsPath                              string
	kubernetesAddons                                 kubermaticv1.AddonList
	openshiftAddons                                  kubermaticv1.AddonList
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
	kubermaticImage                                  string
	etcdLauncherImage                                string
	dnatControllerImage                              string
	namespace                                        string
	apiServerDefaultReplicas                         int
	apiServerEndpointReconcilingDisabled             bool
	controllerManagerDefaultReplicas                 int
	schedulerDefaultReplicas                         int
	validationWebhook                                validation.WebhookOpts
	concurrentClusterUpdate                          int
	addonEnforceInterval                             int
	controlPlaneLeaderElectLeaseDurationSeconds      int
	controlPlaneLeaderElectRenewDeadlineSeconds      int
	controlPlaneLeaderElectRetryPeriodSeconds        int

	// OIDC configuration
	oidcCAFile             string
	oidcIssuerURL          string
	oidcIssuerClientID     string
	oidcIssuerClientSecret string

	// Used in the tunneling expose strategy
	tunnelingAgentIP flagopts.IPValue

	featureGates features.FeatureGate
}

func newControllerRunOptions() (controllerRunOptions, error) {
	c := controllerRunOptions{
		featureGates: features.FeatureGate{},
		// Default IP used by tunneling agents
		tunnelingAgentIP: flagopts.IPValue{IP: net.ParseIP("192.168.30.10")},
	}
	var rawEtcdDiskSize string
	var defaultKubernetesAddonsList string
	var defaultKubernetesAddonsFile string
	var defaultOpenshiftAddonList string
	var defaultOpenshiftAddonsFile string

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
	flag.StringVar(&c.nodePortRange, "nodeport-range", "30000-32767", "NodePort range to use for new clusters. It must be within the NodePort range of the seed-cluster")
	flag.StringVar(&c.nodeAccessNetwork, "node-access-network", kubermaticv1.DefaultNodeAccessNetwork, "A network which allows direct access to nodes via VPN. Uses CIDR notation.")
	flag.StringVar(&c.kubernetesAddonsPath, "kubernetes-addons-path", "/opt/addons/kubernetes", "Path to addon manifests. Should contain sub-folders for each addon")
	flag.StringVar(&c.openshiftAddonsPath, "openshift-addons-path", "/opt/addons/openshift", "Path to addon manifests. Should contain sub-folders for each addon")
	flag.StringVar(&defaultKubernetesAddonsList, "kubernetes-addons-list", "", "Comma separated list of Addons to install into every user-cluster. Mutually exclusive with `--kubernetes-addons-file`")
	flag.StringVar(&defaultKubernetesAddonsFile, "kubernetes-addons-file", "", "File that contains a list of default kubernetes addons. Mutually exclusive with `--kubernetes-addons-list`")
	flag.StringVar(&defaultOpenshiftAddonList, "openshift-addons-list", "", "Comma separated list of addons to install into every openshift user cluster. Mutually exclusive with `--openshift-addons-file`")
	flag.StringVar(&defaultOpenshiftAddonsFile, "openshift-addons-file", "", "File that contains a list of default openshift addons. Mutually exclusive with `--openshift-addons-list`")
	flag.StringVar(&c.backupContainerFile, "backup-container", "", fmt.Sprintf("[Required] Filepath of a backup container yaml. It must mount a volume named %s from which it reads the etcd backups", backupcontroller.SharedVolumeName))
	flag.StringVar(&c.cleanupContainerFile, "cleanup-container", "", "[Required] Filepath of a cleanup container yaml. The container will be used to cleanup the backup directory for a cluster after it got deleted.")
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
	flag.StringVar(&c.oidcCAFile, "oidc-ca-file", "", "The path to the certificate for the CA that signed your identity provider’s web certificate.")
	flag.StringVar(&c.oidcIssuerURL, "oidc-issuer-url", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.StringVar(&c.oidcIssuerClientID, "oidc-issuer-client-id", "", "Issuer client ID")
	flag.StringVar(&c.oidcIssuerClientSecret, "oidc-issuer-client-secret", "", "OpenID client secret")
	flag.StringVar(&c.kubermaticImage, "kubermatic-image", resources.DefaultKubermaticImage, "The location from which to pull the Kubermatic image")
	flag.StringVar(&c.etcdLauncherImage, "etcd-launcher-image", resources.DefaultEtcdLauncherImage, "The location from which to pull the etcd launcher image")
	flag.StringVar(&c.dnatControllerImage, "dnatcontroller-image", resources.DefaultDNATControllerImage, "The location of the dnatcontroller-image")
	flag.StringVar(&c.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources")
	flag.IntVar(&c.apiServerDefaultReplicas, "apiserver-default-replicas", 2, "The default number of replicas for usercluster api servers")
	flag.BoolVar(&c.apiServerEndpointReconcilingDisabled, "apiserver-reconciling-disabled-by-default", false, "Whether to disable reconciling for the apiserver endpoints by default")
	flag.IntVar(&c.controllerManagerDefaultReplicas, "controller-manager-default-replicas", 1, "The default number of replicas for usercluster controller managers")
	flag.IntVar(&c.schedulerDefaultReplicas, "scheduler-default-replicas", 1, "The default number of replicas for usercluster schedulers")
	flag.IntVar(&c.concurrentClusterUpdate, "max-parallel-reconcile", 10, "The default number of resources updates per cluster")
	flag.IntVar(&c.addonEnforceInterval, "addon-enforce-interval", 5, "Check and ensure default usercluster addons are deployed every interval in minutes. Set to 0 to disable.")
	flag.IntVar(&c.controlPlaneLeaderElectLeaseDurationSeconds, "control-plane-leader-elect-lease-duration", 0, "The lease duration in seconds used by control plane components using leader election (i.e. controller-manager and schedule). The default value for the component is used when equal or less than 0.")
	flag.IntVar(&c.controlPlaneLeaderElectRenewDeadlineSeconds, "control-plane-leader-elect-renew-deadline", 0, "The lease renew deadline in seconds by control plane components using leader election (i.e. controller-manager and schedule). Should be smaller or equal than control-plane-leader-elect-lease-duration. The default value for the component is used when equal or less than 0.")
	flag.IntVar(&c.controlPlaneLeaderElectRetryPeriodSeconds, "control-plane-leader-elect-retry-period", 0, "The duration in seconds that control plane components using leader election (i.e. controller-manager and schedule) should wait between attempting acquisition and renewal of a leadership. The default value for the component is used when equal or less than 0.")
	flag.Var(&c.tunnelingAgentIP, "tunneling-agent-ip", "The address used by the tunneling agents.")
	c.validationWebhook.AddFlags(flag.CommandLine, true)
	addFlags(flag.CommandLine)
	flag.Parse()

	if err := c.validationWebhook.Validate(); err != nil {
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

	c.openshiftAddons, err = loadAddons(defaultOpenshiftAddonList, defaultOpenshiftAddonsFile)
	if err != nil {
		return c, err
	}

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
	// Validate OIDC CA file
	if err := o.validateCABundle(); err != nil {
		return fmt.Errorf("validation CA bundle file failed: %v", err)
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

	if err := validation.ValidateLeaderElectionSettings(o.controlPlaneLeaderElectionSettings()); err != nil {
		return fmt.Errorf("the control plane leader election settings are not valid: %w", err)
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

func (o controllerRunOptions) nodeLocalDNSCacheEnabled() bool {
	for _, addon := range o.kubernetesAddons.Items {
		if addon.Name == "nodelocal-dns-cache" {
			return true
		}
	}
	return false
}

func (o controllerRunOptions) controlPlaneLeaderElectionSettings() kubermaticv1.LeaderElectionSettings {
	toPointerIfPositive := func(v int) *int32 {
		if v > 0 {
			c := int32(v)
			return &c
		}
		return nil
	}
	return kubermaticv1.LeaderElectionSettings{
		LeaseDurationSeconds: toPointerIfPositive(o.controlPlaneLeaderElectLeaseDurationSeconds),
		RenewDeadlineSeconds: toPointerIfPositive(o.controlPlaneLeaderElectRenewDeadlineSeconds),
		RetryPeriodSeconds:   toPointerIfPositive(o.controlPlaneLeaderElectRetryPeriodSeconds),
	}
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
