package ginkgo

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/clients"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/tests"
	"k8c.io/machine-controller/sdk/providerconfig"
)

// The ReportAfterEach function has been removed in favor of using AfterEach
// with CurrentSpecReport(). This is a Ginkgo best practice as it allows the
// reporting logic to live alongside the test and have access to its closure
// (e.g., the `cluster` variable) for more detailed, context-aware reporting on failure.

// Following Ginkgo best practices, we iterate over the providers to create a `Describe`
// block for each, rather than using a custom `DescribeTableSubtree` helper. This improves
// readability and aligns with standard Ginkgo patterns.
var _ = Describe("[provider]", func() {
	// Define all known providers that can be tested. This list is static and ensures
	// that all possible provider tests are generated. The decision to run or skip a
	// test is made dynamically inside the test execution based on the configuration.
	allProviders := map[string]providerconfig.CloudProvider{
		"KubeVirt": providerconfig.CloudProviderKubeVirt,
		"Hetzner":  providerconfig.CloudProviderHetzner,
		// Add other providers here as they become available for testing.
	}

	// Convert the configured provider names from the options into a set for efficient lookups.
	// enabledProviders := sets.NewString(opts.Providers...)
	// enabledDistributions := sets.NewString(opts.Distributions...)
	// enabledReleases := sets.NewString(opts.Releases...)

	for description, provider := range allProviders {
		// Capture range variables to ensure they have the correct value in the closure.
		provider := provider
		description := description

		Describe(description, func() {
			// Context("Smoke Tests", Label("TIER:bronze"), func() {
			// A `DescribeTable` is used here for the scenarios, which is the idiomatic
			// way to run the same test logic against multiple data-driven inputs.
			DescribeTableSubtree("with scenario",
				func(scenario scenarios.Scenario) {
					// The function passed to DescribeTable serves as the complete test body for each entry.
					// Unlike Describe/Context blocks, it's not a container for other nodes like
					// `BeforeEach`, `AfterEach`, or `It`. Therefore, setup is performed directly
					// at the beginning of the function, and teardown is managed with a `defer`
					// statement to ensure it executes reliably after the test logic.

					var (
						cluster           *kubermaticv1.Cluster
						userClusterClient ctrlruntimeclient.Client
					)
					BeforeEach(func() {
						cluster, userClusterClient = commonSetup(rootCtx, log, scenario, legacyOpts)
					})

					AfterEach(func() {
						commonCleanup(rootCtx, log, client, scenario, userClusterClient, cluster)
						currentSpecReport := CurrentSpecReport()
						if currentSpecReport.Failed() {
							By("Capturing diagnostics for failed test")
							// e.g., AddReportEntry("Cluster Events", captureClusterEvents(cluster))
						}
						r := NewJUnitReporter(opts.ReportsRoot)
						r.failures = scenarioFailureMap[currentSpecReport.ContainerHierarchyTexts[len(currentSpecReport.ContainerHierarchyTexts)-1]]
						r.SpecDidComplete(currentSpecReport)
						r.AfterSuite(currentSpecReport)
					})

					It("should succeed", func() {
						// This is the actual test logic.
						machineSetup(rootCtx, log, clients.NewKubeClient(legacyOpts), scenario, userClusterClient, cluster, legacyOpts)

						// Individual smoke tests are wrapped in `By` to clearly delineate them in the test report.
						By(KKP(CloudProvider("Test container images not containing k8s.gcr.io on seed cluster")), func() {
							Expect(tests.TestNoK8sGcrImages(rootCtx, log, legacyOpts, cluster)).NotTo(HaveOccurred())
						})

						By(KKP(CloudProvider("Test PersistentVolumes")), func() {
							Expect(tests.TestStorage(rootCtx, log, legacyOpts, cluster, userClusterClient, 1)).NotTo(HaveOccurred())
						})

						By(KKP("Test user cluster RBAC controller"), func() {
							Expect(tests.TestUserclusterControllerRBAC(rootCtx, log, legacyOpts, cluster, userClusterClient, legacyOpts.SeedClusterClient)).NotTo(HaveOccurred())
						})
					})
				},
				// Scenarios are generated dynamically for the current provider in the loop.
				scenarioEntriesByProvider(testSuiteScenarios, provider))
		})
	}
})
