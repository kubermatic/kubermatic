//go:build mla

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

package mla

import (
	"context"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/log"
	mlafeatures "k8c.io/kubermatic/v2/pkg/test/e2e/features/mla"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	mlatestkit "k8c.io/kubermatic/v2/pkg/test/e2e/testkit/mla"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	credentials     jig.AWSCredentials
	logOptions      = utils.DefaultLogOptions
	existingCluster string

	// grafanaAdminEmail defaults to the KKP admin User that hack/ci/run-mla-e2e-tests.sh creates.
	grafanaAdminEmail = "roxy-admin@kubermatic.com"

	// input is filled on every Ginkgo process before any spec runs.
	input mlatestkit.Input
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
	flag.StringVar(&existingCluster, "existing-cluster", "", "name of an existing user cluster to run the checks against, instead of creating a new AWS cluster")
	flag.StringVar(&grafanaAdminEmail, "grafana-admin-email", grafanaAdminEmail, "email of a KKP admin User that is expected to be provisioned into Grafana")
}

func TestMLA(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "User cluster MLA e2e")
}

var _ = ginkgo.SynchronizedBeforeSuite(func(ctx ginkgo.SpecContext) []byte {
	logger := newLogger()

	seedClient, _, err := utils.GetClients()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get seed clients")

	clusterName := existingCluster
	if clusterName == "" {
		gomega.Expect(credentials.Parse()).To(gomega.Succeed(), "failed to get AWS credentials")

		testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
		testJig.ClusterJig.WithTestName("mla")

		// registered before checking the error, so a half-created cluster is
		// removed as well
		ginkgo.DeferCleanup(func(ctx ginkgo.SpecContext) {
			testJig.Cleanup(ctx, ginkgo.GinkgoT(), true)
		}, ginkgo.NodeTimeout(45*time.Minute))

		_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to set up the user cluster")

		clusterName = cluster.Name
	}

	in, err := buildInput(ctx, logger, clusterName)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	gomega.Expect(mlatestkit.EnableMLA(ctx, in)).To(gomega.Succeed(), "failed to enable MLA")

	// started early so the agents' scrape, remote write and log shipping
	// overlap with the other checks
	ginkgo.DeferCleanup(func(ctx ginkgo.SpecContext) {
		// best effort, the cluster is deleted right after
		if err := mlatestkit.CleanupProbes(ctx, in); err != nil {
			logger.Warnw("Failed to clean up probes", zap.Error(err))
		}
		if err := mlatestkit.CleanupGatewayProbe(ctx, in); err != nil {
			logger.Warnw("Failed to clean up gateway probe", zap.Error(err))
		}
	})
	gomega.Expect(mlatestkit.StartProbes(ctx, in)).To(gomega.Succeed())

	return []byte(clusterName)
}, func(ctx ginkgo.SpecContext, data []byte) {
	var err error

	input, err = buildInput(ctx, newLogger(), string(data))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}, ginkgo.NodeTimeout(60*time.Minute))

var _ = mlafeatures.UserClusterSpecs(func() mlatestkit.Input {
	return input
})

func newLogger() *zap.SugaredLogger {
	rawLogger := log.NewFromOptions(logOptions)

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	return rawLogger.Sugar()
}

// buildInput loads the cluster, its project and a user cluster client.
func buildInput(ctx context.Context, logger *zap.SugaredLogger, clusterName string) (mlatestkit.Input, error) {
	seedClient, seedConfig, err := utils.GetClients()
	if err != nil {
		return mlatestkit.Input{}, fmt.Errorf("failed to get seed clients: %w", err)
	}

	return mlatestkit.LoadInput(ctx, seedClient, seedConfig, logger, clusterName, grafanaAdminEmail)
}
