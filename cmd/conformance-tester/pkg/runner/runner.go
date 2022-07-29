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
	"bytes"
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

	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/clients"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/metrics"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/tests"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/util"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type TestRunner struct {
	log       *zap.SugaredLogger
	opts      *ctypes.Options
	kkpClient clients.Client
}

func NewAPIRunner(opts *ctypes.Options, log *zap.SugaredLogger) *TestRunner {
	return &TestRunner{
		log:       log.With("client", "api"),
		opts:      opts,
		kkpClient: clients.NewAPIClient(opts),
	}
}

func NewKubeRunner(opts *ctypes.Options, log *zap.SugaredLogger) *TestRunner {
	return &TestRunner{
		log:       log.With("client", "kube"),
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
	}

	if err := r.kkpClient.CreateSSHKeys(ctx, r.log); err != nil {
		return fmt.Errorf("failed to create SSH keys: %w", err)
	}

	return nil
}

func (r *TestRunner) Run(ctx context.Context, testScenarios []scenarios.Scenario) error {
	scenariosCh := make(chan scenarios.Scenario, len(testScenarios))
	resultsCh := make(chan testResult, len(testScenarios))

	r.log.Info("Test suite:")
	for _, scenario := range testScenarios {
		r.log.Info(scenario.Name())
		scenariosCh <- scenario
	}
	r.log.Infof("Total: %d tests", len(testScenarios))

	for i := 1; i <= r.opts.ClusterParallelCount; i++ {
		go r.scenarioWorker(ctx, scenariosCh, resultsCh)
	}

	close(scenariosCh)

	var results []testResult
	for range testScenarios {
		results = append(results, <-resultsCh)
		r.log.Infof("Finished %d/%d test cases", len(results), len(testScenarios))
	}

	overallResultBuf := &bytes.Buffer{}
	hadFailure := false
	for _, result := range results {
		prefix := "PASS"
		if !result.Passed() {
			prefix = "FAIL"
			hadFailure = true
		}
		scenarioResultMsg := fmt.Sprintf("[%s] - %s", prefix, result.scenario.Name())
		if result.err != nil {
			scenarioResultMsg = fmt.Sprintf("%s : %v", scenarioResultMsg, result.err)
		}

		fmt.Fprintln(overallResultBuf, scenarioResultMsg)
		if result.report != nil {
			printDetailedReport(result.report)
		}
	}

	fmt.Println("========================== RESULT ===========================")
	fmt.Println(overallResultBuf.String())

	if hadFailure {
		return errors.New("some tests failed")
	}

	return nil
}

func (r *TestRunner) scenarioWorker(ctx context.Context, scenarios <-chan scenarios.Scenario, results chan<- testResult) {
	for s := range scenarios {
		var report *reporters.JUnitTestSuite

		scenarioLog := r.log.With("scenario", s.Name())
		scenarioLog.Info("Starting to test scenario...")

		err := metrics.MeasureTime(metrics.ScenarioRuntimeMetric.With(prometheus.Labels{"scenario": s.Name()}), scenarioLog, func() error {
			var err error
			report, err = r.executeScenario(ctx, scenarioLog, s)
			return err
		})
		if err != nil {
			scenarioLog.Warnw("Finished with error", zap.Error(err))
		} else {
			scenarioLog.Info("Finished")
		}

		results <- testResult{
			report:   report,
			scenario: s,
			err:      err,
		}
	}
}

func (r *TestRunner) executeScenario(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario) (*reporters.JUnitTestSuite, error) {
	report := &reporters.JUnitTestSuite{
		Name: scenario.Name(),
	}
	totalStart := time.Now()

	// We'll store the report there and all kinds of logs
	scenarioFolder := path.Join(r.opts.ReportsRoot, scenario.Name())
	if err := os.MkdirAll(scenarioFolder, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create the scenario folder %q: %w", scenarioFolder, err)
	}

	// We need the closure to defer the evaluation of the time.Since(totalStart) call
	defer func() {
		log.Infof("Finished testing cluster after %s", time.Since(totalStart))
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
		return report, err
	}

	log = log.With("cluster", cluster.Name)

	if err := r.executeTests(ctx, log, cluster, report, scenario); err != nil {
		return report, err
	}

	if !r.opts.DeleteClusterAfterTests {
		return report, nil
	}

	deleteTimeout := 15 * time.Minute
	if cluster.Spec.Cloud.Azure != nil {
		// 15 Minutes are not enough for Azure
		deleteTimeout = 30 * time.Minute
	}

	if err := util.JUnitWrapper("[KKP] Delete cluster", report, func() error {
		return r.kkpClient.DeleteCluster(ctx, log, cluster, deleteTimeout)
	}); err != nil {
		return report, fmt.Errorf("failed to delete cluster: %w", err)
	}

	return report, nil
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

	// NB: It's important for this health check loop to refresh the cluster object, as
	// during reconciliation some cloud providers will fill in missing fields in the CloudSpec,
	// and later when we create MachineDeployments we potentially rely on these fields
	// being set in the cluster variable.
	healthCheck := func() error {
		log.Info("Waiting for cluster to be successfully reconciled...")

		return wait.PollLog(log, 5*time.Second, 5*time.Minute, func() (transient error, terminal error) {
			if err := r.opts.SeedClusterClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
				return err, nil
			}

			versions := kubermatic.NewDefaultVersions()

			// ignore Kubermatic version in this check, to allow running against a 3rd party setup
			missingConditions, _ := kubermaticv1helper.ClusterReconciliationSuccessful(cluster, versions, true)
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

	providerName, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
	if err != nil {
		return fmt.Errorf("failed to get cloud provider name from cluster: %w", err)
	}

	log = log.With("cloud-provider", providerName)

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
	if err := wait.PollImmediate(1*time.Second, 15*time.Second, func() (transient error, terminal error) {
		userClusterClient, err = r.opts.ClusterClientProvider.GetClient(ctx, cluster)
		return err, nil
	}); err != nil {
		return fmt.Errorf("failed to get the client for the cluster: %w", err)
	}

	if err := util.JUnitWrapper("[KKP] Create NodeDeployments", report, func() error {
		return r.kkpClient.CreateNodeDeployments(ctx, log, scenario, userClusterClient, cluster)
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
	// Packet is slower at provisioning the instances, presumably because those are actual
	// physical hosts.
	if cluster.Spec.Cloud.Packet != nil {
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

	if r.opts.OnlyTestCreation {
		log.Info("All nodes are ready. Only testing cluster creation, skipping further tests.")
		return nil
	}

	if err := r.testCluster(ctx, log, scenario, cluster, userClusterClient, kubeconfigFilename, cloudConfigFilename, report); err != nil {
		return fmt.Errorf("failed to test cluster: %w", err)
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
		maxTestAttempts,
		func(attempt int) error {
			return tests.TestLoadBalancer(ctx, log, r.opts, cluster, userClusterClient, attempt)
		},
	)); err != nil {
		log.Errorf("Failed to verify that LoadBalancers work: %v", err)
	}

	// Do user cluster RBAC controller test
	if err := util.JUnitWrapper("[KKP] Test user cluster RBAC controller", report, func() error {
		return util.RetryN(maxTestAttempts, func(attempt int) error {
			return tests.TestUserclusterControllerRBAC(ctx, log, r.opts, cluster, userClusterClient, r.opts.SeedClusterClient)
		})
	}); err != nil {
		log.Errorf("Failed to verify that user cluster RBAC controller work: %v", err)
	}

	// Do prometheus metrics available test
	if err := util.JUnitWrapper("[KKP] Test prometheus metrics availability", report, func() error {
		return util.RetryN(maxTestAttempts, func(attempt int) error {
			return tests.TestUserClusterMetrics(ctx, log, r.opts, cluster, r.opts.SeedClusterClient)
		})
	}); err != nil {
		log.Errorf("Failed to verify that prometheus metrics are available: %v", err)
	}

	// Do pod and node metrics availability test
	if err := util.JUnitWrapper("[KKP] Test pod and node metrics availability", report, func() error {
		return util.RetryN(maxTestAttempts, func(attempt int) error {
			return tests.TestUserClusterPodAndNodeMetrics(ctx, log, r.opts, cluster, userClusterClient)
		})
	}); err != nil {
		log.Errorf("Failed to verify that pod and node metrics are available: %v", err)
	}

	// Check seccomp profiles for Pods running on user cluster
	if err := util.JUnitWrapper("[KKP] Test pod seccomp profiles on user cluster", report, func() error {
		return util.RetryN(maxTestAttempts, func(attempt int) error {
			return tests.TestUserClusterSeccompProfiles(ctx, log, r.opts, cluster, userClusterClient)
		})
	}); err != nil {
		log.Errorf("failed to verify that pods have a seccomp profile: %v", err)
	}

	// Check security context (seccomp profiles) for control plane pods running on seed cluster
	if err := util.JUnitWrapper("[KKP] Test pod security context on seed cluster", report, func() error {
		return util.RetryN(maxTestAttempts, func(attempt int) error {
			return tests.TestUserClusterControlPlaneSecurityContext(ctx, log, r.opts, cluster)
		})
	}); err != nil {
		log.Errorf("failed to verify security context for control plane pods: %v", err)
	}

	// Check telemetry is working
	if err := util.JUnitWrapper("[KKP] Test telemetry", report, func() error {
		return util.RetryN(maxTestAttempts, func(attempt int) error {
			return tests.TestTelemetry(ctx, log, r.opts)
		})
	}); err != nil {
		log.Errorf("failed to verify telemetry is working: %v", err)
	}

	log.Info("All tests completed.")

	return nil
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

	name := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.CloudConfigConfigMapName}
	cm := &corev1.ConfigMap{}
	if err := r.opts.SeedClusterClient.Get(ctx, name, cm); err != nil {
		return "", fmt.Errorf("failed to get ConfigMap %s: %w", name.String(), err)
	}

	filename := path.Join(r.opts.HomeDir, fmt.Sprintf("%s-cloud-config", cluster.Name))
	if err := os.WriteFile(filename, []byte(cm.Data["config"]), 0644); err != nil {
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
