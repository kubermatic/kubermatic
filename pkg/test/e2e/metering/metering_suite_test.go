//go:build metering

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

package metering

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	meteringfeatures "k8c.io/kubermatic/v2/pkg/test/e2e/features/metering"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	meteringtestkit "k8c.io/kubermatic/v2/pkg/test/e2e/testkit/metering"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions

	existingCluster string
	noUserCluster   bool
	seedName        = "kubermatic"
	seedNamespace   = "kubermatic"
	bucket          = "kkp-metering-e2e"
	setupOptions    = meteringtestkit.SetupOptions{StorageClassName: "kubermatic-fast", StorageSize: "5Gi"}

	// input is filled on every Ginkgo process before any spec runs.
	input meteringtestkit.Input
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
	flag.StringVar(&existingCluster, "existing-cluster", "", "name of an existing user cluster whose usage the report checks look for, instead of creating a new AWS cluster")
	flag.BoolVar(&noUserCluster, "no-user-cluster", false, "run without a user cluster; combine with --label-filter='!needs-user-cluster'")
	flag.StringVar(&seedName, "seed-name", seedName, "name of the Seed under test")
	flag.StringVar(&seedNamespace, "seed-namespace", seedNamespace, "namespace of the Seed and the metering objects")
	flag.StringVar(&bucket, "s3-bucket", bucket, "bucket to create in the seed MinIO when the seed has no metering-s3 Secret")
	flag.StringVar(&setupOptions.StorageClassName, "storage-class", setupOptions.StorageClassName, "storage class for the metering Prometheus, used when the suite turns metering on")
	flag.StringVar(&setupOptions.StorageSize, "storage-size", setupOptions.StorageSize, "volume size for the metering Prometheus, used when the suite turns metering on")
}

func TestMetering(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Metering e2e")
}

// suiteData is passed from the first Ginkgo process to all others.
type suiteData struct {
	ClusterName  string
	ReportPrefix string
}

var _ = ginkgo.SynchronizedBeforeSuite(func(ctx ginkgo.SpecContext) []byte {
	logger := newLogger()

	seedClient, _, err := utils.GetClients()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get seed clients")

	data := suiteData{
		ClusterName:  existingCluster,
		ReportPrefix: "e2e-" + rand.String(6),
	}

	if data.ClusterName == "" && !noUserCluster {
		gomega.Expect(credentials.Parse()).To(gomega.Succeed(), "failed to get AWS credentials")

		testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
		testJig.ClusterJig.WithTestName("metering")

		// registered before checking the error, so a half-created cluster is
		// removed as well
		ginkgo.DeferCleanup(func(ctx ginkgo.SpecContext) {
			testJig.Cleanup(ctx, ginkgo.GinkgoT(), true)
		}, ginkgo.NodeTimeout(45*time.Minute))

		_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to set up the user cluster")

		data.ClusterName = cluster.Name
	}

	in, err := buildInput(ctx, logger, data)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// cleanups run in reverse order: first the objects, then the Seed, then
	// the Secret the Seed may still need while it reconciles
	createdSecret, err := meteringtestkit.EnsureS3(ctx, in, bucket)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to set up the metering bucket")
	if createdSecret {
		ginkgo.DeferCleanup(meteringtestkit.DeleteS3Secret, in)
	}

	revert, err := meteringtestkit.EnableMetering(ctx, in, setupOptions)
	if revert != nil {
		ginkgo.DeferCleanup(func(ctx context.Context) error {
			return revert(ctx)
		})
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to enable metering")

	ginkgo.DeferCleanup(func(ctx context.Context) {
		// best effort, the bucket is test data
		if err := meteringtestkit.CleanupObjects(ctx, in); err != nil {
			logger.Warnw("Failed to remove the test reports", zap.Error(err))
		}
	})

	if in.Cluster != nil {
		ginkgo.DeferCleanup(func(ctx context.Context) {
			if err := meteringtestkit.CleanupWorkload(ctx, in); err != nil {
				logger.Warnw("Failed to remove the probe workload", zap.Error(err))
			}
		})
		gomega.Expect(meteringtestkit.StartWorkload(ctx, in)).To(gomega.Succeed(), "failed to start the probe workload")
	}

	raw, err := json.Marshal(data)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return raw
}, func(ctx ginkgo.SpecContext, raw []byte) {
	data := suiteData{}
	gomega.Expect(json.Unmarshal(raw, &data)).To(gomega.Succeed())

	var err error
	input, err = buildInput(ctx, newLogger(), data)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}, ginkgo.NodeTimeout(60*time.Minute))

var _ = meteringfeatures.Specs(func() meteringtestkit.Input {
	return input
})

func newLogger() *zap.SugaredLogger {
	rawLogger := log.NewFromOptions(logOptions)

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	return rawLogger.Sugar()
}

// buildInput loads the user cluster, when there is one, and its project.
func buildInput(ctx context.Context, logger *zap.SugaredLogger, data suiteData) (meteringtestkit.Input, error) {
	seedClient, seedConfig, err := utils.GetClients()
	if err != nil {
		return meteringtestkit.Input{}, fmt.Errorf("failed to get seed clients: %w", err)
	}

	in := meteringtestkit.Input{
		SeedClient:    seedClient,
		SeedConfig:    seedConfig,
		SeedName:      seedName,
		SeedNamespace: seedNamespace,
		ReportPrefix:  data.ReportPrefix,
		Log:           logger,
	}

	if data.ClusterName == "" {
		return in, nil
	}

	clusterJig := jig.NewClusterJig(seedClient, logger).WithExistingCluster(data.ClusterName)

	in.Cluster, err = clusterJig.Cluster(ctx)
	if err != nil {
		return in, err
	}

	projectName := in.Cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	in.Project = &kubermaticv1.Project{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: projectName}, in.Project); err != nil {
		return in, fmt.Errorf("failed to get project %q: %w", projectName, err)
	}

	in.UserClient, err = clusterJig.ClusterClient(ctx)
	if err != nil {
		return in, fmt.Errorf("failed to get user cluster client: %w", err)
	}

	return in, nil
}
