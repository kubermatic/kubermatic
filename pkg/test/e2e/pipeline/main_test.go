//go:build e2e

/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package pipeline

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	testEnv    env.Environment
	logOptions = utils.DefaultLogOptions

	// byoCredentials holds the BYO datacenter for the shared base cluster. Defaulted to the
	// byo-kubernetes datacenter the kind seed registers (hack/ci/testdata/seed.yaml) so local
	// and CI runs need no extra flags.
	byoCredentials jig.BYOCredentials

	// withUserCluster gates eager provisioning of the shared BYO base cluster for Tier B/C1
	// features. When false (default) the run is seed-only and provisions nothing.
	withUserCluster bool
)

func init() {
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
	byoCredentials.KKPDatacenter = "byo-kubernetes"
	byoCredentials.AddFlags(flag.CommandLine) // registers -byo-kkp-datacenter
	flag.BoolVar(&withUserCluster, "with-user-cluster", false,
		"provision a shared BYO base user cluster for Tier B/C1 feature tests")
}

func TestMain(m *testing.M) {
	// With a custom TestMain, the testing package does not parse flags itself, so parse
	// before reading -with-user-cluster / -byo-kkp-datacenter registered in init().
	flag.Parse()

	// build the config from KUBECONFIG directly instead of envconf.NewFromFlags();
	// NewFromFlags would register a --namespace flag that collides with jig's.
	cfg := envconf.New().WithKubeconfigFile(os.Getenv("KUBECONFIG"))
	testEnv = env.NewWithConfig(cfg)
	testEnv.Setup(waitForMasterControllerManager)

	// Eagerly provision the shared BYO base cluster only when opted in. Seed-only runs
	// (the default) skip this entirely, so TestUserProjectBinding and other Tier A features
	// pay no cluster cost and leave main_test.go's seed-only behavior unchanged.
	if withUserCluster {
		testEnv.Setup(provisionBaseCluster)
		testEnv.Finish(tearDownBaseCluster)
	}

	os.Exit(testEnv.Run(m))
}

// waitForMasterControllerManager gates the whole suite on the KKP
// master-controller-manager deployment being ready, so seed-only feature tests
// do not race the operator's initial reconcile. This checks the KKP management
// install, not a user cluster's control plane (that is jig.WaitForHealthyControlPlane).
// Cluster teardown stays the run script's job, so no env.Finish is registered.
func waitForMasterControllerManager(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	logger := log.NewFromOptions(logOptions).Sugar()

	client, _, err := utils.GetClients()
	if err != nil {
		return ctx, fmt.Errorf("failed to build client: %w", err)
	}

	ns := jig.KubermaticNamespace()
	if err := utils.WaitForDeploymentReady(ctx, client, logger, ns, common.MasterControllerManagerDeploymentName, 10*time.Minute); err != nil {
		return ctx, fmt.Errorf("KKP master-controller-manager not ready: %w", err)
	}

	return ctx, nil
}
