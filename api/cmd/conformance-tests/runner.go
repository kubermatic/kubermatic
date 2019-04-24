package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/onsi/ginkgo/reporters"
	"github.com/sirupsen/logrus"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machine"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	podIsReady = func(p *corev1.Pod) bool {
		for _, c := range p.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	}
)

type testScenario interface {
	Name() string
	Cluster(secrets secrets) *kubermaticv1.Cluster
	Nodes(num int) *kubermaticapiv1.NodeDeployment
}

func newRunner(scenarios []testScenario, opts *Opts) *testRunner {
	return &testRunner{
		scenarios:                    scenarios,
		clusterLister:                opts.clusterLister,
		controlPlaneReadyWaitTimeout: opts.controlPlaneReadyWaitTimeout,
		deleteClusterAfterTests:      opts.deleteClusterAfterTests,
		kubermaticClient:             opts.kubermaticClient,
		secrets:                      opts.secrets,
		namePrefix:                   opts.namePrefix,
		clusterClientProvider:        opts.clusterClientProvider,
		nodesReadyWaitTimeout:        opts.nodeReadyWaitTimeout,
		dcs:                          opts.dcs,
		nodeCount:                    opts.nodeCount,
		repoRoot:                     opts.repoRoot,
		reportsRoot:                  opts.reportsRoot,
		clusterParallelCount:         opts.clusterParallelCount,
		PublicKeys:                   opts.publicKeys,
		workerName:                   opts.workerName,
		homeDir:                      opts.homeDir,
		seedKubeClient:               opts.seedKubeClient,
		log:                          opts.log,
		existingClusterLabel:         opts.existingClusterLabel,
	}
}

type testRunner struct {
	scenarios   []testScenario
	secrets     secrets
	namePrefix  string
	repoRoot    string
	reportsRoot string
	PublicKeys  [][]byte
	workerName  string
	homeDir     string
	log         *logrus.Entry

	controlPlaneReadyWaitTimeout time.Duration
	deleteClusterAfterTests      bool
	nodesReadyWaitTimeout        time.Duration
	nodeCount                    int
	clusterParallelCount         int

	kubermaticClient      kubermaticclientset.Interface
	seedKubeClient        kubernetes.Interface
	clusterLister         kubermaticv1lister.ClusterLister
	clusterClientProvider clusterclient.UserClusterConnectionProvider
	dcs                   map[string]provider.DatacenterMeta

	// The label to use to select an existing cluster to test against instead of
	// creating a new one
	existingClusterLabel string
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

func (r *testRunner) worker(id int, scenarios <-chan testScenario, results chan<- testResult) {
	for s := range scenarios {
		scenarioLog := r.log.WithFields(logrus.Fields{
			"scenario": s.Name(),
			"worker":   id,
		})
		scenarioLog.Info("Starting to test scenario...")

		report, err := r.executeScenario(scenarioLog, s)
		res := testResult{
			report:   report,
			scenario: s,
			err:      err,
		}
		if err != nil {
			scenarioLog.Infof("Finished with error: %v", err)
		} else {
			scenarioLog.Info("Finished")
		}

		results <- res
	}
}

func (r *testRunner) Run() error {
	scenariosCh := make(chan testScenario, len(r.scenarios))
	resultsCh := make(chan testResult, len(r.scenarios))

	r.log.Infoln("Test suite:")
	for _, scenario := range r.scenarios {
		r.log.Infoln(scenario.Name())
		scenariosCh <- scenario
	}
	r.log.Infoln(fmt.Sprintf("Total: %d tests", len(r.scenarios)))

	for i := 1; i <= r.clusterParallelCount; i++ {
		go r.worker(i, scenariosCh, resultsCh)
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

func (r *testRunner) executeScenario(log *logrus.Entry, scenario testScenario) (*reporters.JUnitTestSuite, error) {
	var err error
	var cluster *kubermaticv1.Cluster

	if r.existingClusterLabel == "" {
		cluster, err = r.createCluster(log, scenario)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster: %v", err)
		}
	} else {
		selector, err := labels.Parse(r.existingClusterLabel)
		if err != nil {
			return nil, fmt.Errorf("failed to parse labelselector %q: %v", r.existingClusterLabel, err)
		}
		foundClusters, err := r.clusterLister.List(selector)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %v", err)
		}
		if foundClusterNum := len(foundClusters); foundClusterNum != 1 {
			return nil, fmt.Errorf("expected to find exactly one existing cluster, but got %d", foundClusterNum)
		}
		cluster = foundClusters[0]
	}

	cluster, err = r.waitForControlPlane(log, cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for control plane to become ready: %v", err)
	}

	// We must store the name here because the cluster object may be nil on error
	clusterName := cluster.Name
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cluster, err = r.kubermaticClient.KubermaticV1().Clusters().Get(clusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		cluster.Finalizers = append(cluster.Finalizers, kubermaticapiv1.InClusterPVCleanupFinalizer, kubermaticapiv1.InClusterLBCleanupFinalizer)
		cluster, err = r.kubermaticClient.KubermaticV1().Clusters().Update(cluster)
		return err

	})
	if err != nil {
		return nil, fmt.Errorf("failed to add PV and LB cleanup finalizers: %v", err)
	}

	providerName, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud provider name from cluster: %v", err)
	}

	log = log.WithFields(logrus.Fields{
		"cluster":        cluster.Name,
		"cloud-provider": providerName,
		"version":        cluster.Spec.Version,
	})

	dc, found := r.dcs[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("invalid cloud datacenter specified '%s'. Not found in datacenters list", cluster.Spec.Cloud.DatacenterName)
	}

	if r.deleteClusterAfterTests {
		defer func() {
			if err := tryToDeleteClusterWithRetries(log, cluster, r.clusterClientProvider, r.kubermaticClient); err != nil {
				log.Errorf("failed to delete cluster: %v", err)
				log.Errorf("Please manually cleanup the cluster. Either by restarting with `-cleanup-on-start=true` or by doing the cleanup by hand: %v", err)
			}
		}()
	}

	kubeconfigFilename, err := r.getKubeconfig(log, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %v", err)
	}
	log = log.WithFields(logrus.Fields{"kubeconfig": kubeconfigFilename})

	cloudConfigFilename, err := r.getCloudConfig(log, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud config: %v", err)
	}

	clusterKubeClient, err := r.clusterClientProvider.GetClient(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get the client for the cluster: %v", err)
	}

	nodeDeployment := scenario.Nodes(r.nodeCount)
	if err := r.setupNodes(log, scenario.Name(), cluster, clusterKubeClient, nodeDeployment, dc); err != nil {
		return nil, fmt.Errorf("failed to setup nodes: %v", err)
	}

	if err := r.waitUntilAllPodsAreReady(log, clusterKubeClient); err != nil {
		return nil, fmt.Errorf("failed to wait until all pods are running after creating the cluster: %v", err)
	}

	report, err := r.testCluster(log, scenario.Name(), cluster, clusterKubeClient, nodeDeployment, dc, kubeconfigFilename, cloudConfigFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to test cluster: %v", err)
	}

	if report == nil {
		return nil, errors.New("no report generated")
	}

	return report, nil
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
	return fmt.Errorf("function did not succeeded after %d attempts: %v", maxAttempts, err)
}

func (r *testRunner) testCluster(
	log *logrus.Entry,
	scenarioName string,
	cluster *kubermaticv1.Cluster,
	clusterKubeClient kubernetes.Interface,
	nd *kubermaticapiv1.NodeDeployment,
	dc provider.DatacenterMeta,
	kubeconfigFilename string,
	cloudConfigFilename string,
) (*reporters.JUnitTestSuite, error) {
	const maxTestAttempts = 3
	var err error
	totalStart := time.Now()
	log.Info("Starting to test cluster...")
	defer log.Infof("Finished testing cluster after %s", time.Since(totalStart))

	// We'll store the report there and all kinds of logs
	scenarioFolder := path.Join(r.reportsRoot, scenarioName)
	if err := os.MkdirAll(scenarioFolder, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create the scenario folder '%s': %v", scenarioFolder, err)
	}

	report := &reporters.JUnitTestSuite{
		Name: scenarioName,
	}

	ginkgoRuns, err := r.getGinkgoRuns(log, scenarioName, kubeconfigFilename, cloudConfigFilename, cluster, nd, dc)
	if err != nil {
		return nil, fmt.Errorf("failed to get Ginkgo runs: %v", err)
	}
	for _, run := range ginkgoRuns {

		ginkgoRes, err := r.executeGinkgoRunWithRetries(log, run, clusterKubeClient)
		if err != nil {
			// Ginkgo failed hard. We don't have any JUnit reports to append, so we appenda custom one to indicate the hard failure
			report.TestCases = append(report.TestCases, reporters.JUnitTestCase{
				Name:           "[Ginkgo] Run ginkgo tests",
				ClassName:      "Ginkgo",
				FailureMessage: &reporters.JUnitFailureMessage{Message: fmt.Sprintf("%v", err)},
			})

			// We still wan't to run potential next runs
			continue
		}

		// We have a valid report from Ginkgo. It might contain failed tests, but that's ok here.
		// The executor if this scenario will later on interpret the junit report and decides for a return code.
		// We append the report from Ginkgo to our scenario wide report
		report = combineReports("Kubernetes Conformance tests", report, ginkgoRes.report)
	}

	// Do a simple PVC test - with retries
	if supportsStorage(cluster) {
		testStart := time.Now()
		testCase := reporters.JUnitTestCase{
			Name:      "[CloudProvider] Test PVC support with the existing StorageClass",
			ClassName: "Kubermatic custom tests",
		}
		err := retryNAttempts(maxTestAttempts, func(attempt int) error { return r.testPVC(log, clusterKubeClient, attempt) })
		if err != nil {
			report.Errors++
			testCase.FailureMessage = &reporters.JUnitFailureMessage{Message: err.Error()}
			log.Errorf("Failed to verify that PVC's work: %v", err)
		}
		testCase.Time = time.Since(testStart).Seconds()
		report.TestCases = append(report.TestCases, testCase)
		report.Tests++
	}

	// Do a simple LB test - with retries
	if supportsLBs(cluster) {
		testStart := time.Now()
		testCase := reporters.JUnitTestCase{
			Name:      "[CloudProvider] Test LB support",
			ClassName: "Kubermatic custom tests",
		}
		err := retryNAttempts(maxTestAttempts, func(attempt int) error { return r.testLB(log, clusterKubeClient, attempt) })
		if err != nil {
			report.Errors++
			testCase.FailureMessage = &reporters.JUnitFailureMessage{Message: err.Error()}
			log.Errorf("Failed to verify that LB's work: %v", err)
		}
		testCase.Time = time.Since(testStart).Seconds()
		report.TestCases = append(report.TestCases, testCase)
		report.Tests++
	}

	// Do user cluster RBAC controller test - with retries
	{
		testStart := time.Now()
		testCase := reporters.JUnitTestCase{
			Name:      "Test user cluster RBAC controller",
			ClassName: "Kubermatic custom tests",
		}
		err = retryNAttempts(maxTestAttempts, func(attempt int) error {
			return r.testUserclusterControllerRBAC(log, cluster, clusterKubeClient, r.seedKubeClient)
		})
		if err != nil {
			report.Errors++
			testCase.FailureMessage = &reporters.JUnitFailureMessage{Message: err.Error()}
			log.Errorf("Failed to verify that user cluster RBAC controller work: %v", err)
		}
		testCase.Time = time.Since(testStart).Seconds()
		report.TestCases = append(report.TestCases, testCase)
		report.Tests++
	}

	report.Time = time.Since(totalStart).Seconds()
	b, err := xml.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal combined report file: %v", err)
	}

	if err := ioutil.WriteFile(path.Join(r.reportsRoot, fmt.Sprintf("junit.%s.xml", scenarioName)), b, 0644); err != nil {
		return nil, fmt.Errorf("failed to write combined report file: %v", err)
	}

	return report, nil
}

// executeGinkgoRunWithRetries executes the passed GinkgoRun and retries if it failed hard(Failed to execute the Ginkgo binary for example)
// Or if the JUnit report from Ginkgo contains failed tests.
// Only if Ginkgo failed hard, an error will be returned. If some tests still failed after retrying the run, the report will reflect that.
func (r *testRunner) executeGinkgoRunWithRetries(log *logrus.Entry, run *ginkgoRun, kubeClient kubernetes.Interface) (ginkgoRes *ginkgoResult, err error) {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ginkgoRes, err = executeGinkgoRun(log, run, kubeClient)
		if err != nil {
			// Something critical happened and we don't have a valid result
			log.Errorf("failed to execute the Ginkgo run '%s': %v", run.name, err)
			continue
		}

		if ginkgoRes.report.Errors > 0 || ginkgoRes.report.Failures > 0 {
			msg := fmt.Sprintf("Ginkgo run '%s' had failed tests.", run.name)
			if attempt < maxAttempts {
				msg = fmt.Sprintf("%s. Retrying...", msg)
			}
			log.Info(msg)
			continue
		}

		// Ginkgo run successfully and no test failed
		return ginkgoRes, err
	}

	return ginkgoRes, err
}

func (r *testRunner) setupNodes(log *logrus.Entry, scenarioName string, cluster *kubermaticv1.Cluster, clusterKubeClient kubernetes.Interface, nodeDeployment *kubermaticapiv1.NodeDeployment, dc provider.DatacenterMeta) error {
	log.Info("Creating machineDeployment..")
	client, err := r.clusterClientProvider.GetDynamicClient(cluster)
	if err != nil {
		return fmt.Errorf("failed to get the machine client for the cluster: %v", err)
	}

	var keys []*kubermaticv1.UserSSHKey
	for _, data := range r.PublicKeys {
		keys = append(keys, &kubermaticv1.UserSSHKey{
			Spec: kubermaticv1.SSHKeySpec{
				PublicKey: string(data),
			},
		})
	}

	// When there are machines not owned by a MachineDeployment, we must decrement
	// the nodeDeployments replicas by their count, this is needed in upgrade tests
	if err := retryNAttempts(defaultAPIRetries, func(_ int) error {
		if r.existingClusterLabel == "" {
			return nil
		}
		machineList := &clusterv1alpha1.MachineList{}
		if err := client.List(context.TODO(), &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}, machineList); err != nil {
			return fmt.Errorf("failed to list machines: %v", err)
		}
		machinesWithoutOwner := int32(0)
		for _, machine := range machineList.Items {
			if len(machine.OwnerReferences) > 0 {
				continue
			}
			machinesWithoutOwner++
		}
		nodeDeployment.Spec.Replicas = nodeDeployment.Spec.Replicas - machinesWithoutOwner
		return nil
	}); err != nil {
		return fmt.Errorf("failed to cleanup machines that are not owned by a MachineDeployment: %v", err)
	}
	// Explicitly set name. machine.MachineDeployment sets generateName if the name
	// is unset but need a deterministic name because we retry creation and dont
	// want to accidentally create multiple MachineDeployments
	nodeDeployment.Name = fmt.Sprintf("md-%s", scenarioName)
	machineDeployment, err := machine.Deployment(cluster, nodeDeployment, dc, keys)
	if err != nil {
		return fmt.Errorf("failed to get MachineDeployment from NodeDeployment: %v", err)
	}

	if err := retryNAttempts(defaultAPIRetries, func(attempt int) error {
		if err := client.Create(context.TODO(), machineDeployment); err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil
			}
			log.WithField("machineDeployment", machineDeployment.Name).
				Warnf("[Attempt %d/%d] Failed to create MachineDeployment: %v. Retrying",
					attempt, defaultAPIRetries, err)
			time.Sleep(defaultUserClusterPollInterval)
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to create MachineDeployment %s after %d attempts: %v", machineDeployment.Name, defaultAPIRetries, err)
	}

	// Make sure replicas matches nodeDeployment replicas, this may differ on the second run on upgrade tests
	// We dont explicitly catch that, as we ignore kerrors.IsAlreadyExists when creating the machineDeployment
	// This is a very poor persons replication of `EnsureResources`, the problem is we want to use the apis `machine.Deployment`
	// func which always returns a new MachineDeployment
	if err := retryNAttempts(defaultAPIRetries, func(_ int) error {
		mdName := machineDeployment.Name
		if err := client.Get(context.TODO(), types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: mdName}, machineDeployment); err != nil {
			return fmt.Errorf("failed to get MachineDeployment %q: %v", mdName, err)
		}
		if *machineDeployment.Spec.Replicas == nodeDeployment.Spec.Replicas {
			return nil
		}
		machineDeployment.Spec.Replicas = &nodeDeployment.Spec.Replicas
		return client.Update(context.TODO(), machineDeployment)
	}); err != nil {
		return fmt.Errorf("failed to ensure machineDeployment has desired number of replicas: %v", err)
	}
	log.Infof("Successfully created %d machine(s)!", machineDeployment.Spec.Replicas)

	if err := r.waitForReadyNodes(log, clusterKubeClient, int(*machineDeployment.Spec.Replicas)); err != nil {
		return fmt.Errorf("failed waiting for nodes to become ready: %v", err)
	}
	return nil
}

func (r *testRunner) getKubeconfig(log *logrus.Entry, cluster *kubermaticv1.Cluster) (string, error) {
	log.Debug("Getting kubeconfig...")
	kubeconfig, err := r.clusterClientProvider.GetAdminKubeconfig(cluster)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig from cluster client provider: %v", err)
	}
	filename := path.Join(r.homeDir, fmt.Sprintf("%s-kubeconfig", cluster.Name))
	if err := ioutil.WriteFile(filename, kubeconfig, 0644); err != nil {
		return "", fmt.Errorf("failed to write kubeconfig to %s: %v", filename, err)
	}

	log.Infof("Successfully wrote kubeconfig to %s", filename)
	return filename, nil
}

func (r *testRunner) getCloudConfig(log *logrus.Entry, cluster *kubermaticv1.Cluster) (string, error) {
	log.Debug("Getting cloud-config...")

	var cmData string
	err := retryNAttempts(defaultAPIRetries, func(attempt int) error {
		cm, err := r.seedKubeClient.CoreV1().ConfigMaps(cluster.Status.NamespaceName).Get(resources.CloudConfigConfigMapName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to load cloud-config: %v", err)
		}
		cmData = cm.Data["config"]
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to get cloud config ConfigMap: %v", err)
	}

	filename := path.Join(r.homeDir, fmt.Sprintf("%s-cloud-config", cluster.Name))
	if err := ioutil.WriteFile(filename, []byte(cmData), 0644); err != nil {
		return "", fmt.Errorf("failed to write cloud config: %v", err)
	}

	log.Infof("Successfully wrote cloud-config to %s", filename)
	return filename, nil
}

func (r *testRunner) createCluster(log *logrus.Entry, scenario testScenario) (*kubermaticv1.Cluster, error) {
	// Always generate a random name
	cluster := scenario.Cluster(r.secrets)
	cluster.Name = rand.String(8)
	if r.namePrefix != "" {
		cluster.Name = fmt.Sprintf("%s-%s", r.namePrefix, cluster.Name)
	}
	log = logrus.WithFields(logrus.Fields{"cluster": cluster.Name})

	if cluster.Labels == nil {
		cluster.Labels = map[string]string{}
	}
	if r.workerName != "" {
		cluster.Labels[kubermaticv1.WorkerNameLabelKey] = r.workerName
	}

	log.Info("Creating cluster...")
	cluster, err := r.kubermaticClient.KubermaticV1().Clusters().Create(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create the cluster resource: %v", err)
	}
	log.Debug("Successfully created cluster!")
	return cluster, nil
}

func (r *testRunner) waitForReadyNodes(log *logrus.Entry, client kubernetes.Interface, num int) error {
	log.Info("Waiting for nodes to become ready...")
	started := time.Now()

	nodeIsReady := func(n corev1.Node) bool {
		for _, condition := range n.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				if condition.Status == corev1.ConditionTrue {
					return true
				}
			}
		}
		return false
	}

	err := wait.Poll(nodesReadyPollPeriod, r.nodesReadyWaitTimeout, func() (done bool, err error) {
		nodeList, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			log.Debugf("failed to list nodes while waiting for them to be ready. %v. Will retry", err)
			return false, nil
		}

		if len(nodeList.Items) != num {
			return false, nil
		}

		for _, node := range nodeList.Items {
			if !nodeIsReady(node) {
				return false, nil
			}
		}

		return true, nil
	})

	if err != nil {
		return err
	}

	log.Infof("Nodes got ready after %.2f seconds", time.Since(started).Seconds())
	return nil
}

func (r *testRunner) waitForControlPlane(log *logrus.Entry, clusterName string) (*kubermaticv1.Cluster, error) {
	log.Debug("Waiting for control plane to become ready...")
	started := time.Now()

	err := wait.Poll(controlPlaneReadyPollPeriod, r.controlPlaneReadyWaitTimeout, func() (done bool, err error) {
		cluster, err := r.clusterLister.Get(clusterName)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, fmt.Errorf("failed to get the cluster %s from the lister: %v", clusterName, err)
		}
		// Check for this first, because otherwise we instantly return as the cluster-controller did not
		// create any pods yet
		if !cluster.Status.Health.AllHealthy() {
			return false, nil
		}
		controlPlanePods, err := r.seedKubeClient.CoreV1().Pods(cluster.Status.NamespaceName).List(metav1.ListOptions{})
		if err != nil {
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
		// Be helpful and log which pods are not ready
		if err := r.logNotReadyControlplanePods(clusterName); err != nil {
			r.log.Infof("failed to log not ready controlplane pods: %v", err)
		}
		return nil, err
	}

	// Get copy of latest version
	cluster, err := r.clusterLister.Get(clusterName)
	if err != nil {
		return nil, err
	}

	log.Debugf("Control plane became ready after %.2f seconds", time.Since(started).Seconds())
	return cluster.DeepCopy(), nil
}

func (r *testRunner) waitUntilAllPodsAreReady(log *logrus.Entry, client kubernetes.Interface) error {
	log.Debug("Waiting for all pods to be ready...")
	started := time.Now()

	err := wait.Poll(defaultUserClusterPollInterval, defaultTimeout, func() (done bool, err error) {
		podList, err := client.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			log.Warnf("failed to load pod list while waiting until all pods are running: %v", err)
			return false, nil
		}

		for _, pod := range podList.Items {
			if !podIsReady(&pod) {
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
}

func (r *testRunner) getGinkgoRuns(
	log *logrus.Entry,
	scenarioName,
	kubeconfigFilename,
	cloudConfigFilename string,
	cluster *kubermaticv1.Cluster,
	nd *kubermaticapiv1.NodeDeployment,
	dc provider.DatacenterMeta,
) ([]*ginkgoRun, error) {
	kubeconfigFilename = path.Clean(kubeconfigFilename)
	repoRoot := path.Clean(r.repoRoot)
	MajorMinor := fmt.Sprintf("%d.%d", cluster.Spec.Version.Major(), cluster.Spec.Version.Minor())

	runs := []struct {
		name          string
		ginkgoFocus   string
		ginkgoSkip    string
		parallelTests int
	}{
		{
			name:          "parallel",
			ginkgoFocus:   `\[Conformance\]`,
			ginkgoSkip:    `\[Serial\]`,
			parallelTests: int(nd.Spec.Replicas) * 10,
		},
		{
			name:          "serial",
			ginkgoFocus:   `\[Serial\].*\[Conformance\]`,
			ginkgoSkip:    `should not cause race condition when used for configmap`,
			parallelTests: 1,
		},
	}
	versionRoot := path.Join(repoRoot, MajorMinor)
	binRoot := path.Join(versionRoot, "/platforms/linux/amd64")
	var ginkgoRuns []*ginkgoRun
	for _, run := range runs {

		reportsDir := path.Join("/tmp", scenarioName, run.name)
		env := []string{
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
			fmt.Sprintf("--num-nodes=%d", nd.Spec.Replicas),
			fmt.Sprintf("--cloud-config-file=%s", cloudConfigFilename),
		}

		if cluster.Spec.Cloud.AWS != nil {
			args = append(args, "--provider=aws")
			args = append(args, fmt.Sprintf("--gce-zone=%s%s", dc.Spec.AWS.Region, dc.Spec.AWS.ZoneCharacter))
		} else if cluster.Spec.Cloud.Azure != nil {
			args = append(args, "--provider=azure")
		} else if cluster.Spec.Cloud.Openstack != nil {
			args = append(args, "--provider=openstack")
		} else if cluster.Spec.Cloud.VSphere != nil {
			args = append(args, "--provider=vsphere")
		} else {
			args = append(args, "--provider=local")
		}

		if nd.Spec.Template.OperatingSystem.Ubuntu != nil {
			args = append(args, "--node-os-distro=ubuntu")
			env = append(env, "KUBE_SSH_USER=ubuntu")
		} else if nd.Spec.Template.OperatingSystem.CentOS != nil {
			args = append(args, "--node-os-distro=centos")
			env = append(env, "KUBE_SSH_USER=centos")
		} else if nd.Spec.Template.OperatingSystem.ContainerLinux != nil {
			args = append(args, "--node-os-distro=coreos")
			env = append(env, "KUBE_SSH_USER=core")
		}

		cmd := exec.Command(path.Join(binRoot, "ginkgo"), args...)
		cmd.Env = env

		ginkgoRuns = append(ginkgoRuns, &ginkgoRun{
			name:       run.name,
			cmd:        cmd,
			reportsDir: reportsDir,
		})
	}

	return ginkgoRuns, nil
}

func executeGinkgoRun(parentLog *logrus.Entry, run *ginkgoRun, kubeClient kubernetes.Interface) (*ginkgoResult, error) {
	started := time.Now()
	log := parentLog.WithField("reports-dir", run.reportsDir)

	if err := deleteAllNonDefaultNamespaces(log, kubeClient); err != nil {
		return nil, fmt.Errorf("failed to cleanup namespaces before the Ginkgo run: %v", err)
	}

	// We're clearing up the temp dir on every run
	if err := os.RemoveAll(run.reportsDir); err != nil {
		log.Errorf("failed to remove temporary reports directory: %v", err)
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
	log = log.WithField("ginkgo-log", file.Name())

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Copy the command as we cannot execute a command twice
	cmd := &exec.Cmd{
		Path:       run.cmd.Path,
		Args:       run.cmd.Args,
		Env:        run.cmd.Env,
		Dir:        run.cmd.Dir,
		ExtraFiles: run.cmd.ExtraFiles,
	}
	if _, err := writer.Write([]byte(strings.Join(cmd.Args, argSeparator))); err != nil {
		return nil, fmt.Errorf("failed to write command to log: %v", err)
	}

	log.Debugf("Starting Ginkgo run '%s'...", run.name)

	// Flush to disk so we can actually watch logs
	stopCh := make(chan struct{}, 1)
	defer close(stopCh)
	go wait.Until(func() {
		if err := writer.Flush(); err != nil {
			log.Warnf("failed to flush log writer: %v", err)
		}
		if err := file.Sync(); err != nil {
			log.Warnf("failed to sync log file: %v", err)
		}
	}, 1*time.Second, stopCh)

	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Debugf("Ginkgo exited with a non 0 return code: %v", exitErr)
		} else {
			return nil, fmt.Errorf("ginkgo failed to start: %T %v", err, err)
		}
	}

	// When running ginkgo in parallel, each ginkgo worker creates a own report, thus we must combine them
	combinedReport, err := collectReports(run.name, run.reportsDir)
	if err != nil {
		return nil, err
	}

	// If we have no junit files, we cannot return a valid report
	if len(combinedReport.TestCases) == 0 {
		return nil, errors.New("ginkgo report is empty. It seems no tests where executed")
	}

	combinedReport.Time = time.Since(started).Seconds()

	log.Debugf("Ginkgo run '%s' took %s", run.name, time.Since(started))
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
		cluster.Spec.Cloud.Hetzner != nil
}

func supportsLBs(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil
}

func (r *testRunner) logNotReadyControlplanePods(clusterName string) error {
	cluster, err := r.clusterLister.Get(clusterName)
	if err != nil {
		return err
	}
	controlPlanePods, err := r.seedKubeClient.CoreV1().Pods(cluster.Status.NamespaceName).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	notReadyControlPlanePods := sets.NewString()
	for _, pod := range controlPlanePods.Items {
		if !podIsReady(&pod) {
			notReadyControlPlanePods.Insert(pod.Name)
		}
	}
	r.log.Infof("Failed because these controlplane pods didn't get ready: %v", notReadyControlPlanePods.List())
	return nil
}
