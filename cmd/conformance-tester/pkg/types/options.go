/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package types

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/go-openapi/runtime"
	"go.uber.org/zap"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/resources"
	kubermativsemver "k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test"
	apiclient "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Options represent combination of flags and ENV options.
type Options struct {
	Client     string
	NamePrefix string

	// these flags control the test scenario matrix;
	// the Enabled... and Exclude... sets are combined
	// together to result in the final ... set

	Providers            sets.String
	Releases             sets.String
	Versions             []*kubermativsemver.Semver
	EnableDistributions  sets.String
	ExcludeDistributions sets.String
	Distributions        sets.String
	EnableTests          sets.String
	ExcludeTests         sets.String
	Tests                sets.String
	ContainerRuntimes    sets.String

	// additional settings identical for all scenarios

	OperatingSystemManagerEnabled bool
	DualStackEnabled              bool
	KonnectivityEnabled           bool
	PspEnabled                    bool
	ScenarioOptions               sets.String

	// additional settings

	ControlPlaneReadyWaitTimeout time.Duration
	NodeReadyTimeout             time.Duration
	CustomTestTimeout            time.Duration
	UserClusterPollInterval      time.Duration
	DeleteClusterAfterTests      bool
	WaitForClusterDeletion       bool
	NodeCount                    int
	PublicKeys                   [][]byte
	ReportsRoot                  string
	LogDirectory                 string
	SeedClusterClient            ctrlruntimeclient.Client
	SeedGeneratedClient          kubernetes.Interface
	ClusterClientProvider        *clusterclient.Provider
	RepoRoot                     string
	Seed                         *kubermaticv1.Seed
	SeedRestConfig               *rest.Config
	ClusterParallelCount         int
	HomeDir                      string
	ExistingClusterLabel         string
	CreateOIDCToken              bool
	DexHelmValuesFile            string
	KubermaticNamespace          string
	KubermaticEndpoint           string
	KubermaticSeedName           string
	KubermaticProject            string
	KubermaticOIDCToken          string
	KubermaticClient             *apiclient.KubermaticKubernetesPlatformAPI
	KubermaticAuthenticator      runtime.ClientAuthInfoWriter
	KubermaticConfiguration      *kubermaticv1.KubermaticConfiguration
	PushgatewayEndpoint          string

	Secrets Secrets
}

func NewDefaultOptions() *Options {
	return &Options{
		Client:                       "kube",
		ScenarioOptions:              sets.NewString(),
		Providers:                    sets.NewString(),
		Releases:                     sets.NewString(getLatestMinorVersions(defaulting.DefaultKubernetesVersioning.Versions)...),
		ContainerRuntimes:            sets.NewString(resources.ContainerRuntimeContainerd),
		EnableDistributions:          sets.NewString(),
		ExcludeDistributions:         sets.NewString(),
		Distributions:                sets.NewString(),
		EnableTests:                  sets.NewString(),
		ExcludeTests:                 sets.NewString(),
		Tests:                        sets.NewString(),
		KubermaticNamespace:          "kubermatic",
		KubermaticSeedName:           "kubermatic",
		ControlPlaneReadyWaitTimeout: 10 * time.Minute,
		NodeReadyTimeout:             20 * time.Minute,
		CustomTestTimeout:            10 * time.Minute,
		UserClusterPollInterval:      5 * time.Second,
	}
}

var (
	pubKeyPath string
)

func (o *Options) AddFlags() {
	// user.Current does not work in Alpine
	pubKeyPath = path.Join(os.Getenv("HOME"), ".ssh/id_rsa.pub")

	flag.StringVar(&o.Client, "client", o.Client, "controls how to interact with KKP; can be either `api` or `kube`")
	flag.StringVar(&o.ExistingClusterLabel, "existing-cluster-label", "", "label to use to select an existing cluster for testing. If provided, no cluster will be created. Sample: my=cluster")
	flag.StringVar(&o.NamePrefix, "name-prefix", "", "prefix used for all cluster names")
	flag.Var(flagopts.SetFlag(o.Providers), "providers", "Comma-separated list of providers to test")
	flag.Var(flagopts.SetFlag(o.Releases), "releases", "Comma-separated list of Kubernetes releases (e.g. '1.24') to test")
	flag.Var(flagopts.SetFlag(o.ContainerRuntimes), "container-runtimes", "Comma-separated list of container runtimes to test")
	flag.Var(flagopts.SetFlag(o.EnableDistributions), "distributions", "Comma-separated list of distributions to test (cannot be used in conjunction with -exclude-distributions)")
	flag.Var(flagopts.SetFlag(o.ExcludeDistributions), "exclude-distributions", "Comma-separated list of distributions that will get excluded from the tests (cannot be used in conjunction with -distributions)")
	flag.Var(flagopts.SetFlag(o.EnableTests), "tests", "Comma-separated list of enabled tests (cannot be used in conjunction with -exclude-tests)")
	flag.Var(flagopts.SetFlag(o.ExcludeTests), "exclude-tests", "Run all the tests except the ones in this comma-separated list (cannot be used in conjunction with -tests)")
	flag.Var(flagopts.SetFlag(o.ScenarioOptions), "scenario-options", "Comma-separated list of additional options to be passed to scenarios, e.g. to configure specific features to be tested.")
	flag.StringVar(&o.RepoRoot, "repo-root", "/opt/kube-test/", "Root path for the different kubernetes repositories")
	flag.StringVar(&o.KubermaticEndpoint, "kubermatic-endpoint", "http://localhost:8080", "scheme://host[:port] of the Kubermatic API endpoint to talk to")
	flag.StringVar(&o.KubermaticProject, "kubermatic-project", "", "Kubermatic project to use; leave empty to create a new one")
	flag.StringVar(&o.KubermaticOIDCToken, "kubermatic-oidc-token", "", "Token to authenticate against the Kubermatic API")
	flag.StringVar(&o.KubermaticSeedName, "kubermatic-seed-cluster", o.KubermaticSeedName, "Seed cluster name to create test cluster in")
	flag.StringVar(&o.KubermaticNamespace, "kubermatic-namespace", o.KubermaticNamespace, "Namespace where Kubermatic is installed to")
	flag.BoolVar(&o.CreateOIDCToken, "create-oidc-token", false, "Whether to create a OIDC token. If false, -kubermatic-project-id and -kubermatic-oidc-token must be set")
	flag.IntVar(&o.NodeCount, "kubermatic-nodes", 3, "number of worker nodes")
	flag.IntVar(&o.ClusterParallelCount, "kubermatic-parallel-clusters", 1, "number of clusters to test in parallel")
	flag.StringVar(&o.ReportsRoot, "reports-root", "/opt/reports", "Root for reports")
	flag.StringVar(&o.LogDirectory, "log-directory", "", "Root directory to place container logs into")
	flag.DurationVar(&o.ControlPlaneReadyWaitTimeout, "kubermatic-cluster-timeout", o.ControlPlaneReadyWaitTimeout, "cluster creation timeout")
	flag.DurationVar(&o.NodeReadyTimeout, "node-ready-timeout", o.NodeReadyTimeout, "base time to wait for machines to join the cluster")
	flag.DurationVar(&o.CustomTestTimeout, "custom-test-timeout", o.CustomTestTimeout, "timeout for Kubermatic-specific PVC/LB tests")
	flag.DurationVar(&o.UserClusterPollInterval, "user-cluster-poll-interval", o.UserClusterPollInterval, "poll interval when checking user-cluster conditions")
	flag.BoolVar(&o.DeleteClusterAfterTests, "kubermatic-delete-cluster", true, "delete test cluster when tests where successful")
	flag.BoolVar(&o.WaitForClusterDeletion, "wait-for-cluster-deletion", true, "wait for the cluster deletion to have finished")
	flag.StringVar(&pubKeyPath, "node-ssh-pub-key", pubKeyPath, "path to a public key which gets deployed onto every node")
	flag.BoolVar(&o.PspEnabled, "enable-psp", false, "When set, enables the Pod Security Policy plugin in the user cluster")
	flag.BoolVar(&o.OperatingSystemManagerEnabled, "enable-osm", true, "When set, enables Operating System Manager in the user cluster")
	flag.BoolVar(&o.DualStackEnabled, "enable-dualstack", false, "When set, enables dualstack (IPv4+IPv6 networking) in the user cluster")
	flag.BoolVar(&o.KonnectivityEnabled, "enable-konnectivity", false, "When set, enables Konnectivity (proxy service for control plane communication) in the user cluster. When set to false, OpenVPN is used")
	flag.StringVar(&o.DexHelmValuesFile, "dex-helm-values-file", "", "Helm values.yaml of the OAuth (Dex) chart to read and configure a matching client for. Only needed if -create-oidc-token is enabled.")
	flag.StringVar(&o.PushgatewayEndpoint, "pushgateway-endpoint", "", "host:port of a Prometheus Pushgateway to send runtime metrics to")
	o.Secrets.AddFlags()
}

func (o *Options) ParseFlags(log *zap.SugaredLogger) error {
	if o.ExistingClusterLabel != "" && o.ClusterParallelCount != 1 {
		return errors.New("-cluster-parallel-count must be 1 when testing an existing cluster")
	}

	if !sets.NewString("api", "kube").Has(o.Client) {
		return fmt.Errorf("invalid -client option %q", o.Client)
	}

	// restrict to known container runtimes
	o.ContainerRuntimes = sets.NewString(resources.ContainerRuntimeDocker, resources.ContainerRuntimeContainerd).Intersection(o.ContainerRuntimes)
	if o.ContainerRuntimes.Len() == 0 {
		return errors.New("no container runtime was enabled")
	}

	o.Providers = AllProviders.Intersection(o.Providers)
	if o.Providers.Len() == 0 {
		return errors.New("no cloud provider was enabled")
	}

	var err error
	o.Tests, err = o.effectiveTests()
	if err != nil {
		return err
	}

	if o.Tests.Len() == 0 {
		log.Warn("All tests have been disabled, will only test cluster creation and whether nodes come up successfully.")
	}

	for _, release := range o.Releases.List() {
		version := test.LatestKubernetesVersionForRelease(release, nil)
		if version == nil {
			return fmt.Errorf("no version found for release %q", release)
		}

		o.Versions = append(o.Versions, version)
	}

	// periodics do not specify a version at all and just rely on us auto-determining
	// the most recent stable (stable = latest-1) supported Kubernetes version
	if len(o.Versions) == 0 {
		o.Versions = append(o.Versions, test.LatestStableKubernetesVersion(nil))
		log.Infow("No -releases specified, defaulting to latest stable Kubernetes version", "version", o.Versions[0])
	}

	o.Distributions, err = o.effectiveDistributions()
	if err != nil {
		return err
	}

	if pubKeyPath != "" {
		keyData, err := os.ReadFile(pubKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load SSH key: %w", err)
		}

		o.PublicKeys = [][]byte{keyData}
	}

	if o.LogDirectory != "" {
		if _, err := os.Stat(o.LogDirectory); err != nil {
			if err := os.MkdirAll(o.LogDirectory, 0755); err != nil {
				return fmt.Errorf("log-directory %q is not a valid directory and could not be created", o.LogDirectory)
			}
		}
	}

	if err := o.Secrets.ParseFlags(); err != nil {
		return err
	}

	return nil
}

func (o *Options) effectiveDistributions() (sets.String, error) {
	all := sets.NewString()
	for _, os := range providerconfig.AllOperatingSystems {
		all.Insert(string(os))
	}

	return combineSets(o.EnableDistributions, o.ExcludeDistributions, all, "distributions")
}

func (o *Options) effectiveTests() (sets.String, error) {
	// Do not force all scripts to keep a list of all tests, just default to running all tests
	// when no relevant CLI flag was given.
	if o.EnableTests.Len() == 0 && o.ExcludeTests.Len() == 0 {
		return AllTests, nil
	}

	// Make it more comfortable to disable all custom tests.
	if o.EnableTests.Len() == 0 && o.ExcludeTests.Has("all") {
		return sets.NewString(), nil
	}

	return combineSets(o.EnableTests, o.ExcludeTests, AllTests, "tests")
}

func combineSets(include, exclude, all sets.String, flagname string) (sets.String, error) {
	if include.Len() == 0 && exclude.Len() == 0 {
		return nil, fmt.Errorf("either -%s or -exclude-%s must be given (each is a comma-separated list of %v)", flagname, flagname, all.List())
	}

	if include.Len() > 0 && exclude.Len() > 0 {
		return nil, fmt.Errorf("-%s and -exclude-%s must not be given at the same time", flagname, flagname)
	}

	var chosen sets.String

	if include.Len() > 0 {
		chosen = include

		if unsupported := chosen.Difference(all); unsupported.Len() > 0 {
			return nil, fmt.Errorf("unknown %s: %v", flagname, unsupported.List())
		}
	} else {
		chosen = all.Difference(exclude)
	}

	if chosen.Len() == 0 {
		return nil, fmt.Errorf("no %s to use in tests remained after evaluating -%s and -exclude-%s", flagname, flagname, flagname)
	}

	return chosen, nil
}

func getLatestMinorVersions(versions []kubermativsemver.Semver) []string {
	minorMap := map[uint64]*semverlib.Version{}

	for _, version := range versions {
		sversion := version.Semver()
		minor := sversion.Minor()

		if existing := minorMap[minor]; existing == nil || existing.LessThan(sversion) {
			minorMap[minor] = sversion
		}
	}

	list := []string{}
	for _, v := range minorMap {
		list = append(list, "v"+v.String())
	}
	sort.Strings(list)

	return list
}
