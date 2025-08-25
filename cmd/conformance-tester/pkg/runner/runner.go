/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package runner

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/onsi/ginkgo/reporters"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/clients"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/metrics"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/tests"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/util"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type TestRunner struct {
	log       *zap.SugaredLogger
	opts      *ctypes.Options
	kkpClient clients.Client

	createdProject bool
}

func NewKubeRunner(opts *ctypes.Options, log *zap.SugaredLogger) *TestRunner {
	return &TestRunner{
		log:       log,
		opts:      opts,
		kkpClient: clients.NewKubeClient(opts),
	}
}

func (r *TestRunner) Setup(ctx context.Context) error {
	return r.kkpClient.Setup(ctx, r.log)
}

func (r *TestRunner) SetupProject(ctx context.Context) error {
	if r.opts.KubermaticProject == "" {
		projectName, err := r.kkpClient.CreateProject(ctx, r.log, "e2e-"+rand.String(5))
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		r.opts.KubermaticProject = projectName
		r.createdProject = true
	}

	if err := r.kkpClient.EnsureSSHKeys(ctx, r.log); err != nil {
		return fmt.Errorf("failed to create SSH keys: %w", err)
	}

	return nil
}

func (r *TestRunner) Run(ctx context.Context, testScenarios []scenarios.Scenario) (*Results, error) {
	scenariosCh := make(chan scenarios.Scenario, len(testScenarios))
	resultsCh := make(chan ScenarioResult, len(testScenarios))

	r.log.Infow("Test suite:", "total", len(testScenarios))
	for _, scenario := range testScenarios {
		scenario.NamedLog(scenario.Log(r.log)).Info("Scenario")
		scenariosCh <- scenario
	}

	for i := 1; i <= r.opts.ClusterParallelCount; i++ {
		go r.scenarioWorker(ctx, scenariosCh, resultsCh)
	}

	close(scenariosCh)

	success := 0
	failures := 0
	skipped := 0
	remaining := len(testScenarios)

	var results []ScenarioResult
	for range testScenarios {
		result := <-resultsCh

		remaining--
		switch result.Status {
		case ScenarioPassed:
			success++
		case ScenarioFailed:
			failures++
		case ScenarioSkipped:
			skipped++
		}

		results = append(results, result)
		r.log.Infow("Scenario finished.", "successful", success, "failed", failures, "skipped", skipped, "remaining", remaining)
	}

	r.log.Info("All scenarios have finished.")

	return &Results{
		Options:   r.opts,
		Scenarios: results,
	}, nil
}

func (r *TestRunner) scenarioWorker(ctx context.Context, scenarios <-chan scenarios.Scenario, results chan<- ScenarioResult) {
	for scenario := range scenarios {
		var (
			report  *reporters.JUnitTestSuite
			cluster *kubermaticv1.Cluster
		)

		clusterVersion := scenario.ClusterVersion()

		result := ScenarioResult{
			scenarioName:      scenario.Name(),
			CloudProvider:     scenario.CloudProvider(),
			OperatingSystem:   scenario.OperatingSystem(),
			KubernetesRelease: clusterVersion.MajorMinor(),
			KubernetesVersion: clusterVersion,
		}

		// This check could be done much earlier, but doing it rather late ensures that
		// skipped scenarios end up in the test result like normal without any special handling.
		if err := r.isValidNewScenario(scenario); err != nil {
			scenario.Log(r.log).Infof("Skipping scenario: %v", err.Error())

			result.Status = ScenarioSkipped
			result.Message = fmt.Sprintf("Skipped: %v", err)
			results <- result

			continue
		}

		scenarioLog := scenario.NamedLog(r.log)
		scenarioLog.Info("Starting to test scenario...")

		start := time.Now()

		err := metrics.MeasureTime(metrics.ScenarioRuntimeMetric.With(prometheus.Labels{"scenario": scenario.Name()}), scenarioLog, func() error {
			var err error
			report, cluster, err = r.executeScenario(ctx, scenarioLog, scenario)
			return err
		})
		if err == nil {
			switch {
			case report == nil:
				err = errors.New("test report is empty")
			case len(report.TestCases) == 0:
				err = errors.New("test report contains no test cases")
			case report.Errors > 0 || report.Failures > 0:
				err = fmt.Errorf("test report contains %d errors and %d failures", report.Errors, report.Failures)
			}
		}

		if err != nil {
			result.Status = ScenarioFailed
			result.Message = err.Error()
			scenarioLog.Warnw("Finished with error", zap.Error(err))
		} else {
			result.Status = ScenarioPassed
			scenarioLog.Info("Finished successfully")
		}

		if cluster != nil {
			result.ClusterName = cluster.Name
			result.KubermaticVersion = cluster.Status.Conditions[kubermaticv1.ClusterConditionClusterInitialized].KubermaticVersion
		}

		result.Duration = time.Since(start)
		result.report = report

		results <- result
	}
}

func (r *TestRunner) isValidNewScenario(scenario scenarios.Scenario) error {
	// check if the OS is enabled by the user
	if !r.opts.Distributions.Has(string(scenario.OperatingSystem())) {
		return fmt.Errorf("OS is not enabled")
	}

	return scenario.IsValid()
}

func (r *TestRunner) executeScenario(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario) (*reporters.JUnitTestSuite, *kubermaticv1.Cluster, error) {
	report := &reporters.JUnitTestSuite{
		Name: scenario.Name(),
	}
	totalStart := time.Now()

	// We'll store the report there and all kinds of logs
	scenarioFolder := path.Join(r.opts.ReportsRoot, scenario.Name())
	if err := os.MkdirAll(scenarioFolder, os.ModePerm); err != nil {
		return nil, nil, fmt.Errorf("failed to create the scenario folder %q: %w", scenarioFolder, err)
	}

	// We need the closure to defer the evaluation of the time.Since(totalStart) call
	defer func() {
		log.Infof("Finished testing cluster after %s", time.Since(totalStart).Round(time.Second))
	}()

	// Always write junit to disk
	defer func() {
		report.Time = time.Since(totalStart).Seconds()
		b, err := xml.Marshal(report)
		if err != nil {
			log.Errorw("failed to marshal junit", zap.Error(err))
			return
		}
		if err := os.WriteFile(path.Join(r.opts.ReportsRoot, fmt.Sprintf("junit.%s.xml", scenario.Name())), b, 0644); err != nil {
			log.Errorw("Failed to write junit", zap.Error(err))
		}
	}()

	// create a cluster if no existing one should be used
	cluster, err := r.ensureCluster(ctx, log, scenario, report)
	if err != nil {
		return report, nil, err
	}

	log = log.With("cluster", cluster.Name)
	testError := r.executeTests(ctx, log, cluster, report, scenario)

	// refresh the variable with the latest state
	if err := r.opts.SeedClusterClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
		return nil, nil, err
	}

	if !r.opts.DeleteClusterAfterTests {
		return report, cluster, testError
	}

	deleteTimeout := 15 * time.Minute
	if cluster.Spec.Cloud.Azure != nil {
		// 15 Minutes are not enough for Azure
		deleteTimeout = 30 * time.Minute
	}

	if !r.opts.WaitForClusterDeletion {
		deleteTimeout = 0
	}

	clusterDeleteError := util.JUnitWrapper("[KKP] Delete cluster", report, func() error {
		// use a background context to ensure that when the test is cancelled using Ctrl-C,
		// the cleanup is still happening
		return r.kkpClient.DeleteCluster(context.Background(), log, cluster, deleteTimeout)
	})

	errs := []error{testError, clusterDeleteError}

	if r.createdProject {
		projectDeleteError := util.JUnitWrapper("[KKP] Delete project", report, func() error {
			// use a background context to ensure that when the test is cancelled using Ctrl-C,
			// the cleanup is still happening
			return r.kkpClient.DeleteProject(context.Background(), log, r.opts.KubermaticProject, deleteTimeout)
		})
		errs = append(errs, projectDeleteError)
	}

	return report, cluster, kerrors.NewAggregate(errs)
}

func (r *TestRunner) ensureCluster(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario, report *reporters.JUnitTestSuite) (*kubermaticv1.Cluster, error) {
	var (
		cluster *kubermaticv1.Cluster
		err     error
	)

	if r.opts.ExistingClusterLabel == "" {
		if err := util.JUnitWrapper("[KKP] Create cluster", report, func() error {
			cluster, err = r.kkpClient.CreateCluster(ctx, log, scenario)
			return err
		}); err != nil {
			return nil, fmt.Errorf("failed to create cluster: %w", err)
		}

		return cluster, nil
	}

	log.Info("Using existing cluster")
	selector, err := labels.Parse(r.opts.ExistingClusterLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to parse labelselector %q: %w", r.opts.ExistingClusterLabel, err)
	}

	clusterList := &kubermaticv1.ClusterList{}
	listOptions := &ctrlruntimeclient.ListOptions{LabelSelector: selector}
	if err := r.opts.SeedClusterClient.List(ctx, clusterList, listOptions); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	if foundClusterNum := len(clusterList.Items); foundClusterNum != 1 {
		return nil, fmt.Errorf("expected to find exactly one existing cluster, but got %d", foundClusterNum)
	}

	return &clusterList.Items[0], nil
}

func (r *TestRunner) executeTests(
	ctx context.Context,
	log *zap.SugaredLogger,
	cluster *kubermaticv1.Cluster,
	report *reporters.JUnitTestSuite,
	scenario scenarios.Scenario,
) error {
	// We must store the name here because the cluster object may be nil on error
	clusterName := cluster.Name

	// Print Cluster events and status information once the test has finished. Logs
	// of the control plane or KKP should be collected outside of the conformance-tester,
	// for example using protokol or similar tools.
	defer r.dumpClusterInformation(ctx, log, clusterName)

	var err error

	versions := kubermatic.GetVersions()

	// NB: It's important for this health check loop to refresh the cluster object, as
	// during reconciliation some cloud providers will fill in missing fields in the CloudSpec,
	// and later when we create MachineDeployments we potentially rely on these fields
	// being set in the cluster variable.
	healthCheck := func() error {
		log.Info("Waiting for cluster to be successfully reconciled...")

		return wait.PollLog(ctx, log, 5*time.Second, 10*time.Minute, func(ctx context.Context) (transient error, terminal error) {
			if err := r.opts.SeedClusterClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
				return err, nil
			}

			// ignore Kubermatic version in this check, to allow running against a 3rd party setup
			missingConditions, _ := controllerutil.ClusterReconciliationSuccessful(cluster, versions, true)
			if len(missingConditions) > 0 {
				return fmt.Errorf("missing conditions: %v", missingConditions), nil
			}

			return nil, nil
		})
	}

	if err := util.JUnitWrapper("[KKP] Wait for successful reconciliation", report, metrics.TimeMeasurementWrapper(
		metrics.KubermaticReconciliationDurationMetric.With(prometheus.Labels{"scenario": scenario.Name()}),
		log,
		healthCheck,
	)); err != nil {
		return fmt.Errorf("failed to wait for successful reconciliation: %w", err)
	}

	if err := util.JUnitWrapper("[KKP] Wait for control plane", report, metrics.TimeMeasurementWrapper(
		metrics.SeedControlplaneDurationMetric.With(prometheus.Labels{"scenario": scenario.Name()}),
		log,
		func() error {
			cluster, err = waitForControlPlane(ctx, log, r.opts, clusterName)
			return err
		},
	)); err != nil {
		return fmt.Errorf("failed waiting for control plane to become ready: %w", err)
	}

	if err := util.JUnitWrapper("[KKP] Add LB and PV Finalizers", report, func() error {
		return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			if err := r.opts.SeedClusterClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
				return err
			}
			cluster.Finalizers = append(cluster.Finalizers,
				kubermaticv1.InClusterPVCleanupFinalizer,
				kubermaticv1.InClusterLBCleanupFinalizer,
			)
			return r.opts.SeedClusterClient.Update(ctx, cluster)
		})
	}); err != nil {
		return fmt.Errorf("failed to add PV and LB cleanup finalizers: %w", err)
	}

	kubeconfigFilename, err := r.getKubeconfig(ctx, log, cluster)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	cloudConfigFilename, err := r.getCloudConfig(ctx, log, cluster)
	if err != nil {
		return fmt.Errorf("failed to get cloud config: %w", err)
	}

	var userClusterClient ctrlruntimeclient.Client

	// This can randomly fail if the apiserver is not 100% fully ready yet:
	//   failed to get the client for the cluster:
	//     failed to create restMapper:
	//       Get "https://<clustername>.kubermatic.worker.ci.k8c.io:<port>/api?timeout=32s":
	//         dial tcp <ciclusternodeip>:<port>:
	//           connect: connection refused
	// To prevent this from stopping a conformance test, we simply retry a couple of times.
	if err := wait.PollImmediate(ctx, 1*time.Second, 15*time.Second, func(ctx context.Context) (transient error, terminal error) {
		userClusterClient, err = r.opts.ClusterClientProvider.GetClient(ctx, cluster)
		return err, nil
	}); err != nil {
		return fmt.Errorf("failed to get the client for the cluster: %w", err)
	}

	if err := util.JUnitWrapper("[KKP] Create MachineDeployments", report, func() error {
		return r.kkpClient.CreateMachineDeployments(ctx, log, scenario, userClusterClient, cluster)
	}); err != nil {
		return fmt.Errorf("failed to setup nodes: %w", err)
	}

	defer logEventsForAllMachines(ctx, log, userClusterClient)
	deferredGatherUserClusterLogs(ctx, log, r.opts, cluster.DeepCopy())

	overallTimeout := r.opts.NodeReadyTimeout
	// The initialization of the external CCM is super slow
	if cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
		overallTimeout += 5 * time.Minute
	}

	var timeoutRemaining time.Duration

	if err := util.JUnitWrapper("[KKP] Wait for machines to get a node", report, metrics.TimeMeasurementWrapper(
		metrics.NodeCreationDuration.With(prometheus.Labels{"scenario": scenario.Name()}),
		log,
		func() error {
			var err error
			timeoutRemaining, err = waitForMachinesToJoinCluster(ctx, log, userClusterClient, overallTimeout)
			return err
		},
	)); err != nil {
		return fmt.Errorf("failed to wait for machines to get a node: %w", err)
	}

	if err := util.JUnitWrapper("[KKP] Wait for nodes to be ready", report, metrics.TimeMeasurementWrapper(
		metrics.NodeRadinessDuration.With(prometheus.Labels{"scenario": scenario.Name()}),
		log,
		func() error {
			// Getting ready just implies starting the CNI deamonset, so that should be quick.
			var err error
			timeoutRemaining, err = waitForNodesToBeReady(ctx, log, userClusterClient, timeoutRemaining)
			return err
		},
	)); err != nil {
		return fmt.Errorf("failed to wait for all nodes to be ready: %w", err)
	}

	if err := util.JUnitWrapper("[KKP] Wait for Pods inside usercluster to be ready", report, metrics.TimeMeasurementWrapper(
		metrics.SeedControlplaneDurationMetric.With(prometheus.Labels{"scenario": scenario.Name()}),
		log,
		func() error {
			return waitUntilAllPodsAreReady(ctx, log, r.opts, userClusterClient, timeoutRemaining)
		},
	)); err != nil {
		return fmt.Errorf("failed to wait for all pods to get ready: %w", err)
	}

	if err := r.checkAddons(ctx, report, log, cluster); err != nil {
		return fmt.Errorf("failed waiting for addons to become ready: %w", err)
	}

	const maxTestAttempts = 3
	if err := util.JUnitWrapper("[KKP] Test Network Policy Enforcement", report, func() error {
		return util.MeasuredRetryNWithSummary(
			func(v float64) {
				metrics.NetworkPolicyTestRuntimeMetric.With(prometheus.Labels{"scenario": scenario.Name()}).Observe(v)
			},
			metrics.NetworkPolicyTestAttemptsMetric.With(prometheus.Labels{"scenario": scenario.Name()}),
			log,
			3*time.Second,
			maxTestAttempts,
			func(attempt int) error {
				return tests.TestNetworkPolicy(ctx, userClusterClient)
			},
		)
	}); err != nil {
		log.Errorf("Failed to verify network policy enforcement: %v", err)
	}

	if err := util.JUnitWrapper("[KKP] Test Pod Disruption Budget enforcement", report, func() error {
		return util.RetryN(5*time.Second, maxTestAttempts, func(attempt int) error {
			return tests.TestPodDisruptionBudget(ctx, userClusterClient)
		})
	}); err != nil {
		log.Errorf("Failed to verify PodDisruptionBudget enforcement: %v", err)
	}

	if err := r.testCluster(ctx, log, scenario, cluster, userClusterClient, kubeconfigFilename, cloudConfigFilename, report); err != nil {
		return fmt.Errorf("failed to test cluster: %w", err)
	}

	if r.opts.TestClusterUpdate {
		if err := r.updateClusterToNextMinor(ctx, log, scenario, cluster, userClusterClient, kubeconfigFilename, cloudConfigFilename, report); err != nil {
			return fmt.Errorf("failed to test cluster: %w", err)
		}

		if err := r.opts.SeedClusterClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
			return err
		}

		if err := r.testCluster(ctx, log, scenario, cluster, userClusterClient, kubeconfigFilename, cloudConfigFilename, report); err != nil {
			return fmt.Errorf("failed to test updated cluster: %w", err)
		}
	}

	return nil
}

func (r *TestRunner) testCluster(
	ctx context.Context,
	log *zap.SugaredLogger,
	scenario scenarios.Scenario,
	cluster *kubermaticv1.Cluster,
	userClusterClient ctrlruntimeclient.Client,
	kubeconfigFilename string,
	cloudConfigFilename string,
	report *reporters.JUnitTestSuite,
) error {
	const maxTestAttempts = 3

	defaultLabels := prometheus.Labels{
		"scenario": scenario.Name(),
	}

	log.Info("Starting to test cluster...")

	// Run the Kubernetes conformance tests
	if err := tests.TestKubernetesConformance(ctx, log, r.opts, scenario, cluster, userClusterClient, kubeconfigFilename, cloudConfigFilename, report); err != nil {
		log.Errorf("Conformance tests failed: %v", err)
	}

	// Do a simple PVC test
	if err := util.JUnitWrapper("[KKP] [CloudProvider] Test PersistentVolumes", report, util.MeasuredRetryN(
		metrics.PVCTestRuntimeMetric.MustCurryWith(defaultLabels),
		metrics.PVCTestAttemptsMetric.With(defaultLabels),
		log,
		3*time.Second,
		maxTestAttempts,
		func(attempt int) error {
			return tests.TestStorage(ctx, log, r.opts, cluster, userClusterClient, attempt)
		},
	)); err != nil {
		log.Errorf("Failed to verify that PVC's work: %v", err)
	}

	// Do a simple LoadBalancer test
	if err := util.JUnitWrapper("[KKP] [CloudProvider] Test LoadBalancers", report, util.MeasuredRetryN(
		metrics.LBTestRuntimeMetric.MustCurryWith(defaultLabels),
		metrics.LBTestAttemptsMetric.With(defaultLabels),
		log,
		3*time.Second,
		maxTestAttempts,
		func(attempt int) error {
			return tests.TestLoadBalancer(ctx, log, r.opts, cluster, userClusterClient, attempt)
		},
	)); err != nil {
		log.Errorf("Failed to verify that LoadBalancers work: %v", err)
	}

	// Do user cluster RBAC controller test
	if err := util.JUnitWrapper("[KKP] Test user cluster RBAC controller", report, func() error {
		return util.RetryN(5*time.Second, maxTestAttempts, func(attempt int) error {
			return tests.TestUserclusterControllerRBAC(ctx, log, r.opts, cluster, userClusterClient, r.opts.SeedClusterClient)
		})
	}); err != nil {
		log.Errorf("Failed to verify that user cluster RBAC controller work: %v", err)
	}

	// Metrics availability can take forever so for these tests we need
	// to try for longer than normal.
	maxMetrixAttempts := 15

	// Do prometheus metrics available test
	if err := util.JUnitWrapper("[KKP] Test prometheus metrics availability", report, func() error {
		return util.RetryN(5*time.Second, maxMetrixAttempts, func(attempt int) error {
			return tests.TestUserClusterMetrics(ctx, log, r.opts, cluster, r.opts.SeedClusterClient)
		})
	}); err != nil {
		log.Errorf("Failed to verify that prometheus metrics are available: %v", err)
	}

	// Do pod and node metrics availability test
	if err := util.JUnitWrapper("[KKP] Test pod and node metrics availability", report, func() error {
		return util.RetryN(5*time.Second, maxMetrixAttempts, func(attempt int) error {
			return tests.TestUserClusterPodAndNodeMetrics(ctx, log, r.opts, cluster, userClusterClient)
		})
	}); err != nil {
		log.Errorf("Failed to verify that pod and node metrics are available: %v", err)
	}

	// Check seccomp profiles for Pods running on user cluster
	if err := util.JUnitWrapper("[KKP] Test pod seccomp profiles on user cluster", report, func() error {
		return util.RetryN(5*time.Second, maxTestAttempts, func(attempt int) error {
			return tests.TestUserClusterSeccompProfiles(ctx, log, r.opts, cluster, userClusterClient)
		})
	}); err != nil {
		log.Errorf("failed to verify that pods have a seccomp profile: %v", err)
	}

	// Check for Pods with k8s.gcr.io images on user cluster
	if err := util.JUnitWrapper("[KKP] Test container images not containing k8s.gcr.io on user cluster", report, func() error {
		return util.RetryN(5*time.Second, maxTestAttempts, func(attempt int) error {
			return tests.TestUserClusterNoK8sGcrImages(ctx, log, r.opts, cluster, userClusterClient)
		})
	}); err != nil {
		log.Errorf("failed to verify that no user cluster containers has k8s.gcr.io image: %v", err)
	}

	// Check security context (seccomp profiles) for control plane pods running on seed cluster
	if err := util.JUnitWrapper("[KKP] Test pod security context on seed cluster", report, func() error {
		return util.RetryN(5*time.Second, maxTestAttempts, func(attempt int) error {
			return tests.TestUserClusterControlPlaneSecurityContext(ctx, log, r.opts, cluster)
		})
	}); err != nil {
		log.Errorf("failed to verify security context for control plane pods: %v", err)
	}

	// Check for Pods with k8s.gcr.io images on user cluster
	if err := util.JUnitWrapper("[KKP] Test container images not containing k8s.gcr.io on seed cluster", report, func() error {
		return util.RetryN(5*time.Second, maxTestAttempts, func(attempt int) error {
			return tests.TestNoK8sGcrImages(ctx, log, r.opts, cluster)
		})
	}); err != nil {
		log.Errorf("failed to verify that no seed cluster containers has k8s.gcr.io image: %v", err)
	}

	// Check telemetry is working
	if err := util.JUnitWrapper("[KKP] Test telemetry", report, func() error {
		return util.RetryN(5*time.Second, maxTestAttempts, func(attempt int) error {
			return tests.TestTelemetry(ctx, log, r.opts)
		})
	}); err != nil {
		log.Errorf("failed to verify telemetry is working: %v", err)
	}

	log.Info("All tests completed.")

	return nil
}

func (r *TestRunner) updateClusterToNextMinor(
	ctx context.Context,
	log *zap.SugaredLogger,
	scenario scenarios.Scenario,
	cluster *kubermaticv1.Cluster,
	userClusterClient ctrlruntimeclient.Client,
	kubeconfigFilename string,
	cloudConfigFilename string,
	report *reporters.JUnitTestSuite,
) error {
	var err error

	currentVersion := cluster.Spec.Version.Semver()
	nextRelease := fmt.Sprintf("%d.%d", currentVersion.Major(), currentVersion.Minor()+1)

	nextVersion := test.LatestKubernetesVersionForRelease(nextRelease, r.opts.KubermaticConfiguration)
	if nextVersion == nil {
		return fmt.Errorf("cannot update cluster as there is no %s.x version configured", nextRelease)
	}

	oldCluster := cluster.DeepCopy()
	cluster.Spec.Version = *nextVersion

	log.Infow("Updating cluster to next release", "from", currentVersion.String(), "to", nextVersion.String())
	if err := r.opts.SeedClusterClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to patch cluster with new version %q: %w", nextVersion, err)
	}

	// Wait a moment to let the controllers begin reconciling, otherwise the wait loop below
	// might just see the current healthy state and not actually wait for the reconciliation to complete.
	time.Sleep(10 * time.Second)

	clusterName := cluster.Name

	if err := util.JUnitWrapper("[KKP] Wait for control plane", report, metrics.TimeMeasurementWrapper(
		metrics.SeedControlplaneDurationMetric.With(prometheus.Labels{"scenario": scenario.Name()}),
		log,
		func() error {
			cluster, err = waitForControlPlane(ctx, log, r.opts, clusterName)
			return err
		},
	)); err != nil {
		return fmt.Errorf("failed waiting for control plane to become ready: %w", err)
	}

	// Check that we actually upgraded.
	if !cluster.Status.Versions.ControlPlane.Equal(nextVersion) {
		return fmt.Errorf("cluster control plane version is %q after the update to %q, something did not work", cluster.Status.Versions.ControlPlane, nextVersion)
	}

	// Upgrade all MDs to the new cluster version.
	mdList := &clusterv1alpha1.MachineDeploymentList{}
	if err := userClusterClient.List(ctx, mdList); err != nil {
		return fmt.Errorf("failed to list MachineDeployments: %w", err)
	}

	log.Info("Updating MachineDeployments...")
	for _, md := range mdList.Items {
		err = wait.PollLog(ctx, log, 1*time.Second, 30*time.Second, func(ctx context.Context) (transient error, terminal error) {
			oldMD := md.DeepCopy()
			md.Spec.Template.Spec.Versions.Kubelet = nextVersion.String()

			return userClusterClient.Patch(ctx, &md, ctrlruntimeclient.MergeFrom(oldMD)), nil
		})
		if err != nil {
			return fmt.Errorf("failed to update MachineDeployment %q: %w", md.Name, err)
		}
	}

	// Wait for all nodes to reach the new version.
	err = wait.PollLog(ctx, log, 30*time.Second, 2*r.opts.NodeReadyTimeout, func(ctx context.Context) (transient error, terminal error) {
		nodeList := &corev1.NodeList{}
		if err := userClusterClient.List(ctx, nodeList); err != nil {
			return fmt.Errorf("failed to list nodes: %w", err), nil
		}

		outdated := sets.New[string]()
		unready := sets.New[string]()

		for _, node := range nodeList.Items {
			if !util.NodeIsReady(node) {
				unready.Insert(node.Name)
				continue
			}

			kubeletVersion := semver.NewSemverOrDie(node.Status.NodeInfo.KubeletVersion)
			if !kubeletVersion.Equal(nextVersion) {
				outdated.Insert(node.Name)
			}
		}

		if outdated.Len() > 0 || unready.Len() > 0 {
			return fmt.Errorf("not all nodes rotated: %v are unready, %v are outdated", sets.List(unready), sets.List(outdated)), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for all nodes to be rotated: %w", err)
	}

	// And now wait for all pods.
	if err := util.JUnitWrapper("[KKP] Wait for Pods inside usercluster to be ready", report, metrics.TimeMeasurementWrapper(
		metrics.SeedControlplaneDurationMetric.With(prometheus.Labels{"scenario": scenario.Name()}),
		log,
		func() error {
			return waitUntilAllPodsAreReady(ctx, log, r.opts, userClusterClient, r.opts.NodeReadyTimeout)
		},
	)); err != nil {
		return fmt.Errorf("failed to wait for all pods to get ready: %w", err)
	}

	if err := r.checkAddons(ctx, report, log, cluster); err != nil {
		return fmt.Errorf("failed waiting for addons to become ready: %w", err)
	}

	log.Info("Cluster update complete.")

	return nil
}

func (r *TestRunner) checkAddons(ctx context.Context, report *reporters.JUnitTestSuite, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	return util.JUnitWrapper("[KKP] Wait for addons", report, func() error {
		return wait.PollLog(ctx, log, 2*time.Second, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
			addons := kubermaticv1.AddonList{}
			if err := r.opts.SeedClusterClient.List(ctx, &addons, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
				return err, nil
			}

			unhealthyAddons := sets.New[string]()
			for _, addon := range addons.Items {
				if addon.Status.Conditions[kubermaticv1.AddonReconciledSuccessfully].Status != corev1.ConditionTrue {
					unhealthyAddons.Insert(addon.Name)
				}
			}

			if unhealthyAddons.Len() > 0 {
				return fmt.Errorf("unhealthy addons: %v", sets.List(unhealthyAddons)), nil
			}

			return nil, nil
		})
	})
}

func (r *TestRunner) getKubeconfig(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (string, error) {
	log.Debug("Getting kubeconfig...")

	kubeconfig, err := r.opts.ClusterClientProvider.GetAdminKubeconfig(ctx, cluster)
	if err != nil {
		return "", fmt.Errorf("failed to wait for kubeconfig: %w", err)
	}

	filename := path.Join(r.opts.HomeDir, fmt.Sprintf("%s-kubeconfig", cluster.Name))
	if err := os.WriteFile(filename, kubeconfig, 0644); err != nil {
		return "", fmt.Errorf("failed to write kubeconfig to %s: %w", filename, err)
	}

	log.Infof("Successfully wrote kubeconfig to %s", filename)
	return filename, nil
}

func (r *TestRunner) getCloudConfig(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (string, error) {
	log.Debug("Getting cloud-config...")

	name := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.CloudConfigSeedSecretName}
	cm := &corev1.Secret{}
	if err := r.opts.SeedClusterClient.Get(ctx, name, cm); err != nil {
		return "", fmt.Errorf("failed to get Secret %s: %w", name.String(), err)
	}

	filename := path.Join(r.opts.HomeDir, fmt.Sprintf("%s-cloud-config", cluster.Name))
	if err := os.WriteFile(filename, cm.Data["config"], 0644); err != nil {
		return "", fmt.Errorf("failed to write cloud config: %w", err)
	}

	log.Infof("Successfully wrote cloud-config to %s", filename)
	return filename, nil
}

func (r *TestRunner) dumpClusterInformation(ctx context.Context, log *zap.SugaredLogger, clusterName string) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.opts.SeedClusterClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		log.Errorw("Failed to get cluster", zap.Error(err))
		return
	}

	log.Infow("Cluster health status", "status", cluster.Status.ExtendedHealth, "phase", cluster.Status.Phase)

	log.Info("Logging events for cluster")
	if err := logEventsObject(ctx, log, r.opts.SeedClusterClient, "default", cluster.UID); err != nil {
		log.Errorw("Failed to log cluster events", zap.Error(err))
	}
}
