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
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type controllerRunOptions struct {
	internalAddr            string
	enableLeaderElection    bool
	leaderElectionNamespace string

	externalURL              string
	seedName                 string
	workerName               string
	workerCount              int
	overwriteRegistry        string
	nodeAccessNetwork        string
	addonsPath               string
	backupInterval           string
	etcdDiskSize             resource.Quantity
	dockerPullConfigJSONFile string
	kubermaticImage          string
	etcdLauncherImage        string
	dnatControllerImage      string
	namespace                string
	concurrentClusterUpdate  int
	addonEnforceInterval     int
	systemAppEnforceInterval int
	caBundle                 *certificates.CABundle

	// for development purposes, a local configuration file
	// can be used to provide the KubermaticConfiguration
	kubermaticConfiguration *kubermaticv1.KubermaticConfiguration

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

	// disabledCollectors is a list of comma-separated collectors that should be disabled
	disabledCollectors string
}

func newControllerRunOptions() (controllerRunOptions, error) {
	c := controllerRunOptions{
		featureGates: features.FeatureGate{},
		// Default IP used by tunneling agents
		tunnelingAgentIP: flagopts.IPValue{IP: net.ParseIP(resources.DefaultTunnelingAgentIP)},
	}

	var (
		rawEtcdDiskSize string
		caBundleFile    string
		configFile      string
	)

	flag.BoolVar(&c.enableLeaderElection, "enable-leader-election", true, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&c.leaderElectionNamespace, "leader-election-namespace", "", "Leader election namespace. In-cluster discovery will be attempted in such case.")
	flag.StringVar(&c.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal server is running on")
	flag.StringVar(&c.externalURL, "external-url", "", "The external url for the apiserver host and the the dc.(Required)")
	flag.StringVar(&c.seedName, "seed-name", "", "The name of the seed this controller is running in. It will be used to build the absolute url for a user cluster.")
	flag.StringVar(&c.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&c.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&c.overwriteRegistry, "overwrite-registry", "", "registry to use for all images")
	flag.StringVar(&c.nodeAccessNetwork, "node-access-network", kubermaticv1.DefaultNodeAccessNetwork, "A network which allows direct access to nodes via VPN. Uses CIDR notation.")
	flag.StringVar(&c.addonsPath, "addons-path", "/opt/addons", "Path to addon manifests. Should contain sub-folders for each addon")
	flag.StringVar(&c.backupInterval, "backup-interval", defaulting.DefaultBackupInterval, "Interval in which the etcd gets backed up")
	flag.StringVar(&rawEtcdDiskSize, "etcd-disk-size", "5Gi", "Size for the etcd PV's. Only applies to new clusters.")
	flag.StringVar(&c.dockerPullConfigJSONFile, "docker-pull-config-json-file", "", "The file containing the docker auth config.")
	flag.Var(&c.featureGates, "feature-gates", "A set of key=value pairs that describe feature gates for various features.")
	flag.StringVar(&c.oidcIssuerURL, "oidc-issuer-url", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.StringVar(&c.oidcIssuerClientID, "oidc-issuer-client-id", "", "Issuer client ID")
	flag.StringVar(&c.oidcIssuerClientSecret, "oidc-issuer-client-secret", "", "OpenID client secret")
	flag.StringVar(&c.kubermaticImage, "kubermatic-image", defaulting.DefaultKubermaticImage, "The location from which to pull the Kubermatic image")
	flag.StringVar(&c.etcdLauncherImage, "etcd-launcher-image", defaulting.DefaultEtcdLauncherImage, "The location from which to pull the etcd launcher image")
	flag.StringVar(&c.dnatControllerImage, "dnatcontroller-image", defaulting.DefaultDNATControllerImage, "The location of the dnatcontroller-image")
	flag.StringVar(&c.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for Seed resources")
	flag.IntVar(&c.concurrentClusterUpdate, "max-parallel-reconcile", 10, "The default number of resources updates per cluster")
	flag.IntVar(&c.addonEnforceInterval, "addon-enforce-interval", 5, "Check and ensure default usercluster addons are deployed every interval in minutes. Set to 0 to disable.")
	flag.IntVar(&c.systemAppEnforceInterval, "system-app-enforce-interval", 5, "Check and ensure system ApplicationInstallations in user cluster every interval in minutes. Set to 0 to disable.")
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
	flag.StringVar(&configFile, "kubermatic-configuration-file", "", "(for development only) path to a KubermaticConfiguration YAML file")
	flag.StringVar(&c.disabledCollectors, "disabled-collectors", "", "Disables metrics collectors in the seed. The value should be a comma-separated list of collector names.")

	addFlags(flag.CommandLine)
	flag.Parse()

	etcdDiskSize, err := resource.ParseQuantity(rawEtcdDiskSize)
	if err != nil {
		return c, fmt.Errorf("failed to parse value of flag etcd-disk-size (%q): %w", rawEtcdDiskSize, err)
	}
	c.etcdDiskSize = etcdDiskSize

	if c.overwriteRegistry != "" {
		c.overwriteRegistry = path.Clean(strings.TrimSpace(c.overwriteRegistry))
	}

	if configFile != "" {
		if c.kubermaticConfiguration, err = loadKubermaticConfiguration(configFile); err != nil {
			return c, fmt.Errorf("invalid KubermaticConfiguration: %w", err)
		}
	}

	caBundle, err := certificates.NewCABundleFromFile(caBundleFile)
	if err != nil {
		return c, fmt.Errorf("invalid CA bundle file (%q): %w", caBundleFile, err)
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
			return fmt.Errorf("wrong format of \"oidc-issuer-url\" flag: %w", err)
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

	if o.seedName == "" {
		return fmt.Errorf("seed-name is undefined")
	}

	return nil
}

// controllerContext holds all controllerRunOptions plus everything that
// needs to be initialized first.
type controllerContext struct {
	ctx                  context.Context
	runOptions           controllerRunOptions
	mgr                  manager.Manager
	clientProvider       *client.Provider
	seedGetter           provider.SeedGetter
	configGetter         provider.KubermaticConfigurationGetter
	dockerPullConfigJSON []byte
	log                  *zap.SugaredLogger
	versions             kubermatic.Versions
}

func loadKubermaticConfiguration(filename string) (*kubermaticv1.KubermaticConfiguration, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer f.Close()

	config := &kubermaticv1.KubermaticConfiguration{}
	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse file as YAML: %w", err)
	}

	defaulted, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to process: %w", err)
	}

	return defaulted, nil
}
