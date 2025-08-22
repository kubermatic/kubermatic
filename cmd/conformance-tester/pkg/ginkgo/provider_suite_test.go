package ginkgo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-logr/zapr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/clients"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/metrics"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	legacytypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	rootCtx            context.Context
	log                *zap.SugaredLogger
	opts               *Options
	runtimeOpts        *RuntimeOptions
	legacyOpts         *legacytypes.Options
	client             clients.Client
	testSuiteScenarios []scenarios.Scenario
	scenarioFailureMap map[string][]Failure
	// junitReporter      *JUnitReporter
)

func init() {
	// setup logging
	logOpts := kubermaticlog.NewDefaultOptions()
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log = rawLog.Sugar()
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))
	kkpreconciling.Configure(log)

	// load options
	var err error
	opts, err = newOptionsFromYAML(log)
	if err != nil {
		log.Fatalw("Failed to load options", zap.Error(err))
	}

	// setup context
	rootCtx = signals.SetupSignalHandler()
	runtimeOpts, err = NewRuntimeOptions(rootCtx, log, opts)
	if err != nil {
		log.Fatalw("Failed to create runtime options", zap.Error(err))
	}

	legacyOpts = toLegacyOptions(opts, runtimeOpts)
	// testSuiteScenarios, err = scenarios.NewGenerator().
	// 	WithCloudProviders(string(providerconfig.CloudProviderKubeVirt), string(providerconfig.CloudProviderHetzner)).
	// 	WithDualstack(opts.DualStackEnabled).
	// 	WithOperatingSystems(string(providerconfig.OperatingSystemUbuntu), string(providerconfig.OperatingSystemRockyLinux)).
	// 	WithVersions(semver.NewSemverOrDie("v1.33.3"), semver.NewSemverOrDie("v1.32.7"), semver.NewSemverOrDie("v1.31.11")).
	// 	Scenarios(rootCtx, legacyOpts, log)
	testSuiteScenarios, err = scenarios.NewGenerator().
		WithCloudProviders(legacyOpts.Providers.UnsortedList()...).
		WithDualstack(opts.DualStackEnabled).
		WithOperatingSystems(legacyOpts.Distributions.UnsortedList()...).
		WithVersions(legacyOpts.Versions...).
		Scenarios(rootCtx, legacyOpts, log)
	if err != nil {
		log.Fatalw("Failed to generate scenarios", zap.Error(err))
	}

	scenarioFailureMap = make(map[string][]Failure)
	// junitReporter = NewJUnitReporter(opts.ReportsRoot)
}

func TestScenarios(t *testing.T) {
	RegisterFailHandler(CustomFailHandler)

	// To replicate the per-scenario JUnit reporting from the original implementation,
	// we add our custom JUnit reporter. It writes one XML file per spec to the `reports`
	// directory.
	if err := os.MkdirAll(opts.ReportsRoot, 0755); err != nil {
		t.Fatalf("Failed to create reports directory: %v", err)
	}

	suiteConfig, reporterConfig := GinkgoConfiguration()
	format.RegisterCustomFormatter(formatter)

	RunSpecs(t, "Conformance Tester Scenarios Suite", suiteConfig, reporterConfig)
}

func CustomFailHandler(message string, callerSkip ...int) {
	skip := 0
	if len(callerSkip) > 0 {
		skip = callerSkip[0]
	}
	currentSpecReport := CurrentSpecReport()
	scenarioFailureMap[currentSpecReport.ContainerHierarchyTexts[len(currentSpecReport.ContainerHierarchyTexts)-1]] = append(scenarioFailureMap[currentSpecReport.ContainerHierarchyTexts[len(currentSpecReport.ContainerHierarchyTexts)-1]], Failure{
		Message: message,
		Step:    currentSpecReport.SpecEvents[len(currentSpecReport.SpecEvents)-1].Message,
	})
	// currentSpecReport = append(currentSpecReport.Failure.AdditionalFailures, types.AdditionalFailure{Failure: types.Failure{Message: message}})
	log.Infof("Skipping %d message %v", skip, message)
}

func formatter(value any) (string, bool) {
	// handle github.com/pkg/errors with a stack
	pkgErr, isPkgError := value.(interface{ Cause() error })
	if isPkgError {
		return fmt.Sprintf("%+v", pkgErr), true
	}

	return "", false
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// This function runs once on a single process.
	// It's responsible for setting up the global environment, like creating
	// the KKP project and ensuring SSH keys exist.
	By(KKP("Setting up metrics"))
	if legacyOpts.PushgatewayEndpoint != "" {
		metrics.Setup(legacyOpts.PushgatewayEndpoint, os.Getenv("JOB_NAME"), os.Getenv("PROW_JOB_ID"))
	}

	By(KKP("Creating a KKP client"))
	client = clients.NewKubeClient(legacyOpts)
	Expect(client.Setup(rootCtx, log)).To(Succeed())

	By(KKP("Ensuring a project exists"))
	if opts.KubermaticProject == "" {
		projectName := "e2e-" + rand.String(5)
		p, err := client.CreateProject(rootCtx, log, projectName)
		Expect(err).NotTo(HaveOccurred())
		projectName = p
		opts.KubermaticProject = projectName
		legacyOpts.KubermaticProject = projectName
	}

	By(KKP("Ensuring SSH keys exist"))
	Expect(client.EnsureSSHKeys(rootCtx, log)).To(Succeed())

	kubermaticProject := ptr.String(legacyOpts.KubermaticProject)
	data, err := json.Marshal(kubermaticProject)
	Expect(err).NotTo(HaveOccurred())
	return data
}, func(data []byte) {
	// This function runs on every parallel process.
	// It's responsible for setting up the process-specific environment.
	var kubermaticProject *string
	err := json.Unmarshal(data, &kubermaticProject)
	Expect(err).NotTo(HaveOccurred())
	legacyOpts.KubermaticProject = *kubermaticProject
	client = clients.NewKubeClient(legacyOpts)
	Expect(client.Setup(rootCtx, log)).To(Succeed())
	// fmt.Fprintf(GinkgoWriter, "Using project %q\n", *kubermaticProject)
})

var _ = SynchronizedAfterSuite(func() {
	// This function runs on every parallel process.
	// Here we could clean up resources created by each process.
}, func() {
	// This function runs once on a single process after all tests have finished.
	if opts.KubermaticProject == "" {
		By(KKP("Deleting the project"))
		deleteTimeout := 15 * time.Minute
		Expect(client.DeleteProject(rootCtx, log, opts.KubermaticProject, deleteTimeout)).To(Succeed())
	}

	By(KKP("Updating metrics"))
	// if legacyOpts.PushgatewayEndpoint != "" {
	// 	metrics.UpdateMetrics(log)
	// }
})

// var _ = ReportBeforeSuite(func(r Report) {

// })

var _ = ReportAfterSuite("ReportAfterSuite", func(r Report) {
	By("Report after smoke tests")
	// junitReporter.AfterSuite(r)
})
