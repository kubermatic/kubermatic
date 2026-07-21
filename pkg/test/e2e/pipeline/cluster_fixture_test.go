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
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// clusterFixture holds the shared BYO base cluster for Tier B/C1 features. It is
// provisioned once, eagerly, by TestMain when -with-user-cluster is set, and left nil
// otherwise so seed-only runs pay no cluster cost. There is no lazy init: the cluster is
// warm before any feature runs, so features grab it without a first-call race.
type clusterFixture struct {
	testJig  *jig.TestJig
	seed     ctrlruntimeclient.Client
	user     ctrlruntimeclient.Client
	log      *zap.SugaredLogger
	cluster  *kubermaticv1.Cluster
	baseline *kubermaticv1.ClusterSpec
}

// userCluster is the package-level handle to the shared base cluster. nil unless TestMain
// provisioned one (i.e. -with-user-cluster was set).
var userCluster *clusterFixture

// provisionBaseCluster is the TestMain Setup step. It creates the shared BYO base cluster
// and captures its initial Spec as the isolation baseline every feature must restore to.
func provisionBaseCluster(ctx context.Context, _ *envconf.Config) (context.Context, error) {
	logger := log.NewFromOptions(logOptions).Sugar()
	seedClient, _, err := utils.GetClients()
	if err != nil {
		return ctx, fmt.Errorf("failed to build seed client: %w", err)
	}

	testJig := jig.NewBYOCluster(seedClient, logger, byoCredentials)

	// Disable the Kubernetes Dashboard for the e2e base cluster: it is irrelevant to these
	// tests and only adds reconcile time and images. KubernetesDashboard is mutable, but
	// setting it at creation avoids deploying it at all. (NewBYOCluster already disables the
	// user-ssh-key-agent, which is unneeded without Machines.)
	testJig.ClusterJig.WithPatch(func(spec *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
		spec.KubernetesDashboard = &kubermaticv1.KubernetesDashboard{Enabled: false}
		return spec
	})

	_, cluster, err := testJig.Setup(ctx, jig.WaitForNothing)
	if err != nil {
		return ctx, fmt.Errorf("failed to set up base cluster: %w", err)
	}

	userClient, err := testJig.ClusterClient(ctx)
	if err != nil {
		// best-effort cleanup so a half-provisioned cluster does not leak into the seed.
		_ = testJig.ClusterJig.Delete(ctx, false)
		return ctx, fmt.Errorf("failed to build user-cluster client: %w", err)
	}

	userCluster = &clusterFixture{
		testJig:  testJig,
		seed:     seedClient,
		user:     userClient,
		log:      logger,
		cluster:  cluster,
		baseline: cluster.Spec.DeepCopy(),
	}

	logger.Info("Base user cluster ready for Tier B/C1 features", "cluster", cluster.Name)
	return ctx, nil
}

// tearDownBaseCluster is the TestMain Finish step. It deletes the cluster and project
// synchronously, tolerating a nil (never-provisioned) or partially-provisioned fixture so
// a seed-only run or an interrupted provisioning both clean up without panicking.
func tearDownBaseCluster(ctx context.Context, _ *envconf.Config) (context.Context, error) {
	if userCluster == nil {
		return ctx, nil
	}

	logger := userCluster.log
	logger.Info("Tearing down base user cluster", "cluster", userCluster.cluster.Name)

	uc := userCluster
	userCluster = nil // guard against double-entry

	uc.testJig.Cleanup(ctx, &errorPrinter{}, true)
	return ctx, nil
}

// errorPrinter satisfies jig.TestJig's ErrorPrinter so Cleanup can report teardown errors
// without a *testing.T (Finish runs outside a specific test).
type errorPrinter struct {
	errs []error
}

func (e *errorPrinter) Errorf(format string, args ...interface{}) {
	e.errs = append(e.errs, fmt.Errorf(format, args...))
}
