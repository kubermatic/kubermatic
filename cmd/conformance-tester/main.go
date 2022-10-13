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

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/metrics"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/runner"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/cli"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	rootCtx := signals.SetupSignalHandler()

	// setup flags
	opts := types.NewDefaultOptions()
	opts.AddFlags()

	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	flag.Parse()

	// setup logging
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	// parse our CLI flags
	if err := opts.ParseFlags(log); err != nil {
		log.Fatalw("Invalid flags", zap.Error(err))
	}

	// collect runtime metrics if there is a pushgateway URL configured
	// and these variables are set
	metrics.Setup(opts.PushgatewayEndpoint, os.Getenv("JOB_NAME"), os.Getenv("PROW_JOB_ID"))
	defer metrics.UpdateMetrics(log)

	// say hello
	cli.Hello(log, "Conformance Tests", true, nil)
	log.Infow("Kubermatic API Endpoint", "endpoint", opts.KubermaticEndpoint)
	log.Infow("Runner configuration",
		"providers", opts.Providers.List(),
		"operatingsystems", opts.Distributions.List(),
		"versions", opts.Versions,
		"containerruntimes", opts.ContainerRuntimes.List(),
		"tests", opts.Tests.List(),
		"osm", opts.OperatingSystemManagerEnabled,
		"dualstack", opts.DualStackEnabled,
		"konnectivity", opts.KonnectivityEnabled,
	)

	// setup kube client, ctrl-runtime client, clientgetter, seedgetter etc.
	if err := setupKubeClients(rootCtx, opts); err != nil {
		log.Fatalw("Failed to setup kube clients", zap.Error(err))
	}

	// create a temporary home directory and a fresh SSH key
	homeDir, dynamicSSHPublicKey, err := setupHomeDir(log)
	if err != nil {
		log.Fatalw("Failed to setup temporary home dir", zap.Error(err))
	}
	opts.PublicKeys = append(opts.PublicKeys, dynamicSSHPublicKey)
	opts.HomeDir = homeDir

	// setup test runner
	testRunner := runner.NewKubeRunner(opts, log)

	// setup runner and KKP clients
	log.Info("Preparing project...")
	if err := testRunner.Setup(rootCtx); err != nil {
		log.Fatalw("Failed to setup runner", zap.Error(err))
	}

	// determine what's to do
	scenarios, err := scenarios.NewGenerator().
		WithCloudProviders(opts.Providers.List()...).
		WithOperatingSystems(opts.Distributions.List()...).
		WithContainerRuntimes(opts.ContainerRuntimes.List()...).
		WithOSM(opts.OperatingSystemManagerEnabled).
		WithDualstack(opts.DualStackEnabled).
		WithVersions(opts.Versions...).
		Scenarios(rootCtx, opts, log)
	if err != nil {
		log.Fatalw("Failed to determine test scenarios", zap.Error(err))
	}

	if len(scenarios) == 0 {
		// Fatalw() because Fatal() trips up the linter because of the previous defer.
		log.Fatalw("No scenarios match the given criteria.")
	}

	if err := testRunner.SetupProject(rootCtx); err != nil {
		log.Fatalw("Failed to setup project", zap.Error(err))
	}

	log.Infow("Using project", "project", opts.KubermaticProject)

	// let the magic happen!
	log.Info("Running E2E tests...")
	start := time.Now()
	if err := testRunner.Run(rootCtx, scenarios); err != nil {
		log.Fatalw("Test failed", zap.Error(err))
	}

	log.Infow("Test suite has completed successfully", "runtime", time.Since(start))
}

func setupKubeClients(ctx context.Context, opts *types.Options) error {
	_, _, config, err := utils.GetClients()
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

	clusterClientProvider, err := clusterclient.NewExternal(seedClusterClient)
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
