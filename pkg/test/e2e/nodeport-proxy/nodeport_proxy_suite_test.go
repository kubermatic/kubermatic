// +build e2e

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
	"flag"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
)

var deployer *Deployer
var networkingTest *NetworkingTestConfig
var skipCleanup bool
var debugLog bool
var versions common.Versions = common.NewDefaultVersions()

func init() {
	flag.StringVar(&versions.Kubermatic, "kubermatic-tag", "latest", "Kubermatic image tag to be used for the tests.")
	flag.BoolVar(&debugLog, "debug-log", false, "Activate debug logs.")
	flag.BoolVar(&skipCleanup, "skip-cleanup", false, "Skip clean-up of resources.")
}

func TestNodeportProxy(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "NodeportProxy Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	logger = CreateLogger(debugLog)
	k8scli, podRestCli, config := GetClientsOrDie()
	deployer = &Deployer{
		Log:      logger,
		Client:   k8scli,
		Versions: versions,
	}
	gomega.Expect(deployer.SetUp()).NotTo(gomega.HaveOccurred(), "nodeport-proxy should deploy successfully")
	networkingTest = &NetworkingTestConfig{
		Log:           logger,
		Namespace:     deployer.Namespace,
		Client:        k8scli,
		Config:        config,
		PodRestClient: podRestCli,
	}
	gomega.Expect(networkingTest.DeployTestPod()).NotTo(gomega.HaveOccurred(), "test pod should deploy successfully")
})

var _ = ginkgo.AfterSuite(func() {
	if !skipCleanup {
		gomega.Expect(networkingTest.CleanUp()).NotTo(gomega.HaveOccurred(), "failed to clean-up networkingTest")
		gomega.Expect(deployer.CleanUp()).NotTo(gomega.HaveOccurred(), "failed to clean-up deployer")
	}
})
