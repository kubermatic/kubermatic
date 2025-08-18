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

package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/config"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/metrics"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/runner"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	options    *types.Options
	logOptions kubermaticlog.Options
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "conformance-tester",
		Short: "A tool for running Kubermatic conformance tests.",
		RunE:  runTests,
	}

	options = types.NewDefaultOptions()
	options.AddFlags(rootCmd.Flags())

	logOptions = kubermaticlog.NewDefaultOptions()
	goFlags := flag.NewFlagSet("goflags", flag.ContinueOnError)
	logOptions.AddFlags(goFlags)
	rootCmd.Flags().AddGoFlagSet(goFlags)

	var (
		generateTemplateFile string
		generateOutputFile   string
	)

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a scenario file from a template.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return scenarios.GenerateScenarios(generateTemplateFile, generateOutputFile)
		},
	}
	generateCmd.Flags().StringVar(&generateTemplateFile, "from", "", "The template file to generate scenarios from.")
	generateCmd.Flags().StringVar(&generateOutputFile, "to", "scenarios.yaml", "The output file for the generated scenarios.")
	rootCmd.AddCommand(generateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Failed to execute command: %v\n", err)
		os.Exit(1)
	}
}

func runTests(cmd *cobra.Command, args []string) error {
	rootCtx := signals.SetupSignalHandler()

	// setup logging
	rawLog := kubermaticlog.New(logOptions.Debug, logOptions.Format)
	log := rawLog.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	// parse our CLI flags
	if err := options.ParseFlags(log); err != nil {
		return fmt.Errorf("invalid flags: %w", err)
	}

	reconciling.Configure(log)

	// collect runtime metrics if there is a pushgateway URL configured
	// and these variables are set
	metrics.Setup(options.PushgatewayEndpoint, os.Getenv("JOB_NAME"), os.Getenv("PROW_JOB_ID"))
	defer metrics.UpdateMetrics(log)

	// say hello
	cli.Hello(log, "Conformance Tests", nil)
	log.Infow("Runner configuration",
		"providers", sets.List(options.Providers),
		"operatingsystems", sets.List(options.Distributions),
		"versions", options.Versions,
		"tests", sets.List(options.Tests),
		"dualstack", options.DualStackEnabled,
		"konnectivity", options.KonnectivityEnabled,
		"updates", options.TestClusterUpdate,
	)

	// setup kube client, ctrl-runtime client, clientgetter, seedgetter etc.
	if err := setupKubeClients(rootCtx, options); err != nil {
		return fmt.Errorf("failed to setup kube clients: %w", err)
	}

	// create a temporary home directory and a fresh SSH key
	homeDir, dynamicSSHPublicKey, err := setupHomeDir(log)
	if err != nil {
		return fmt.Errorf("failed to setup temporary home dir: %w", err)
	}
	options.PublicKeys = append(options.PublicKeys, dynamicSSHPublicKey)
	options.HomeDir = homeDir

	// setup runner and KKP clients
	log.Info("Preparing project...")
	testRunner := runner.NewKubeRunner(options, log)
	if err := testRunner.Setup(rootCtx); err != nil {
		return fmt.Errorf("failed to setup runner: %w", err)
	}

	// determine what's to do
	generator := scenarios.NewGenerator()
	if options.ScenarioFile != "" {
		cfg, err := config.Load(options.ScenarioFile)
		if err != nil {
			return fmt.Errorf("failed to load scenario file %q: %w", options.ScenarioFile, err)
		}
		generator.WithConfig(cfg)
	} else {
		generator.WithCloudProviders(sets.List(options.Providers)...).
			WithOperatingSystems(sets.List(options.Distributions)...).
			WithDualstack(options.DualStackEnabled).
			WithVersions(options.Versions...)
	}

	scenarios, err := generator.Scenarios(rootCtx, options, log)
	if err != nil {
		return fmt.Errorf("failed to determine test scenarios: %w", err)
	}

	if len(scenarios) == 0 {
		return fmt.Errorf("no scenarios match the given criteria")
	}

	// optionally restrict the full set of scenarios to those that previously did not succeed
	var previousResults *runner.ResultsFile
	if options.RetryFailedScenarios {
		previousResults, err = loadPreviousResults(options)
		if err != nil {
			return fmt.Errorf("failed to load previous test results: %w", err)
		}

		scenarios = keepOnlyFailedScenarios(log, scenarios, previousResults, *options)
	}

	if err := testRunner.SetupProject(rootCtx); err != nil {
		return fmt.Errorf("failed to setup project: %w", err)
	}

	log.Infow("Using project", "project", options.KubermaticProject)

	// let the magic happen!
	log.Info("Running E2E tests...")
	start := time.Now()

	results, err := testRunner.Run(rootCtx, scenarios)

	// always print the test results
	if results != nil {
		results.PrintJUnitDetails()
		results.PrintSummary()

		if filename := options.ResultsFile; filename != "" {
			log.Infow("Writing results file", "filename", filename)

			// Merge the previous tests with the new, current results; otherwise if we'd only
			// dump the new results, those would not contain skipped/successful scenarios from
			// the previous run, effectively shrinking the results file every time it is used.
			if previousResults != nil {
				results = runner.MergeResults(previousResults, results)
			}

			if err := results.WriteToFile(filename); err != nil {
				log.Warnw("Failed to write results file", zap.Error(err))
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to execute tests: %w", err)
	}

	if results.HasFailures() {
		return fmt.Errorf("not all tests have passed")
	}

	log.Infow("Test suite has completed successfully", "runtime", time.Since(start))
	return nil
}

func setupKubeClients(ctx context.Context, opts *types.Options) error {
	_, config, err := utils.GetClients()
	if err != nil {
		return fmt.Errorf("failed to get client config: %w", err)
	}
	opts.SeedRestConfig = config

	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("failed to add clusterv1alpha1 to scheme: %w", err)
	}

	if err := metricsv1beta1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("failed to add metrics v1beta1 to scheme: %w", err)
	}

	seedClusterClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		return err
	}
	opts.SeedClusterClient = seedClusterClient

	seedGeneratedClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	opts.SeedGeneratedClient = seedGeneratedClient

	seedGetter, err := kubernetesprovider.SeedGetterFactory(ctx, seedClusterClient, opts.KubermaticSeedName, opts.KubermaticNamespace)
	if err != nil {
		return fmt.Errorf("failed to construct seedGetter: %w", err)
	}
	opts.Seed, err = seedGetter()
	if err != nil {
		return fmt.Errorf("failed to get seed: %w", err)
	}

	configGetter, err := kubernetesprovider.DynamicKubermaticConfigurationGetterFactory(opts.SeedClusterClient, opts.KubermaticNamespace)
	if err != nil {
		return fmt.Errorf("failed to construct configGetter: %w", err)
	}

	opts.KubermaticConfiguration, err = configGetter(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Kubermatic config: %w", err)
	}

	clusterClientProvider, err := clusterclient.NewExternalWithProxy(seedClusterClient, opts.Seed.GetManagementProxyURL())
	if err != nil {
		return fmt.Errorf("failed to get clusterClientProvider: %w", err)
	}
	opts.ClusterClientProvider = clusterClientProvider

	return nil
}

// setupHomeDir set up a temporary home dir (because the e2e tests have some filenames hardcoded,
// which might conflict with the user files).
func setupHomeDir(log *zap.SugaredLogger) (string, []byte, error) {
	// We'll set the env-var $HOME to this directory when executing the tests
	homeDir, err := os.MkdirTemp("/tmp", "e2e-home-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to setup temporary home dir: %w", err)
	}
	log.Infof("Setting up temporary home directory with SSH keys at %s...", homeDir)

	if err := os.MkdirAll(path.Join(homeDir, ".ssh"), os.ModePerm); err != nil {
		return "", nil, err
	}

	// Setup temporary home dir with filepath.Join(os.Getenv("HOME"), ".ssh")
	// Make sure to create relevant ssh keys (because they are hardcoded in the e2e tests...). They must not be password protected
	log.Debug("Generating SSH keys...")
	// Private Key generation
	privateKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return "", nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return "", nil, err
	}

	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	privatePEM := pem.EncodeToMemory(&privBlock)
	// Needs to be google_compute_engine as its hardcoded in the kubernetes e2e tests
	if err := os.WriteFile(path.Join(homeDir, ".ssh", "google_compute_engine"), privatePEM, 0400); err != nil {
		return "", nil, err
	}

	publicRsaKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return "", nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)
	if err := os.WriteFile(path.Join(homeDir, ".ssh", "google_compute_engine.pub"), pubKeyBytes, 0400); err != nil {
		return "", nil, err
	}

	log.Infof("Finished setting up temporary home dir %s", homeDir)
	return homeDir, pubKeyBytes, nil
}

func loadPreviousResults(opts *types.Options) (*runner.ResultsFile, error) {
	if opts.ResultsFile == "" {
		return nil, nil
	}

	// non-existing or empty files are okay
	stat, err := os.Stat(opts.ResultsFile)
	if err != nil || stat.Size() == 0 {
		return nil, nil
	}

	return runner.LoadResultsFile(opts.ResultsFile)
}

func keepOnlyFailedScenarios(log *zap.SugaredLogger, allScenarios []scenarios.Scenario, previousResults *runner.ResultsFile, opts types.Options) []scenarios.Scenario {
	if optionsChanged(previousResults.Configuration, opts) {
		log.Warn("Disregarding previous test results as current options do not match previous options.")
		return allScenarios
	}

	filtered := []scenarios.Scenario{}
	for i, scenario := range allScenarios {
		hasSuccess := false

		for _, previous := range previousResults.Results {
			if previous.MatchesScenario(scenario) && previous.Status == runner.ScenarioPassed {
				hasSuccess = true
				break
			}
		}

		if hasSuccess {
			scenario.Log(log).Info("Skipping because scenario succeeded in a previous run.")
			continue
		}

		filtered = append(filtered, allScenarios[i])
	}

	return filtered
}

func optionsChanged(previous runner.TestConfiguration, current types.Options) bool {
	return false ||
		previous.KonnectivityEnabled != current.KonnectivityEnabled ||
		previous.DualstackEnabled != current.DualStackEnabled ||
		previous.TestClusterUpdate != current.TestClusterUpdate ||
		!sets.New(previous.Tests...).IsSuperset(current.Tests)
}
