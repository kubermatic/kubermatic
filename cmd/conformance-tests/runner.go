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
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/onsi/ginkgo/reporters"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

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
		containerLogsDirectory:       opts.containerLogsDirectory,
	}
}

type testRunner struct {
	log                    *zap.SugaredLogger
	scenarios              []testScenario
	secrets                secrets
	namePrefix             string
	repoRoot               string
	reportsRoot            string
	PublicKeys             [][]byte
	workerName             string
	homeDir                string
	printGinkoLogs         bool
	printContainerLogs     bool
	containerLogsDirectory string
	onlyTestCreation       bool
	pspEnabled             bool

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

	var logExport logExporter
	eventExport := &eventLogger{}

	if r.printContainerLogs {
		logExport = &logPrinter{}
	} else if r.containerLogsDirectory != "" {
		logExport = &logDumper{directory: r.containerLogsDirectory}
	}

	if logExport != nil {
		// Print all controlplane logs to both make debugging easier and show issues
		// that didn't result in test failures.
		defer exportControlPlane(ctx, log, r.seedClusterClient, r.seedGeneratedClient, clusterName, logExport, eventExport)
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
		defer exportEventsForAllMachines(ctx, log, userClusterClient, eventExport)
		defer exportUserCluster(ctx, log, r.clusterClientProvider, cluster.DeepCopy(), logExport, eventExport)
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
