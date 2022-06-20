//go:build e2e

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

package nodeportproxy

import (
	"context"
	"flag"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/log"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
)

var deployer *Deployer
var networkingTest *networkingTestConfig
var logger *zap.SugaredLogger
var skipCleanup bool
var versions = kubermatic.NewDefaultVersions()
var logOptions = e2eutils.DefaultLogOptions

func init() {
	flag.StringVar(&versions.Kubermatic, "kubermatic-tag", "latest", "Kubermatic image tag to be used for the tests")
	flag.BoolVar(&skipCleanup, "skip-cleanup", false, "Skip clean-up of resources")

	logOptions.AddFlags(flag.CommandLine)
}

func TestNodeportProxy(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "NodeportProxy Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	logger = log.NewFromOptions(logOptions).Sugar()
	k8scli, podRestCli, config := e2eutils.GetClientsOrDie()
	deployer = &Deployer{
		Log:      logger,
		Client:   k8scli,
		Versions: versions,
	}
	networkingTest = &networkingTestConfig{
		TestPodConfig: e2eutils.TestPodConfig{
			Log:           logger,
			Client:        k8scli,
			Config:        config,
			PodRestClient: podRestCli,
			CreatePodFunc: newAgnhostPod,
		},
	}
	gomega.Expect(deployer.SetUp(context.Background())).NotTo(gomega.HaveOccurred(), "nodeport-proxy should deploy successfully")
	// We put the test pod in same namespace as the nodeport proxy
	networkingTest.Namespace = deployer.Namespace
	gomega.Expect(networkingTest.DeployTestPod(context.Background(), logger)).NotTo(gomega.HaveOccurred(), "test pod should deploy successfully")
})

var _ = ginkgo.AfterSuite(func() {
	if !skipCleanup {
		gomega.Expect(networkingTest.CleanUp(context.Background())).NotTo(gomega.HaveOccurred(), "failed to clean-up networkingTest")
		gomega.Expect(deployer.CleanUp(context.Background())).NotTo(gomega.HaveOccurred(), "failed to clean-up deployer")
	}
})
