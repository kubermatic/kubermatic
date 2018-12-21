package main

import (
	"bufio"
	"bytes"
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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type testScenario interface {
	Name() string
	Cluster(secrets secrets) *kubermaticv1.Cluster
	Nodes(num int) []*kubermaticapiv1.Node
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
		kubeClient:                   opts.kubeClient,
		log:                          opts.log,
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
	kubeClient            kubernetes.Interface
	clusterLister         kubermaticv1lister.ClusterLister
	clusterClientProvider *clusterclient.Provider
	dcs                   map[string]provider.DatacenterMeta
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
	cluster, err := r.setupCluster(log, scenario)
	if err != nil {
		return nil, fmt.Errorf("failed to setup cluster: %v", err)
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

	apiNodes := scenario.Nodes(r.nodeCount)
	if err := r.setupNodes(log, scenario.Name(), cluster, clusterKubeClient, apiNodes, dc); err != nil {
		return nil, fmt.Errorf("failed to setup nodes: %v", err)
	}

	if err := r.waitUntilAllPodsAreReady(log, clusterKubeClient); err != nil {
		return nil, fmt.Errorf("failed to wait until all pods are running after creating the cluster: %v", err)
	}

	report, err := r.testCluster(log, scenario.Name(), cluster, clusterKubeClient, apiNodes, dc, kubeconfigFilename, cloudConfigFilename)
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
	apiNodes []*kubermaticapiv1.Node,
	dc provider.DatacenterMeta,
	kubeconfigFilename string,
	cloudConfigFilename string,
) (*reporters.JUnitTestSuite, error) {
	const maxTestAttempts = 3
	var err error
	totalStart := time.Now()

	report := &reporters.JUnitTestSuite{
		Name: scenarioName,
	}
	// Do a simple PVC test - with retries
	if supportsStorage(cluster) {
		testStart := time.Now()
		testCase := reporters.JUnitTestCase{
			Name:      "[CloudProvider] Test PVC support with the existing StorageClass",
			ClassName: "Kubermatic custom tests",
		}
		err := retryNAttempts(maxTestAttempts, func(attempt int) error { return r.testPVC(log, clusterKubeClient) })
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
		err := retryNAttempts(maxTestAttempts, func(attempt int) error { return r.testLB(log, clusterKubeClient) })
		if err != nil {
			report.Errors++
			testCase.FailureMessage = &reporters.JUnitFailureMessage{Message: err.Error()}
			log.Errorf("Failed to verify that LB's work: %v", err)
		}
		testCase.Time = time.Since(testStart).Seconds()
		report.TestCases = append(report.TestCases, testCase)
		report.Tests++
	}

	var reportsDir string
	var ginkgoReport *reporters.JUnitTestSuite
	err = retryNAttempts(maxTestAttempts, func(attempt int) error {
		startedE2E := time.Now()

		reportsDir = path.Join(r.reportsRoot, scenarioName, fmt.Sprintf("attempt-%d", attempt))
		if err := os.MkdirAll(reportsDir, 0755); err != nil {
			return fmt.Errorf("failed to create reports dir: %v", err)
		}

		log.Infof("[Attempt: %d/%d] Starting E2E tests...", attempt, maxTestAttempts)
		if err := r.runE2E(log, cluster, kubeconfigFilename, cloudConfigFilename, reportsDir, apiNodes, dc); err != nil {
			log.Warnf("[Attempt: %d/%d] failed to run E2E tests: %v", attempt, maxTestAttempts, err)
			return err
		}
		log.Infof("[Attempt: %d/%d] E2E tests finished after %.2f seconds", attempt, maxTestAttempts, time.Since(startedE2E).Seconds())

		ginkgoReport, err = collectReports(scenarioName, reportsDir, time.Since(startedE2E))
		if err != nil {
			log.Warnf("[Attempt: %d/%d] failed to combine reports: %v. Restarting...", attempt, maxTestAttempts, err)
			return err
		}

		if ginkgoReport.Errors > 0 || ginkgoReport.Failures > 0 {
			log.Warnf("[Attempt: %d/%d] e2e tests had some failures (See %s for details). Restarting...", attempt, maxTestAttempts, reportsDir)
			return errors.New("encountered partial errors during ginkgo run")
		}

		return nil
	})
	if err != nil {
		log.Errorf("Failed to successfully run kubernetes e2e tests: %v", err)
	}

	if ginkgoReport != nil {
		report.Time = time.Since(totalStart).Seconds()
		report.Tests += ginkgoReport.Tests
		report.Errors += ginkgoReport.Errors
		report.Failures += ginkgoReport.Failures
		report.TestCases = append(report.TestCases, ginkgoReport.TestCases...)
	}

	b, err := xml.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal combined report file: %v", err)
	}

	if err := ioutil.WriteFile(path.Join(reportsDir, "junit.xml"), b, 0644); err != nil {
		return nil, fmt.Errorf("failed to wrte combined report file: %v", err)
	}

	return report, nil
}

func (r *testRunner) setupNodes(log *logrus.Entry, scenarioName string, cluster *kubermaticv1.Cluster, clusterKubeClient kubernetes.Interface, apiNodes []*kubermaticapiv1.Node, dc provider.DatacenterMeta) error {
	log.Info("Creating machines...")
	kubeMachineClient, err := r.clusterClientProvider.GetMachineClient(cluster)
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

	for i, node := range apiNodes {
		m, err := machine.Machine(cluster, node, dc, keys)
		if err != nil {
			return fmt.Errorf("failed to create Machine from scenario node '%s': %v", node.Name, err)
		}
		// Make sure all nodes have different names across all scenarios - otherwise the Kubelet might not come up (OpenStack has this...)
		m.Name = fmt.Sprintf("%s-machine-%d", scenarioName, i)
		m.Spec.Name = strings.Replace(fmt.Sprintf("%s-node-%d", scenarioName, i), ".", "-", -1)

		err = retryNAttempts(defaultAPIRetries, func(attempt int) error {
			machineLog := log.WithFields(logrus.Fields{"machine": m.Name})
			_, err := kubeMachineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).Create(m)
			if err != nil {
				if kerrors.IsAlreadyExists(err) {
					return nil
				}

				machineLog.Warnf("[Attempt %d/100] Failed to create the machine: %v. Retrying...", attempt, err)
				time.Sleep(defaultUserClusterPollInterval)
				return err
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to create machine '%s' after %d attempts: %v", m.Name, defaultAPIRetries, err)
		}
	}
	log.Infof("Successfully created %d machine(s)!", len(apiNodes))

	if err := r.waitForReadyNodes(log, clusterKubeClient, len(apiNodes)); err != nil {
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
		cm, err := r.kubeClient.CoreV1().ConfigMaps(cluster.Status.NamespaceName).Get(resources.CloudConfigConfigMapName, metav1.GetOptions{})
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

func (r *testRunner) setupCluster(log *logrus.Entry, scenario testScenario) (*kubermaticv1.Cluster, error) {
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

	// We setting higher resource requests to avoid running into issues due to throttling
	cluster.Spec.ComponentsOverride.Apiserver.Resources = &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("250m"),
		},
	}

	// We set the replicas for the control plane to 2 to avoid issues with "bind: address already in use" on control plane components.
	// Somehow this happens for <1% of the pods - though this seems to be a GKE specific issue: https://github.com/kubernetes/kubernetes/issues/69364
	var replicas int32 = 2
	cluster.Spec.ComponentsOverride.Apiserver.Replicas = &replicas
	cluster.Spec.ComponentsOverride.ControllerManager.Replicas = &replicas
	cluster.Spec.ComponentsOverride.Scheduler.Replicas = &replicas

	log.Info("Creating cluster...")
	cluster, err := r.kubermaticClient.KubermaticV1().Clusters().Create(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create the cluster resource: %v", err)
	}
	log.Debug("Successfully created cluster!")

	cluster, err = r.waitForControlPlane(log, cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for control plane to become ready: %v", err)
	}

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

		return cluster.Status.Health.AllHealthy(), nil
	})
	// Timeout or other error
	if err != nil {
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

	podIsReady := func(p *corev1.Pod) bool {
		for _, c := range p.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	}

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

func (r *testRunner) runE2E(
	log *logrus.Entry,
	cluster *kubermaticv1.Cluster,
	kubeconfigFilename,
	cloudConfigFilename,
	reportsDir string,
	nodes []*kubermaticapiv1.Node,
	dc provider.DatacenterMeta,
) error {
	kubeconfigFilename = path.Clean(kubeconfigFilename)
	repoRoot := path.Clean(r.repoRoot)
	MajorMinor := fmt.Sprintf("%d.%d", cluster.Spec.Version.Major(), cluster.Spec.Version.Minor())

	// TODO: Figure out why they fail & potentially fix. Otherwise, explain why they are deactivated
	//brokenTests := []string{
	//	"should set TCP CLOSE_WAIT timeout",                // AWS
	//	"should support exec through an HTTP proxy",        // AWS
	//	"should proxy to cadvisor",                         // AWS
	//	"should handle in-cluster config",                  // AWS
	//	"should proxy to cadvisor using proxy subresource", // AWS
	//	"should support exec through kubectl proxy",        // AWS
	//}

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
			parallelTests: 25,
		},
		{
			name:          "serial",
			ginkgoFocus:   `\[Serial\].*\[Conformance\]`,
			ginkgoSkip:    ``,
			parallelTests: 1,
		},
		// TODO: Enable more e2e tests
		//{
		//	name:          "parallel",
		//	ginkgoFocus:   `.*`,
		//	ginkgoSkip:    `\[Slow\]|\[Serial\]|\[Disruptive\]|\[Flaky\]|\[Feature:.+\]|\[HPA\]|Dashboard|Services.*functioning.*NodePort|` + strings.Join(brokenTests, "|"),
		//	parallelTests: 25,
		//},
		//{
		//	name:          "serial",
		//	ginkgoFocus:   `\[Serial\].*`,
		//	ginkgoSkip:    `\[Slow\]|\[Disruptive\]|\[Flaky\]|\[Feature:.+\]|\[HPA\]|Dashboard|Services.*functioning.*NodePort` + strings.Join(brokenTests, "|"),
		//	parallelTests: 1,
		//},
	}
	versionRoot := path.Join(repoRoot, MajorMinor)
	binRoot := path.Join(versionRoot, "/platforms/linux/amd64")
	for _, run := range runs {
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
			fmt.Sprintf("--num-nodes=%d", len(nodes)),
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

		if nodes[0].Spec.OperatingSystem.Ubuntu != nil {
			args = append(args, "--node-os-distro=ubuntu")
			env = append(env, "KUBE_SSH_USER=ubuntu")
		} else if nodes[0].Spec.OperatingSystem.CentOS != nil {
			args = append(args, "--node-os-distro=centos")
			env = append(env, "KUBE_SSH_USER=centos")
		} else if nodes[0].Spec.OperatingSystem.ContainerLinux != nil {
			args = append(args, "--node-os-distro=coreos")
			env = append(env, "KUBE_SSH_USER=core")
		}

		cmd := exec.Command(path.Join(binRoot, "ginkgo"), args...)
		cmd.Env = env

		run := func() error {
			logFile := path.Join(reportsDir, "e2e.log")
			f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				return fmt.Errorf("failed to open logfile: %v", err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					log.Errorf("failed to close file handle on '%s': %v", logFile, err)
				}
			}()

			writer := bufio.NewWriter(f)
			defer func() {
				if err := writer.Flush(); err != nil {
					log.Errorf("failed to flush log file buffer: %v", err)
				}
			}()

			err = ioutil.WriteFile(
				path.Join(reportsDir, fmt.Sprintf("script-%s.sh", run.name)),
				[]byte(strings.Join(cmd.Args, ` \
    `)),
				0644,
			)
			if err != nil {
				return fmt.Errorf("failed to write script: %v", err)
			}

			log.Debugf("Starting ginkgo run '%s'...", run.name)

			cmd.Stdout = writer
			cmd.Stderr = writer
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to execute ginkgo run '%s': %v. Output can be found at %s", run.name, err, logFile)
			}
			return nil
		}

		if err := run(); err != nil {
			return err
		}
	}

	return nil
}

func supportsStorage(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Openstack != nil ||
		cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil ||
		cluster.Spec.Cloud.VSphere != nil
}

func supportsLBs(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil
}
