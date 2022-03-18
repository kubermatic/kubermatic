//go:build e2e

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
	"os"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/providers"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
)

// Options holds the e2e test options.
type testOptions struct {
	skipCleanup       bool
	debugLog          bool
	kubernetesVersion semver.Semver

	provider kubermaticv1.ProviderType

	vsphereSeedDatacenter string
	osSeedDatacenter      string

	osCredentials      providers.OpenstackCredentialsType
	vSphereCredentials providers.VsphereCredentialsType
}

var (
	options = testOptions{
		kubernetesVersion: *semver.NewSemverOrDie(os.Getenv("VERSION_TO_TEST")),
	}
	provider string
)

func init() {
	flag.BoolVar(&options.debugLog, "debug-log", false, "Activate debug logs.")
	flag.BoolVar(&options.skipCleanup, "skip-cleanup", false, "Skip clean-up of resources.")

	flag.StringVar(&provider, "provider", "", "Cloud provider to test")

	flag.StringVar(&options.osSeedDatacenter, "openstack-seed-datacenter", "", "openstack datacenter")
	flag.StringVar(&options.vsphereSeedDatacenter, "vsphere-seed-datacenter", "", "vsphere seed datacenter")

	flag.StringVar(&options.osCredentials.AuthURL, "openstack-auth-url", "", "openstack auth url")
	flag.StringVar(&options.osCredentials.Username, "openstack-username", "", "openstack username")
	flag.StringVar(&options.osCredentials.Password, "openstack-password", "", "openstack password")
	flag.StringVar(&options.osCredentials.Tenant, "openstack-tenant", "", "openstack tenant")
	flag.StringVar(&options.osCredentials.Domain, "openstack-domain", "", "openstack domain")
	flag.StringVar(&options.osCredentials.Region, "openstack-region", "", "openstack region")
	flag.StringVar(&options.osCredentials.FloatingIPPool, "openstack-floating-ip-pool", "", "openstack floating ip pool")
	flag.StringVar(&options.osCredentials.Network, "openstack-network", "", "openstack network")

	flag.StringVar(&options.vSphereCredentials.AuthURL, "vsphere-auth-url", "", "vsphere auth-url")
	flag.StringVar(&options.vSphereCredentials.Username, "vsphere-username", "", "vsphere username")
	flag.StringVar(&options.vSphereCredentials.Password, "vsphere-password", "", "vsphere password")
	flag.StringVar(&options.vSphereCredentials.Datacenter, "vsphere-datacenter", "", "vsphere datacenter")
	flag.StringVar(&options.vSphereCredentials.Cluster, "vsphere-cluster", "", "vsphere cluster")

	options.provider = kubermaticv1.ProviderType(provider)
}

func TestCCMMigration(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "CCM migration suite")
}

var _ = ginkgo.BeforeSuite(func() {
	e2eutils.DefaultLogger = e2eutils.CreateLogger(options.debugLog)
})
