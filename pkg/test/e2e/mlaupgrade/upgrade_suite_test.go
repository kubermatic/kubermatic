//go:build mlaupgrade

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

package mlaupgrade

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	mlatestkit "k8c.io/kubermatic/v2/pkg/test/e2e/testkit/mla"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

// The suite runs twice around a KKP upgrade: with --label-filter=pre-upgrade
// on the old release, and with --label-filter=post-upgrade after the upgrade.
// The state file carries what the first run recorded to the second.

var (
	credentials       jig.AWSCredentials
	logOptions        = utils.DefaultLogOptions
	existingCluster   string
	stateFile         string
	grafanaAdminEmail = "roxy-admin@kubermatic.com"
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
	flag.StringVar(&existingCluster, "existing-cluster", "", "name of an existing user cluster; without it the pre-upgrade run creates an AWS cluster that the post-upgrade run deletes")
	flag.StringVar(&stateFile, "upgrade-state-file", "", "file the pre-upgrade run writes and the post-upgrade run reads")
	flag.StringVar(&grafanaAdminEmail, "grafana-admin-email", grafanaAdminEmail, "email of a KKP admin User that is expected to be provisioned into Grafana")
}

// state is what the pre-upgrade run hands to the post-upgrade run.
type state struct {
	mlatestkit.UpgradeState

	// CreatedCluster is set when the pre-upgrade run created the cluster.
	CreatedCluster bool   `json:"createdCluster"`
	ProjectName    string `json:"projectName"`
}

func TestMLAUpgrade(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "MLA upgrade e2e")
}

func newLogger() *zap.SugaredLogger {
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	return rawLogger.Sugar()
}

// ContinueOnFailure keeps one failing post-upgrade check from hiding the
// results of the others.
var _ = ginkgo.Describe("MLA upgrade", ginkgo.Ordered, ginkgo.ContinueOnFailure, func() {
	ginkgo.BeforeAll(func() {
		gomega.Expect(stateFile).NotTo(gomega.BeEmpty(), "-upgrade-state-file is required")
	})

	ginkgo.It("UP0 records data and configuration before the upgrade", ginkgo.Label("pre-upgrade"), func(ctx ginkgo.SpecContext) {
		logger := newLogger()

		seedClient, seedConfig, err := utils.GetClients()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		s := &state{}
		clusterName := existingCluster

		if clusterName == "" {
			gomega.Expect(credentials.Parse()).To(gomega.Succeed(), "failed to get AWS credentials")

			testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
			testJig.ClusterJig.WithTestName("mlaup")

			// delete the cluster when this run fails; on success the
			// post-upgrade run needs it and deletes it
			ginkgo.DeferCleanup(func(ctx ginkgo.SpecContext) {
				if ginkgo.CurrentSpecReport().Failed() {
					testJig.Cleanup(ctx, ginkgo.GinkgoT(), true)
				}
			}, ginkgo.NodeTimeout(45*time.Minute))

			project, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to set up the user cluster")

			clusterName = cluster.Name
			s.CreatedCluster = true
			s.ProjectName = project.Name
			s.ClusterName = clusterName

			// written early, so a failed setup below still leaves a record
			// of the cluster
			gomega.Expect(saveState(stateFile, s)).To(gomega.Succeed())
		}

		in, err := mlatestkit.LoadInput(ctx, seedClient, seedConfig, logger, clusterName, grafanaAdminEmail)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		gomega.Expect(mlatestkit.EnableMLA(ctx, in)).To(gomega.Succeed())

		upgradeState, err := mlatestkit.SetupUpgrade(ctx, in)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		s.UpgradeState = *upgradeState
		gomega.Expect(saveState(stateFile, s)).To(gomega.Succeed())
	}, ginkgo.NodeTimeout(60*time.Minute))

	// UP3 runs before UP1, because UP1 deletes the user cluster in its cleanup.
	ginkgo.It("UP3 ships the user cluster control plane logs to seed Loki after the upgrade", ginkgo.Label("post-upgrade", "needs-seed-mla"), func(ctx ginkgo.SpecContext) {
		seedClient, seedConfig, err := utils.GetClients()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		in := mlatestkit.SeedInput{SeedClient: seedClient, SeedConfig: seedConfig, Log: newLogger()}
		gomega.Expect(mlatestkit.CheckControlPlaneLogs(ctx, in)).To(gomega.Succeed())
		gomega.Expect(mlatestkit.CheckSeedLogDiscards(ctx, in)).To(gomega.Succeed())
	}, ginkgo.NodeTimeout(15*time.Minute))

	ginkgo.It("UP1 keeps user cluster MLA data and configuration through the upgrade", ginkgo.Label("post-upgrade"), func(ctx ginkgo.SpecContext) {
		logger := newLogger()

		s, err := loadState(stateFile)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		seedClient, seedConfig, err := utils.GetClients()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		if s.CreatedCluster {
			ginkgo.DeferCleanup(func(ctx ginkgo.SpecContext) {
				clusterJig := jig.NewClusterJig(seedClient, logger).WithExistingCluster(s.ClusterName)
				projectJig := jig.NewProjectJig(seedClient, logger).WithExistingProject(s.ProjectName)
				testJig := &jig.TestJig{ClusterJig: clusterJig, ProjectJig: projectJig}
				testJig.Cleanup(ctx, ginkgo.GinkgoT(), true)
			}, ginkgo.NodeTimeout(45*time.Minute))
		}

		in, err := mlatestkit.LoadInput(ctx, seedClient, seedConfig, logger, s.ClusterName, grafanaAdminEmail)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ginkgo.DeferCleanup(func(ctx ginkgo.SpecContext) {
			mlatestkit.CleanupUpgrade(ctx, in, &s.UpgradeState)
		})

		gomega.Expect(mlatestkit.VerifyUpgrade(ctx, in, &s.UpgradeState)).To(gomega.Succeed())
	}, ginkgo.NodeTimeout(45*time.Minute))

	ginkgo.It("UP2 keeps seed MLA data through the upgrade", ginkgo.Label("post-upgrade", "needs-seed-mla"), func(ctx ginkgo.SpecContext) {
		s, err := loadState(stateFile)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		seedClient, seedConfig, err := utils.GetClients()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		in := mlatestkit.SeedInput{SeedClient: seedClient, SeedConfig: seedConfig, Log: newLogger()}
		gomega.Expect(mlatestkit.VerifySeedRetention(ctx, in, s.SampleTime)).To(gomega.Succeed())
	}, ginkgo.NodeTimeout(10*time.Minute))
})

func saveState(path string, s *state) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func loadState(path string) (*state, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	s := &state{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("failed to parse upgrade state: %w", err)
	}

	if s.ClusterName == "" {
		return nil, errors.New("upgrade state has no cluster name")
	}

	return s, nil
}
