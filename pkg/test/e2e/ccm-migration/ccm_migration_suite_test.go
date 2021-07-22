// +build e2e

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package ccmmigration

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"k8c.io/kubermatic/v2/pkg/semver"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
)

// Options holds the e2e test options.
type testOptions struct {
	skipCleanup       bool
	debugLog          bool
	datacenter        string
	kubernetesVersion semver.Semver

	osCredentials credentials
}

var options = testOptions{
	kubernetesVersion: *semver.NewSemverOrDie("v1.20.2"),
}

func init() {
	flag.StringVar(&options.datacenter, "datacenter", "byo-kubernetes", "Name of the datacenter used by the user clusters created for the test.")
	flag.Var(&options.kubernetesVersion, "kubernetes-version", "Kubernetes version for the user cluster")
	flag.BoolVar(&options.debugLog, "debug-log", false, "Activate debug logs.")
	flag.BoolVar(&options.skipCleanup, "skip-cleanup", false, "Skip clean-up of resources.")

	flag.StringVar(&options.osCredentials.authUrl, "openstack-auth-url", "", "openstack auth url")
	flag.StringVar(&options.osCredentials.username, "openstack-username", "", "openstack username")
	flag.StringVar(&options.osCredentials.password, "openstack-password", "", "openstack password")
	flag.StringVar(&options.osCredentials.tenant, "openstack-tenant", "", "openstack tenant")
	flag.StringVar(&options.osCredentials.domain, "openstack-domain", "", "openstack domain")
	flag.StringVar(&options.osCredentials.region, "openstack-region", "", "openstack region")
	flag.StringVar(&options.osCredentials.floatingIpPool, "openstack-floating-ip-pool", "", "openstack floating ip pool")
	flag.StringVar(&options.osCredentials.network, "openstack-network", "", "openstack network")
}

func TestCCMMigration(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "CCM migration suite")
}

var _ = ginkgo.BeforeSuite(func() {
	e2eutils.DefaultLogger = e2eutils.CreateLogger(options.debugLog)
})
