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

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/onsi/ginkgo/reporters"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	apiclient "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client"
	projectclient "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/project"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilerror "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func podIsReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

type testScenario interface {
	Name() string
	Cluster(secrets secrets) *apimodels.CreateClusterSpec
	NodeDeployments(ctx context.Context, num int, secrets secrets) ([]apimodels.NodeDeployment, error)
	OS() apimodels.OperatingSystemSpec
}

func newRunner(scenarios []testScenario, opts *Opts, log *zap.SugaredLogger) *testRunner {
	return &testRunner{
		log:                          log,
		scenarios:                    scenarios,
		controlPlaneReadyWaitTimeout: opts.controlPlaneReadyWaitTimeout,
		nodeReadyTimeout:             opts.nodeReadyTimeout,
		customTestTimeout:            opts.customTestTimeout,
		userClusterPollInterval:      opts.userClusterPollInterval,
		deleteClusterAfterTests:      opts.deleteClusterAfterTests,
		secrets:                      opts.secrets,
		namePrefix:                   opts.namePrefix,
		clusterClientProvider:        opts.clusterClientProvider,
		seed:                         opts.seed,
		seedRestConfig:               opts.seedRestConfig,
		nodeCount:                    opts.nodeCount,
		repoRoot:                     opts.repoRoot,
		reportsRoot:                  opts.reportsRoot,
		clusterParallelCount:         opts.clusterParallelCount,
		PublicKeys:                   opts.publicKeys,
		workerName:                   opts.workerName,
		homeDir:                      opts.homeDir,
		seedClusterClient:            opts.seedClusterClient,
		seedGeneratedClient:          opts.seedGeneratedClient,
		existingClusterLabel:         opts.existingClusterLabel,
		printGinkoLogs:               opts.printGinkoLogs,
		printContainerLogs:           opts.printContainerLogs,
		onlyTestCreation:             opts.onlyTestCreation,
		pspEnabled:                   opts.pspEnabled,
		kubermaticProjectID:          opts.kubermaticProjectID,
		kubermaticClient:             opts.kubermaticClient,
		kubermaticAuthenticator:      opts.kubermaticAuthenticator,
	}
}

type testRunner struct {
	log                *zap.SugaredLogger
	scenarios          []testScenario
	secrets            secrets
	namePrefix         string
	repoRoot           string
	reportsRoot        string
	PublicKeys         [][]byte
	workerName         string
	homeDir            string
	printGinkoLogs     bool
	printContainerLogs bool
	onlyTestCreation   bool
	pspEnabled         bool

	controlPlaneReadyWaitTimeout time.Duration
	nodeReadyTimeout             time.Duration
	customTestTimeout            time.Duration
	userClusterPollInterval      time.Duration
	deleteClusterAfterTests      bool
	nodeCount                    int
	clusterParallelCount         int

	seedClusterClient     ctrlruntimeclient.Client
	seedGeneratedClient   kubernetes.Interface
	clusterClientProvider *clusterclient.Provider
	seed                  *kubermaticv1.Seed
	seedRestConfig        *rest.Config

	// The label to use to select an existing cluster to test against instead of
	// creating a new one
	existingClusterLabel string

	kubermaticProjectID     string
	kubermaticClient        *apiclient.KubermaticAPI
	kubermaticAuthenticator runtime.ClientAuthInfoWriter
}

type testResult struct {
	report   *reporters.JUnitTestSuite
	err      error
	scenario testScenario
}

func (t *testResult) Passed() bool {
	if t.err != nil {
		return false
	}

	if t.report == nil {
		return false
	}

	if len(t.report.TestCases) == 0 {
		return false
	}

	if t.report.Errors > 0 || t.report.Failures > 0 {
		return false
	}

	return true
}

func (r *testRunner) worker(ctx context.Context, scenarios <-chan testScenario, results chan<- testResult) {
	for s := range scenarios {
		var report *reporters.JUnitTestSuite

		scenarioLog := r.log.With("scenario", s.Name())
		scenarioLog.Info("Starting to test scenario...")

		err := measureTime(scenarioRuntimeMetric.With(prometheus.Labels{"scenario": s.Name()}), scenarioLog, func() error {
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

func (r *testRunner) Run(ctx context.Context) error {
	scenariosCh := make(chan testScenario, len(r.scenarios))
	resultsCh := make(chan testResult, len(r.scenarios))

	r.log.Info("Test suite:")
	for _, scenario := range r.scenarios {
		r.log.Info(scenario.Name())
		scenariosCh <- scenario
	}
	r.log.Infof("Total: %d tests", len(r.scenarios))

	for i := 1; i <= r.clusterParallelCount; i++ {
		go r.worker(ctx, scenariosCh, resultsCh)
	}

	close(scenariosCh)

	var results []testResult
	for range r.scenarios {
		results = append(results, <-resultsCh)
		r.log.Infof("Finished %d/%d test cases", len(results), len(r.scenarios))
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

func (r *testRunner) executeScenario(ctx context.Context, log *zap.SugaredLogger, scenario testScenario) (*reporters.JUnitTestSuite, error) {
	var err error
	var cluster *kubermaticv1.Cluster

	report := &reporters.JUnitTestSuite{
		Name: scenario.Name(),
	}
	totalStart := time.Now()

	// We'll store the report there and all kinds of logs
	scenarioFolder := path.Join(r.reportsRoot, scenario.Name())
	if err := os.MkdirAll(scenarioFolder, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create the scenario folder '%s': %v", scenarioFolder, err)
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
		if err := ioutil.WriteFile(path.Join(r.reportsRoot, fmt.Sprintf("junit.%s.xml", scenario.Name())), b, 0644); err != nil {
			log.Errorw("Failed to write junit", zap.Error(err))
		}
	}()

	if r.existingClusterLabel == "" {
		if err := junitReporterWrapper(
			"[Kubermatic] Create cluster",
			report,
			func() error {
				cluster, err = r.createCluster(ctx, log, scenario)
				return err
			}); err != nil {
			return report, fmt.Errorf("failed to create cluster: %v", err)
		}
	} else {
		log.Info("Using existing cluster")
		selector, err := labels.Parse(r.existingClusterLabel)
		if err != nil {
			return nil, fmt.Errorf("failed to parse labelselector %q: %v", r.existingClusterLabel, err)
		}
		clusterList := &kubermaticv1.ClusterList{}
		listOptions := &ctrlruntimeclient.ListOptions{LabelSelector: selector}
		if err := r.seedClusterClient.List(ctx, clusterList, listOptions); err != nil {
			return nil, fmt.Errorf("failed to list clusters: %v", err)
		}
		if foundClusterNum := len(clusterList.Items); foundClusterNum != 1 {
			return nil, fmt.Errorf("expected to find exactly one existing cluster, but got %d", foundClusterNum)
		}
		cluster = &clusterList.Items[0]
	}
	clusterName := cluster.Name
	log = log.With("cluster", cluster.Name)

	if err := junitReporterWrapper(
		"[Kubermatic] Wait for successful reconciliation",
		report,
		timeMeasurementWrapper(
			kubermaticReconciliationDurationMetric.With(prometheus.Labels{"scenario": scenario.Name()}),
			log,
			func() error {
				return wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
					if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
						log.Errorw("Failed to get cluster when waiting for successful reconciliation", zap.Error(err))
						return false, nil
					}

					versions := kubermatic.NewDefaultVersions()
					// ignore Kubermatic version in this check, to allow running against a 3rd party setup
					missingConditions, success := kubermaticv1helper.ClusterReconciliationSuccessful(cluster, versions, true)
					if len(missingConditions) > 0 {
						log.Infof("Waiting for the following conditions: %v", missingConditions)
					}
					return success, nil
				})
			},
		),
	); err != nil {
		return report, fmt.Errorf("failed to wait for successful reconciliation: %v", err)
	}

	if err := r.executeTests(ctx, log, cluster, report, scenario); err != nil {
		return report, err
	}

	if !r.deleteClusterAfterTests {
		return report, nil
	}

	return report, r.deleteCluster(ctx, report, cluster, log)
}

func (r *testRunner) executeTests(
	ctx context.Context,
	log *zap.SugaredLogger,
	cluster *kubermaticv1.Cluster,
	report *reporters.JUnitTestSuite,
	scenario testScenario,
) error {
	// We must store the name here because the cluster object may be nil on error
	clusterName := cluster.Name

	if r.printContainerLogs {
		// Print all controlplane logs to both make debugging easier and show issues
		// that didn't result in test failures.
		defer r.printAllControlPlaneLogs(ctx, log, clusterName)
	}

	var err error

	if err := junitReporterWrapper(
		"[Kubermatic] Wait for control plane",
		report,
		timeMeasurementWrapper(
			seedControlplaneDurationMetric.With(prometheus.Labels{"scenario": scenario.Name()}),
			log,
			func() error {
				cluster, err = r.waitForControlPlane(ctx, log, clusterName)
				return err
			},
		),
	); err != nil {
		return fmt.Errorf("failed waiting for control plane to become ready: %v", err)
	}

	if err := junitReporterWrapper(
		"[Kubermatic] Add LB and PV Finalizers",
		report,
		func() error {
			return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
					return err
				}
				cluster.Finalizers = append(cluster.Finalizers,
					kubermaticapiv1.InClusterPVCleanupFinalizer,
					kubermaticapiv1.InClusterLBCleanupFinalizer,
				)
				return r.seedClusterClient.Update(ctx, cluster)
			})
		},
	); err != nil {
		return fmt.Errorf("failed to add PV and LB cleanup finalizers: %v", err)
	}

	providerName, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
	if err != nil {
		return fmt.Errorf("failed to get cloud provider name from cluster: %v", err)
	}

	log = log.With("cloud-provider", providerName)

	_, exists := r.seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !exists {
		return fmt.Errorf("datacenter %q doesn't exist", cluster.Spec.Cloud.DatacenterName)
	}

	kubeconfigFilename, err := r.getKubeconfig(ctx, log, cluster)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %v", err)
	}

	cloudConfigFilename, err := r.getCloudConfig(ctx, log, cluster)
	if err != nil {
		return fmt.Errorf("failed to get cloud config: %v", err)
	}

	userClusterClient, err := r.clusterClientProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get the client for the cluster: %v", err)
	}

	if err := junitReporterWrapper(
		"[Kubermatic] Create NodeDeployments",
		report,
		func() error {
			return r.createNodeDeployments(ctx, log, scenario, clusterName)
		},
	); err != nil {
		return fmt.Errorf("failed to setup nodes: %v", err)
	}

	if r.printContainerLogs {
		defer logEventsForAllMachines(ctx, log, userClusterClient)
		defer logUserClusterPodEventsAndLogs(ctx, log, r.clusterClientProvider, cluster.DeepCopy())
	}

	overallTimeout := r.nodeReadyTimeout
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

	if err := junitReporterWrapper(
		"[Kubermatic] Wait for machines to get a node",
		report,
		timeMeasurementWrapper(
			nodeCreationDuration.With(prometheus.Labels{"scenario": scenario.Name()}),
			log,
			func() error {
				var err error
				timeoutRemaining, err = waitForMachinesToJoinCluster(ctx, log, userClusterClient, overallTimeout)
				return err
			},
		),
	); err != nil {
		return fmt.Errorf("failed to wait for machines to get a node: %v", err)
	}

	if err := junitReporterWrapper(
		"[Kubermatic] Wait for nodes to be ready",
		report,
		timeMeasurementWrapper(
			nodeRadinessDuration.With(prometheus.Labels{"scenario": scenario.Name()}),
			log,
			func() error {
				// Getting ready just implies starting the CNI deamonset, so that should
				// be quick.
				var err error
				timeoutRemaining, err = waitForNodesToBeReady(ctx, log, userClusterClient, timeoutRemaining)
				return err
			},
		),
	); err != nil {
		return fmt.Errorf("failed to wait for all nodes to be ready: %v", err)
	}

	if err := junitReporterWrapper(
		"[Kubermatic] Wait for Pods inside usercluster to be ready",
		report,
		timeMeasurementWrapper(
			seedControlplaneDurationMetric.With(prometheus.Labels{"scenario": scenario.Name()}),
			log,
			func() error {
				return r.waitUntilAllPodsAreReady(ctx, log, userClusterClient, timeoutRemaining)
			},
		),
	); err != nil {
		return fmt.Errorf("failed to wait for all pods to get ready: %v", err)
	}

	if r.onlyTestCreation {
		return nil
	}

	if err := r.testCluster(
		ctx,
		log,
		scenario,
		cluster,
		userClusterClient,
		kubeconfigFilename,
		cloudConfigFilename,
		report,
	); err != nil {
		return fmt.Errorf("failed to test cluster: %v", err)
	}

	return nil
}

func (r *testRunner) deleteCluster(ctx context.Context, report *reporters.JUnitTestSuite, cluster *kubermaticv1.Cluster, log *zap.SugaredLogger) error {
	deleteTimeout := 15 * time.Minute
	if cluster.Spec.Cloud.Azure != nil {
		// 15 Minutes are not enough for Azure
		deleteTimeout = 30 * time.Minute
	}

	if err := junitReporterWrapper(
		"[Kubermatic] Delete cluster",
		report,
		func() error {
			var selector labels.Selector
			var err error
			if r.workerName != "" {
				selector, err = labels.Parse(fmt.Sprintf("worker-name=%s", r.workerName))
				if err != nil {
					return fmt.Errorf("failed to parse selector: %v", err)
				}
			}
			return wait.PollImmediate(5*time.Second, deleteTimeout, func() (bool, error) {
				clusterList := &kubermaticv1.ClusterList{}
				listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: selector}
				if err := r.seedClusterClient.List(ctx, clusterList, listOpts); err != nil {
					log.Errorw("Listing clusters failed", zap.Error(err))
					return false, nil
				}

				// Success!
				if len(clusterList.Items) == 0 {
					return true, nil
				}

				// Should never happen
				if len(clusterList.Items) > 1 {
					return false, fmt.Errorf("expected to find zero or one cluster, got %d", len(clusterList.Items))
				}

				// Cluster is currently being deleted
				if clusterList.Items[0].DeletionTimestamp != nil {
					return false, nil
				}

				// Issue Delete call
				log.With("cluster", clusterList.Items[0].Name).Info("Deleting user cluster now...")

				deleteParms := &projectclient.DeleteClusterParams{
					Context:   ctx,
					ProjectID: r.kubermaticProjectID,
					ClusterID: clusterList.Items[0].Name,
					DC:        r.seed.Name,
				}
				utils.SetupParams(nil, deleteParms, 3*time.Second, deleteTimeout)

				if _, err := r.kubermaticClient.Project.DeleteCluster(deleteParms, r.kubermaticAuthenticator); err != nil {
					log.Warnw("Failed to delete cluster", zap.Error(err))
				}

				return false, nil
			})
		},
	); err != nil {
		log.Errorw("Failed to delete cluster", zap.Error(err))
		return err
	}

	return nil
}

func retryNAttempts(maxAttempts int, f func(attempt int) error) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = f(attempt)
		if err != nil {
			continue
		}
		return nil
	}
	return fmt.Errorf("function did not succeed after %d attempts: %v", maxAttempts, err)
}

// measuredRetryNAttempts wraps retryNAttempts with code that counts
// the executed number of attempts and the runtimes for each
// attempt.
func measuredRetryNAttempts(
	runtimeMetric *prometheus.GaugeVec,
	//nolint:interfacer
	attemptsMetric prometheus.Gauge,
	log *zap.SugaredLogger,
	maxAttempts int,
	f func(attempt int) error,
) func() error {
	return func() error {
		attempts := 0

		err := retryNAttempts(maxAttempts, func(attempt int) error {
			attempts++
			metric := runtimeMetric.With(prometheus.Labels{"attempt": strconv.Itoa(attempt)})

			return measureTime(metric, log, func() error {
				return f(attempt)
			})
		})

		attemptsMetric.Set(float64(attempts))
		updateMetrics(log)

		return err
	}
}

func (r *testRunner) testCluster(
	ctx context.Context,
	log *zap.SugaredLogger,
	scenario testScenario,
	cluster *kubermaticv1.Cluster,
	userClusterClient ctrlruntimeclient.Client,
	kubeconfigFilename string,
	cloudConfigFilename string,
	report *reporters.JUnitTestSuite,
) error {
	const maxTestAttempts = 3
	// var err error
	log.Info("Starting to test cluster...")

	ginkgoRuns, err := r.getGinkgoRuns(log, scenario, kubeconfigFilename, cloudConfigFilename, cluster)
	if err != nil {
		return fmt.Errorf("failed to get Ginkgo runs: %v", err)
	}
	for _, run := range ginkgoRuns {
		if err := junitReporterWrapper(
			fmt.Sprintf("[Ginkgo] Run ginkgo tests %q", run.name),
			report,
			func() error {
				ginkgoRes, err := r.executeGinkgoRunWithRetries(ctx, log, scenario, run, userClusterClient)
				if ginkgoRes != nil {
					// We append the report from Ginkgo to our scenario wide report
					appendReport(report, ginkgoRes.report)
				}
				return err
			},
		); err != nil {
			log.Errorf("Ginkgo scenario '%s' failed, giving up retrying: %v", err)
			// We still want to run potential next runs
			continue
		}
	}

	defaultLabels := prometheus.Labels{
		"scenario": scenario.Name(),
	}

	// Do a simple PVC test - with retries
	if supportsStorage(cluster) {
		if err := junitReporterWrapper(
			"[Kubermatic] [CloudProvider] Test PersistentVolumes",
			report,
			measuredRetryNAttempts(
				pvctestRuntimeMetric.MustCurryWith(defaultLabels),
				pvctestAttemptsMetric.With(defaultLabels),
				log,
				maxTestAttempts,
				func(attempt int) error {
					return r.testPVC(ctx, log, userClusterClient, attempt)
				},
			),
		); err != nil {
			log.Errorf("Failed to verify that PVC's work: %v", err)
		}
	}

	// Do a simple LB test - with retries
	if supportsLBs(cluster) {
		if err := junitReporterWrapper(
			"[Kubermatic] [CloudProvider] Test LoadBalancers",
			report,
			measuredRetryNAttempts(
				lbtestRuntimeMetric.MustCurryWith(defaultLabels),
				lbtestAttemptsMetric.With(defaultLabels),
				log,
				maxTestAttempts,
				func(attempt int) error {
					return r.testLB(ctx, log, userClusterClient, attempt)
				},
			),
		); err != nil {
			log.Errorf("Failed to verify that LB's work: %v", err)
		}
	}

	// Do user cluster RBAC controller test - with retries
	if err := junitReporterWrapper(
		"[Kubermatic] Test user cluster RBAC controller",
		report,
		func() error {
			return retryNAttempts(maxTestAttempts, func(attempt int) error {
				return r.testUserclusterControllerRBAC(ctx, log, cluster, userClusterClient, r.seedClusterClient)
			})
		}); err != nil {
		log.Errorf("Failed to verify that user cluster RBAC controller work: %v", err)
	}

	// Do prometheus metrics available test - with retries
	if err := junitReporterWrapper(
		"[Kubermatic] Test prometheus metrics availability", report, func() error {
			return retryNAttempts(maxTestAttempts, func(attempt int) error {
				return r.testUserClusterMetrics(ctx, log, cluster, r.seedClusterClient)
			})
		}); err != nil {
		log.Errorf("Failed to verify that prometheus metrics are available: %v", err)
	}

	// Do pod and node metrics availability test - with retries
	if err := junitReporterWrapper(
		"[Kubermatic] Test pod and node metrics availability", report, func() error {
			return retryNAttempts(maxTestAttempts, func(attempt int) error {
				return r.testUserClusterPodAndNodeMetrics(ctx, log, cluster, userClusterClient)
			})
		}); err != nil {
		log.Errorf("Failed to verify that pod and node metrics are available: %v", err)
	}

	return nil
}

// executeGinkgoRunWithRetries executes the passed GinkgoRun and retries if it failed hard(Failed to execute the Ginkgo binary for example)
// Or if the JUnit report from Ginkgo contains failed tests.
// Only if Ginkgo failed hard, an error will be returned. If some tests still failed after retrying the run, the report will reflect that.
func (r *testRunner) executeGinkgoRunWithRetries(ctx context.Context, log *zap.SugaredLogger, scenario testScenario, run *ginkgoRun, client ctrlruntimeclient.Client) (ginkgoRes *ginkgoResult, err error) {
	const maxAttempts = 3

	attempts := 1
	defer func() {
		ginkgoAttemptsMetric.With(prometheus.Labels{
			"scenario": scenario.Name(),
			"run":      run.name,
		}).Set(float64(attempts))
		updateMetrics(log)
	}()

	for attempts = 1; attempts <= maxAttempts; attempts++ {
		ginkgoRes, err = r.executeGinkgoRun(ctx, log, run, client)

		if ginkgoRes != nil {
			ginkgoRuntimeMetric.With(prometheus.Labels{
				"scenario": scenario.Name(),
				"run":      run.name,
				"attempt":  strconv.Itoa(attempts),
			}).Set(ginkgoRes.duration.Seconds())
			updateMetrics(log)
		}

		if err != nil {
			// Something critical happened and we don't have a valid result
			log.Errorf("Failed to execute the Ginkgo run '%s': %v", run.name, err)
			continue
		}

		if ginkgoRes.report.Errors > 0 || ginkgoRes.report.Failures > 0 {
			msg := fmt.Sprintf("Ginkgo run '%s' had failed tests.", run.name)
			if attempts < maxAttempts {
				msg = fmt.Sprintf("%s. Retrying...", msg)
			}
			log.Info(msg)
			if r.printGinkoLogs {
				if err := printFileUnbuffered(ginkgoRes.logfile); err != nil {
					log.Infof("Error printing ginkgo logfile: %v", err)
				}
				log.Info("Successfully printed logfile")
			}
			continue
		}

		// Ginkgo run successfully and no test failed
		return ginkgoRes, err
	}

	return ginkgoRes, err
}

func (r *testRunner) createNodeDeployments(ctx context.Context, log *zap.SugaredLogger, scenario testScenario, clusterName string) error {
	nodeDeploymentGetParams := &projectclient.ListNodeDeploymentsParams{
		Context:   ctx,
		ProjectID: r.kubermaticProjectID,
		ClusterID: clusterName,
		DC:        r.seed.Name,
	}
	utils.SetupParams(nil, nodeDeploymentGetParams, 5*time.Second, 1*time.Minute)

	log.Info("Getting existing NodeDeployments")
	resp, err := r.kubermaticClient.Project.ListNodeDeployments(nodeDeploymentGetParams, r.kubermaticAuthenticator)
	if err != nil {
		return fmt.Errorf("failed to get existing NodeDeployments: %v", err)
	}

	existingReplicas := 0
	for _, nodeDeployment := range resp.Payload {
		existingReplicas += int(*nodeDeployment.Spec.Replicas)
	}
	log.Infof("Found %d pre-existing node replicas", existingReplicas)

	nodeCount := r.nodeCount - existingReplicas
	if nodeCount < 0 {
		return fmt.Errorf("found %d existing replicas and want %d, scaledown not supported", existingReplicas, r.nodeCount)
	}
	if nodeCount == 0 {
		return nil
	}

	log.Info("Preparing NodeDeployments")
	var nodeDeployments []apimodels.NodeDeployment
	if err := wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) {
		var err error
		nodeDeployments, err = scenario.NodeDeployments(ctx, nodeCount, r.secrets)
		if err != nil {
			log.Warnw("Getting NodeDeployments from scenario failed", zap.Error(err))
			return false, nil
		}
		return true, nil
	}); err != nil {
		return fmt.Errorf("didn't get NodeDeployments from scenario within a minute: %v", err)
	}

	log.Info("Creating NodeDeployments via Kubermatic API")
	for _, nd := range nodeDeployments {
		params := &projectclient.CreateNodeDeploymentParams{
			Context:   ctx,
			ProjectID: r.kubermaticProjectID,
			ClusterID: clusterName,
			DC:        r.seed.Name,
			Body:      &nd,
		}
		utils.SetupParams(nil, params, 5*time.Second, 1*time.Minute, http.StatusConflict)

		if _, err := r.kubermaticClient.Project.CreateNodeDeployment(params, r.kubermaticAuthenticator); err != nil {
			return fmt.Errorf("failed to create NodeDeployment %s: %v", nd.Name, err)
		}
	}

	log.Infof("Successfully created %d NodeDeployments via Kubermatic API", nodeCount)
	return nil
}

func (r *testRunner) getKubeconfig(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (string, error) {
	log.Debug("Getting kubeconfig...")
	var kubeconfig []byte
	// Needed for Openshift where we have to create a SA and bindings inside the cluster
	// which can only be done after the APIServer is up and ready
	if err := wait.PollImmediate(1*time.Second, 1*time.Minute, func() (bool, error) {
		var err error
		kubeconfig, err = r.clusterClientProvider.GetAdminKubeconfig(ctx, cluster)
		if err != nil {
			log.Debugw("Failed to get Kubeconfig", zap.Error(err))
			return false, nil
		}
		return true, nil
	}); err != nil {
		return "", fmt.Errorf("failed to wait for kubeconfig: %v", err)
	}

	filename := path.Join(r.homeDir, fmt.Sprintf("%s-kubeconfig", cluster.Name))
	if err := ioutil.WriteFile(filename, kubeconfig, 0644); err != nil {
		return "", fmt.Errorf("failed to write kubeconfig to %s: %v", filename, err)
	}

	log.Infof("Successfully wrote kubeconfig to %s", filename)
	return filename, nil
}

func (r *testRunner) getCloudConfig(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (string, error) {
	log.Debug("Getting cloud-config...")

	name := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.CloudConfigConfigMapName}
	cmData := ""

	if err := wait.PollImmediate(3*time.Second, 5*time.Minute, func() (bool, error) {
		cm := &corev1.ConfigMap{}
		if err := r.seedClusterClient.Get(ctx, name, cm); err != nil {
			log.Warnw("Failed to load cloud-config", zap.Error(err))
			return false, nil
		}

		cmData = cm.Data["config"]
		return true, nil
	}); err != nil {
		return "", fmt.Errorf("failed to get ConfigMap %s: %v", name.String(), err)
	}

	filename := path.Join(r.homeDir, fmt.Sprintf("%s-cloud-config", cluster.Name))
	if err := ioutil.WriteFile(filename, []byte(cmData), 0644); err != nil {
		return "", fmt.Errorf("failed to write cloud config: %v", err)
	}

	log.Infof("Successfully wrote cloud-config to %s", filename)
	return filename, nil
}

func (r *testRunner) createCluster(ctx context.Context, log *zap.SugaredLogger, scenario testScenario) (*kubermaticv1.Cluster, error) {
	log.Info("Creating cluster via Kubermatic API")

	cluster := scenario.Cluster(r.secrets)
	// The cluster name must be unique per project.
	// We build up a readable name with the various cli parameters & add a random string in the end to ensure
	// we really have a unique name
	if r.namePrefix != "" {
		cluster.Cluster.Name = r.namePrefix + "-"
	}
	if r.workerName != "" {
		cluster.Cluster.Name += r.workerName + "-"
	}
	cluster.Cluster.Name += scenario.Name() + "-"
	cluster.Cluster.Name += rand.String(8)

	cluster.Cluster.Spec.UsePodSecurityPolicyAdmissionPlugin = r.pspEnabled

	params := &projectclient.CreateClusterParams{
		Context:   ctx,
		ProjectID: r.kubermaticProjectID,
		DC:        r.seed.Name,
		Body:      cluster,
	}
	utils.SetupParams(nil, params, 3*time.Second, 1*time.Minute, http.StatusConflict)

	response, err := r.kubermaticClient.Project.CreateCluster(params, r.kubermaticAuthenticator)
	if err != nil {
		return nil, err
	}

	clusterID := response.Payload.ID
	crCluster := &kubermaticv1.Cluster{}

	if err := wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
		key := types.NamespacedName{Name: clusterID}

		if err := r.seedClusterClient.Get(ctx, key, crCluster); err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to wait for Cluster to appear: %v", err)
	}

	// fetch all existing SSH keys
	listKeysBody := &projectclient.ListSSHKeysParams{
		Context:   ctx,
		ProjectID: r.kubermaticProjectID,
	}
	utils.SetupParams(nil, listKeysBody, 3*time.Second, 1*time.Minute, http.StatusConflict, http.StatusNotFound)

	result, err := r.kubermaticClient.Project.ListSSHKeys(listKeysBody, r.kubermaticAuthenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to list project's SSH keys: %v", err)
	}

	keyIDs := []string{}
	for _, key := range result.Payload {
		keyIDs = append(keyIDs, key.ID)
	}

	// assign all keys to the new cluster
	for _, keyID := range keyIDs {
		assignKeyBody := &projectclient.AssignSSHKeyToClusterParams{
			Context:   ctx,
			ProjectID: r.kubermaticProjectID,
			DC:        r.seed.Name,
			ClusterID: crCluster.Name,
			KeyID:     keyID,
		}
		utils.SetupParams(nil, assignKeyBody, 3*time.Second, 1*time.Minute, http.StatusConflict, http.StatusNotFound, http.StatusForbidden)

		if _, err := r.kubermaticClient.Project.AssignSSHKeyToCluster(assignKeyBody, r.kubermaticAuthenticator); err != nil {
			return nil, fmt.Errorf("failed to assign SSH key to cluster: %v", err)
		}
	}

	log.Infof("Successfully created cluster %s", crCluster.Name)
	return crCluster, nil
}

func (r *testRunner) waitForControlPlane(ctx context.Context, log *zap.SugaredLogger, clusterName string) (*kubermaticv1.Cluster, error) {
	log.Debug("Waiting for control plane to become ready...")
	started := time.Now()
	namespacedClusterName := types.NamespacedName{Name: clusterName}

	err := wait.Poll(3*time.Second, r.controlPlaneReadyWaitTimeout, func() (done bool, err error) {
		newCluster := &kubermaticv1.Cluster{}

		if err := r.seedClusterClient.Get(ctx, namespacedClusterName, newCluster); err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
		}
		// Check for this first, because otherwise we instantly return as the cluster-controller did not
		// create any pods yet
		if !newCluster.Status.ExtendedHealth.AllHealthy() {
			return false, nil
		}

		controlPlanePods := &corev1.PodList{}
		if err := r.seedClusterClient.List(
			ctx,
			controlPlanePods,
			&ctrlruntimeclient.ListOptions{Namespace: newCluster.Status.NamespaceName},
		); err != nil {
			return false, fmt.Errorf("failed to list controlplane pods: %v", err)
		}
		for _, pod := range controlPlanePods.Items {
			if !podIsReady(&pod) {
				return false, nil
			}
		}

		return true, nil
	})
	// Timeout or other error
	if err != nil {
		return nil, err
	}

	// Get copy of latest version
	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClusterClient.Get(ctx, namespacedClusterName, cluster); err != nil {
		return nil, err
	}

	log.Debugf("Control plane became ready after %.2f seconds", time.Since(started).Seconds())
	return cluster, nil
}

// podFailedKubeletAdmissionDueToNodeAffinityPredicate detects a condition in
// which a pod is scheduled but fails kubelet admission due to a race condition
// between scheduler and kubelet.
// see: https://github.com/kubernetes/kubernetes/issues/93338
func (r *testRunner) podFailedKubeletAdmissionDueToNodeAffinityPredicate(p *corev1.Pod) bool {
	failedAdmission := p.Status.Phase == "Failed" && p.Status.Reason == "NodeAffinity"
	if failedAdmission {
		r.log.Infow(
			"pod failed kubelet admission due to NodeAffinity predicate",
			"pod", *p,
		)
	}
	return failedAdmission
}

func (r *testRunner) waitUntilAllPodsAreReady(ctx context.Context, log *zap.SugaredLogger, userClusterClient ctrlruntimeclient.Client, timeout time.Duration) error {
	log.Debug("Waiting for all pods to be ready...")
	started := time.Now()

	err := wait.Poll(r.userClusterPollInterval, timeout, func() (done bool, err error) {
		podList := &corev1.PodList{}
		if err := userClusterClient.List(ctx, podList); err != nil {
			log.Warnw("Failed to load pod list while waiting until all pods are running", zap.Error(err))
			return false, nil
		}

		for _, pod := range podList.Items {
			// Ignore pods failing kubelet admission #6185
			if !podIsReady(&pod) && !r.podFailedKubeletAdmissionDueToNodeAffinityPredicate(&pod) {
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	log.Debugf("All pods became ready after %.2f seconds", time.Since(started).Seconds())
	return nil
}

type ginkgoResult struct {
	logfile  string
	report   *reporters.JUnitTestSuite
	duration time.Duration
}

const (
	argSeparator = ` \
    `
)

type ginkgoRun struct {
	name       string
	cmd        *exec.Cmd
	reportsDir string
	timeout    time.Duration
}

func (r *testRunner) getGinkgoRuns(
	log *zap.SugaredLogger,
	scenario testScenario,
	kubeconfigFilename,
	cloudConfigFilename string,
	cluster *kubermaticv1.Cluster,
) ([]*ginkgoRun, error) {
	kubeconfigFilename = path.Clean(kubeconfigFilename)
	repoRoot := path.Clean(r.repoRoot)
	MajorMinor := fmt.Sprintf("%d.%d", cluster.Spec.Version.Major(), cluster.Spec.Version.Minor())

	nodeNumberTotal := int32(r.nodeCount)

	ginkgoSkipParallel := `\[Serial\]`
	if minor := cluster.Spec.Version.Minor(); minor >= 16 && minor <= 20 {
		// These require the nodes NodePort to be available from the tester, which is not the case for us.
		// TODO: Maybe add an option to allow the NodePorts in the SecurityGroup?
		ginkgoSkipParallel = strings.Join([]string{
			ginkgoSkipParallel,
			"Services should be able to change the type from ExternalName to NodePort",
			"Services should be able to create a functioning NodePort service",
			"Services should be able to switch session affinity for NodePort service",
			"Services should have session affinity timeout work for NodePort service",
			"Services should have session affinity work for NodePort service",
		}, "|")
	}

	runs := []struct {
		name          string
		ginkgoFocus   string
		ginkgoSkip    string
		parallelTests int
		timeout       time.Duration
	}{
		{
			name:          "parallel",
			ginkgoFocus:   `\[Conformance\]`,
			ginkgoSkip:    ginkgoSkipParallel,
			parallelTests: int(nodeNumberTotal) * 3,
			timeout:       30 * time.Minute,
		},
		{
			name:          "serial",
			ginkgoFocus:   `\[Serial\].*\[Conformance\]`,
			ginkgoSkip:    `should not cause race condition when used for configmap`,
			parallelTests: 1,
			timeout:       30 * time.Minute,
		},
	}
	versionRoot := path.Join(repoRoot, MajorMinor)
	binRoot := path.Join(versionRoot, "/platforms/linux/amd64")
	var ginkgoRuns []*ginkgoRun
	for _, run := range runs {

		reportsDir := path.Join("/tmp", scenario.Name(), run.name)
		env := []string{
			// `kubectl diff` needs to find /usr/bin/diff
			fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
			fmt.Sprintf("HOME=%s", r.homeDir),
			fmt.Sprintf("AWS_SSH_KEY=%s", path.Join(r.homeDir, ".ssh", "google_compute_engine")),
			fmt.Sprintf("LOCAL_SSH_KEY=%s", path.Join(r.homeDir, ".ssh", "google_compute_engine")),
			fmt.Sprintf("KUBE_SSH_KEY=%s", path.Join(r.homeDir, ".ssh", "google_compute_engine")),
		}

		args := []string{
			"-progress",
			fmt.Sprintf("-nodes=%d", run.parallelTests),
			"-noColor=true",
			"-flakeAttempts=2",
			fmt.Sprintf(`-focus=%s`, run.ginkgoFocus),
			fmt.Sprintf(`-skip=%s`, run.ginkgoSkip),
			path.Join(binRoot, "e2e.test"),
			"--",
			"--disable-log-dump",
			fmt.Sprintf("--repo-root=%s", versionRoot),
			fmt.Sprintf("--report-dir=%s", reportsDir),
			fmt.Sprintf("--report-prefix=%s", run.name),
			fmt.Sprintf("--kubectl-path=%s", path.Join(binRoot, "kubectl")),
			fmt.Sprintf("--kubeconfig=%s", kubeconfigFilename),
			fmt.Sprintf("--num-nodes=%d", nodeNumberTotal),
			fmt.Sprintf("--cloud-config-file=%s", cloudConfigFilename),
		}

		args = append(args, "--provider=local")

		osSpec := scenario.OS()
		switch {
		case osSpec.Ubuntu != nil:
			args = append(args, "--node-os-distro=ubuntu")
			env = append(env, "KUBE_SSH_USER=ubuntu")
		case osSpec.Centos != nil:
			args = append(args, "--node-os-distro=centos")
			env = append(env, "KUBE_SSH_USER=centos")
		case osSpec.Flatcar != nil:
			args = append(args, "--node-os-distro=flatcar")
			env = append(env, "KUBE_SSH_USER=core")
		}

		cmd := exec.Command(path.Join(binRoot, "ginkgo"), args...)
		cmd.Env = env

		ginkgoRuns = append(ginkgoRuns, &ginkgoRun{
			name:       run.name,
			cmd:        cmd,
			reportsDir: reportsDir,
			timeout:    run.timeout,
		})
	}

	return ginkgoRuns, nil
}

func (r *testRunner) executeGinkgoRun(ctx context.Context, parentLog *zap.SugaredLogger, run *ginkgoRun, client ctrlruntimeclient.Client) (*ginkgoResult, error) {
	log := parentLog.With("reports-dir", run.reportsDir)

	if err := r.deleteAllNonDefaultNamespaces(ctx, log, client); err != nil {
		return nil, fmt.Errorf("failed to cleanup namespaces before the Ginkgo run: %v", err)
	}

	timedCtx, cancel := context.WithTimeout(ctx, run.timeout)
	defer cancel()

	// We're clearing up the temp dir on every run
	if err := os.RemoveAll(run.reportsDir); err != nil {
		log.Errorw("Failed to remove temporary reports directory", zap.Error(err))
	}
	if err := os.MkdirAll(run.reportsDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create temporary reports directory: %v", err)
	}

	// Make sure we write to a file instead of a byte buffer as the logs are pretty big
	file, err := ioutil.TempFile("/tmp", run.name+"-log")
	if err != nil {
		return nil, fmt.Errorf("failed to open logfile: %v", err)
	}
	defer file.Close()
	log = log.With("ginkgo-log", file.Name())

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	started := time.Now()

	// Copy the command as we cannot execute a command twice
	cmd := exec.CommandContext(timedCtx, "")
	cmd.Path = run.cmd.Path
	cmd.Args = run.cmd.Args
	cmd.Env = run.cmd.Env
	cmd.Dir = run.cmd.Dir
	cmd.ExtraFiles = run.cmd.ExtraFiles
	if _, err := writer.Write([]byte(strings.Join(cmd.Args, argSeparator))); err != nil {
		return nil, fmt.Errorf("failed to write command to log: %v", err)
	}

	log.Infof("Starting Ginkgo run '%s'...", run.name)

	// Flush to disk so we can actually watch logs
	stopCh := make(chan struct{}, 1)
	defer close(stopCh)
	go wait.Until(func() {
		if err := writer.Flush(); err != nil {
			log.Warnw("Failed to flush log writer", zap.Error(err))
		}
		if err := file.Sync(); err != nil {
			log.Warnw("Failed to sync log file", zap.Error(err))
		}
	}, 1*time.Second, stopCh)

	cmd.Stdout = writer
	cmd.Stderr = writer

	if err := cmd.Run(); err != nil {
		// did the context's timeout kick in?
		if ctxErr := timedCtx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Debugf("Ginkgo exited with a non-zero return code %d: %v", exitErr.ExitCode(), exitErr)
		} else {
			return nil, fmt.Errorf("ginkgo failed to start: %T %v", err, err)
		}
	}

	log.Debug("Ginkgo run completed, collecting reports...")

	// When running ginkgo in parallel, each ginkgo worker creates a own report, thus we must combine them
	combinedReport, err := collectReports(run.name, run.reportsDir)
	if err != nil {
		return nil, err
	}

	// If we have no junit files, we cannot return a valid report
	if len(combinedReport.TestCases) == 0 {
		return nil, errors.New("Ginkgo report is empty, it seems no tests where executed")
	}

	combinedReport.Time = time.Since(started).Seconds()

	log.Infof("Ginkgo run '%s' took %s", run.name, time.Since(started))
	return &ginkgoResult{
		logfile:  file.Name(),
		report:   combinedReport,
		duration: time.Since(started),
	}, nil
}

func supportsStorage(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Openstack != nil ||
		cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil ||
		cluster.Spec.Cloud.VSphere != nil ||
		cluster.Spec.Cloud.GCP != nil

	// Currently broken, see https://github.com/kubermatic/kubermatic/issues/3312
	//cluster.Spec.Cloud.Hetzner != nil
}

func supportsLBs(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil ||
		cluster.Spec.Cloud.GCP != nil
}

func (r *testRunner) printAllControlPlaneLogs(ctx context.Context, log *zap.SugaredLogger, clusterName string) {
	log.Info("Printing control plane logs")

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClusterClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		log.Errorw("Failed to get cluster", zap.Error(err))
		return
	}

	log.Debugw("Cluster health status", "status", cluster.Status.ExtendedHealth)

	log.Info("Logging events for cluster")
	if err := logEventsObject(ctx, log, r.seedClusterClient, "default", cluster.UID); err != nil {
		log.Errorw("Failed to log cluster events", zap.Error(err))
	}

	if err := printEventsAndLogsForAllPods(
		ctx,
		log,
		r.seedClusterClient,
		r.seedGeneratedClient,
		cluster.Status.NamespaceName,
	); err != nil {
		log.Errorw("Failed to print events and logs of pods", zap.Error(err))
	}
}

// waitForMachinesToJoinCluster waits for machines to join the cluster. It does so by checking
// if the machines have a nodeRef. It does not check if the nodeRef is valid.
// All errors are swallowed, only the timeout error is returned.
func waitForMachinesToJoinCluster(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, timeout time.Duration) (time.Duration, error) {
	startTime := time.Now()
	err := wait.Poll(10*time.Second, timeout, func() (bool, error) {
		machineList := &clusterv1alpha1.MachineList{}
		if err := client.List(ctx, machineList); err != nil {
			log.Warnw("Failed to list machines", zap.Error(err))
			return false, nil
		}
		for _, machine := range machineList.Items {
			if !machineHasNodeRef(machine) {
				log.Infow("Machine has no nodeRef yet", "machine", machine.Name)
				return false, nil
			}
		}
		log.Infow("All machines got a Node", "duration-in-seconds", time.Since(startTime).Seconds())
		return true, nil
	})
	return timeout - time.Since(startTime), err
}

func machineHasNodeRef(machine clusterv1alpha1.Machine) bool {
	return machine.Status.NodeRef != nil && machine.Status.NodeRef.Name != ""
}

// WaitForNodesToBeReady waits for all nodes to be ready. It does so by checking the Nodes "Ready"
// condition. It swallows all errors except for the timeout.
func waitForNodesToBeReady(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, timeout time.Duration) (time.Duration, error) {
	startTime := time.Now()
	err := wait.Poll(10*time.Second, timeout, func() (bool, error) {
		nodeList := &corev1.NodeList{}
		if err := client.List(ctx, nodeList); err != nil {
			log.Warnw("Failed to list nodes", zap.Error(err))
			return false, nil
		}
		for _, node := range nodeList.Items {
			if !nodeIsReady(node) {
				log.Infow("Node is not ready", "node", node.Name)
				return false, nil
			}
		}
		log.Infow("All nodes got ready", "duration-in-seconds", time.Since(startTime).Seconds())
		return true, nil
	})
	return timeout - time.Since(startTime), err
}

func nodeIsReady(node corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func printFileUnbuffered(filename string) error {
	fd, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fd.Close()
	return printUnbuffered(fd)
}

// printUnbuffered uses io.Copy to print data to stdout.
// It should be used for all bigger logs, to avoid buffering
// them in memory and getting oom killed because of that.
func printUnbuffered(src io.Reader) error {
	_, err := io.Copy(os.Stdout, src)
	return err
}

// junitReporterWrapper is a convenience func to get junit results for a step
// It will create a report, append it to the passed in testsuite and propagate
// the error of the executor back up
// TODO: Should we add optional retrying here to limit the amount of wrappers we need?
func junitReporterWrapper(
	testCaseName string,
	report *reporters.JUnitTestSuite,
	executor func() error,
	extraErrOutputFn ...func() string,
) error {
	junitTestCase := reporters.JUnitTestCase{
		Name:      testCaseName,
		ClassName: testCaseName,
	}

	startTime := time.Now()
	err := executor()
	junitTestCase.Time = time.Since(startTime).Seconds()
	if err != nil {
		junitTestCase.FailureMessage = &reporters.JUnitFailureMessage{Message: err.Error()}
		report.Failures++
		for _, extraOut := range extraErrOutputFn {
			extraOutString := extraOut()
			err = fmt.Errorf("%v\n%s", err, extraOutString)
			junitTestCase.FailureMessage.Message += "\n" + extraOutString
		}
	}

	report.TestCases = append(report.TestCases, junitTestCase)
	report.Tests++

	return err
}

// printEvents and logs for all pods. Include ready pods, because they may still contain useful information.
func printEventsAndLogsForAllPods(
	ctx context.Context,
	log *zap.SugaredLogger,
	client ctrlruntimeclient.Client,
	k8sclient kubernetes.Interface,
	namespace string,
) error {
	log.Infow("Printing logs for all pods", "namespace", namespace)

	pods := &corev1.PodList{}
	if err := client.List(ctx, pods, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}

	var errs []error
	for _, pod := range pods.Items {
		log := log.With("pod", pod.Name)
		if !podIsReady(&pod) {
			log.Error("Pod is not ready")
		}
		log.Info("Logging events for pod")
		if err := logEventsObject(ctx, log, client, pod.Namespace, pod.UID); err != nil {
			log.Errorw("Failed to log events for pod", zap.Error(err))
			errs = append(errs, err)
		}
		log.Info("Printing logs for pod")
		if err := printLogsForPod(ctx, log, k8sclient, &pod); err != nil {
			log.Errorw("Failed to print logs for pod", zap.Error(utilerror.NewAggregate(err)))
			errs = append(errs, err...)
		}
	}

	return utilerror.NewAggregate(errs)
}

func printLogsForPod(ctx context.Context, log *zap.SugaredLogger, k8sclient kubernetes.Interface, pod *corev1.Pod) []error {
	var errs []error
	for _, container := range pod.Spec.Containers {
		containerLog := log.With("container", container.Name)
		containerLog.Info("Printing logs for container")
		if err := printLogsForContainer(ctx, k8sclient, pod, container.Name); err != nil {
			containerLog.Errorw("Failed to print logs for container", zap.Error(err))
			errs = append(errs, err)
		}
	}
	for _, initContainer := range pod.Spec.InitContainers {
		containerLog := log.With("initContainer", initContainer.Name)
		containerLog.Infow("Printing logs for initContainer")
		if err := printLogsForContainer(ctx, k8sclient, pod, initContainer.Name); err != nil {
			containerLog.Errorw("Failed to print logs for initContainer", zap.Error(err))
			errs = append(errs, err)
		}
	}
	return errs
}

func printLogsForContainer(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod, containerName string) error {
	readCloser, err := client.
		CoreV1().
		Pods(pod.Namespace).
		GetLogs(pod.Name, &corev1.PodLogOptions{Container: containerName}).
		Stream(ctx)
	if err != nil {
		return err
	}
	defer readCloser.Close()
	return printUnbuffered(readCloser)
}

func logEventsForAllMachines(
	ctx context.Context,
	log *zap.SugaredLogger,
	client ctrlruntimeclient.Client,
) {
	machines := &clusterv1alpha1.MachineList{}
	if err := client.List(ctx, machines); err != nil {
		log.Errorw("Failed to list machines", zap.Error(err))
		return
	}

	for _, machine := range machines.Items {
		machineLog := log.With("name", machine.Name)
		machineLog.Infow("Logging events for machine")
		if err := logEventsObject(ctx, log, client, machine.Namespace, machine.UID); err != nil {
			machineLog.Errorw("Failed to log events for machine", "namespace", machine.Namespace, zap.Error(err))
		}
	}
}

func logEventsObject(
	ctx context.Context,
	log *zap.SugaredLogger,
	client ctrlruntimeclient.Client,
	namespace string,
	uid types.UID,
) error {
	events := &corev1.EventList{}
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     namespace,
		FieldSelector: fields.OneTermEqualSelector("involvedObject.uid", string(uid)),
	}
	if err := client.List(ctx, events, listOpts); err != nil {
		return fmt.Errorf("failed to get events: %v", err)
	}

	for _, event := range events.Items {
		var msg string
		if event.Type == corev1.EventTypeWarning {
			// Make sure this gets highlighted
			msg = "ERROR"
		}
		log.Infow(
			msg,
			"EventType", event.Type,
			"Number", event.Count,
			"Reason", event.Reason,
			"Message", event.Message,
			"Source", event.Source.Component,
		)
	}
	return nil
}

func logUserClusterPodEventsAndLogs(
	ctx context.Context,
	log *zap.SugaredLogger,
	connProvider *clusterclient.Provider,
	cluster *kubermaticv1.Cluster,
) {
	log.Info("Attempting to log usercluster pod events and logs")
	cfg, err := connProvider.GetClientConfig(ctx, cluster)
	if err != nil {
		log.Errorw("Failed to get usercluster admin kubeconfig", zap.Error(err))
		return
	}
	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Errorw("Failed to construct k8sClient for usercluster", zap.Error(err))
		return
	}
	client, err := connProvider.GetClient(ctx, cluster)
	if err != nil {
		log.Errorw("Failed to construct client for usercluster", zap.Error(err))
		return
	}
	if err := printEventsAndLogsForAllPods(
		ctx,
		log,
		client,
		k8sClient,
		"",
	); err != nil {
		log.Errorw("Failed to print events and logs for usercluster pods", zap.Error(err))
	}
}
