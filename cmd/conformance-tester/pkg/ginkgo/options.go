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

package ginkgo

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"gopkg.in/yaml.v3"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	legacytypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermativsemver "k8c.io/kubermatic/sdk/v2/semver"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8s.io/client-go/kubernetes/scheme"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// Options represent combination of flags and ENV options.
type Options struct {
	Client     string `yaml:"client,omitempty"`
	NamePrefix string `yaml:"namePrefix,omitempty"`

	// these flags control the test scenario matrix;
	// the Enabled... and Exclude... sets are combined
	// together to result in the final ... set

	Providers            []string                   `yaml:"providers,omitempty"`
	Releases             []string                   `yaml:"releases,omitempty"`
	Versions             []*kubermativsemver.Semver `yaml:"-"` // populated based on releases
	EnableDistributions  []string                   `yaml:"enableDistributions,omitempty"`
	ExcludeDistributions []string                   `yaml:"excludeDistributions,omitempty"`
	Distributions        []string                   `yaml:"-"` // populated based on enable/exclude distributions
	EnableTests          []string                   `yaml:"enableTests,omitempty"`
	ExcludeTests         []string                   `yaml:"excludeTests,omitempty"`
	Tests                []string                   `yaml:"-"` // populated based on enable/exclude tests

	// The tester can export the result status for all executed scenarios
	// into a JSON file and then re-read that to retry failed runs.
	ResultsFile          string `yaml:"resultsFile,omitempty"`
	RetryFailedScenarios bool   `yaml:"retryFailedScenarios,omitempty"`

	// additional settings identical for all scenarios

	DualStackEnabled    bool     `yaml:"dualStackEnabled,omitempty"`
	KonnectivityEnabled bool     `yaml:"konnectivityEnabled,omitempty"`
	ScenarioOptions     []string `yaml:"scenarioOptions,omitempty"`
	TestClusterUpdate   bool     `yaml:"testClusterUpdate,omitempty"`

	// additional settings

	ControlPlaneReadyWaitTimeout time.Duration `yaml:"controlPlaneReadyWaitTimeout,omitempty"`
	NodeReadyTimeout             time.Duration `yaml:"nodeReadyTimeout,omitempty"`
	CustomTestTimeout            time.Duration `yaml:"customTestTimeout,omitempty"`
	UserClusterPollInterval      time.Duration `yaml:"userClusterPollInterval,omitempty"`
	DeleteClusterAfterTests      bool          `yaml:"deleteClusterAfterTests,omitempty"`
	WaitForClusterDeletion       bool          `yaml:"waitForClusterDeletion,omitempty"`
	NodeCount                    int           `yaml:"nodeCount,omitempty"`
	PublicKeys                   [][]byte      `yaml:"-"` // populated from NodeSSHPublicKeyFile
	NodeSSHPublicKeyFile         string        `yaml:"nodeSSHPublicKeyFile,omitempty"`
	ReportsRoot                  string        `yaml:"reportsRoot,omitempty"`
	LogDirectory                 string        `yaml:"logDirectory,omitempty"`
	RepoRoot                     string        `yaml:"repoRoot,omitempty"`
	ClusterParallelCount         int           `yaml:"clusterParallelCount,omitempty"`
	HomeDir                      string        `yaml:"-"`
	ExistingClusterLabel         string        `yaml:"existingClusterLabel,omitempty"`
	KubermaticNamespace          string        `yaml:"kubermaticNamespace,omitempty"`
	KubermaticSeedName           string        `yaml:"kubermaticSeedName,omitempty"`
	KubermaticProject            string        `yaml:"kubermaticProject,omitempty"`
	PushgatewayEndpoint          string        `yaml:"pushgatewayEndpoint,omitempty"`

	JUnitFile string `yaml:"junitFile,omitempty"`

	Secrets types.Secrets `yaml:"secrets,omitempty"`
}

// RuntimeOptions holds runtime-specific objects that are not serializable.
type RuntimeOptions struct {
	SeedClusterClient       ctrlruntimeclient.Client
	SeedGeneratedClient     kubernetes.Interface
	ClusterClientProvider   *clusterclient.Provider
	Seed                    *kubermaticv1.Seed
	SeedRestConfig          *rest.Config
	KubermaticConfiguration *kubermaticv1.KubermaticConfiguration
}

func MergeOptions(base Options, override *types.Options) *types.Options {
	return &types.Options{
		Client:                       override.Client,
		Providers:                    override.Providers,
		Releases:                     override.Releases,
		EnableDistributions:          override.EnableDistributions,
		ExcludeDistributions:         override.ExcludeDistributions,
		EnableTests:                  override.EnableTests,
		ExcludeTests:                 override.ExcludeTests,
		KubermaticNamespace:          override.KubermaticNamespace,
		KubermaticSeedName:           override.KubermaticSeedName,
		KonnectivityEnabled:          override.KonnectivityEnabled,
		NodeCount:                    override.NodeCount,
		ControlPlaneReadyWaitTimeout: override.ControlPlaneReadyWaitTimeout,
		NodeReadyTimeout:             override.NodeReadyTimeout,
		CustomTestTimeout:            override.CustomTestTimeout,
		UserClusterPollInterval:      override.UserClusterPollInterval,
		NamePrefix:                   override.NamePrefix,
		Secrets:                      override.Secrets,
		LogDirectory:                 override.LogDirectory,
		ReportsRoot:                  override.ReportsRoot,
		DeleteClusterAfterTests:      override.DeleteClusterAfterTests,
	}
}

func NewDefaultOptions() *Options {
	return &Options{
		Client:                       "kube",
		Providers:                    []string{string(providerconfig.CloudProviderKubeVirt)},
		Releases:                     []string{utils.KubernetesVersion()},
		EnableDistributions:          []string{string(providerconfig.OperatingSystemUbuntu)},
		ExcludeDistributions:         []string{},
		EnableTests:                  []string{},
		ExcludeTests:                 []string{},
		KubermaticNamespace:          "kubermatic",
		KubermaticSeedName:           "kubermatic",
		KonnectivityEnabled:          true,
		NodeCount:                    1,
		ControlPlaneReadyWaitTimeout: 10 * time.Minute,
		NodeReadyTimeout:             20 * time.Minute,
		CustomTestTimeout:            10 * time.Minute,
		UserClusterPollInterval:      5 * time.Second,
		NamePrefix:                   "dev-conformance-tester",
		Secrets: types.Secrets{
			Kubevirt: types.KubevirtSecrets{
				KKPDatacenter:  "kubevirt",
				KubeconfigFile: "/home/soer3n/vscode/mybackup/kubermatic-work/kubermatic/local-kkp",
			},
			// Hetzner: types.HetznerSecrets{
			// 	KKPDatacenter: "hetzner",
			// 	Token:         "your-hetzner-api-token",
			// },
		},
		LogDirectory: "_logs",
		ReportsRoot:  "_reports",
		// PushgatewayEndpoint:     "http://pushgateway.monitoring.svc.cluster.local:9091",
		DeleteClusterAfterTests: true,
	}
}

func newOptionsFromYAML(log *zap.SugaredLogger) (*Options, error) {
	// opts will be populated with defaults first, and then overwritten by the YAML inside UnmarshalYAML.
	opts := &Options{}

	configPath := os.Getenv("CONFORMANCE_TESTER_CONFIG_FILE")
	if configPath == "" {
		log.Info("CONFORMANCE_TESTER_CONFIG_FILE not set, using default options.")
		// No config file, so just use the defaults.
		opts = NewDefaultOptions()
	} else {
		log.Infow("Loading configuration from YAML file", "path", configPath)
		yamlFile, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(yamlFile, opts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config yaml: %w", err)
		}
	}

	// setup metrics
	// metrics.Setup(opts.PushgatewayEndpoint, os.Getenv("JOB_NAME"), os.Getenv("PROW_JOB_ID"))

	// Post-process secrets to load from files if specified
	if err := opts.Secrets.ProcessFileSecrets(); err != nil {
		return nil, fmt.Errorf("failed to process secrets from files: %w", err)
	}

	return opts, nil
}

func NewRuntimeOptions(ctx context.Context, log *zap.SugaredLogger, o *Options) (*RuntimeOptions, error) {
	runtimeOpts := &RuntimeOptions{}

	_, config, err := utils.GetClients()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	runtimeOpts.SeedRestConfig = config
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return nil, (fmt.Errorf("failed to add clusterv1alpha1 to scheme: %w", err))
	}
	if err := metricsv1beta1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return nil, (fmt.Errorf("failed to add metrics v1beta1 to scheme: %w", err))
	}

	seedClusterClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		return nil, err
	}
	runtimeOpts.SeedClusterClient = seedClusterClient

	seedGeneratedClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	runtimeOpts.SeedGeneratedClient = seedGeneratedClient

	seedGetter, err := kubernetesprovider.SeedGetterFactory(ctx, seedClusterClient, o.KubermaticSeedName, o.KubermaticNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to construct seedGetter: %w", err)
	}
	runtimeOpts.Seed, err = seedGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get seed: %w", err)
	}

	configGetter, err := kubernetesprovider.DynamicKubermaticConfigurationGetterFactory(runtimeOpts.SeedClusterClient, o.KubermaticNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to construct configGetter: %w", err)
	}

	runtimeOpts.KubermaticConfiguration, err = configGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubermatic config: %w", err)
	}

	clusterClientProvider, err := clusterclient.NewExternalWithProxy(seedClusterClient, runtimeOpts.Seed.GetManagementProxyURL())
	if err != nil {
		return nil, fmt.Errorf("failed to get clusterClientProvider: %w", err)
	}
	runtimeOpts.ClusterClientProvider = clusterClientProvider

	return runtimeOpts, nil
}

func mergeOptions(log *zap.SugaredLogger, yamlOpts *Options, flagOpts *legacytypes.Options, runtimeOpts *RuntimeOptions) *legacytypes.Options {
	// Start with a base of legacy options derived from the YAML file.
	merged := toLegacyOptions(yamlOpts, runtimeOpts)

	// Now, selectively override with values from the command-line flags,
	// but only if the flags were actually set by the user.
	flag.Visit(func(f *flag.Flag) {
		log.Debugw("Merging command-line flag", "flag", f.Name, "value", f.Value)
		switch f.Name {
		case "client":
			merged.Client = flagOpts.Client
		case "existing-cluster-label":
			merged.ExistingClusterLabel = flagOpts.ExistingClusterLabel
		case "name-prefix":
			merged.NamePrefix = flagOpts.NamePrefix
		case "providers":
			merged.Providers = flagOpts.Providers
		case "releases":
			merged.Releases = flagOpts.Releases
		case "distributions":
			merged.EnableDistributions = flagOpts.EnableDistributions
		case "exclude-distributions":
			merged.ExcludeDistributions = flagOpts.ExcludeDistributions
		case "tests":
			merged.EnableTests = flagOpts.EnableTests
		case "exclude-tests":
			merged.ExcludeTests = flagOpts.ExcludeTests
		case "scenario-options":
			merged.ScenarioOptions = flagOpts.ScenarioOptions
		case "repo-root":
			merged.RepoRoot = flagOpts.RepoRoot
		case "kubermatic-project":
			merged.KubermaticProject = flagOpts.KubermaticProject
		case "kubermatic-seed-cluster":
			merged.KubermaticSeedName = flagOpts.KubermaticSeedName
		case "kubermatic-namespace":
			merged.KubermaticNamespace = flagOpts.KubermaticNamespace
		case "kubermatic-nodes":
			merged.NodeCount = flagOpts.NodeCount
		case "kubermatic-parallel-clusters":
			merged.ClusterParallelCount = flagOpts.ClusterParallelCount
		case "reports-root":
			merged.ReportsRoot = flagOpts.ReportsRoot
		case "log-directory":
			merged.LogDirectory = flagOpts.LogDirectory
		case "kubermatic-cluster-timeout":
			merged.ControlPlaneReadyWaitTimeout = flagOpts.ControlPlaneReadyWaitTimeout
		case "node-ready-timeout":
			merged.NodeReadyTimeout = flagOpts.NodeReadyTimeout
		case "custom-test-timeout":
			merged.CustomTestTimeout = flagOpts.CustomTestTimeout
		case "user-cluster-poll-interval":
			merged.UserClusterPollInterval = flagOpts.UserClusterPollInterval
		case "kubermatic-delete-cluster":
			merged.DeleteClusterAfterTests = flagOpts.DeleteClusterAfterTests
		case "wait-for-cluster-deletion":
			merged.WaitForClusterDeletion = flagOpts.WaitForClusterDeletion
		case "node-ssh-pub-key":
			// This is handled by pubKeyPath and loaded in ParseFlags, so we just copy the result.
			merged.PublicKeys = flagOpts.PublicKeys
		case "enable-dualstack":
			merged.DualStackEnabled = flagOpts.DualStackEnabled
		case "enable-konnectivity":
			merged.KonnectivityEnabled = flagOpts.KonnectivityEnabled
		case "update-cluster":
			merged.TestClusterUpdate = flagOpts.TestClusterUpdate
		case "pushgateway-endpoint":
			merged.PushgatewayEndpoint = flagOpts.PushgatewayEndpoint
		case "results-file":
			merged.ResultsFile = flagOpts.ResultsFile
		case "retry":
			merged.RetryFailedScenarios = flagOpts.RetryFailedScenarios
		}
	})

	// Secrets are complex; for now, we assume flags override the entire secret structure if provided.
	// A more granular merge might be needed if users can specify partial secrets via flags.
	if flagOpts.Secrets.AreAnySet() {
		merged.Secrets = flagOpts.Secrets
	}

	return merged
}

// toLegacyOptions converts the ginkgo options to the old ctypes options for compatibility.
func toLegacyOptions(opts *Options, runtimeOpts *RuntimeOptions) *types.Options {
	// The various scenario and test functions have not been updated to the new
	// ginkgo.Options struct yet, so we need to convert.
	legacyOpts := &types.Options{
		Client:                       opts.Client,
		NamePrefix:                   opts.NamePrefix,
		Providers:                    sets.New[string](opts.Providers...),
		Releases:                     sets.New[string](opts.Releases...),
		Versions:                     opts.Versions,
		EnableDistributions:          sets.New[string](opts.EnableDistributions...),
		ExcludeDistributions:         sets.New[string](opts.ExcludeDistributions...),
		Distributions:                sets.New[string](opts.Distributions...),
		EnableTests:                  sets.New[string](opts.EnableTests...),
		ExcludeTests:                 sets.New[string](opts.ExcludeTests...),
		Tests:                        sets.New[string](opts.Tests...),
		ResultsFile:                  opts.ResultsFile,
		RetryFailedScenarios:         opts.RetryFailedScenarios,
		DualStackEnabled:             opts.DualStackEnabled,
		KonnectivityEnabled:          true,
		ScenarioOptions:              sets.New[string](opts.ScenarioOptions...),
		TestClusterUpdate:            opts.TestClusterUpdate,
		ControlPlaneReadyWaitTimeout: opts.ControlPlaneReadyWaitTimeout,
		NodeReadyTimeout:             opts.NodeReadyTimeout,
		CustomTestTimeout:            opts.CustomTestTimeout,
		UserClusterPollInterval:      opts.UserClusterPollInterval,
		DeleteClusterAfterTests:      opts.DeleteClusterAfterTests,
		WaitForClusterDeletion:       opts.WaitForClusterDeletion,
		NodeCount:                    opts.NodeCount,
		PublicKeys:                   opts.PublicKeys,
		ReportsRoot:                  opts.ReportsRoot,
		LogDirectory:                 opts.LogDirectory,
		RepoRoot:                     opts.RepoRoot,
		ClusterParallelCount:         opts.ClusterParallelCount,
		HomeDir:                      opts.HomeDir,
		ExistingClusterLabel:         opts.ExistingClusterLabel,
		KubermaticNamespace:          opts.KubermaticNamespace,
		KubermaticSeedName:           opts.KubermaticSeedName,
		KubermaticProject:            opts.KubermaticProject,
		PushgatewayEndpoint:          opts.PushgatewayEndpoint,
	}

	// Copy secrets
	legacyOpts.Secrets = opts.Secrets

	if runtimeOpts == nil {
		return legacyOpts
	}

	// Copy runtime objects
	legacyOpts.SeedClusterClient = runtimeOpts.SeedClusterClient
	legacyOpts.SeedGeneratedClient = runtimeOpts.SeedGeneratedClient
	legacyOpts.ClusterClientProvider = runtimeOpts.ClusterClientProvider
	legacyOpts.Seed = runtimeOpts.Seed
	legacyOpts.SeedRestConfig = runtimeOpts.SeedRestConfig
	legacyOpts.KubermaticConfiguration = runtimeOpts.KubermaticConfiguration
	return legacyOpts
}
